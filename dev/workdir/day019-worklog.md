# DAY019 作業ログ

## 日付

2026-01-30

---

## 作業記録

| 時刻 | タスク ID | 内容 | 備考 |
|------|-----------|------|------|
|  | Stripe-001 | Stripe アカウント作成 | サンドボックスモード |
|  | Stripe-002 | APIキー取得・.env.dev 設定 | `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY` |
|  | Stripe-003 | Product/Price 作成 | $0, 100 credits |
|  | Stripe-004 | DBマイグレーション 029-031 作成 | `stripe_customer_id` カラム、RPC関数 |
|  | Stripe-005 | Console Stripe クライアント実装 | `src/lib/stripe.ts` |
|  | Stripe-006 | Checkout Session API 実装 | `/api/stripe/checkout` |
|  | Stripe-007 | Webhook API 実装 | `/api/stripe/webhook` |
|  | Stripe-008 | Billing ページ UI 更新 | 購入ボタン、クレジット表示 |
|  | Stripe-009 | database.types.ts Stripe RPC 型追加 | 4関数の型定義 |
|  | Stripe-010 | Vercel 環境変数設定 | 5変数追加 |
|  | Stripe-011 | Webhook エンドポイント登録 | Stripe Dashboard |
|  | Stripe-012 | E2E テスト完了 | 100 paid_credits 付与確認 |

---

## 完了タスク

- [x] Stripe Phase 1: 無料クレジット購入フロー
  - Stripe アカウント作成・設定
  - DBマイグレーション（029-031）
  - Checkout Session API
  - Webhook API（クレジット付与）
  - Console UI更新
  - E2Eテスト完了

---

## 変更ファイル概要

| カテゴリ | ファイル数 | 主な変更 |
|----------|------------|----------|
| DB Migration | 3 | 029: stripe_customer_id + RPC, 030: get_stripe_customer_id, 031: public schema wrappers |
| Console API | 2 | checkout/route.ts, webhook/route.ts |
| Console Lib | 3 | stripe.ts, admin.ts, database.types.ts |
| Console UI | 1 | billing/page.tsx |

---

## DBマイグレーション詳細

| ファイル | 内容 |
|----------|------|
| 00000000000029_stripe_integration.sql | `stripe_customer_id` カラム追加、`add_paid_credits`, `link_stripe_customer`, `get_user_by_stripe_customer` RPC |
| 00000000000030_rpc_get_stripe_customer_id.sql | `get_stripe_customer_id` RPC |
| 00000000000031_public_stripe_rpc_wrappers.sql | public スキーマラッパー関数（Supabase クライアント互換） |

---

## 環境変数（Vercel 追加）

| 変数名 | 用途 |
|--------|------|
| `STRIPE_SECRET_KEY` | Stripe API シークレットキー |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | Stripe 公開キー |
| `STRIPE_WEBHOOK_SECRET` | Webhook 署名検証 |
| `STRIPE_FREE_CREDIT_PRICE_ID` | $0 Price ID |
| `STRIPE_FREE_CREDIT_PROD_ID` | Product ID |

---

## 解決した問題

| 問題 | 原因 | 解決策 |
|------|------|--------|
| `stripe_customer_id` 型エラー | database.types.ts に型定義なし | RPC方式に変更、型定義追加 |
| RPC function not found | mcpist スキーマに作成、public から参照不可 | public スキーマにラッパー関数作成 |
| Missing Supabase credentials | `SUPABASE_SERVICE_ROLE_KEY` vs `SUPABASE_SECRET_KEY` | admin.ts で両方対応 |
| /credits 404 | 成功URL が /credits を指定 | /billing に修正 |

---

## E2Eテスト結果

1. /billing ページで「無料クレジットを取得」ボタンクリック
2. Stripe Checkout ($0) 表示
3. テストカード入力、決済完了
4. /billing?success=true にリダイレクト
5. DB確認: `paid_credits: 100`, `stripe_customer_id: "cus_Tsuu5YWkYoHV9a"`

---

## メモ

- Stripe サンドボックスモード使用（本番移行時は prod キーに切替）
- Webhook 冪等性: `processed_webhook_events` テーブルで event_id 重複チェック
- public スキーマラッパーは Supabase JS クライアントが public スキーマをデフォルトで参照するため必要
