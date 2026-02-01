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

---

## 作業記録（追加2）

| 時刻 | タスク ID | 内容 | 備考 |
|------|-----------|------|------|
|  | Bonus-001 | 初回クレジット付与UI実装 | billing ページに `pre_active` ユーザー向けカード |
|  | Bonus-002 | テストクレジットカードUI更新 | `active` ユーザー向け、テスト期間中は何度でも取得可能 |
|  | Dashboard-004 | オンボーディングステップからtools削除 | サービス接続後にデフォルトツールが有効化されるため不要 |
|  | Dashboard-005 | オンボーディング条件を pre_active ベースに変更 | クレジット数ではなく account_status で判定 |
|  | Dashboard-006 | 残高アラート実装 | active ユーザーで残高50以下の時に警告バナー表示 |
|  | MCP-001 | クレジット不足エラーにbilling URL追加 | authz.go, handler.go |

---

## 完了タスク（追加2）

- [x] 初回クレジット付与（Signup Bonus）
  - billing ページに `pre_active` ユーザー向け初回クレジットカード
  - 「100クレジットを受け取る」ボタンで grant-signup-bonus API 呼び出し
  - 付与後 account_status が active に遷移

- [x] ダッシュボードオンボーディング改善
  - オンボーディングステップから tools 削除（connections → billing → complete）
  - オンボーディング条件を account_status ベースに変更
  - 残高アラート追加（active ユーザー、残高50以下で警告）

- [x] MCPサーバー改善
  - クレジット不足エラーメッセージに billing URL 追加
  - 単一ツール実行（authz.go）とバッチ実行（handler.go）両方対応

---

## 変更ファイル概要（追加2）

| カテゴリ | ファイル | 主な変更 |
|----------|----------|----------|
| Console | billing/page.tsx | pre_active 向け初回クレジットカード、テストクレジットカードUI更新 |
| Console | dashboard/page.tsx | tools step 削除、pre_active 条件、残高アラート |
| Console | credits.ts | UserContext 型、getUserContext() 関数 |
| Server | authz.go | INSUFFICIENT_CREDITS エラーに billing URL 追加 |
| Server | handler.go | バッチ実行のクレジット不足エラーに billing URL 追加 |

---

## オンボーディングフロー（更新版）

1. ログイン（Google/Microsoft OAuth）
2. `/onboarding` でサービス選択（6サービスから複数選択可）
3. 選択結果を `mcpist.users.preferences.preferred_services` に保存
4. `/dashboard` にリダイレクト
5. ダッシュボードで「連携中のサービス」カードが光る（connections = 0 の場合）
6. `/tools` で選択したサービスが先頭に表示され、未接続なら光る
7. サービス接続後、「クレジット残高」カードが光る（pre_active の場合）
8. `/billing` で「初回クレジットを受け取る」カードが光る
9. 100クレジット付与後、account_status が active に遷移
10. 以降は残高50以下でアラート表示

---

## 作業記録（追加3：RPC設計・マイグレーション統合）

| 時刻 | タスク ID | 内容 | 備考 |
|------|-----------|------|------|
|  | Migration-001 | マイグレーション統合（36→9ファイル） | 再設計に伴う統合 |
|  | RPC-001 | RPC命名規則統一 | `_my_`=Console(User), `_user_`=Router/API Server |
|  | RPC-002 | `consume_credits` → `consume_user_credits` | API Server 呼び出し |
|  | RPC-003 | `add_credits` → `add_user_credits` | Console Router 呼び出し |
|  | RPC-004 | `complete_onboarding` → `complete_user_onboarding` | Console Router 呼び出し |
|  | RPC-005 | `lookup_user_by_key_hash` 呼び出し元を Gateway に変更 | API Server → Gateway |
|  | RPC-006 | `list_modules` 呼び出し元を Console (Router) に変更 | |
|  | RPC-007 | OAuth Apps RPCの呼び出し元を Console (Router) に変更 | Admin → Router |
|  | RPC-008 | `preferences` → `settings` 統一 | カラム名・RPC名変更 |
|  | Table-001 | users テーブルに `display_name`, `avatar_url` 追加 | |
|  | Prompts-001 | prompts テーブル用 RPC マイグレーション作成 | 00000000000010_rpc_prompts.sql |
|  | Canvas-001 | grh-rpc-design.canvas 更新 | 全変更をCanvas図に反映 |

---

## 完了タスク（追加3）

- [x] マイグレーション統合・再設計
  - 36ファイル → 9ファイルに統合
  - RPC命名規則の統一（`_my_` / `_user_` プレフィックス）
  - 呼び出し元の整理（Console User/Router、API Server、Gateway）

- [x] RPC名変更
  - `consume_credits` → `consume_user_credits`
  - `add_credits` → `add_user_credits`
  - `complete_onboarding` → `complete_user_onboarding`
  - `get_my_preferences` → `get_my_settings`
  - `update_my_preferences` → `update_my_settings`

- [x] 呼び出し元変更
  - `lookup_user_by_key_hash`: API Server → Gateway
  - `list_modules`: API Server / Console (User) → Console (Router)
  - OAuth Apps RPC: Console (Admin) → Console (Router)

- [x] テーブル変更
  - users: `display_name`, `avatar_url` カラム追加
  - users: `preferences` → `settings` 統一

- [x] prompts RPC 作成
  - `list_my_prompts` - プロンプト一覧取得
  - `get_my_prompt` - プロンプト詳細取得
  - `upsert_my_prompt` - プロンプト作成/更新
  - `delete_my_prompt` - プロンプト削除

- [x] Canvas 更新
  - grh-rpc-design.canvas に全変更を反映
  - prompts グループ・テーブル・RPC追加

---

## 変更ファイル概要（追加3）

| カテゴリ | ファイル | 主な変更 |
|----------|----------|----------|
| DB Migration | 00000000000002_tables.sql | users に display_name, avatar_url 追加 |
| DB Migration | 00000000000006_rpc_mcp_server.sql | consume_credits → consume_user_credits |
| DB Migration | 00000000000007_rpc_console.sql | preferences → settings |
| DB Migration | 00000000000009_stripe_integration.sql | add_credits → add_user_credits, complete_onboarding → complete_user_onboarding |
| DB Migration | 00000000000010_rpc_prompts.sql | 新規作成（prompts CRUD RPC） |
| Docs | grh-rpc-design.canvas | 全変更反映、prompts追加 |

---

## RPC命名規則

| プレフィックス | 呼び出し元 | 認証方式 | ユーザー特定 |
|---|---|---|---|
| `_my_` | Console (User) | `auth.uid()` | 自分自身のみ |
| `_user_` | Console (Router) / API Server | `service_role` | `p_user_id` で指定 |
| なし | 公開 / Trigger | 状況による | - |

---

## 呼び出し元一覧

| 呼び出し元 | 説明 | 認証 |
|---|---|---|
| Console (User) | ユーザー自身の操作 | authenticated role |
| Console (Router) | サーバーサイドAPI Route | service_role |
| API Server | MCP Server | service_role |
| Gateway | 認証ゲートウェイ | service_role |
| Trigger | DB トリガー | - |
