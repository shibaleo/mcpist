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

## 作業記録（追加）

| 時刻 | タスク ID | 内容 | 備考 |
|------|-----------|------|------|
|  | Onboard-001 | オンボーディングフロー改善 | サービス連携→クレジット付与の順序変更 |
|  | Onboard-002 | OAuth callback URL バグ修正 | `?step=3?success=` → `searchParams.set()` で修正 |
|  | Onboard-003 | 利用規約・プライバシーポリシーページ作成 | `/terms`, `/privacy` |
|  | Onboard-004 | DBマイグレーション 035-036 作成 | credit_transaction_type 型キャスト修正、bonus enum追加 |
|  | Dashboard-001 | ダッシュボードカードをクリック可能に | `/connections`, `/tools`, `/billing` へリンク |
|  | Dashboard-002 | オンボーディングステップ検出機能 | connections → tools → billing の順で誘導 |
|  | Dashboard-003 | カード光るアニメーション実装 | `animate-pulse-border` CSS keyframes |
|  | Onboard-005 | オンボーディングページ簡素化 | プロダクトツアー用プレースホルダーに変更 |
|  | Onboard-006 | サービス選択チェックボックス実装 | 6サービス選択可能 |
|  | Onboard-007 | preferences API 作成 | `/api/user/preferences` GET/POST |
|  | Tools-001 | カルーセル優先ソート実装 | `preferred_services` で先頭に配置 |
|  | Tools-002 | 優先サービスカード光る機能 | 未接続の優先サービスに pulse-border 適用 |

---

## 完了タスク（追加）

- [x] オンボーディングUX改善
  - オンボーディングページ簡素化（プロダクトツアー用プレースホルダー）
  - サービス選択チェックボックス追加（6サービス）
  - preferences API 作成
  - /tools カルーセル優先ソート
  - 優先サービスカードのハイライトアニメーション

- [x] ダッシュボード改善
  - カードをクリック可能に（/tools, /billing へリンク）
  - オンボーディングステップ検出（connections → tools → billing）
  - 次のアクションカードを光らせるアニメーション
  - セットアップ進捗バナー表示

- [x] 法的ページ追加
  - 利用規約ページ `/terms`
  - プライバシーポリシーページ `/privacy`
  - LP フッター、ログインページからリンク

- [x] バグ修正
  - OAuth callback URL の query param 重複問題修正
  - credit_transaction_type 型キャストエラー修正
  - bonus enum 値追加

---

## 変更ファイル概要（追加）

| カテゴリ | ファイル数 | 主な変更 |
|----------|------------|----------|
| DB Migration | 2 | 035: type cast fix, 036: bonus enum |
| Console Pages | 4 | onboarding, dashboard, terms, privacy |
| Console API | 3 | oauth/google/callback, oauth/microsoft/callback, user/preferences |
| Console CSS | 1 | globals.css (pulse-border animation) |

---

## DBマイグレーション詳細（追加）

| ファイル | 内容 |
|----------|------|
| 00000000000035_fix_add_credits_type_cast.sql | `credit_transaction_type` 型への明示的キャスト |
| 00000000000036_add_bonus_transaction_type.sql | `bonus` enum 値追加 |

---

## 新規作成ファイル

| ファイル | 内容 |
|----------|------|
| `apps/console/src/app/terms/page.tsx` | 利用規約ページ |
| `apps/console/src/app/privacy/page.tsx` | プライバシーポリシーページ |
| `apps/console/src/app/api/user/preferences/route.ts` | ユーザー設定 API |

---

## オンボーディングフロー

1. ログイン（Google/Microsoft OAuth）
2. `/onboarding` でサービス選択（6サービスから複数選択可）
3. 選択結果を `mcpist.users.preferences.preferred_services` に保存
4. `/dashboard` にリダイレクト
5. ダッシュボードで「連携中のサービス」カードが光る
6. `/tools` で選択したサービスが先頭に表示され、未接続なら光る
7. サービス接続後、「有効なツール」→「クレジット残高」と順に誘導

---

## メモ

- Stripe サンドボックスモード使用（本番移行時は prod キーに切替）
- Webhook 冪等性: `processed_webhook_events` テーブルで event_id 重複チェック
- public スキーマラッパーは Supabase JS クライアントが public スキーマをデフォルトで参照するため必要
- `animate-pulse-border` はOKLCHカラーと互換性を持たせるため `var(--primary)` を直接使用
- オンボーディングのプロダクトツアー部分は未実装（TODO）
