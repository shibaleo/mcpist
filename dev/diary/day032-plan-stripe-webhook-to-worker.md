# DAY032 計画: Stripe Webhook を Worker に移行

## 日付

2026-02-18

---

## 背景

Stripe webhook は現在 Console (Next.js) の Route Handler で処理している。
しかし webhook にはユーザーセッション (JWT) が存在せず、Console は `rpcDirect` で
PostgREST を直接呼んでいる。これは Worker 経由の統一アーキテクチャに反する。

Worker は既に PostgREST への接続情報 (`POSTGREST_URL`, `POSTGREST_API_KEY`) を持ち、
Stripe signature 検証 + RPC 呼び出しだけで完結するため、Worker で直接処理するのが最適。

## 現状

### Console 側 (`apps/console/src/app/api/stripe/webhook/route.ts`)

- `POST /api/stripe/webhook` で Stripe イベントを受信
- `stripe.webhooks.constructEvent()` で署名検証
- 2つのイベントを処理:
  - `invoice.paid` → `activate_subscription` RPC
  - `customer.subscription.deleted` → `activate_subscription` RPC (plan=free)
- `invoice.paid` のフォールバックで `get_user_by_stripe_customer` RPC
- すべて `rpcDirect` (PostgREST 直接呼び出し) を使用

### 依存する RPC

| RPC | 用途 |
|-----|------|
| `activate_subscription(p_user_id, p_plan_id, p_event_id)` | プラン有効化/ダウングレード |
| `get_user_by_stripe_customer(p_stripe_customer_id)` | Stripe customer ID → user_id 解決 |

---

## 実装計画

### Step 1: Worker に Stripe webhook エンドポイント追加

**ファイル**: `apps/worker/src/v1/routes/stripe.ts` (新規)

```
POST /v1/stripe/webhook
```

- 認証: Stripe signature 検証のみ (JWT 不要)
- `STRIPE_WEBHOOK_SECRET` を Worker の環境変数に追加
- イベント処理ロジックは Console からほぼそのまま移植
- `rpcDirect` → `callPostgRESTRpc` に置き換え

### Step 2: Worker の環境変数追加

```
STRIPE_WEBHOOK_SECRET=whsec_xxx
```

- `wrangler.toml` の `[vars]` または Cloudflare ダッシュボードの Secrets に追加
- `types.ts` の `Env` インターフェースに追加

### Step 3: Worker の v1 ルーターに stripe ルートを登録

**ファイル**: `apps/worker/src/v1/index.ts`

```typescript
import { stripe } from "./routes/stripe";
v1.route("/stripe", stripe);
```

### Step 4: OpenAPI spec 更新

**ファイル**: `apps/worker/src/openapi.yaml`

```yaml
/v1/stripe/webhook:
  post:
    operationId: handleStripeWebhook
    summary: Handle Stripe webhook events
    tags: [stripe]
    security: []  # Stripe signature で検証、JWT 不要
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
    responses:
      "200":
        description: Webhook received
        content:
          application/json:
            schema:
              type: object
              properties:
                received:
                  type: boolean
```

### Step 5: Console の webhook ルート削除

- `apps/console/src/app/api/stripe/webhook/route.ts` を削除
- `rpcDirect` の最後の利用箇所なので、`rpcDirect` 自体が不要になる可能性を確認

### Step 6: Stripe ダッシュボードで Webhook URL 変更

```
旧: https://console.mcpist.com/api/stripe/webhook
新: https://api.mcpist.com/v1/stripe/webhook
```

### Step 7: `rpcDirect` / 旧 `worker-client.ts` のクリーンアップ

- `rpcDirect` の他の利用箇所がなければ `@/lib/worker-client` から削除
- 未使用の export があれば整理

---

## 検証

1. Worker: `pnpm type-check` 通過
2. Worker: dev 環境にデプロイ
3. Stripe CLI でローカルテスト: `stripe trigger invoice.paid`
4. Stripe ダッシュボードでテスト webhook 送信
5. `invoice.paid` → プラン有効化が動作すること
6. `customer.subscription.deleted` → free にダウングレードされること
7. Console: `pnpm build` 通過 (webhook ルート削除後)

---

## リスク

| リスク | 対策 |
|--------|------|
| Worker デプロイ〜Stripe URL 変更の間にイベントが来る | 旧 Console ルートを先に残し、Worker 側を先にデプロイ・検証してから URL を切り替え |
| Stripe SDK の Cloudflare Workers 互換性 | `stripe.webhooks.constructEvent` は Web Crypto API を使うので問題ないはず。ただし `stripe` npm パッケージのバージョン確認が必要 |
| `STRIPE_WEBHOOK_SECRET` の管理 | `wrangler secret put` で暗号化して保存 |
