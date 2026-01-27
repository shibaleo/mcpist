# MCPist サブスクリプション・使用量制御設計

## 概要

サブスクリプション管理と使用量制御（Rate Limit、Quota、Credit）の詳細設計。

**このドキュメントはオリジン（ビジネスロジック層）の責務をカバーする。**

関連ドキュメント:
- [adr-usage-control-architecture.md](./adr-usage-control-architecture.md) - 使用量制御アーキテクチャに関するADR
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計（**Burst制限はこちら**）
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [dsn-billing.md](./dsn-billing.md) - 課金システム設計

---

## 使用量制御の全体像

### インフラ層 vs ビジネスロジック層

```
┌─────────────────────────────────────────────────────────┐
│                 Cloudflare Worker（インフラ層）           │
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
│  → 詳細は dsn-load-management.md を参照                 │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ↓
┌─────────────────────────────────────────────────────────┐
│           オリジン（ビジネスロジック層）← このドキュメント  │
│                                                          │
│  4. Rate Limit（ユーザー単位、DB）                       │
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
└─────────────────────────────────────────────────────────┘
```

### オリジン側の処理フロー

```
リクエスト（Worker通過後）
    │
    ├─ 1. Rate Limit（DB）
    │     └─ 超過 → 429 rate_limit_exceeded
    │
    ├─ 2. 権限チェック（キャッシュ）
    │     ├─ suspended → 403
    │     ├─ not_subscribed → 403
    │     └─ user_disabled → 403
    │
    ├─ 3. Quotaチェック（DB）
    │     └─ 超過 → 403 quota_exceeded
    │
    ├─ 4. Creditチェック（DB）
    │     └─ 不足 → 403 insufficient_credits
    │
    └─ 5. ツール実行
          │
          └─ 使用量記録 / クレジット消費
```

---

## 1. Rate Limit（ユーザー別・プラン別）

### 目的

分単位での過負荷を防止。プラン別に上限を設定。

### 仕様

| プラン | 上限 |
|--------|------|
| Free | 30 req/min |
| Starter | 60 req/min |
| Pro | 120 req/min |
| Unlimited | 無制限 |

| 項目 | 値 |
|------|-----|
| アルゴリズム | 固定ウィンドウカウンター |
| ウィンドウ | 1分 |
| 保存場所 | ローカルメモリ + Supabase DB（非同期同期） |
| 精度 | 10%誤差許容 |

### 計算方法

```
┌─────────────────────────────────────────────────────────────────┐
│                    Rate Limit 計算フロー                         │
│                                                                  │
│  リクエスト到着                                                   │
│      │                                                           │
│      ▼                                                           │
│  ┌──────────────────────────────────┐                           │
│  │ 1. メモリ内カウント確認           │                           │
│  │    key = userID + 現在の分        │                           │
│  │    例: "user123:2026-01-16T10:05" │                           │
│  └──────────────────────────────────┘                           │
│      │                                                           │
│      ├─ カウント >= 60 ─────────────────▶ 429 返却              │
│      │                                                           │
│      ▼                                                           │
│  ┌──────────────────────────────────┐                           │
│  │ 2. カウント +1 してツール実行     │                           │
│  └──────────────────────────────────┘                           │
│      │                                                           │
│      ▼                                                           │
│  ┌──────────────────────────────────┐                           │
│  │ 3. DBへ非同期書き込み             │  ← fire-and-forget       │
│  │    (goroutine / setTimeout)      │                           │
│  └──────────────────────────────────┘                           │
│      │                                                           │
│      ▼                                                           │
│  ┌──────────────────────────────────┐                           │
│  │ 4. 次の分になったら古いエントリ破棄│                           │
│  └──────────────────────────────────┘                           │
└─────────────────────────────────────────────────────────────────┘
```

**DB同期間隔と理論的誤差**:

| 同期間隔 | 理論的最大誤差 | 説明 |
|----------|---------------|------|
| リアルタイム | 0% | Redis等の共有ストレージ必要 |
| 1分ごと | 最大100%（120 req/min） | 両インスタンスが独立カウント |
| 10分ごと | 最大100%（120 req/min） | 同上、同期頻度は誤差に影響しない |

**なぜ誤差が問題にならないか**:

1. **正常時はPrimaryのみ**: Koyeb健全時はトラフィックが集中、誤差0%
2. **フェイルオーバー時のみ分散**: 一時的に両インスタンスにトラフィックが分散
3. **同一ユーザーの均等分散は稀**: LBはstickyではないが、実際には偏る
4. **Phase 1規模**: 5-10人では両インスタンス均等分散の確率は極めて低い

**Phase 1 設定**: DB同期間隔 = 10分（シンプルさ優先）

### スケール時の誤差分析（1,000 MAU）

```
前提:
- 1,000 MAU
- 平均 10 req/日/ユーザー = 10,000 req/日
- ピーク時（日本時間 20-22時）に60%集中

ピーク時トラフィック:
- 6,000 req / 2時間 = 50 req/min（システム全体）
- 1ユーザーあたり平均 0.05 req/min
```

**シナリオ別の誤差**:

| シナリオ | 発生確率 | 誤差 | 説明 |
|----------|----------|------|------|
| 正常運用（Primary健全） | 99%+ | 0% | Koyebに集中 |
| フェイルオーバー中 | <1% | 最大100%（一時的） | 両インスタンスに分散 |
| 悪意あるユーザーの攻撃 | 極稀 | 最大100% | 意図的な両方へのアクセス |

**なぜ1,000 MAUでも問題にならないか**:

1. 同一ユーザーが60 req/min出すこと自体が稀（平均0.05 req/min）
2. 両インスタンスに均等分散させるには意図的な攻撃が必要
3. 仮に120 req/minになっても、nanoインスタンスの処理能力内

**本当に問題になる条件**（全て揃う必要あり）:

```
1. 同一ユーザーが60+ req/min出す（ヘビーユーザー）
2. フェイルオーバー中 or LBが両方に分散
3. 10分以上継続

→ Phase 1〜2では考慮不要
→ 問題化したらRedis導入を検討
```

**マルチインスタンス共有**:

```
Koyeb (メモリ: count=55)  ──┐
                           ├──▶ Supabase DB (count=55)
Fly.io (メモリ: count=55) ──┘
                               ↑
                               │ 非同期同期（数百ms遅延）
                               │
                        ┌──────┴──────┐
                        │ 起動時にDB  │
                        │ から読み込み │
                        └─────────────┘
```

### 実装

```go
package ratelimit

import (
    "sync"
    "time"
)

type RateLimiter struct {
    mu       sync.Mutex
    windows  map[string]*slidingWindow
}

type slidingWindow struct {
    counts    map[int64]int  // timestamp(秒) → カウント
    lastClean time.Time
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        windows: make(map[string]*slidingWindow),
    }
}

type RateLimitResult struct {
    Allowed   bool
    Limit     int
    Remaining int
    Reset     time.Time
}

func (l *RateLimiter) Check(userID string, limit int) RateLimitResult {
    // limit <= 0 は無制限
    if limit <= 0 {
        return RateLimitResult{
            Allowed:   true,
            Limit:     0,
            Remaining: -1,  // 無制限を示す
            Reset:     time.Time{},
        }
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now()
    windowStart := now.Add(-time.Minute)
    reset := now.Truncate(time.Minute).Add(time.Minute)

    window, exists := l.windows[userID]
    if !exists {
        window = &slidingWindow{
            counts:    make(map[int64]int),
            lastClean: now,
        }
        l.windows[userID] = window
    }

    // 古いカウントを削除
    if now.Sub(window.lastClean) > 10*time.Second {
        for ts := range window.counts {
            if ts < windowStart.Unix() {
                delete(window.counts, ts)
            }
        }
        window.lastClean = now
    }

    // 現在のウィンドウ内のカウント
    var total int
    for ts, count := range window.counts {
        if ts >= windowStart.Unix() {
            total += count
        }
    }

    if total >= limit {
        return RateLimitResult{
            Allowed:   false,
            Limit:     limit,
            Remaining: 0,
            Reset:     reset,
        }
    }

    // カウント追加
    window.counts[now.Unix()]++
    total++

    return RateLimitResult{
        Allowed:   true,
        Limit:     limit,
        Remaining: limit - total,
        Reset:     reset,
    }
}
```

### レスポンス

```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Try again later.",
    "rate_limit": {
      "limit": 30,
      "remaining": 0,
      "reset": 1705398060
    }
  }
}
```

**HTTPステータス**: 429 Too Many Requests

**レスポンスヘッダー**:
```
X-RateLimit-Limit: 30
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705398060
Retry-After: 45
```

---

## 2. Quota

### 目的

月間の利用量を制限。プランごとに上限を設定。

### 仕様

| プラン | 月間上限 |
|--------|----------|
| Free | 100 req/月 |
| Starter | 1,000 req/月 |
| Pro | 10,000 req/月 |
| Unlimited | **無制限** |

**Phase 1**: 全ユーザーを Unlimited に設定（quota_limit = -1）

### DBスキーマ

```sql
-- 使用量テーブル
CREATE TABLE usage (
    user_id UUID REFERENCES users(id),
    month TEXT NOT NULL,           -- '2026-01'
    count INT DEFAULT 0,
    PRIMARY KEY (user_id, month)
);

CREATE INDEX idx_usage_user_month ON usage(user_id, month);
```

### 実装

```go
package quota

import (
    "context"
    "database/sql"
    "time"
)

type QuotaChecker struct {
    db *sql.DB
}

type QuotaResult struct {
    Allowed   bool
    Limit     int   // -1 = 無制限
    Used      int
    Remaining int   // -1 = 無制限
    Reset     string // '2026-02-01'
}

func (c *QuotaChecker) Check(ctx context.Context, userID string, limit int) (*QuotaResult, error) {
    // limit < 0 は無制限
    if limit < 0 {
        return &QuotaResult{
            Allowed:   true,
            Limit:     -1,
            Used:      0,
            Remaining: -1,
        }, nil
    }

    month := time.Now().Format("2006-01")
    nextMonth := time.Now().AddDate(0, 1, 0).Format("2006-01") + "-01"

    var used int
    err := c.db.QueryRowContext(ctx, `
        SELECT COALESCE(count, 0) FROM usage
        WHERE user_id = $1 AND month = $2
    `, userID, month).Scan(&used)

    if err != nil && err != sql.ErrNoRows {
        return nil, err
    }

    remaining := limit - used
    if remaining < 0 {
        remaining = 0
    }

    return &QuotaResult{
        Allowed:   used < limit,
        Limit:     limit,
        Used:      used,
        Remaining: remaining,
        Reset:     nextMonth,
    }, nil
}

func (c *QuotaChecker) Increment(ctx context.Context, userID string) error {
    month := time.Now().Format("2006-01")

    _, err := c.db.ExecContext(ctx, `
        INSERT INTO usage (user_id, month, count)
        VALUES ($1, $2, 1)
        ON CONFLICT (user_id, month)
        DO UPDATE SET count = usage.count + 1
    `, userID, month)

    return err
}
```

### レスポンス

```json
{
  "error": {
    "code": "quota_exceeded",
    "message": "Monthly quota exceeded. Upgrade your plan for more requests.",
    "quota": {
      "limit": 100,
      "used": 100,
      "remaining": 0,
      "reset": "2026-02-01"
    },
    "hint": "Upgrade to Starter plan: https://mcpist.com/billing"
  }
}
```

**HTTPステータス**: 403 Forbidden

---

## 3. Credit

### 目的

従量課金モデル。ツールごとにコストを設定し、クレジット残高から消費。

### 仕様

| 項目 | 値 |
|------|-----|
| 初期クレジット | 0 |
| チャージ単位 | 100クレジット = ¥100 |
| ツールコスト | 1〜10（ツールによる） |

**Phase 1**: 全ユーザーの credit_limit = -1（クレジット制を使用しない）

### DBスキーマ

```sql
-- クレジット残高
CREATE TABLE credits (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    balance INT DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- クレジット履歴（監査用）
CREATE TABLE credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    amount INT NOT NULL,           -- 正=チャージ、負=消費
    reason TEXT NOT NULL,          -- 'charge', 'tool:notion:create_page'
    balance_after INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_credit_tx_user ON credit_transactions(user_id);
CREATE INDEX idx_credit_tx_created ON credit_transactions(created_at);

-- ツールコスト定義
CREATE TABLE tool_costs (
    tool_name TEXT PRIMARY KEY,    -- 'notion:create_page'
    cost INT DEFAULT 1             -- 消費クレジット
);
```

### 実装

```go
package credit

import (
    "context"
    "database/sql"
    "errors"
)

var ErrInsufficientCredits = errors.New("insufficient credits")

type CreditChecker struct {
    db    *sql.DB
    costs map[string]int  // ツール名 → コスト
}

type CreditResult struct {
    Allowed bool
    Balance int   // -1 = クレジット制を使用しない
    Cost    int
}

func (c *CreditChecker) Check(ctx context.Context, userID string, toolName string, useCredit bool) (*CreditResult, error) {
    // クレジット制を使用しない場合
    if !useCredit {
        return &CreditResult{
            Allowed: true,
            Balance: -1,
            Cost:    0,
        }, nil
    }

    cost := c.GetCost(toolName)

    var balance int
    err := c.db.QueryRowContext(ctx, `
        SELECT balance FROM credits WHERE user_id = $1
    `, userID).Scan(&balance)

    if err == sql.ErrNoRows {
        balance = 0
    } else if err != nil {
        return nil, err
    }

    return &CreditResult{
        Allowed: balance >= cost,
        Balance: balance,
        Cost:    cost,
    }, nil
}

func (c *CreditChecker) GetCost(toolName string) int {
    if cost, exists := c.costs[toolName]; exists {
        return cost
    }
    return 1  // デフォルト
}

func (c *CreditChecker) Consume(ctx context.Context, userID string, toolName string) error {
    cost := c.GetCost(toolName)

    tx, err := c.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 残高取得（FOR UPDATE でロック）
    var balance int
    err = tx.QueryRowContext(ctx, `
        SELECT balance FROM credits
        WHERE user_id = $1
        FOR UPDATE
    `, userID).Scan(&balance)

    if err == sql.ErrNoRows {
        // レコードがない場合は作成
        balance = 0
        _, err = tx.ExecContext(ctx, `
            INSERT INTO credits (user_id, balance) VALUES ($1, 0)
        `, userID)
        if err != nil {
            return err
        }
    } else if err != nil {
        return err
    }

    if balance < cost {
        return ErrInsufficientCredits
    }

    newBalance := balance - cost

    // 残高更新
    _, err = tx.ExecContext(ctx, `
        UPDATE credits SET balance = $1, updated_at = NOW()
        WHERE user_id = $2
    `, newBalance, userID)
    if err != nil {
        return err
    }

    // 履歴記録
    _, err = tx.ExecContext(ctx, `
        INSERT INTO credit_transactions (user_id, amount, reason, balance_after)
        VALUES ($1, $2, $3, $4)
    `, userID, -cost, "tool:"+toolName, newBalance)
    if err != nil {
        return err
    }

    return tx.Commit()
}

func (c *CreditChecker) Charge(ctx context.Context, userID string, amount int, reason string) error {
    tx, err := c.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // UPSERT で残高更新
    var newBalance int
    err = tx.QueryRowContext(ctx, `
        INSERT INTO credits (user_id, balance, updated_at)
        VALUES ($1, $2, NOW())
        ON CONFLICT (user_id)
        DO UPDATE SET balance = credits.balance + $2, updated_at = NOW()
        RETURNING balance
    `, userID, amount).Scan(&newBalance)
    if err != nil {
        return err
    }

    // 履歴記録
    _, err = tx.ExecContext(ctx, `
        INSERT INTO credit_transactions (user_id, amount, reason, balance_after)
        VALUES ($1, $2, $3, $4)
    `, userID, amount, reason, newBalance)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

### レスポンス

```json
{
  "error": {
    "code": "insufficient_credits",
    "message": "Not enough credits. Please top up your account.",
    "credit": {
      "balance": 2,
      "cost": 5
    },
    "hint": "Top up credits: https://mcpist.com/billing/credits"
  }
}
```

**HTTPステータス**: 403 Forbidden

---

## プラン定義

### DBスキーマ

```sql
-- プラン定義
CREATE TABLE plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    rate_limit INT DEFAULT 30,       -- req/min（0以下=無制限）
    quota_limit INT DEFAULT -1,      -- req/month（-1=無制限）
    use_credit BOOLEAN DEFAULT FALSE, -- クレジット制を使用するか
    price_monthly INT DEFAULT 0,     -- 月額（円）
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Phase 1 プラン
INSERT INTO plans (id, name, rate_limit, quota_limit, use_credit, price_monthly) VALUES
    ('free', 'Free', 30, 100, FALSE, 0),
    ('starter', 'Starter', 60, 1000, FALSE, 500),
    ('pro', 'Pro', 120, 10000, FALSE, 2000),
    ('unlimited', 'Unlimited', -1, -1, FALSE, 5000),
    ('credit', 'Pay as you go', 120, -1, TRUE, 0);

-- ユーザーのプラン
ALTER TABLE users ADD COLUMN plan_id TEXT REFERENCES plans(id) DEFAULT 'unlimited';
```

### Phase 1 設定

```sql
-- Phase 1: 全ユーザーを unlimited に設定
UPDATE users SET plan_id = 'unlimited';

-- unlimited プランの定義
-- rate_limit = -1 (無制限)
-- quota_limit = -1 (無制限)
-- use_credit = FALSE (クレジット制を使わない)
```

---

## 統合: UsageController

オリジン側の使用量制御を統合するコントローラー。

**注**: Burst制限はWorker側で実行済みのため、オリジンでは扱わない。

```go
package usage

import (
    "context"
    "time"
)

type Controller struct {
    rateLimit *ratelimit.RateLimiter
    quota     *quota.QuotaChecker
    credit    *credit.CreditChecker
    plans     *PlanCache
}

type CheckResult struct {
    Allowed bool
    Reason  string  // rate_limit_exceeded, quota_exceeded, insufficient_credits

    // 各制御の詳細（常に含める）
    RateLimit *RateLimitInfo
    Quota     *QuotaInfo
    Credit    *CreditInfo
}

type RateLimitInfo struct {
    Limit     int
    Remaining int
    Reset     time.Time
}

type QuotaInfo struct {
    Limit     int    // -1 = 無制限
    Used      int
    Remaining int    // -1 = 無制限
    Reset     string
}

type CreditInfo struct {
    Balance int  // -1 = クレジット制を使用しない
    Cost    int
}

func (c *Controller) Check(ctx context.Context, userID, toolName string) (*CheckResult, error) {
    result := &CheckResult{Allowed: true}

    // プラン取得
    plan, err := c.plans.Get(ctx, userID)
    if err != nil {
        return nil, err
    }

    // 1. Rate Limit（ユーザー別・プラン別）
    rl := c.rateLimit.Check(userID, plan.RateLimit)
    result.RateLimit = &RateLimitInfo{
        Limit:     rl.Limit,
        Remaining: rl.Remaining,
        Reset:     rl.Reset,
    }
    if !rl.Allowed {
        result.Allowed = false
        result.Reason = "rate_limit_exceeded"
        return result, nil
    }

    // 2. Quota
    q, err := c.quota.Check(ctx, userID, plan.QuotaLimit)
    if err != nil {
        return nil, err
    }
    result.Quota = &QuotaInfo{
        Limit:     q.Limit,
        Used:      q.Used,
        Remaining: q.Remaining,
        Reset:     q.Reset,
    }
    if !q.Allowed {
        result.Allowed = false
        result.Reason = "quota_exceeded"
        return result, nil
    }

    // 3. Credit
    cr, err := c.credit.Check(ctx, userID, toolName, plan.UseCredit)
    if err != nil {
        return nil, err
    }
    result.Credit = &CreditInfo{
        Balance: cr.Balance,
        Cost:    cr.Cost,
    }
    if !cr.Allowed {
        result.Allowed = false
        result.Reason = "insufficient_credits"
        return result, nil
    }

    return result, nil
}

func (c *Controller) Record(ctx context.Context, userID, toolName string) error {
    plan, err := c.plans.Get(ctx, userID)
    if err != nil {
        return err
    }

    // Quota記録（無制限でも記録しておく：統計用）
    if err := c.quota.Increment(ctx, userID); err != nil {
        return err
    }

    // Credit消費（クレジット制の場合のみ）
    if plan.UseCredit {
        if err := c.credit.Consume(ctx, userID, toolName); err != nil {
            return err
        }
    }

    return nil
}
```

---

## APIレスポンス

### 成功時

```json
{
  "result": "...",
  "usage": {
    "rate_limit": {
      "limit": 30,
      "remaining": 25,
      "reset": 1705398060
    },
    "quota": {
      "limit": -1,
      "used": 150,
      "remaining": -1,
      "reset": "2026-02-01"
    }
  }
}
```

**Phase 1**（Unlimited プラン）では:
- `rate_limit.limit = -1`（無制限なので remaining も -1）
- `quota.limit = -1`, `quota.remaining = -1`
- `credit` フィールドは含まない（use_credit = false）

### エラー時

HTTPステータスと拒否理由の対応:

| 拒否理由 | HTTPステータス |
|----------|---------------|
| burst_exceeded | 429 |
| rate_limit_exceeded | 429 |
| quota_exceeded | 403 |
| insufficient_credits | 403 |

---

## 拒否理由一覧

```go
type DenyReason string

const (
    // 認証・認可（dsn-permission-system.md）
    DenyReasonSuspended           DenyReason = "suspended"
    DenyReasonNotSubscribed       DenyReason = "not_subscribed"
    DenyReasonUserDisabled        DenyReason = "user_disabled"

    // インフラ層（dsn-load-management.md）
    DenyReasonBurstExceeded       DenyReason = "burst_exceeded"

    // ビジネスロジック層（このドキュメント）
    DenyReasonRateLimitExceeded   DenyReason = "rate_limit_exceeded"
    DenyReasonQuotaExceeded       DenyReason = "quota_exceeded"
    DenyReasonInsufficientCredits DenyReason = "insufficient_credits"
)
```

### オリジン側の拒否理由

| 理由 | メッセージ | アクション |
|------|-----------|-----------|
| `rate_limit_exceeded` | レート制限に達しました | Reset時刻まで待つ |
| `quota_exceeded` | 月間上限に達しました | プランアップグレード |
| `insufficient_credits` | クレジット不足 | クレジットチャージ |

**注**: `burst_exceeded` はWorker側で返却。詳細は [dsn-load-management.md](./dsn-load-management.md) を参照。

---

## Phase 1 の状態

### 全体

| 制御 | 実装場所 | 実装 | 有効 | 設定 |
|------|----------|------|------|------|
| Burst | Worker | ○ | ○ | 5 req/s |
| Rate Limit | Origin | ○ | ○ | 60 req/min |
| Quota | Origin | ○ | ○（無制限） | limit = -1 |
| Credit | Origin | ○ | × | use_credit = false |

### オリジン側（このドキュメントのスコープ）

**全ユーザーを `phase1` プランに設定**:
- Rate Limit: 60 req/min
- Quota: -1（無制限）、使用量は記録（統計用）
- Credit: チェックしない

**注**: Burstは Worker 側で 5 req/s を有効化。詳細は [dsn-load-management.md](./dsn-load-management.md) を参照。

---

## 関連ドキュメント

- [adr-usage-control-architecture.md](./adr-usage-control-architecture.md) - 使用量制御アーキテクチャ
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [dsn-billing.md](./dsn-billing.md) - 課金システム設計
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
