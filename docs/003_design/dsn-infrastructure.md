# MCPist インフラストラクチャ設計

## 概要

MCPサーバーのインフラ構成と運用戦略。放置運用を前提とした高可用性設計。

関連ドキュメント:
- [adr-rate-limit-architecture.md](./adr-rate-limit-architecture.md) - Rate Limitアーキテクチャに関するADR
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計（Rate Limit、監視、ヘルスチェック）
- [dsn-deployment.md](./dsn-deployment.md) - デプロイ戦略（CI/CD、シークレット管理）
- [dsn-billing.md](./dsn-billing.md) - 課金システム設計
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [adr-b2c-focus.md](./adr-b2c-focus.md) - B2Cフォーカスに関するADR

---

## 設計原則

### 放置運用

- 副業プロジェクトのため、平日昼間（本業中）に対応できない
- 障害発生時に自動でフェイルオーバー
- 人的介入を最小化

### ベンダー分散

- 単一ベンダーに依存しない
- 「卵を複数のカゴに」

---

## インフラ構成

### 全体図

```
┌─────────────────────────────────────────────────────────┐
│                     Cloudflare                          │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Worker (API Gateway)                │   │
│  │                                                  │   │
│  │  1. JWT署名検証（登録ユーザーか）                │   │
│  │  2. Rate Limit（KVカウンター）                  │   │
│  │  3. サブスクリプション確認（KV）                │   │
│  │  4. 有効 → オリジンに転送                       │   │
│  └──────────────────────┬──────────────────────────┘   │
│                         │                               │
│              (DNS + ヘルスチェック + LB)                │
│                         │                               │
│              ┌──────────┴──────────┐                    │
│              ↓                     ↓                    │
│         ┌─────────┐           ┌─────────┐              │
│         │  Koyeb  │           │ Fly.io  │              │
│         │(Primary)│           │(Standby)│              │
│         └────┬────┘           └────┬────┘              │
│              │                     │                    │
│              └──────────┬──────────┘                    │
│                         ↓                               │
│                   Supabase DB                           │
│                   (共通DB)                              │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                    課金処理（独立）                      │
│                                                         │
│  Stripe Checkout → Webhook → Supabase Edge Function    │
│                                    │                    │
│                         ┌──────────┴──────────┐        │
│                         ↓                     ↓        │
│                    DB更新              Cloudflare KV   │
│                                        (キャッシュ更新) │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                    監視（外部）                          │
│                                                         │
│  Grafana Cloud (Prometheus + Loki + アラート)          │
│       ↑                                                 │
│       │ /metrics, ログ                                  │
│       │                                                 │
│  MCPサーバー (Koyeb / Fly.io)                          │
└─────────────────────────────────────────────────────────┘
```

### コンポーネント

| コンポーネント | サービス | 役割 |
|---------------|----------|------|
| API Gateway | Cloudflare Worker | JWT検証、Rate Limit、課金チェック |
| KVキャッシュ | Cloudflare KV | サブスクリプション状態、Rate Limitカウンター |
| DNS/LB | Cloudflare | ヘルスチェック、フェイルオーバー |
| Primary | Koyeb nano | Primary MCPサーバー |
| Standby | Fly.io | ホットスタンバイ MCPサーバー |
| DB | Supabase | 共通データベース |
| 課金 | Stripe + Supabase Edge | 課金処理（→ [dsn-billing.md](./dsn-billing.md)） |
| 監視 | Grafana Cloud | メトリクス、ログ、アラート |
| UI | Vercel | 管理UI、ユーザーUI |

---

## API Gateway（Cloudflare Worker）

### 目的

- **未登録ユーザーの遮断**: MCPサーバーに到達する前にエッジで拒否
- **DDoS対策**: 無効なリクエストをCloudflareで吸収
- **負荷軽減**: オリジンサーバーへの不要なトラフィックを削減
- **Rate Limiting**: CPU過負荷を防止（Worker側で制御）
- **課金チェック**: サブスクリプション状態をKVから確認

### 処理フロー

```
MCPクライアント
    │
    │ Authorization: Bearer <JWT>
    ↓
Cloudflare Worker
    │
    ├─ 1. Authorizationヘッダーなし → 401 Unauthorized
    │
    ├─ 2. JWT署名検証
    │     └─ 無効/期限切れ → 403 Forbidden
    │
    ├─ 3. Rate Limit チェック（Cloudflare KV）
    │     └─ 超過 → 429 Too Many Requests
    │
    ├─ 4. サブスクリプション確認（Cloudflare KV）
    │     └─ 無効 → レート制限を厳格化
    │
    └─ 5. 有効 → オリジン（Koyeb/Fly.io）に転送
          ↓
    MCPサーバー
        └─ ツール権限チェック（PermissionGate）
```

### JWT検証の判定基準

```
有効なJWT = Supabase Authに登録済み = MCPistの登録ユーザー

検証項目:
1. 署名が正しいか（JWKSで検証）
2. 有効期限（exp）が切れていないか
3. 発行者（iss）が正しいか
```

詳細実装は [dsn-load-management.md](./dsn-load-management.md) を参照。

### MCPサーバー側の変更

Workerで検証済みなので、MCPサーバーは`X-User-ID`ヘッダーを信頼できる。

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Workerで検証済み、X-User-IDを信頼
        userID := r.Header.Get("X-User-ID")
        if userID == "" {
            // 直接アクセス（Worker経由でない）は拒否
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }

        ctx := context.WithValue(r.Context(), "user_id", userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### セキュリティ考慮

**オリジン直接アクセスの防止**:

```
Cloudflare経由でない直接アクセスを防ぐ

方法:
1. オリジンはCloudflareのIPのみ許可
2. または、Worker→オリジン間で共有シークレットを検証
```

```go
// MCPサーバー側: シークレット検証
func GatewayMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ヘルスチェックはバイパス
        if r.URL.Path == "/health" {
            next.ServeHTTP(w, r)
            return
        }

        secret := r.Header.Get("X-Gateway-Secret")
        if secret != os.Getenv("GATEWAY_SECRET") {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 無料枠

| リソース | 無料枠 |
|----------|--------|
| Worker リクエスト | 10万/日 |
| Worker CPU時間 | 10ms/リクエスト |
| KV 読み取り | 10万/日 |
| KV 書き込み | 1,000/日 |

Phase 1の規模では十分。

---

## コスト試算

### Phase 1（5-10人）

| サービス | プラン | コスト |
|----------|--------|--------|
| Koyeb | nano（永年無料） | $0 |
| Fly.io | 無料枠（3 VM） | $0 |
| Supabase | 無料枠 | $0 |
| Cloudflare | 無料枠 | $0 |
| Grafana Cloud | 無料枠 | $0 |
| Vercel | 無料枠 | $0 |
| Stripe | 従量課金 | 決済額の3.6% |

**月額固定費: $0**（Stripe手数料のみ）

### Fly.io 無料枠

| リソース | 無料枠 |
|----------|--------|
| 共有CPU VM | 最大3台 |
| メモリ | 256MB/VM |
| 帯域 | 160GB/月 |

Phase 1の規模（5-10人）なら十分。

### 障害時のコスト

```
Koyeb障害 → Fly.ioにトラフィック集中
    │
    │ 仮に1週間Fly.ioのみで運用
    ↓
無料枠内で収まる可能性が高い
（超えても数ドル/月）
```

**「いざというときの課金はやむを得ない」**

---

## Cloudflare障害時のフォールバック

### 選択肢

| 方法 | 複雑度 | 切替時間 | 備考 |
|------|--------|----------|------|
| DNSを直接変更 | 低 | 数分〜数時間 | TTL依存 |
| 別のDNSプロバイダに切替 | 中 | 数時間 | 事前設定が必要 |
| クライアント側で直接接続 | 低 | 即時 | ユーザー手動対応 |

### 推奨対応

```
1. 即時対応（ユーザー向けアナウンス）
   - Koyeb直接URL: mcpist.koyeb.app
   - Fly.io直接URL: mcpist.fly.dev

   MCPクライアント設定を一時的に直接URLに変更

2. 短期対応（DNS切替）
   - Cloudflare以外のDNSに切替
   - 事前にRoute 53等でバックアップゾーン準備（オプション）

3. 長期対応
   - Cloudflare復旧を待つ
   - 世界規模のCloudflare障害は稀（年1-2回程度、数時間で復旧）
```

### Phase 1 方針

- Cloudflare障害時は**直接URL案内**で対応
- 追加インフラは不要（費用・複雑度の観点）
- 「数時間の障害は許容」（放置運用の原則）

---

## Phase 1 スコープ

### 実装する

- [x] Koyeb デプロイ（既存）
- [ ] Fly.io デプロイ
- [ ] Cloudflare DNS設定
- [ ] Cloudflare Worker（API Gateway）
- [ ] Cloudflare KV設定
- [ ] Cloudflare Terraform化
- [ ] ヘルスチェックエンドポイント
- [ ] Grafana Cloud設定

詳細:
- Rate Limit、監視 → [dsn-load-management.md](./dsn-load-management.md)
- CI/CD、シークレット管理 → [dsn-deployment.md](./dsn-deployment.md)
- 課金 → [dsn-billing.md](./dsn-billing.md)

### 実装しない

- Staging環境
- 自動スケーリング
- マルチリージョン
- バックアップDNSプロバイダ

---

## 関連ドキュメント

- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
- [dsn-deployment.md](./dsn-deployment.md) - デプロイ戦略
- [dsn-billing.md](./dsn-billing.md) - 課金システム設計
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [dsn-cache-share.md](./dsn-cache-share.md) - キャッシュ共有設計
- [dsn-tool-sieve.md](./dsn-tool-sieve.md) - 認証・認可アーキテクチャ
