# MCPist 負荷対策設計

## 概要

MCPサーバーの負荷対策。Rate Limiting、監視、ヘルスチェックを含む。

関連ドキュメント:
- [adr-rate-limit-architecture.md](./adr-rate-limit-architecture.md) - Rate Limitアーキテクチャに関するADR
- [adr-usage-control-architecture.md](./adr-usage-control-architecture.md) - 使用量制御アーキテクチャに関するADR
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-subscription.md](./dsn-subscription.md) - サブスクリプション・使用量制御設計
- [dsn-deployment.md](./dsn-deployment.md) - デプロイ戦略

---

## 設計方針

### インフラリスクとビジネスリスクの分離

| リスク種別 | 対処層 | 例 |
|-----------|--------|-----|
| **インフラリスク** | CDN/LB層（Worker） | DDoS、未認証アクセス、IP単位の異常 |
| **ビジネスリスク** | オリジン | 認証済みユーザーの異常利用、課金回避 |

### ビジネスロジックはオリジンに集約

```
Worker（CDN/LB層）: インフラ保護のみ
オリジン: ビジネスロジック（課金別Rate Limit、権限チェック）
```

### Phase 1の位置づけ

Phase 1は「友人5-10人」だが、**これはテストユーザー**である。

- 最終的には**想定しない匿名ユーザー**が利用可能になる
- その時点でどんなことが起きるかは予測不可能
- **過剰でもリスクは徹底的に考慮する必要がある**

---

## Rate Limiting アーキテクチャ

### 2層構成

```
┌─────────────────────────────────────────────────────────┐
│                 Cloudflare Worker                        │
│                                                          │
│  責務: インフラ保護                                      │
│                                                          │
│  1. JWT署名検証（未登録ユーザー遮断）                   │
│  2. グローバルRate Limit（IP単位、DDoS対策）            │
│     - 1000 req/min/IP                                   │
│  3. Burst制限（ユーザー単位、スパイク防止）             │
│     - 5 req/s                                           │
│  4. 有効 → オリジンに転送                               │
│                                                          │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ↓
┌─────────────────────────────────────────────────────────┐
│                 オリジン（MCPサーバー）                   │
│                                                          │
│  責務: ビジネスロジック                                  │
│                                                          │
│  1. Rate Limit（ユーザー単位、プラン別）                │
│  2. 権限チェック（PermissionGate）                      │
│  3. Quotaチェック（月間使用量）                         │
│  4. Creditチェック（従量課金）                          │
│  5. ツール実行                                          │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

詳細は [dsn-subscription.md](./dsn-subscription.md) を参照。

### なぜこの構成か

詳細は [adr-rate-limit-architecture.md](./adr-rate-limit-architecture.md) を参照。

要点:
- 一般的なSaaSアーキテクチャとの整合性
- 責務の明確化
- KV同期問題の回避
- ビジネスロジックの一元管理

---

## Worker側: インフラ保護

### 目的

- **DDoS防御**: 未認証攻撃の吸収
- **IP単位の異常検知**: 単一IPからの大量アクセス遮断
- **瞬間スパイク防止**: ユーザー単位のBurst制限
- **オリジン保護**: 不正リクエストがオリジンに到達しない

### レート設定

| 対象 | 制限 | 用途 |
|------|------|------|
| IP単位（グローバル） | 1000 req/min | DDoS対策 |
| ユーザー単位（Burst） | 5 req/s | 瞬間スパイク防止 |

### Worker実装

```typescript
// worker.ts
import { jwtVerify, createRemoteJWKSet } from 'jose';

interface Env {
  RATE_LIMITS: KVNamespace;    // Rate Limit / Burst用
  GATEWAY_SECRET: string;
}

const JWKS_URL = 'https://xxx.supabase.co/auth/v1/.well-known/jwks.json';
const ISSUER = 'https://xxx.supabase.co/auth/v1';
const JWKS = createRemoteJWKSet(new URL(JWKS_URL));

// グローバルRate Limit（IP単位）
const GLOBAL_RATE_LIMIT = { requests: 1000, windowSec: 60 };

// Burst制限（ユーザー単位）
const BURST_LIMIT = { requests: 5, windowSec: 1 };

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const clientIP = request.headers.get('CF-Connecting-IP') || 'unknown';

    // ヘルスチェックはスルー
    if (url.pathname === '/health') {
      return fetch(request);
    }

    // 1. グローバルRate Limit（IP単位、DDoS対策）
    const globalRateOk = await checkGlobalRateLimit(env.RATE_LIMITS, clientIP);
    if (!globalRateOk) {
      return jsonResponse({
        error: 'rate_limit_exceeded',
        message: 'Too many requests from this IP',
      }, 429);
    }

    // 2. Authorizationヘッダー確認
    const authHeader = request.headers.get('Authorization');
    if (!authHeader?.startsWith('Bearer ')) {
      return jsonResponse({
        error: 'unauthorized',
        message: 'Authorization header required',
      }, 401);
    }

    const token = authHeader.slice(7);
    let userId: string;

    // 3. JWT署名検証
    try {
      const { payload } = await jwtVerify(token, JWKS, { issuer: ISSUER });
      userId = payload.sub as string;
    } catch (err) {
      return jsonResponse({
        error: 'forbidden',
        message: 'Invalid or expired token',
      }, 403);
    }

    // 4. Burst制限（ユーザー単位、瞬間スパイク防止）
    const burstResult = await checkBurst(env.RATE_LIMITS, userId);
    if (!burstResult.allowed) {
      return jsonResponse({
        error: 'burst_exceeded',
        message: 'Too many requests. Please slow down.',
        retry_after: burstResult.retryAfter,
      }, 429, { 'Retry-After': String(Math.ceil(burstResult.retryAfter)) });
    }

    // 5. オリジンに転送
    const modifiedRequest = new Request(request, {
      headers: new Headers(request.headers)
    });
    modifiedRequest.headers.set('X-User-ID', userId);
    modifiedRequest.headers.set('X-Gateway-Secret', env.GATEWAY_SECRET);

    return fetch(modifiedRequest);
  }
};

// グローバルRate Limitチェック
async function checkGlobalRateLimit(kv: KVNamespace, ip: string): Promise<boolean> {
  const windowKey = `global:${ip}:${Math.floor(Date.now() / 1000 / GLOBAL_RATE_LIMIT.windowSec)}`;
  const currentCount = parseInt(await kv.get(windowKey) || '0', 10);

  if (currentCount >= GLOBAL_RATE_LIMIT.requests) {
    return false;
  }

  await kv.put(windowKey, String(currentCount + 1), {
    expirationTtl: GLOBAL_RATE_LIMIT.windowSec * 2,
  });

  return true;
}

// Burst制限チェック（ユーザー単位）
interface BurstResult {
  allowed: boolean;
  retryAfter: number;
}

async function checkBurst(kv: KVNamespace, userID: string): Promise<BurstResult> {
  const now = Math.floor(Date.now() / 1000);
  const key = `burst:${userID}`;

  const data = await kv.get<{ count: number; windowStart: number }>(key, 'json');

  if (!data || now - data.windowStart >= BURST_LIMIT.windowSec) {
    // 新しいウィンドウ
    await kv.put(key, JSON.stringify({
      count: 1,
      windowStart: now
    }), { expirationTtl: 60 });
    return { allowed: true, retryAfter: 0 };
  }

  if (data.count >= BURST_LIMIT.requests) {
    const retryAfter = BURST_LIMIT.windowSec - (now - data.windowStart);
    return { allowed: false, retryAfter: Math.max(0.1, retryAfter) };
  }

  // カウント増加
  await kv.put(key, JSON.stringify({
    count: data.count + 1,
    windowStart: data.windowStart
  }), { expirationTtl: 60 });

  return { allowed: true, retryAfter: 0 };
}

function jsonResponse(data: object, status: number, headers?: Record<string, string>): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  });
}
```

### Cloudflare KV

| Namespace | 用途 | キー形式 | 値 |
|-----------|------|----------|-----|
| `RATE_LIMITS` | グローバルRate Limit | `global:{ip}:{window}` | リクエスト数 |
| `RATE_LIMITS` | Burst制限 | `burst:{userID}` | `{count, windowStart}` |

**注**: `SUBSCRIPTIONS` namespaceは廃止。課金状態はオリジンでDB参照。

### wrangler.toml

```toml
# wrangler.toml
name = "mcpist-gateway"
main = "src/worker.ts"
compatibility_date = "2024-01-01"

[[kv_namespaces]]
binding = "RATE_LIMITS"
id = "xxxxx"  # グローバルRate Limit用

[vars]
# 環境変数（非機密）

# シークレットは wrangler secret で設定
# wrangler secret put GATEWAY_SECRET
```

---

## オリジン側: ビジネスロジック

オリジン側の使用量制御（Rate Limit、Quota、Credit）は [dsn-subscription.md](./dsn-subscription.md) を参照。

ここでは概要のみ記載。

### 制御一覧

| 制御 | 目的 | 保存場所 |
|------|------|----------|
| Rate Limit | 分単位の過負荷防止（プラン別） | メモリ |
| Quota | 月間使用量制限 | DB |
| Credit | 従量課金 | DB |

### Rate Limit概要

| プラン | 制限 |
|--------|------|
| Free | 30 req/min |
| Starter | 60 req/min |
| Pro | 120 req/min |
| Unlimited | 無制限（-1） |

### 実装（簡略版）

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

// 定期的に古いエントリを削除
func (r *RateLimiter) Cleanup() {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    for userID, c := range r.counters {
        if now.After(c.windowEnd) {
            delete(r.counters, userID)
        }
    }
}
```

### Middleware

```go
package middleware

import (
    "net/http"
    "strconv"
    "time"

    "mcpist/internal/permission"
    "mcpist/internal/ratelimit"
)

func RateLimitMiddleware(limiter *ratelimit.RateLimiter, cache *permission.Cache) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Header.Get("X-User-ID")

            // 課金状態を取得（PermissionCacheから、DB参照）
            isPaid := cache.IsSubscribed(userID)

            // Rate Limit判定
            limit := ratelimit.DefaultConfig.Free
            if isPaid {
                limit = ratelimit.DefaultConfig.Paid
            }

            allowed, remaining, reset := limiter.Allow(userID, isPaid)

            // Rate Limitヘッダーを設定
            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

            if !allowed {
                retryAfter := int(time.Until(reset).Seconds())
                if retryAfter < 1 {
                    retryAfter = 1
                }
                w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many requests"}`))
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### レスポンスヘッダー

```
X-RateLimit-Limit: 30
X-RateLimit-Remaining: 25
X-RateLimit-Reset: 1704067200
```

429レスポンス時:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests"
}
```

---

## 想定リスクと対策

### Phase 1後、一般公開時のリスク

| リスク | 発生可能性 | 影響度 | 対策 | 層 |
|--------|-----------|--------|------|-----|
| 未認証DDoS | 高 | 高 | グローバルRate Limit | Worker |
| JWT偽造 | 中 | 高 | JWT署名検証 | Worker |
| 認証済みDDoS | 中 | 中 | ユーザー単位Rate Limit | オリジン |
| アカウント共有 | 中 | 低 | Rate Limit + 監視 | オリジン |
| 自動化ツール暴走 | 高 | 中 | Rate Limit | オリジン |
| 課金回避 | 低 | 高 | 権限チェック（DB） | オリジン |

### 認証済みユーザーの異常アクセス

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

---

## 監視・アラート（Grafana Cloud）

### 設計方針

- **ヘルスチェック（/health）はシンプルに**: DB接続のみ（フェイルオーバー判断用）
- **CPU/メモリ監視は外部メトリクスに委譲**: Grafana Cloudで監視・アラート
- **安全性・安定性第一**: 外部メトリクスで客観的に監視

### Grafana Cloud構成

```
┌─────────────────────────────────────────────────────────┐
│                    Grafana Cloud                         │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Prometheus  │  │     Loki     │  │   Grafana    │  │
│  │  (メトリクス) │  │   (ログ)     │  │(ダッシュボード)│  │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘  │
│         │                  │                             │
│         │    Prometheus Remote Write                     │
│         │    Loki Push                                   │
└─────────┼──────────────────┼─────────────────────────────┘
          │                  │
          ↑                  ↑
┌─────────┼──────────────────┼─────────────────────────────┐
│         │                  │                             │
│    ┌────┴────┐        ┌────┴────┐                       │
│    │ /metrics│        │ログ出力  │                       │
│    │エンドポ │        │         │                       │
│    └────┬────┘        └────┬────┘                       │
│         │                  │                             │
│         └──────┬───────────┘                             │
│                │                                         │
│    ┌───────────┴───────────┐                            │
│    │     MCPサーバー        │                            │
│    │  (Koyeb / Fly.io)     │                            │
│    └───────────────────────┘                            │
└─────────────────────────────────────────────────────────┘
```

### Prometheus Exporter実装（Go）

MCPサーバーに `/metrics` エンドポイントを追加。

```go
package metrics

import (
    "net/http"
    "runtime"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    // リクエスト関連
    RequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mcpist_requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "path", "status"},
    )

    RequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mcpist_request_duration_seconds",
            Help:    "Request duration in seconds",
            Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
        },
        []string{"method", "path"},
    )

    // Rate Limit関連
    RateLimitHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mcpist_rate_limit_hits_total",
            Help: "Total number of rate limit hits",
        },
        []string{"user_type"},  // "free" or "paid"
    )

    // リソース関連
    MemoryUsage = prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "mcpist_memory_bytes",
            Help: "Current memory usage in bytes",
        },
        func() float64 {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            return float64(m.Alloc)
        },
    )

    GoroutineCount = prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "mcpist_goroutines",
            Help: "Current number of goroutines",
        },
        func() float64 {
            return float64(runtime.NumGoroutine())
        },
    )

    // DB関連
    DBQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mcpist_db_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
        },
        []string{"query_type"},
    )
)

func init() {
    prometheus.MustRegister(
        RequestsTotal,
        RequestDuration,
        RateLimitHits,
        MemoryUsage,
        GoroutineCount,
        DBQueryDuration,
    )
}

func Handler() http.Handler {
    return promhttp.Handler()
}
```

### アラート設定

| アラート | 条件 | 重要度 |
|----------|------|--------|
| CPU高負荷 | CPU > 80% (5分間) | Warning |
| メモリ逼迫 | Memory > 200MB (Fly.io上限256MB) | Critical |
| レイテンシ悪化 | p95 > 2s (5分間) | Warning |
| エラー率上昇 | 5xx > 5% (5分間) | Critical |
| オリジン停止 | /health 失敗 (3回連続) | Critical |
| Rate Limit多発 | Rate Limit > 100/min | Warning |

### ログ収集（Loki）

```go
// ログは構造化JSON形式で出力（Lokiが解析しやすい）
type LogEntry struct {
    Timestamp string `json:"ts"`
    Level     string `json:"level"`
    Message   string `json:"msg"`
    UserID    string `json:"user_id,omitempty"`
    Module    string `json:"module,omitempty"`
    Duration  int64  `json:"duration_ms,omitempty"`
    Error     string `json:"error,omitempty"`
}
```

### コスト（Grafana Cloud無料枠）

| リソース | 無料枠 |
|----------|--------|
| メトリクス | 10,000シリーズ |
| ログ | 50GB/月 |
| 保持期間 | 14日間 |

Phase 1（5-10人）では十分。

---

## ヘルスチェック

### 設計方針

- **/health はシンプルに**: DB接続チェックのみ
- **フェイルオーバー判断用**: Cloudflare LBが参照
- **CPU/メモリは外部監視**: Grafana Cloudに委譲

### 2種類のヘルスチェック

| 用途 | 対象 | 方法 | 判定 |
|------|------|------|------|
| **Cloudflare LB** | オリジン直接 | `/health` | どちらか一方がhealthyならOK |
| **CI/CD検証** | 各オリジン直接 | `GATEWAY_SECRET`付き | 全オリジンがhealthy必須 |

### /health の判定項目

| 項目 | チェック方法 | 失敗時 |
|------|-------------|--------|
| DB接続 | `db.PingContext` (3秒タイムアウト) | 503 |

### 実装

```go
package health

import (
    "context"
    "database/sql"
    "net/http"
    "time"
)

type HealthChecker struct {
    db *sql.DB
}

func NewHealthChecker(db *sql.DB) *HealthChecker {
    return &HealthChecker{db: db}
}

func (h *HealthChecker) Handler(w http.ResponseWriter, r *http.Request) {
    // DB接続チェックのみ（CPU/メモリは外部メトリクスで監視）
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    if err := h.db.PingContext(ctx); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte(`{"status":"unhealthy","db":"error"}`))
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}
```

---

## フェイルオーバー

### Cloudflare ヘルスチェック

```
Cloudflare
    │
    ├─→ Koyeb: GET /health（30秒ごと）
    │     └─ 3回連続失敗 → unhealthy
    │
    └─→ Fly.io: GET /health（30秒ごと）
          └─ 3回連続失敗 → unhealthy
```

### 切替ロジック

```
if (Koyeb == healthy && Fly.io == healthy) {
    // 通常: Primary (現在の設定) にルーティング
}

if (Koyeb == unhealthy && Fly.io == healthy) {
    // 自動フェイルオーバー: Fly.io にルーティング
}

if (Koyeb == healthy && Fly.io == unhealthy) {
    // Fly.io障害: Koyeb にルーティング（問題なし）
}

if (Koyeb == unhealthy && Fly.io == unhealthy) {
    // 両方障害: エラーページ表示
    // → アラート通知（後で対応）
}
```

---

## Phase 1 スコープ

### 実装する

- [ ] グローバルRate Limit（Worker）
- [ ] Burst制限（Worker）
- [ ] ユーザー単位Rate Limit（オリジン）
- [ ] Cloudflare KV設定
- [ ] ヘルスチェックエンドポイント
- [ ] Grafana Cloud設定
  - [ ] Prometheus（メトリクス）
  - [ ] Loki（ログ）
  - [ ] アラート設定

### 実装しない

- オートスケーリング
- 複雑な負荷分散アルゴリズム

---

## 関連ドキュメント

- [adr-rate-limit-architecture.md](./adr-rate-limit-architecture.md) - Rate Limitアーキテクチャに関するADR
- [adr-usage-control-architecture.md](./adr-usage-control-architecture.md) - 使用量制御アーキテクチャに関するADR
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-subscription.md](./dsn-subscription.md) - サブスクリプション・使用量制御設計
- [dsn-deployment.md](./dsn-deployment.md) - デプロイ戦略
