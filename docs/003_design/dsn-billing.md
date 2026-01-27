# MCPist 課金システム設計

## 概要

Stripe Checkoutを使用した課金システム。MCPサーバーと独立して動作する。

関連ドキュメント:
- [adr-b2c-focus.md](./adr-b2c-focus.md) - B2Cフォーカスに関するADR
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラストラクチャ設計
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計

---

## 設計方針

課金処理はMCPサーバーと**独立**させる。

**理由**:
- MCPサーバーが両系統ダウンしても課金処理は継続
- Stripe Webhookの受け口が常に稼働
- 障害の影響範囲を限定

---

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                    課金処理（独立）                      │
│                                                         │
│  ユーザーUI (Vercel)                                    │
│       │                                                 │
│       │ 購入ボタン                                      │
│       ↓                                                 │
│  Stripe Checkout（ホスト型決済画面）                    │
│       │                                                 │
│       │ 決済完了                                        │
│       ↓                                                 │
│  Stripe Webhook                                         │
│       │                                                 │
│       │ POST /webhook                                   │
│       ↓                                                 │
│  Supabase Edge Function                                 │
│       │                                                 │
│       │ DB更新（subscription_ok = true）                │
│       ↓                                                 │
│  完了（MCPサーバーは関与しない）                        │
└─────────────────────────────────────────────────────────┘
```

**ポイント**: MCPサーバーは課金処理に関与しない。

---

## Stripe Checkout

### 採用理由

- **PCI DSS準拠不要**: カード情報を扱わない
- **放置運用と相性が良い**: Stripeが全て処理
- **サブスクリプション管理**: Stripe側で完結

### フロー

```
ユーザー
    │
    │ 1. 購入ボタンクリック（UI）
    ↓
Stripe Checkout Session作成
    │
    │ 2. Stripeホスト決済画面にリダイレクト
    ↓
ユーザーがカード情報入力
    │
    │ 3. 決済処理（Stripe側）
    ↓
成功/失敗
    │
    │ 4. コールバックURLにリダイレクト
    ↓
ユーザーUI（購入完了画面）
```

### Checkout Session作成（UI側）

```typescript
// pages/api/create-checkout-session.ts
import Stripe from 'stripe';

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!);

export default async function handler(req, res) {
  const { priceId, userId } = req.body;

  const session = await stripe.checkout.sessions.create({
    mode: 'subscription',
    payment_method_types: ['card'],
    line_items: [{ price: priceId, quantity: 1 }],
    success_url: `${process.env.BASE_URL}/billing/success?session_id={CHECKOUT_SESSION_ID}`,
    cancel_url: `${process.env.BASE_URL}/billing/cancel`,
    client_reference_id: userId,  // MCPistのユーザーID
    metadata: {
      user_id: userId,
    },
  });

  res.json({ url: session.url });
}
```

---

## Webhook処理

### 受け口の選択

| 選択肢 | 評価 |
|--------|------|
| MCPサーバー | × MCPサーバー障害時に課金処理も止まる |
| Vercel API Routes | △ UIと結合、コールドスタート |
| Supabase Edge Function | ○ MCPサーバーと独立、DB直結、常時稼働 |

**Supabase Edge Function を採用**。

### Edge Function実装

```typescript
// supabase/functions/stripe-webhook/index.ts
import { serve } from 'https://deno.land/std@0.168.0/http/server.ts';
import Stripe from 'https://esm.sh/stripe@13.0.0';
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2';

const stripe = new Stripe(Deno.env.get('STRIPE_SECRET_KEY')!, {
  apiVersion: '2023-10-16',
});

const supabase = createClient(
  Deno.env.get('SUPABASE_URL')!,
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')!
);

serve(async (req) => {
  const signature = req.headers.get('stripe-signature')!;
  const body = await req.text();

  let event: Stripe.Event;
  try {
    event = stripe.webhooks.constructEvent(
      body,
      signature,
      Deno.env.get('STRIPE_WEBHOOK_SECRET')!
    );
  } catch (err) {
    return new Response(`Webhook Error: ${err.message}`, { status: 400 });
  }

  switch (event.type) {
    case 'checkout.session.completed': {
      const session = event.data.object as Stripe.Checkout.Session;
      const userId = session.client_reference_id;
      const subscriptionId = session.subscription as string;

      // DB更新: サブスクリプション有効化
      await supabase
        .from('subscriptions')
        .upsert({
          user_id: userId,
          stripe_subscription_id: subscriptionId,
          status: 'active',
          updated_at: new Date().toISOString(),
        });

      // 権限テーブル更新
      await supabase
        .from('user_tool_permissions')
        .update({ subscription_ok: true })
        .eq('user_id', userId);

      break;
    }

    case 'customer.subscription.deleted': {
      const subscription = event.data.object as Stripe.Subscription;
      const { data } = await supabase
        .from('subscriptions')
        .select('user_id')
        .eq('stripe_subscription_id', subscription.id)
        .single();

      if (data) {
        // DB更新: サブスクリプション無効化
        await supabase
          .from('subscriptions')
          .update({ status: 'canceled' })
          .eq('stripe_subscription_id', subscription.id);

        // 権限テーブル更新
        await supabase
          .from('user_tool_permissions')
          .update({ subscription_ok: false })
          .eq('user_id', data.user_id);
      }

      break;
    }

    case 'invoice.payment_failed': {
      const invoice = event.data.object as Stripe.Invoice;
      // 支払い失敗時の処理（メール通知等）
      console.log(`Payment failed for subscription: ${invoice.subscription}`);
      break;
    }
  }

  return new Response(JSON.stringify({ received: true }), {
    headers: { 'Content-Type': 'application/json' },
  });
});
```

### 処理するイベント

| イベント | 処理 |
|----------|------|
| `checkout.session.completed` | サブスクリプション有効化 |
| `customer.subscription.deleted` | サブスクリプション無効化 |
| `customer.subscription.updated` | プラン変更反映 |
| `invoice.payment_failed` | 支払い失敗通知 |

---

## 課金モデル

### Phase 1: サービス単位課金

```
モジュール単位で月額課金

Notionモジュール: ¥500/月
Jiraモジュール: ¥500/月
GitHubモジュール: ¥500/月
...

ユーザーは必要なモジュールだけ購入
```

### Stripe Products/Prices

```
Product: MCPist Notion Module
  └─ Price: ¥500/月 (recurring)

Product: MCPist Jira Module
  └─ Price: ¥500/月 (recurring)

Product: MCPist GitHub Module
  └─ Price: ¥500/月 (recurring)
```

### 将来: バンドルプラン（検討中）

```
Phase 2以降で検討:

Starterプラン: ¥1,000/月（3モジュールまで）
Proプラン: ¥2,000/月（全モジュール）
```

---

## DBスキーマ

### subscriptions テーブル

```sql
CREATE TABLE subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id),
  stripe_subscription_id TEXT NOT NULL UNIQUE,
  stripe_customer_id TEXT,
  status TEXT NOT NULL DEFAULT 'active',  -- active, canceled, past_due
  current_period_end TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_stripe_id ON subscriptions(stripe_subscription_id);
```

### user_tool_permissions テーブル（権限と連携）

```sql
-- subscription_ok カラムは課金状態を反映
-- Webhook処理で更新される

ALTER TABLE user_tool_permissions
ADD COLUMN subscription_ok BOOLEAN DEFAULT FALSE;
```

---

## キャッシュ連携

課金状態変更時、MCPサーバーのPermissionCacheを無効化する必要がある。

### 方法1: Supabase Realtime（推奨）

```typescript
// MCPサーバー側: Supabase Realtimeでsubscriptions変更を購読
supabase
  .channel('subscriptions')
  .on('postgres_changes', { event: '*', schema: 'public', table: 'subscriptions' }, (payload) => {
    const userId = payload.new.user_id;
    permissionCache.invalidateUser(userId);
  })
  .subscribe();
```

### 方法2: TTLに任せる

```
課金変更 → DB更新
    ↓
キャッシュTTL（5分）経過
    ↓
次回アクセス時にDB再取得
```

**Phase 1**: TTLに任せる（シンプル）。リアルタイム性が必要になったらRealtimeを追加。

---

## コスト

| 項目 | コスト |
|------|--------|
| Stripe手数料 | 決済額の3.6% |
| Supabase Edge Function | 無料枠内 |
| Vercel API Routes | 無料枠内 |

---

## Phase 1 スコープ

### 実装する

- [ ] Stripe Products/Prices設定
- [ ] Checkout Session作成API（Vercel）
- [ ] Webhook Edge Function（Supabase）
- [ ] subscriptionsテーブル
- [ ] 購入完了/キャンセル画面

### 実装しない

- バンドルプラン
- クーポン/割引
- 請求書払い
- 返金処理（手動対応）

---

## セキュリティ

### Webhook署名検証

```typescript
// 必ず署名を検証する
const event = stripe.webhooks.constructEvent(
  body,
  signature,
  webhookSecret
);
```

### 環境変数

| 変数 | 保存場所 |
|------|----------|
| `STRIPE_SECRET_KEY` | Supabase Secrets, Vercel Env |
| `STRIPE_WEBHOOK_SECRET` | Supabase Secrets |
| `STRIPE_PUBLISHABLE_KEY` | Vercel Env（公開可） |

---

## 関連ドキュメント

- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラストラクチャ設計
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [adr-b2c-focus.md](./adr-b2c-focus.md) - B2Cフォーカスに関するADR
