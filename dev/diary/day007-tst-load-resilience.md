# MCPist 負荷耐性検証シナリオ

## 概要

MCPistのインフラ・アプリケーション層の負荷耐性を検証するためのテストシナリオ。
意図的にシステムを過負荷状態にし、保護機構の有効性と限界を確認する。

関連ドキュメント:
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
- [dsn-subscription.md](./dsn-subscription.md) - Rate Limit計算方法
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ構成

---

## 検証優先度

| シナリオ | 発生確率 | 影響度 | 優先度 | 対策状況 |
|----------|----------|--------|--------|----------|
| 外部API呼び出しによるリソース枯渇 | 高 | 高 | **P0** | 要対策 |
| KV書き込み枯渇 | 中（最適化で回避可） | 中 | P1 | 最適化案あり |
| 複数ユーザー同時Burst | 中 | 高 | P1 | 部分的 |
| DB接続プール枯渇 | 中 | 高 | P1 | 未検証 |
| フェイルオーバー中の誤差 | 低 | 低 | P3 | 許容 |
| JWT検証CPU負荷 | 低 | 低 | P3 | 対策済み |

---

## P0: 外部API呼び出しによるリソース枯渇

### シナリオ

```
攻撃: 遅い外部APIの連続呼び出しによるgoroutine/メモリ枯渇

発生パターン:
1. AIエージェントがループに入り、同じツールを連続呼び出し
2. 悪意あるユーザーが意図的に遅いAPIを選択して連打
3. 外部APIの応答遅延（障害時）による滞留

前提:
- Notion API等: レスポンス時間 2-3秒
- Burst上限: 5 req/s/user
- 各リクエストがgoroutineを消費

計算（単一ユーザー攻撃）:
- 5 req/s × 3秒/req = 15 goroutine同時滞留
- 60秒間継続: 最大300 req、ピーク15 goroutine

計算（100ユーザー同時攻撃）:
- 100 × 15 = 1,500 goroutine同時滞留
- nanoインスタンス（256MB）で限界に近い
```

### 検証ポイント

- [ ] 遅いツール（Notion, JIRA等）の同時呼び出し数上限
- [ ] goroutine数増加時のメモリ使用量
- [ ] Koyeb nanoの実効限界（何goroutineまで耐えるか）
- [ ] 外部API障害時のカスケード影響

### 対策案

| 対策 | コスト | 効果 | Phase |
|------|--------|------|-------|
| リクエストタイムアウト（30秒） | 0 | 滞留を制限 | Phase 1 |
| ユーザー単位同時実行数制限（50並列） | 低 | 単一ユーザーからの枯渇を防止 | Phase 1 |
| グローバル同時実行数制限（100並列） | 低 | 全体の上限を設定 | Phase 2 |
| 外部API毎のCircuit Breaker | 中 | 障害時のカスケードを防止 | Phase 2 |

### 50並列の根拠

**ボトルネック分析（50並列 × 100ユーザー同時攻撃）**:

| リソース | 計算 | 結果 | 判定 |
|----------|------|------|------|
| goroutine数 | 50 × 100 = 5,000 | 限界10,000+の50% | OK |
| メモリ（goroutine） | 5,000 × 8KB = 40MB | 256MB中 | OK |
| メモリ（HTTPバッファ） | 5,000 × 10KB = 50MB | 合計90MB | OK |
| DB接続 | 5,000同時 vs 60接続 | Phase 1ではDB負荷低い | OK |
| スケジューラ負荷 | 5,000 goroutine | 中程度 | OK |
| GCプレッシャー | 90MB | 低〜中 | OK |

**なぜ50並列か**:
- 限界（10,000 goroutine）の50%で安全マージン確保
- 100並列（10,000 goroutine）は「動くが余裕がない」
- 30並列では「10個のNotion page一括取得」のようなバッチ処理に制約
- 問題発生時は監視で検知 → 30に下げるか、インスタンス増強（課金）で対応

### 対策実装案（Phase 1）

```go
// ユーザー単位同時実行数制限
type UserConcurrencyLimiter struct {
    limits sync.Map // userID -> *semaphore
    max    int
}

func (l *UserConcurrencyLimiter) Acquire(userID string) bool {
    sem, _ := l.limits.LoadOrStore(userID, make(chan struct{}, l.max))
    select {
    case sem.(chan struct{}) <- struct{}{}:
        return true
    default:
        return false // 同時実行数超過
    }
}

func (l *UserConcurrencyLimiter) Release(userID string) {
    if sem, ok := l.limits.Load(userID); ok {
        <-sem.(chan struct{})
    }
}
```

---

## P1: KV書き込み枯渇

### シナリオ

```
攻撃: Cloudflare KV書き込み上限を使い切る

前提:
- Cloudflare KV無料枠: 1,000 write/日
- 現状: Burst制限で毎リクエストKV書き込み

計算（最適化なしの場合）:
- Phase 1（5-10人）: 50-100 write/日 → 余裕
- 1,000 MAU: 10,000 write/日 → 超過

計算（最適化ありの場合）:
- 初回のみ書き込み + メモリキャッシュ
- 1 write/user/second → 最大1,000 write/日
- 1,000 MAUでも無料枠内に収まる
```

### 検証ポイント

- [ ] KV書き込み枯渇時の挙動
- [ ] Burst制限が効かなくなった場合のOrigin負荷
- [ ] エラーハンドリング（KV書き込み失敗時）
- [ ] メモリキャッシュ併用時のWorker間整合性

### 対策案

| 対策 | コスト | 効果 | 推奨 |
|------|--------|------|------|
| メモリキャッシュ併用（初回のみKV書き込み） | 0 | 10倍削減 | **Phase 1推奨** |
| 有料プラン移行（$5/月で100万write） | $5/月 | 根本解決 | 1,000 MAU超過時 |

### 対策実装案（メモリキャッシュ併用）

```typescript
// 改善: 初回のみ書き込み、以降はメモリで管理
const memoryCache = new Map<string, {count: number, windowStart: number}>();

async function checkBurst(kv: KVNamespace, userID: string): Promise<BurstResult> {
  const now = Math.floor(Date.now() / 1000);
  const key = `burst:${userID}`;

  let data = memoryCache.get(key);

  if (!data || now - data.windowStart >= 1) {
    // 新しいウィンドウ: KVに書き込み（1回/秒/ユーザー）
    data = { count: 1, windowStart: now };
    memoryCache.set(key, data);
    await kv.put(key, JSON.stringify(data), { expirationTtl: 60 });
    return { allowed: true, retryAfter: 0 };
  }

  // 同一ウィンドウ内: メモリのみ更新（KV書き込みなし）
  if (data.count >= 5) {
    return { allowed: false, retryAfter: 1 - (now - data.windowStart) };
  }

  data.count++;
  memoryCache.set(key, data);
  // KV書き込みなし

  return { allowed: true, retryAfter: 0 };
}
```

### 書き込み数比較

| シナリオ | 最適化なし | 最適化あり |
|----------|------------|------------|
| Phase 1（5-10人） | 50-100/日 | 5-10/日 |
| 100 MAU | 1,000/日 | 100/日 |
| 1,000 MAU | 10,000/日 | 1,000/日 |

**結論**: メモリキャッシュ併用で1,000 MAUまで無料枠内で運用可能

---

## P1: 複数ユーザー同時Burst

### シナリオ

```
攻撃: 複数ユーザーによる同時Burst

方法:
- 100アカウント作成
- 各アカウントで5 req/s（Burst上限）を同時発行
- 合計: 500 req/s がOriginに到達

検証ポイント:
- nanoインスタンスが500 req/sを処理できるか
- DB接続プールが枯渇しないか
- レスポンスタイムの劣化
```

### 検証ポイント

- [ ] 同時100ユーザー × 5 req/s のOrigin負荷
- [ ] Koyeb nano のCPU/メモリ使用率
- [ ] レスポンスタイム（p50, p95, p99）
- [ ] エラー率

### 対策の有無

| 保護機構 | 効果 |
|----------|------|
| IP単位Rate Limit（1000 req/min） | 同一IPからは効く、分散IPは防げない |
| Burst制限（5 req/s/user） | ユーザー単位なので複数ユーザーは防げない |

### テスト方法

```bash
# k6 または vegeta で負荷テスト
# 100ユーザー × 5 req/s = 500 req/s を1分間

k6 run --vus 100 --duration 60s load-test.js
```

---

## P1: DB接続プール枯渇

### シナリオ

```
攻撃: 長時間トランザクションの蓄積

方法:
- Quota/Credit更新でDBトランザクションを発生
- 同時に多数のリクエストを送信
- 接続プールが枯渇

検証ポイント:
- Supabase無料枠の接続数上限
- コネクションプールの設定
- タイムアウト設定
```

### 検証ポイント

- [ ] Supabase接続数上限（無料枠: 60接続）
- [ ] 接続枯渇時のエラーメッセージ
- [ ] 復旧までの時間

### 対策の有無

| 保護機構 | 効果 |
|----------|------|
| Rate Limit（60 req/min/user） | 単一ユーザーからは保護 |
| Burst（5 req/s/user） | 瞬間的なスパイクは保護 |
| 複数ユーザー同時 | 防げない |

### テスト方法

```bash
# 同時接続数を増やしながらDB応答を監視
# Supabase Dashboard で接続数を確認
```

---

## P3: フェイルオーバー中のRate Limit誤差

### シナリオ

```
攻撃: フェイルオーバー中の両インスタンスアクセス

方法:
1. Koyebのヘルスチェックを意図的に失敗させる（内部テスト用）
2. LBが両インスタンスに分散し始める
3. 単一ユーザーが60 req/minを両方に送る
4. 合計120 req/minが通過
```

### 検証ポイント

- [ ] フェイルオーバー中のRate Limit誤差（実測値）
- [ ] 10分DB同期で実際に何%誤差が出るか
- [ ] 監視アラートで検知できるか

### 対策の有無

- 現状は「許容」の判断
- 最大100%誤差（120 req/min）でもサービス影響は軽微
- 監視で検知し、必要に応じてRedis導入

---

## P3: JWT検証CPU負荷

### シナリオ

```
攻撃: 大量の無効JWT

方法:
- 署名が無効なJWTを大量生成
- Worker側で検証処理が走る
- CPU時間10ms/リクエストの上限に当たるか
```

### 検証ポイント

- [ ] Worker CPU使用量
- [ ] JWKSキャッシュが効いているか
- [ ] 無料枠（10万req/日）を超えるか

### 対策の有無

| 保護機構 | 効果 |
|----------|------|
| IP Rate Limit | 同一IPからの大量リクエストを遮断 |
| Worker CPU制限 | 10ms/req で自動的に制限 |
| JWKSキャッシュ | 検証コストを削減 |

**結論**: 無効JWTはWorkerで止まり、Originは保護される

---

## テスト実行計画

### Phase 1（MVP前）

| テスト | 優先度 | 実施時期 |
|--------|--------|----------|
| 単一ユーザーBurst検証 | P1 | MVP前 |
| Rate Limit動作確認 | P1 | MVP前 |
| ヘルスチェック動作確認 | P1 | MVP前 |

### Phase 2（1,000 MAU前）

| テスト | 優先度 | 実施時期 |
|--------|--------|----------|
| 外部API呼び出しによるリソース枯渇検証 | P0 | 100 MAU到達時 |
| KV書き込み枯渇シミュレーション | P1 | 500 MAU到達時 |
| 複数ユーザー同時Burst | P1 | 500 MAU到達時 |
| DB接続プール負荷テスト | P1 | 500 MAU到達時 |

### 実施しない（Phase 1-2）

- 大規模DDoSシミュレーション
- マルチリージョン負荷テスト
- 長期間（24時間+）負荷テスト

---

## 関連ドキュメント

- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
- [dsn-subscription.md](./dsn-subscription.md) - Rate Limit計算方法
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ構成
