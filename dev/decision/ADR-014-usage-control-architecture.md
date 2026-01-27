# ADR: 使用量制御アーキテクチャ

## ステータス

**承認済み** - 2026-01-16

## コンテキスト

MCPサーバーにおける使用量制御（Burst、Rate Limit、Quota、Credit）の実装方針を決定する必要がある。

### 背景

1. **MCPはAPIサーバー**: 認証さえあれば自動アクセス可能
2. **AIによる自動化リスク**: ユーザーが意図しないプログラムを実行する可能性
3. **課金モデルが未確定**: サービス単位月額固定、クオータ制、クレジット制のいずれも選択肢
4. **UX設計の一貫性**: 後から課金モデルを変更するとUXが変わり、改修が困難

### 使用量制御の種類

| 制御 | 目的 | 時間軸 |
|------|------|--------|
| **Burst** | 瞬間的スパイク防止 | 秒単位 |
| **Rate Limit** | 分単位の過負荷防止 | 分単位 |
| **Quota** | 月間利用量制限 | 月単位 |
| **Credit** | 従量課金の消費管理 | 累積 |

## 決定

### 1. 全ての使用量制御をPhase 1から実装する

```
Phase 1で実装するもの:
  - Burst制限（3-5 req/s）
  - Rate Limit（ユーザー単位、課金状態別）
  - Quota（月間使用量）
  - Credit（従量課金）

Phase 1で有効化するもの:
  - Burst制限: 有効
  - Rate Limit: 有効
  - Quota: 無効（limit=0で無制限）
  - Credit: 無効（limit=0で未使用）
```

### 2. インフラ保護とビジネスロジックを分離する

```
Cloudflare Worker（インフラ保護層）
  │
  │ - JWT署名検証
  │ - グローバルRate Limit（IP単位、DDoS対策）
  │ - Burst制限（ユーザー単位、5 req/s）
  │
  ↓
オリジン（ビジネスロジック層）
  │
  │ - Rate Limit（ユーザー単位、プラン別）
  │ - Quota（月間使用量）
  │ - Credit（従量課金）
  │ - 権限チェック（PermissionGate）
  │
  ↓
ツール実行
```

### 3. Burstはインフラ層（Worker）に配置する

```
理由:
  - 瞬間的スパイクはオリジンに到達する前に止めるべき
  - オリジンのリソースを保護
  - JWT検証後にuser_idが分かるためユーザー単位制御が可能
  - KVの結果整合性は秒単位の制御では許容範囲
```

### 4. UXを最初から全機能前提で設計する

```
理由:
  - 後から課金モデルを変更するとUXが変わる
  - UXが変わると改修範囲の特定が困難
  - 「機能は実装済み、UIに反映しないだけ」が最も安全
```

## 理由

### 1. 自動アクセスのリスク

MCPはAPIサーバーであり、認証さえあれば自動アクセス可能。

```
リスクシナリオ:
  - ユーザーがAIにプログラムを生成させ、意図せず大量アクセス
  - 自動化ツールの暴走
  - 悪意あるユーザーによるリソース枯渇攻撃

対策:
  - Burst: 瞬間的なスパイクを防止
  - Rate Limit: 継続的な過負荷を防止
  - Quota/Credit: ビジネス的な上限を設定
```

### 2. 課金モデルの柔軟性

Phase 1の段階で課金モデルを確定する必要がない。

```
選択肢:
  A. サービス単位月額固定（例: Notion ¥500/月）
  B. クオータ制（例: 月間1000リクエストまで）
  C. クレジット制（例: 1リクエスト = 1クレジット消費）
  D. ハイブリッド（基本料金 + 従量課金）

Phase 1では:
  - 全ユーザーを「無制限」プランに設定
  - 実際の利用状況を観察
  - Phase 2で最適な課金モデルを選択
```

### 3. UXの一貫性

後から機能を追加するとUXが変わり、改修が困難になる。

```
問題:
  - APIレスポンスに新しいフィールドが追加される
  - UIに新しい表示が必要になる
  - エラーメッセージが変わる
  - ユーザーフローが変わる

解決:
  - 最初から全てのフィールドをAPIに含める
  - UIコンポーネントも最初から用意（非表示でもOK）
  - エラーメッセージも最初から定義
```

### 4. 実装コストの比較

| 方針 | 初期コスト | 追加コスト | リスク |
|------|-----------|-----------|--------|
| 必要になったら実装 | 低 | **高** | UX変更による影響大 |
| 最初から全て実装 | 中 | **低** | 使わない機能がある |

**「最初から全て実装」を採用**。使わない機能があっても、後から追加するコストより低い。

## 検討した代替案

### 案A: 必要になったら実装（却下）

```
Phase 1: Rate Limitのみ
Phase 2: Quotaを追加
Phase 3: Creditを追加

却下理由:
  - 各フェーズでAPIスキーマが変わる
  - UIの改修が必要
  - テストの再実行が必要
  - UX設計のやり直しが発生
```

### 案B: インターフェースだけ定義（却下）

```
Phase 1: インターフェース定義、実装はNoOp
Phase 2以降: 実装を追加

却下理由:
  - 実装がないとテストできない
  - 実装時に設計の問題が発覚する可能性
  - 「動くコード」がない状態は危険
```

### 案C: 最初から全て実装（採用）

```
Phase 1: 全て実装、一部は無効化
Phase 2以降: 設定変更のみで有効化

採用理由:
  - UXが変わらない
  - テスト可能
  - 設定変更のみで有効化できる
```

## 結果

### 使用量制御の全体フロー

```
┌─────────────────────────────────────────────────────────┐
│                 Cloudflare Worker                        │
│                                                          │
│  1. JWT署名検証                                         │
│       └─ 無効 → 401/403                                 │
│                                                          │
│  2. グローバルRate Limit（IP単位）                       │
│       └─ 超過 → 429                                     │
│                                                          │
│  3. Burst制限（ユーザー単位、KV）                       │
│       └─ 超過 → 429 burst_exceeded                      │
│                                                          │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ↓
┌─────────────────────────────────────────────────────────┐
│                 オリジン（MCPサーバー）                   │
│                                                          │
│  4. Rate Limit（ユーザー単位、メモリ）                   │
│       └─ 超過 → 429 rate_limit_exceeded                 │
│                                                          │
│  5. 権限チェック（キャッシュ）                           │
│       ├─ suspended → 403                                │
│       ├─ not_subscribed → 403                           │
│       └─ user_disabled → 403                            │
│                                                          │
│  6. Quotaチェック（DB）                                  │
│       └─ 超過 → 403 quota_exceeded                      │
│                                                          │
│  7. Creditチェック（DB）                                 │
│       └─ 不足 → 403 insufficient_credits                │
│                                                          │
│  8. ツール実行 → 使用量記録 / クレジット消費            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 各制御の実装場所

| 制御 | 実装場所 | 保存場所 | 理由 |
|------|----------|----------|------|
| Burst | Worker | KV | 瞬間スパイクはオリジン到達前に止める |
| Rate Limit（グローバル） | Worker | KV | DDoS対策、IP単位 |
| Rate Limit（ユーザー別） | Origin | メモリ | プラン別、ビジネスロジック |
| Quota | Origin | DB | 月間使用量、ビジネスロジック |
| Credit | Origin | DB | 従量課金、トランザクション必須 |

### APIレスポンスの拡張

```json
{
  "result": "...",
  "usage": {
    "burst": {
      "limit": 5,
      "remaining": 4
    },
    "rate_limit": {
      "limit": 30,
      "remaining": 25,
      "reset": 1705398000
    },
    "quota": {
      "limit": 1000,
      "used": 150,
      "remaining": 850,
      "reset": "2026-02-01"
    },
    "credit": {
      "balance": 500,
      "cost": 1
    }
  }
}
```

**Phase 1では**: `quota.limit=-1`（無制限）、`credit`は含まない（use_credit=false）。

### 拒否理由の一覧

```go
type DenyReason string

const (
    // 認証・認可
    DenyReasonSuspended           DenyReason = "suspended"
    DenyReasonNotSubscribed       DenyReason = "not_subscribed"
    DenyReasonUserDisabled        DenyReason = "user_disabled"

    // 使用量制御
    DenyReasonBurstExceeded       DenyReason = "burst_exceeded"
    DenyReasonRateLimitExceeded   DenyReason = "rate_limit_exceeded"
    DenyReasonQuotaExceeded       DenyReason = "quota_exceeded"
    DenyReasonInsufficientCredits DenyReason = "insufficient_credits"
)
```

### Phase 1の設定

```sql
-- 全ユーザーを無制限プランに設定
INSERT INTO plans (id, name, rate_limit, quota_limit, use_credit) VALUES
    ('unlimited', 'Unlimited', -1, -1, FALSE);  -- -1 = 無制限

-- Burst: Worker側で有効（5 req/s）
-- Rate Limit: -1 で無制限（チェックはするが常に許可）
-- Quota: -1 で無制限（使用量は記録、統計用）
-- Credit: use_credit=FALSE でスキップ
```

### 影響

**dsn-load-management.md に追加**:
- Burst制限の実装（Worker側）
- Workerとオリジンの責務分離の明確化

**dsn-subscription.md に追加**:
- Rate Limitの実装（オリジン側）
- Quota/Creditのチェックと消費ロジック
- プラン定義

**dsn-permission-system.md に追加**:
- 拒否理由の拡張

## 関連ドキュメント

- [adr-rate-limit-architecture.md](./adr-rate-limit-architecture.md) - Rate Limitのインフラ/ビジネス分離
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計（Burst含む）
- [dsn-subscription.md](./dsn-subscription.md) - サブスクリプション・使用量制御設計
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [dsn-billing.md](./dsn-billing.md) - 課金システム設計
