# Worker LB設計 - Render Primary / Koyeb Failover

## 概要

Render を Primary、Koyeb を Failover として、負荷ベースの動的LBを実装。

## LB戦略

```
                        p95 レイテンシ
                             │
    ┌────────────────────────┼────────────────────────┐
    │                        │                        │
  p95 < 300ms           300ms ≤ p95 < 600ms       p95 ≥ 600ms
    │                        │                        │
    ▼                        ▼                        ▼
┌─────────┐            ┌──────────┐            ┌──────────┐
│ NORMAL  │            │ WARMUP   │            │ BALANCE  │
│         │            │          │            │          │
│ Render  │            │ Render   │            │ Render   │
│  100%   │            │  100%    │            │  50%     │
│         │            │          │            │          │
│ Koyeb   │            │ Koyeb    │            │ Koyeb    │
│ (sleep) │            │ (起動中)  │            │  50%     │
└─────────┘            └──────────┘            └──────────┘
```

## 状態遷移

### 主指標: p95レイテンシ（rolling window 50req）

| 現状態 | 条件 | 次状態 |
|--------|------|--------|
| NORMAL | p95 ≥ 600ms | BALANCE |
| NORMAL | p95 ≥ 300ms | WARMUP |
| WARMUP | p95 ≥ 600ms | BALANCE |
| WARMUP | p95 < 300ms | NORMAL |
| BALANCE | p95 < 500ms | WARMUP |

### 致命指標（即座にFAILOVER）

```
timeout OR health NG OR fetch error → FAILOVER
```

- タイムアウト（3秒）
- ヘルスチェック失敗
- fetch例外

### ヒステリシス

| 遷移 | 閾値 |
|------|------|
| BALANCE → WARMUP | p95 < 500ms |
| WARMUP → NORMAL | p95 < 300ms |

## 閾値まとめ

```typescript
const THRESHOLDS = {
  WARMUP: 300,           // p95 ≥ 300ms → WARMUP
  BALANCE: 600,          // p95 ≥ 600ms → BALANCE
  BALANCE_TO_WARMUP: 500,// p95 < 500ms → WARMUP
  WARMUP_TO_NORMAL: 300, // p95 < 300ms → NORMAL
};
```

## Koyeb状態管理

| 状態 | 説明 |
|------|------|
| `sleeping` | スリープ中（普段の状態） |
| `waking` | 起動中（コールドスタート中） |
| `ready` | 起動完了、トラフィック受け入れ可能 |

### Koyeb起動タイミング

1. **WARMUP状態** になったら `waking` 開始
2. `/health` に60秒タイムアウトでリクエスト
3. 成功したら `ready`

## KV構造

```typescript
interface Metrics {
  latencies: number[];        // 直近50件のレイテンシ配列
  state: LBState;             // NORMAL | WARMUP | BALANCE | FAILOVER
  koyebState: KoyebState;     // sleeping | waking | ready
  error5xxCount: number;      // 直近20req中の5xx数
  requestCount: number;       // 直近20req用カウンタ
  lastUpdated: number;
  renderHealthy: boolean;
  koyebHealthy: boolean;
}
```

## 環境変数（wrangler.toml）

```toml
[vars]
RENDER_URL = "REDACTED_URL"
KOYEB_URL = "REDACTED_URL"
```

## スケジュール

```toml
[triggers]
crons = ["* * * * *"]  # 毎分実行
```

- Renderヘルスチェック（ウォーム維持）
- Koyeb状態確認（BALANCE/FAILOVER時のみ）

## レスポンスヘッダー

デバッグ用にヘッダー付与：

```
X-LB-State: NORMAL | WARMUP | BALANCE | FAILOVER
X-Backend: render | koyeb
```

## フロー図

```
Request
   │
   ▼
┌─────────────┐
│ 認証チェック  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Rate Limit  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Metrics取得  │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────┐
│ バックエンド選択              │
│                             │
│ FAILOVER → Koyeb            │
│ BALANCE  → 50/50            │
│ WARMUP   → Render (+Koyeb起動)│
│ NORMAL   → Render           │
└──────┬──────────────────────┘
       │
       ▼
┌─────────────┐    timeout/error
│ fetch       │──────────────┐
└──────┬──────┘              │
       │ ok                   │
       ▼                      ▼
┌─────────────┐        ┌─────────────┐
│ Metrics更新  │        │ Koyebリトライ │
│ (非同期)     │        │ or 503返却   │
└──────┬──────┘        └─────────────┘
       │
       ▼
   Response
```
