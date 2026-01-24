# ADR: Rate Limit アーキテクチャ

## ステータス

**承認済み** - 2026-01-16

## コンテキスト

Rate Limitの実装場所について検討した。

当初の設計では、Cloudflare Worker側でKVを使った課金状態別Rate Limitを実装していた。しかし、これが一般的なSaaSアーキテクチャに沿っているか、また適切な責務分離になっているかを再検討した。

### Phase 1の位置づけ

Phase 1は「友人5-10人」という規模だが、**これはテストユーザーである**。

重要な視点：
- 友人は信頼できるが、それは「テスト」に協力してくれる人
- 最終的には**想定しない匿名ユーザー**が利用可能になる
- その時点でどんなことが起きるかは予測不可能
- **過剰でもリスクは徹底的に考慮する必要がある**

## 決定

### 1. インフラリスクとビジネスリスクを分離する

| リスク種別 | 対処層 | 例 |
|-----------|--------|-----|
| **インフラリスク** | CDN/LB層（Worker） | DDoS、未認証アクセス、IP単位の異常 |
| **ビジネスリスク** | オリジン | 認証済みユーザーの異常利用、課金回避 |

**理由**：
- 責務が明確になる
- 各層で適切なデータにアクセスできる
- デバッグ・監視が容易

### 2. ビジネスロジックはオリジンに集約する

```
Worker（CDN/LB層）
  │
  │ 責務: インフラ保護
  │ - JWT署名検証（未登録ユーザー遮断）
  │ - グローバルRate Limit（全ユーザー共通、DDoS対策）
  │ - IP単位の異常検知
  │
  ↓
オリジン（MCPサーバー）
  │
  │ 責務: ビジネスロジック
  │ - ユーザー単位Rate Limit（課金状態別）
  │ - 権限チェック（PermissionGate）
  │ - 課金状態確認
  │ - アカウント状態確認
  │
  ↓
DB（Supabase）
```

### 3. Worker側のKV課金チェックは廃止

当初設計：
```
Worker: JWT + KV参照 + 課金別Rate Limit
    ↓
オリジン: 権限チェック
```

新設計：
```
Worker: JWT + グローバルRate Limit（インフラ保護のみ）
    ↓
オリジン: 課金別Rate Limit + 権限チェック（ビジネスロジック集約）
```

**廃止するもの**：
- Cloudflare KV SUBSCRIPTIONS namespace
- Worker側の課金状態チェック
- Stripe Webhook → KV更新フロー

**維持するもの**：
- Cloudflare KV RATE_LIMITS namespace（グローバルRate Limit用）

## 理由

### 1. 一般的なSaaSアーキテクチャとの整合性

多くのSaaSサービスでは：

| 層 | Rate Limit種別 | 目的 |
|---|---|---|
| CDN/LB層 | グローバル制限、IP制限 | DDoS防御、インフラ保護 |
| オリジン | ユーザー単位、課金状態別 | ビジネスロジック制御 |

MCPistもこのパターンに従う。

### 2. CDN/LB層の制約

Cloudflare Worker（CDN/LB層）の特性：
- DBへの直接アクセスがない（KVは結果整合性）
- ビジネスロジックを持たせると複雑化
- 認証済みユーザー情報へのアクセスが限定的

オリジンの特性：
- 認証後のユーザー情報が完全に揃っている
- DBアクセスが容易（キャッシュも含め）
- ビジネスロジックの一元管理が可能

### 3. KV同期問題の解消

当初設計の問題：
```
Stripe Webhook → DB更新 → KV更新

KVが古い場合:
  → 課金したのにfreeレートで制限される
  → ユーザー体験の低下
```

新設計：
```
Stripe Webhook → DB更新

オリジンは常にDB（Source of Truth）を参照
  → 同期問題なし
```

### 4. 認証済みユーザーからの異常アクセス

「認証済みユーザーからの大量アクセス」は**ビジネスリスク**であり、オリジンで対処すべき。

```
認証済みユーザーの異常アクセス
    │
    ├─ 悪意がある場合
    │     → Rate Limitで一時的に制限
    │     → 継続する場合はアカウント停止（suspend）
    │
    ├─ 自動化ツールの暴走
    │     → Rate Limitでオリジン保護
    │     → ログで検知、ユーザーに連絡
    │
    └─ 正当な利用で大量アクセス
          → 課金プランのアップグレードを促す
```

### 5. 匿名ユーザーへの対応

Phase 1後、一般公開された際のリスク：

| リスク | 対処 | 層 |
|--------|------|-----|
| 未認証DDoS | グローバルRate Limit | Worker |
| JWT偽造攻撃 | JWT署名検証 | Worker |
| 認証済みユーザーの異常利用 | ユーザー単位Rate Limit | オリジン |
| アカウント共有（JWT流出） | ユーザー単位Rate Limit + 監視 | オリジン |
| 課金回避の試み | 権限チェック（DB参照） | オリジン |

**想定しない匿名ユーザーのリスクは全て考慮**し、適切な層で対処する。

## 検討した代替案

### 案A: Worker側で全てのRate Limit（却下）

```
Worker: JWT + KV課金チェック + 課金別Rate Limit
オリジン: 権限チェックのみ
```

**却下理由**：
- KVとDBの同期問題
- ビジネスロジックがWorkerに漏れる
- 一般的なパターンから逸脱

### 案B: オリジン側で全てのRate Limit（却下）

```
Worker: JWTのみ
オリジン: 全Rate Limit + 権限チェック
```

**却下理由**：
- 未認証DDoSがオリジンに到達
- インフラ保護が不十分

### 案C: 層ごとに責務分離（採用）

```
Worker: JWT + グローバルRate Limit（インフラ保護）
オリジン: 課金別Rate Limit + 権限チェック（ビジネスロジック）
```

**採用理由**：
- 責務が明確
- 一般的なSaaSパターン
- 同期問題なし

## 結果

### 新しいアーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                 Cloudflare Worker                        │
│                                                          │
│  責務: インフラ保護                                      │
│                                                          │
│  1. JWT署名検証（未登録ユーザー遮断）                   │
│  2. グローバルRate Limit（全ユーザー共通）              │
│     - 1000 req/min/IP（DDoS対策）                       │
│  3. 有効 → オリジンに転送                               │
│                                                          │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ↓
┌─────────────────────────────────────────────────────────┐
│                 オリジン（MCPサーバー）                   │
│                                                          │
│  責務: ビジネスロジック                                  │
│                                                          │
│  1. ユーザー単位Rate Limit（課金状態別）                │
│     - Free: 30 req/min                                   │
│     - Paid: 120 req/min                                  │
│  2. アカウント状態確認（suspended等）                   │
│  3. 権限チェック（PermissionGate）                      │
│  4. ツール実行                                          │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Cloudflare KVの使用

| Namespace | 用途 | 維持/廃止 |
|-----------|------|-----------|
| `RATE_LIMITS` | グローバルRate Limit（IP単位） | 維持 |
| `SUBSCRIPTIONS` | 課金状態キャッシュ | **廃止** |

### オリジン側Rate Limit実装

```go
package ratelimit

import (
    "sync"
    "time"
)

type RateLimiter struct {
    mu       sync.Mutex
    counters map[string]*counter
}

type counter struct {
    count     int
    windowEnd time.Time
}

type Config struct {
    Free int // requests per minute
    Paid int // requests per minute
}

var DefaultConfig = Config{
    Free: 30,
    Paid: 120,
}

func New() *RateLimiter {
    return &RateLimiter{
        counters: make(map[string]*counter),
    }
}

func (r *RateLimiter) Allow(userID string, isPaid bool) (allowed bool, remaining int, reset time.Time) {
    r.mu.Lock()
    defer r.mu.Unlock()

    limit := DefaultConfig.Free
    if isPaid {
        limit = DefaultConfig.Paid
    }

    now := time.Now()
    windowEnd := now.Truncate(time.Minute).Add(time.Minute)

    c, exists := r.counters[userID]
    if !exists || now.After(c.windowEnd) {
        c = &counter{
            count:     0,
            windowEnd: windowEnd,
        }
        r.counters[userID] = c
    }

    if c.count >= limit {
        return false, 0, c.windowEnd
    }

    c.count++
    return true, limit - c.count, c.windowEnd
}
```

### Middleware実装

```go
func RateLimitMiddleware(limiter *ratelimit.RateLimiter, cache *permission.Cache) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Header.Get("X-User-ID")

            // 課金状態を取得（PermissionCacheから）
            isPaid := cache.IsSubscribed(userID)

            allowed, remaining, reset := limiter.Allow(userID, isPaid)

            // Rate Limitヘッダーを設定
            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

            if !allowed {
                w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(reset).Seconds())))
                http.Error(w, `{"error":"rate_limit_exceeded"}`, http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### 影響

**削除するもの**：
- dsn-load-management.md の KV SUBSCRIPTIONS 関連セクション
- Stripe Webhook → KV更新コード
- Worker側の課金状態チェックコード

**追加するもの**：
- オリジン側のRate Limiter実装
- PermissionCacheにIsSubscribed()メソッド

## リスク考慮

### 想定しない匿名ユーザーへの対応

Phase 1後、一般公開時に想定されるリスクと対策：

| リスク | 発生可能性 | 影響度 | 対策 | 層 |
|--------|-----------|--------|------|-----|
| 未認証DDoS | 高 | 高 | グローバルRate Limit | Worker |
| JWT偽造 | 中 | 高 | JWT署名検証 | Worker |
| 認証済みDDoS | 中 | 中 | ユーザー単位Rate Limit | オリジン |
| アカウント共有 | 中 | 低 | Rate Limit + 監視 | オリジン |
| 自動化ツール暴走 | 高 | 中 | Rate Limit | オリジン |
| 課金回避 | 低 | 高 | 権限チェック（DB） | オリジン |

**全てのリスクに対して、適切な層で対策を実装**する。

## 関連ドキュメント

- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
