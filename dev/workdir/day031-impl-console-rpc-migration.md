# Console RPC 移行計画: auth.uid() 廃止・命名統一

## 日付

2026-02-16

---

## 背景

### 現状の問題

1. **Supabase 依存**: Console RPC (24個) が全て `auth.uid()` を使用。素の PostgREST には `auth.uid()` が存在しないため、Neon 移行時に全て壊れる
2. **ブラウザ直接呼出**: 大半の RPC がブラウザから Supabase SDK 経由で PostgREST に直接通信。Supabase SDK 廃止方針と矛盾
3. **Server との設計不一致**: Go Server は `p_user_id` パラメータ + service_role key で PostgREST に直接 HTTP。Console は `auth.uid()` + Supabase SDK。同じ DB に対して2つの認証パターンが混在
4. **命名の不統一**: `_my_` (Console用)、`_user_` (Server用)、なし (共通) が混在

### あるべき姿

```
Go Server:   Worker(JWT検証) → Go Server → PostgREST /rpc/xxx(p_user_id)
Console:     Browser(JWT)    → Next.js   → PostgREST /rpc/xxx(p_user_id)
```

- **全 RPC が `p_user_id` パラメータを受け取る**（`auth.uid()` ゼロ）
- **ブラウザは PostgREST と直接通信しない**（Next.js Backend が中継）
- **Supabase SDK への依存はゼロ**（`fetch()` で PostgREST に直接 HTTP）
- **RPC 名から `_my_` / `_user_` を除去**（`p_user_id` パラメータで自明）
- **全 DB アクセスは RPC 経由**（スキーマ非公開を維持。Console/Server はテーブル名・カラム名を知らない）

---

## 命名規則

### 方針

- `_my_` 廃止: `auth.uid()` がないので「自分の」は意味がない
- `_user_` 廃止: `p_user_id` パラメータがあるので冗長
- ドメイン名（動詞 + リソース）で統一
- admin 用は `_all_` で区別（全ユーザー対象を明示）

### 命名変更一覧

#### Server RPC (Go Server が呼ぶもの)

| 現在 | 変更後 | 備考 |
|------|--------|------|
| `lookup_user_by_key_hash` | `lookup_user_by_key_hash` | 変更なし (検索対象が user) |
| `get_user_context` | `get_user_context` | 変更なし (検索対象が user) |
| `record_usage` | `record_usage` | 変更なし |
| `get_user_prompts` | `get_prompts` | `_user_` 除去 |
| `get_user_credential` | `get_credential` | `_user_` 除去 |
| `get_oauth_app_credentials` | `get_oauth_app_credentials` | 変更なし (user 関係ない) |
| `upsert_user_credential` | `upsert_credential` | `_user_` 除去。Console 用と統合 |
| `sync_modules` | `sync_modules` | 変更なし |
| `list_modules_with_tools` | `list_modules_with_tools` | 変更なし |

#### Console RPC → RPC として残すもの

| 現在 | 変更後 |
|------|--------|
| `generate_my_api_key` | `generate_api_key` |
| `upsert_my_tool_settings` | `upsert_tool_settings` |
| `upsert_my_module_description` | `upsert_module_description` |
| `upsert_my_prompt` | `upsert_prompt` |
| `update_my_settings` | `update_settings` |
| `upsert_my_credential` | `upsert_credential` (Server用と統合) |
| `get_my_usage` | `get_usage` |
| `list_my_oauth_consents` | `list_oauth_consents` |
| `revoke_my_oauth_consent` | `revoke_oauth_consent` |
| `list_all_oauth_consents` | `list_all_oauth_consents` |

#### Console RPC → RPC 維持 (リネーム + p_user_id 追加)

| 現在 | 変更後 |
|------|--------|
| `list_my_api_keys` | `list_api_keys` |
| `revoke_my_api_key` | `revoke_api_key` |
| `list_my_credentials` | `list_credentials` |
| `delete_my_credential` | `delete_credential` |
| `list_my_prompts` | `list_prompts` |
| `get_my_prompt` | `get_prompt` |
| `delete_my_prompt` | `delete_prompt` |

#### Console RPC → 廃止 (統合)

| 現在 | 統合先 |
|------|--------|
| `get_my_role` | `get_user_context` に role を含める |
| `get_my_settings` | `get_user_context` に settings を含める |
| `get_my_tool_settings` | `get_module_config` に統合 |
| `get_my_module_descriptions` | `get_module_config` に統合 |

#### 新設

| RPC | パラメータ | 処理 |
|-----|----------|------|
| `get_module_config` | `p_user_id, p_module_name?` | ツール設定 + モジュール説明を一括取得 |

---

## 最終 RPC リスト

### 全 RPC (移行後)

| # | RPC | パラメータ | 呼出元 | 処理 |
|---|-----|----------|--------|------|
| 1 | `lookup_user_by_key_hash` | `p_key_hash` | Worker | API Key → user_id 解決 |
| 2 | `get_user_context` | `p_user_id` | Server + Console | ユーザー認可情報一括取得 (role, settings 含む) |
| 3 | `record_usage` | `p_user_id, p_meta_tool, p_request_id, p_details` | Server | 利用記録 |
| 4 | `get_prompts` | `p_user_id, p_prompt_name?, p_enabled_only?` | Server | プロンプト取得 |
| 5 | `get_credential` | `p_user_id, p_module` | Server | モジュール認証情報取得 |
| 6 | `upsert_credential` | `p_user_id, p_module, p_credentials` | Server + Console | トークン保存 |
| 7 | `get_oauth_app_credentials` | `p_provider` | Server + Console | OAuth アプリ設定取得 (Vault) |
| 8 | `sync_modules` | `p_modules` | Server 起動時 | モジュール+ツール DB 同期 |
| 9 | `list_modules_with_tools` | なし | Console | モジュール+ツール一覧 |
| 10 | `generate_api_key` | `p_user_id, p_display_name, p_expires_at?` | Console | API Key 生成 (暗号鍵生成) |
| 11 | `list_api_keys` | `p_user_id` | Console | API Key 一覧 |
| 12 | `revoke_api_key` | `p_user_id, p_key_id` | Console | API Key 失効 |
| 13 | `list_credentials` | `p_user_id` | Console | 接続サービス一覧 |
| 14 | `delete_credential` | `p_user_id, p_module` | Console | トークン削除 |
| 15 | `list_prompts` | `p_user_id, p_module_name?` | Console | プロンプト一覧 |
| 16 | `get_prompt` | `p_user_id, p_prompt_id` | Console | プロンプト取得 |
| 17 | `upsert_prompt` | `p_user_id, p_name, p_content, p_module_name?, p_prompt_id?, p_enabled?, p_description?` | Console | プロンプト作成/更新 |
| 18 | `delete_prompt` | `p_user_id, p_prompt_id` | Console | プロンプト削除 |
| 19 | `upsert_tool_settings` | `p_user_id, p_module_name, p_enabled_tools, p_disabled_tools` | Console | ツール設定更新 |
| 20 | `upsert_module_description` | `p_user_id, p_module_name, p_description` | Console | モジュール説明更新 |
| 21 | `get_module_config` | `p_user_id, p_module_name?` | Console | ツール設定 + モジュール説明一括取得 |
| 22 | `update_settings` | `p_user_id, p_settings` | Console | ユーザー設定更新 (JSONB merge) |
| 23 | `get_usage` | `p_user_id, p_start_date, p_end_date` | Console | 利用統計 |
| 24 | `list_oauth_consents` | `p_user_id` | Console | OAuth 同意一覧 |
| 25 | `revoke_oauth_consent` | `p_user_id, p_consent_id` | Console | OAuth 同意取消 |
| 26 | `list_all_oauth_consents` | なし | Console (admin) | 全ユーザー OAuth 同意 |
| 27 | `list_oauth_apps` | なし | Console (admin) | OAuth アプリ一覧 (Vault) |
| 28 | `upsert_oauth_app` | `p_provider, p_client_id, p_client_secret, p_redirect_uri, p_enabled` | Console (admin) | OAuth アプリ登録/更新 (Vault) |
| 29 | `delete_oauth_app` | `p_provider` | Console (admin) | OAuth アプリ削除 (Vault) |
| 30 | `activate_subscription` | `p_user_id, p_plan_id, p_event_id` | Stripe webhook | プラン有効化 |
| 31 | `complete_user_onboarding` | `p_user_id, p_event_id` | Signup handler | 新規ユーザー有効化 |
| 32 | `get_stripe_customer_id` | `p_user_id` | Stripe checkout | Stripe Customer ID 取得 |
| 33 | `link_stripe_customer` | `p_user_id, p_stripe_customer_id` | Stripe checkout | Stripe Customer 紐付け |
| 34 | `get_user_by_stripe_customer` | `p_stripe_customer_id` | Stripe webhook | Stripe Customer → user_id 逆引き |

**合計: 34 RPC** (現状 33 からリネーム + 統合4廃止 + 1新設。スキーマ非公開を維持し全アクセスを RPC に閉じ込める)

---

## 移行フェーズ

### Phase 0: PostgREST クライアント基盤

Supabase SDK を置き換える PostgREST HTTP クライアントを作成。

```typescript
// lib/postgrest.ts — Next.js サーバーサイドでのみ使用
const POSTGREST_URL = process.env.POSTGREST_URL!
const POSTGREST_API_KEY = process.env.POSTGREST_API_KEY!

const headers = {
  "Content-Type": "application/json",
  "apikey": POSTGREST_API_KEY,
  "Authorization": `Bearer ${POSTGREST_API_KEY}`,
}

/** RPC 呼出 — 全 DB アクセスはこの関数経由 */
export async function rpc<T>(name: string, params: Record<string, unknown> = {}): Promise<T> {
  const res = await fetch(`${POSTGREST_URL}/rpc/${name}`, {
    method: "POST",
    headers,
    body: JSON.stringify(params),
  })
  if (!res.ok) throw new PostgRESTError(res.status, await res.text())
  return res.json()
}
```

> テーブル直接アクセス (`query`, `mutate`) は提供しない。
> 全 DB アクセスを RPC に閉じ込め、スキーマ非公開を維持する。

**成果物:**

| ファイル | 内容 |
|---------|------|
| `apps/console/src/lib/postgrest.ts` | PostgREST HTTP クライアント (rpc のみ) |
| `apps/console/src/lib/auth.ts` | `getUserId()` — JWT から user_id 取得ヘルパー |
| `.env` | `POSTGREST_URL`, `POSTGREST_API_KEY` 追加 |

---

### Phase 1: SQL リネーム + `auth.uid()` → `p_user_id`

全 RPC の SQL を一括変更するマイグレーションを作成。

**変更パターン:**

```sql
-- Before: auth.uid() で暗黙的にユーザー特定
CREATE OR REPLACE FUNCTION public.list_my_api_keys()
RETURNS TABLE (...) LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE v_user_id UUID := auth.uid();
BEGIN
    RETURN QUERY SELECT ... WHERE k.user_id = v_user_id;
END; $$;

-- After: p_user_id パラメータ + リネーム
CREATE OR REPLACE FUNCTION public.list_api_keys(p_user_id UUID)
RETURNS TABLE (...) LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    RETURN QUERY SELECT ... WHERE k.user_id = p_user_id;
END; $$;
-- 旧関数を DROP
DROP FUNCTION IF EXISTS public.list_my_api_keys();
```

```sql
-- Before: auth.uid() + _my_ プレフィックス
CREATE OR REPLACE FUNCTION public.generate_my_api_key(p_display_name TEXT, ...)
RETURNS JSONB LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE v_user_id UUID := auth.uid();
...

-- After: p_user_id パラメータ + リネーム
CREATE OR REPLACE FUNCTION public.generate_api_key(p_user_id UUID, p_display_name TEXT, ...)
RETURNS JSONB LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    -- auth.uid() の呼出を全て p_user_id に置換
    ...
END; $$;
DROP FUNCTION IF EXISTS public.generate_my_api_key(TEXT);
```

**変更対象 (RPC ごと):**

| 旧名 | 新名 | 変更種別 |
|------|------|---------|
| `get_user_prompts` | `get_prompts` | リネーム |
| `get_user_credential` | `get_credential` | リネーム |
| `upsert_user_credential` + `upsert_my_credential` | `upsert_credential` | 統合 + リネーム |
| `generate_my_api_key` | `generate_api_key` | リネーム + p_user_id 追加 |
| `upsert_my_tool_settings` | `upsert_tool_settings` | リネーム + p_user_id 追加 |
| `upsert_my_module_description` | `upsert_module_description` | リネーム + p_user_id 追加 |
| `upsert_my_prompt` | `upsert_prompt` | リネーム + p_user_id 追加 |
| `update_my_settings` | `update_settings` | リネーム + p_user_id 追加 |
| `get_my_usage` | `get_usage` | リネーム + p_user_id 追加 |
| `list_my_oauth_consents` | `list_oauth_consents` | リネーム + p_user_id 追加 |
| `revoke_my_oauth_consent` | `revoke_oauth_consent` | リネーム + p_user_id 追加 |
| `get_my_role` | — | **廃止** (`get_user_context` に統合) |
| `get_my_settings` | — | **廃止** (`get_user_context` に統合) |
| `get_my_tool_settings` | — | **廃止** (`get_module_config` に統合) |
| `get_my_module_descriptions` | — | **廃止** (`get_module_config` に統合) |
| `list_my_api_keys` | `list_api_keys` | リネーム + p_user_id 追加 |
| `revoke_my_api_key` | `revoke_api_key` | リネーム + p_user_id 追加 |
| `list_my_credentials` | `list_credentials` | リネーム + p_user_id 追加 |
| `delete_my_credential` | `delete_credential` | リネーム + p_user_id 追加 |
| `list_my_prompts` | `list_prompts` | リネーム + p_user_id 追加 |
| `get_my_prompt` | `get_prompt` | リネーム + p_user_id 追加 |
| `delete_my_prompt` | `delete_prompt` | リネーム + p_user_id 追加 |

**新設:**

| RPC | 処理 |
|-----|------|
| `get_module_config(p_user_id, p_module_name?)` | tool_settings + module_descriptions 一括取得 |

**拡張:**

| RPC | 追加フィールド |
|-----|--------------|
| `get_user_context` | role, settings, connected_count |

**成果物:**

| ファイル | 内容 |
|---------|------|
| `supabase/migrations/2026XXXX_rpc_rename_and_cleanup.sql` | 全 RPC リネーム・廃止・新設 |

**Go Server 側の同時変更:**

| ファイル | 変更 |
|---------|------|
| `internal/broker/user.go` | `get_user_prompts` → `get_prompts` |
| `internal/broker/token.go` | `get_user_credential` → `get_credential`, `upsert_user_credential` → `upsert_credential` |
| `internal/broker/user.go` | `get_user_context` のレスポンス型に role, settings 追加 |

---

### Phase 2: Console 画面単位で Server Action 化

Phase 1 の SQL 変更に合わせて、Console 側を画面単位で移行。

#### 2-1. API Key 画面

| 操作 | 現在 | 移行後 |
|------|------|--------|
| 一覧 | `supabase.rpc('list_my_api_keys')` | `rpc('list_api_keys', {p_user_id})` |
| 生成 | `supabase.rpc('generate_my_api_key')` | `rpc('generate_api_key', {p_user_id, ...})` |
| 失効 | Server Action → `supabase.rpc('revoke_my_api_key')` | `rpc('revoke_api_key', {p_user_id, ...})` |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/(console)/my/api-keys/actions.ts` | Server Action: listApiKeys, generateApiKey, revokeApiKey |
| `lib/api-keys.ts` | 削除 (actions.ts に統合) |

#### 2-2. サービス接続画面

| 操作 | 現在 | 移行後 |
|------|------|--------|
| 一覧 | `supabase.rpc('list_my_credentials')` | `rpc('list_credentials', {p_user_id})` |
| 保存 | `supabase.rpc('upsert_my_credential')` | `rpc('upsert_credential', {p_user_id, ...})` |
| 削除 | `supabase.rpc('delete_my_credential')` | `rpc('delete_credential', {p_user_id, ...})` |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/(console)/services/actions.ts` | Server Action: listCredentials, upsertCredential, deleteCredential |
| `lib/token-vault.ts` | Supabase SDK 依存を除去、Server Action 呼出に変更 |

#### 2-3. プロンプト画面

| 操作 | 現在 | 移行後 |
|------|------|--------|
| 一覧 | `supabase.rpc('list_my_prompts')` | `rpc('list_prompts', {p_user_id, ...})` |
| 取得 | `supabase.rpc('get_my_prompt')` | `rpc('get_prompt', {p_user_id, p_prompt_id})` |
| 作成/更新 | `supabase.rpc('upsert_my_prompt')` | `rpc('upsert_prompt', {p_user_id, ...})` |
| 削除 | `supabase.rpc('delete_my_prompt')` | `rpc('delete_prompt', {p_user_id, p_prompt_id})` |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/(console)/prompts/actions.ts` | Server Action: listPrompts, getPrompt, upsertPrompt, deletePrompt |
| `lib/prompts.ts` | 削除 (actions.ts に統合) |

#### 2-4. モジュール設定画面

| 操作 | 現在 | 移行後 |
|------|------|--------|
| 設定取得 | `supabase.rpc('get_my_tool_settings')` + `supabase.rpc('get_my_module_descriptions')` | `rpc('get_module_config', {p_user_id, ...})` |
| ツール設定更新 | `supabase.rpc('upsert_my_tool_settings')` | `rpc('upsert_tool_settings', {p_user_id, ...})` |
| 説明更新 | `supabase.rpc('upsert_my_module_description')` | `rpc('upsert_module_description', {p_user_id, ...})` |
| モジュール一覧 | `supabase.rpc('list_modules_with_tools')` | `rpc('list_modules_with_tools')` |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/(console)/modules/actions.ts` | Server Action: getModuleConfig, upsertToolSettings, upsertModuleDescription, listModules |
| `lib/tool-settings.ts` | 削除 (actions.ts に統合) |
| `lib/module-data.ts` | Supabase SDK 依存を除去 |

#### 2-5. ユーザー設定・利用状況

| 操作 | 現在 | 移行後 |
|------|------|--------|
| ログイン時 | `get_my_role` + `get_my_settings` | `rpc('get_user_context', {p_user_id})` 1回で済む |
| 設定更新 | Server Action → `supabase.rpc('update_my_settings')` | `rpc('update_settings', {p_user_id, ...})` |
| 利用統計 | `supabase.rpc('get_my_usage')` | `rpc('get_usage', {p_user_id, ...})` |
| プラン情報 | `supabase.rpc('get_user_context')` | 同上 (既に拡張済み) |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/(console)/my/settings/actions.ts` | Server Action: updateSettings, getUsage |
| `lib/user-settings.ts` | 削除 (actions.ts に統合) |
| `lib/plan.ts` | Supabase SDK 依存を除去、Server Action 呼出に変更 |
| `lib/auth-context.tsx` | `get_my_role` + `get_my_settings` → `get_user_context` 1回呼出に変更 |

#### 2-6. OAuth 同意画面

| 操作 | 現在 | 移行後 |
|------|------|--------|
| 一覧 | `supabase.rpc('list_my_oauth_consents')` | `rpc('list_oauth_consents', {p_user_id})` |
| 取消 | `supabase.rpc('revoke_my_oauth_consent')` | `rpc('revoke_oauth_consent', {p_user_id, ...})` |
| 全一覧 (admin) | `supabase.rpc('list_all_oauth_consents')` | Route Handler → `rpc('list_all_oauth_consents')` |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/(console)/my/oauth-consents/actions.ts` | Server Action: listOAuthConsents, revokeOAuthConsent |
| `app/api/admin/oauth-consents/route.ts` | Route Handler (admin) |
| `lib/oauth-consents.ts` | 削除 (actions.ts + route.ts に統合) |

#### 2-7. OAuth callback (Admin Route Handler)

| 操作 | 現在 | 移行後 |
|------|------|--------|
| アプリ認証情報取得 | `adminClient.rpc('get_oauth_app_credentials')` | `rpc('get_oauth_app_credentials', ...)` |
| トークン保存 | `supabase.rpc('upsert_my_credential')` | `rpc('upsert_credential', {p_user_id, ...})` |

**成果物:**

| ファイル | 変更 |
|---------|------|
| `app/api/oauth/*/route.ts` (全プロバイダ) | Supabase SDK → postgrest.rpc に変更 |

---

### Phase 3: Supabase SDK 完全除去

Phase 2 完了後、Console から Supabase SDK 依存を除去。

| # | タスク |
|---|--------|
| 3-1 | `lib/supabase/client.ts` 削除 |
| 3-2 | `lib/supabase/server.ts` 削除 |
| 3-3 | `lib/supabase/admin.ts` 削除 |
| 3-4 | `@supabase/supabase-js`, `@supabase/ssr` を devDependencies に移動 (型生成のみ) |
| 3-5 | RLS ポリシーを全削除 (全アクセスが service_role 経由) |

> 認証基盤移行 (Supabase Auth → Better Auth 等) は別スプリント。

---

## 実施順序と依存関係

```
Phase 0 (基盤: postgrest.ts, auth.ts)
  │
  ├─ Phase 1 (SQL: リネーム + auth.uid() 廃止 + Go Server 同時変更)
  │
  └─ Phase 2 (Console: 画面単位で移行)
       ├─ 2-1. API Key 画面
       ├─ 2-2. サービス接続画面
       ├─ 2-3. プロンプト画面
       ├─ 2-4. モジュール設定画面
       ├─ 2-5. ユーザー設定・利用状況
       ├─ 2-6. OAuth 同意画面
       └─ 2-7. OAuth callback
            │
            Phase 3 (Supabase SDK 除去)
```

Phase 1 と Phase 2 は同時に進める（SQL 変更 + Console 変更をセットでコミット）。
Phase 2 内の画面は独立しており、任意の順序で作業可能。

---

## 影響範囲サマリ

| カテゴリ | ファイル数 | 概要 |
|---------|-----------|------|
| SQL migration | 1 | 全 RPC リネーム・廃止・新設・p_user_id 化 |
| lib/postgrest.ts (新規) | 1 | PostgREST HTTP クライアント |
| lib/auth.ts (新規) | 1 | JWT → user_id 取得ヘルパー |
| Server Actions (新規) | 6 | 各画面の actions.ts |
| Route Handler (新規/変更) | 2 | admin 系 |
| lib/*.ts (削除) | 6 | api-keys, prompts, tool-settings, user-settings, oauth-consents, plan |
| lib/supabase/* (削除) | 3 | Phase 3 で除去 |
| Go Server (変更) | 2 | broker/user.go, broker/token.go (RPC名変更) |
| OAuth callback routes (変更) | 10+ | Supabase SDK → postgrest.rpc |

---

## 参考

- [ADR-005: RLS に依存しない認可設計](../decision/ADR-005-no-rls-dependency.md)
- [grh-rpc-design.canvas](../../docs/graph/grh-rpc-design.canvas) — RPC 設計図
