# DAY032: PostgREST → Drizzle ORM + Hyperdrive 移行計画

## 日付

2026-02-18

---

## 概要

Worker を PostgREST RPC プロキシから Drizzle ORM + Cloudflare Hyperdrive ベースの
アプリケーションサーバーに移行する。同時に URL パターン再構成・アプリ層暗号化・
OpenAPI spec リライト・Console/Go Server クライアント更新を実施する。

### 設計判断 (合意済み)

| 項目 | 決定 |
|------|------|
| DB Driver | Cloudflare Hyperdrive + Drizzle ORM + `pg` |
| URL Pattern | `/v1/me/*` (Console) + `/v1/users/{user_id}/*` (Go Server) |
| 暗号化 | アプリ層 AES-256-GCM + key versioning |
| vault.secrets | アプリ層暗号化に移行 (廃止) |
| OpenAPI spec | ルート実装と同時にリライト |
| Go Server 型生成 | ogen |
| リリース | Worker + Console + Go Server 同時デプロイ |

### 関連ドキュメント

- [dsn-restapi.md](../../docs/003_design/interface/dsn-restapi.md) — REST API 設計書
- [dsn-rpc.md](../../docs/003_design/interface/dsn-rpc.md) — 旧 RPC 関数設計書
- [day031-backlog.md](day031-backlog.md) — バックログ (#9 に該当)

---

## Phase 0: DB マイグレーション (後方互換、事前デプロイ可)

### 0A. `mcpist.users` に `email`, `role` カラム追加

**背景:** `get_user_context` RPC は `auth.users` を JOIN して email/role を取得する。
Drizzle via Hyperdrive は `auth.users` にアクセスできないため、`mcpist.users` に複製する。

**マイグレーション:** `supabase/migrations/2026MMDD000000_users_email_role.sql`

```sql
ALTER TABLE mcpist.users ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE mcpist.users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

-- 既存データのバックフィル
UPDATE mcpist.users u
SET email = au.email,
    role = COALESCE(au.raw_app_meta_data->>'role', 'user')
FROM auth.users au
WHERE u.id = au.id;

-- トリガー更新 (新規ユーザー作成時に email/role をコピー)
CREATE OR REPLACE FUNCTION mcpist.handle_new_user() ...
```

**検証:** `SELECT id, email, role FROM mcpist.users LIMIT 5` で確認

### 0B. `oauth_apps` に `encrypted_credentials`, `key_version` カラム追加

**マイグレーション:** `supabase/migrations/2026MMDD000001_oauth_apps_encrypted.sql`

```sql
ALTER TABLE mcpist.oauth_apps ADD COLUMN IF NOT EXISTS encrypted_credentials TEXT;
ALTER TABLE mcpist.oauth_apps ADD COLUMN IF NOT EXISTS key_version INTEGER DEFAULT 1;
```

### 0C. `user_credentials` に `encrypted_credentials`, `key_version` カラム追加

**マイグレーション:** `supabase/migrations/2026MMDD000002_user_credentials_encrypted.sql`

```sql
ALTER TABLE mcpist.user_credentials ADD COLUMN IF NOT EXISTS encrypted_credentials TEXT;
ALTER TABLE mcpist.user_credentials ADD COLUMN IF NOT EXISTS key_version INTEGER DEFAULT 1;
```

### 0D. スキーマクリーンアップ

**背景:** `drizzle-kit introspect` で auto-generate したところ、以下の汚れが発覚:

| 問題 | 原因 | 対処 |
|------|------|------|
| `usage_log` のインデックス名が `idx_credit_transactions_*` | テーブルリネーム時にインデックスはリネームされなかった | `ALTER INDEX ... RENAME TO` |
| チェック制約 `credit_transactions_meta_tool_check` | 同上 | `ALTER TABLE ... RENAME CONSTRAINT` |
| FK 名 `credit_transactions_user_id_fkey` | 同上 | `ALTER TABLE ... RENAME CONSTRAINT` |
| `oauth_apps.secret_id` → `vault.secrets` FK | cross-schema FK が introspect で vault スキーマを巻き込む | FK を DROP (アプリ層暗号化に移行済み) |
| `users.id` → `auth.users` FK | Hyperdrive から auth スキーマにアクセス不可。introspect が self-FK と誤認 | FK を DROP (トリガーで整合性保証) |
| `module_settings.description` の `DEFAULT ''` | drizzle-kit が空文字列のエスケープに失敗するバグ | 手書き schema.ts で対処 (DB 側は変更不要) |

**マイグレーション:** `supabase/migrations/20260218200000_schema_cleanup.sql`

```sql
-- usage_log: rename indexes from credit_transactions era
ALTER INDEX mcpist.idx_credit_transactions_user_id RENAME TO idx_usage_log_user_id;
ALTER INDEX mcpist.idx_credit_transactions_created_at RENAME TO idx_usage_log_created_at;
-- ... (6 indexes total)

-- usage_log: rename constraint + FK
ALTER TABLE mcpist.usage_log RENAME CONSTRAINT credit_transactions_meta_tool_check TO usage_log_meta_tool_check;
ALTER TABLE mcpist.usage_log RENAME CONSTRAINT credit_transactions_user_id_fkey TO usage_log_user_id_fkey;

-- Drop cross-schema FKs
ALTER TABLE mcpist.oauth_apps DROP CONSTRAINT IF EXISTS oauth_apps_secret_id_fkey;
ALTER TABLE mcpist.users DROP CONSTRAINT IF EXISTS users_id_fkey;
```

**introspect 結果の検証:**
- `vault.secrets`, `auth.users` への参照が消えた
- `relations.ts` から `secretsInVault`, `usersInAuth` の import が消えた
- FK 数: 13 → 11 (vault, auth の 2 つが除去)

**残存する drizzle-kit バグ:**
- `module_settings.description` の `DEFAULT ''` → `.default(')` と生成される (構文エラー)
- 手書き `schema.ts` では `.default("")` で正しく定義

### 0E. 暗号化移行スクリプト (ワンタイム)

Phase 0A-0D デプロイ後に実行:

1. `user_credentials.credentials` (平文) → AES-256-GCM で暗号化 → `encrypted_credentials` に保存
2. `vault.decrypted_secrets` から OAuth app credentials を読み出し → 暗号化 → `oauth_apps.encrypted_credentials` に保存

**スクリプト:** `scripts/migrate-encryption.ts`

**検証:** 両テーブルで `encrypted_credentials IS NOT NULL` を確認

---

## Phase 1: Worker インフラ層

### 1A. 依存パッケージ追加

**ファイル:** `apps/worker/package.json`

```
drizzle-orm, pg, @types/pg (devDep)
```

### 1B. Hyperdrive バインディング設定

**ファイル:** `apps/worker/wrangler.toml`

```toml
compatibility_flags = ["nodejs_compat"]

[[hyperdrive]]
binding = "HYPERDRIVE"
id = "<hyperdrive-config-id>"
```

**ファイル:** `apps/worker/src/types.ts` — Env インターフェース更新

```typescript
export interface Env {
  HYPERDRIVE: Hyperdrive;
  ENCRYPTION_KEY: string;       // base64 AES-256 key
  ENCRYPTION_KEY_V2?: string;   // ローテーション用
  // POSTGREST_URL, POSTGREST_API_KEY は Phase 5 で削除
  // ... 既存バインディング
}
```

### 1C. Drizzle スキーマ定義

**新規ファイル:** `apps/worker/src/db/schema.ts`

**方針:** `drizzle-kit introspect` で DB から自動生成 → 手書きで整形
- 自動生成で正確なカラム定義・FK・インデックスを取得
- 手書きで export 名の整理 (`xxxInMcpist` → `xxx`)、pgPolicy 除去、
  drizzle-kit バグ修正 (`module_settings.description` の空文字列デフォルト)

12 テーブルを定義:
- `users`, `plans`, `modules`, `module_settings`, `tool_settings`
- `prompts`, `api_keys`, `user_credentials`, `oauth_apps`
- `usage_log`, `processed_webhook_events`, `admin_emails`

`pgSchema('mcpist')` で mcpist スキーマを使用。

### 1D. DB 接続ファクトリ

**新規ファイル:** `apps/worker/src/db/client.ts`

```typescript
import { drizzle } from 'drizzle-orm/node-postgres';
import pg from 'pg';

export function createDb(env: Env) {
  const pool = new pg.Pool({
    connectionString: env.HYPERDRIVE.connectionString,
  });
  return drizzle(pool, { schema });
}
```

### 1E. 暗号化モジュール

**新規ファイル:** `apps/worker/src/crypto/encryption.ts`

- AES-256-GCM (Web Crypto API)
- Key versioning: `{v: number, iv: string, ct: string, tag: string}`
- `encrypt(plaintext, key, keyVersion)` → 暗号化 JSON 文字列
- `decrypt(encryptedJson, keys)` → 平文
- 鍵は Cloudflare Worker secrets (base64 エンコード 32 bytes)

### 1F. クエリ層 (ドメイン別)

**新規ディレクトリ:** `apps/worker/src/db/queries/`

| ファイル | 関数 | 対応する旧 RPC |
|---------|------|---------------|
| `users.ts` | `getMyProfile`, `getUserContext`, `updateSettings`, `completeOnboarding`, `getStripeCustomerId`, `linkStripeCustomer`, `getUserByStripeCustomer`, `lookupUserByKeyHash` | get_user_context, update_settings, etc. |
| `credentials.ts` | `listCredentials`, `getCredential`, `upsertCredential`, `deleteCredential` | get_credential, upsert_credential, etc. |
| `apikeys.ts` | `listApiKeys`, `generateApiKey`, `revokeApiKey` | list_api_keys, generate_api_key, revoke_api_key |
| `prompts.ts` | `listPrompts`, `getPrompt`, `createPrompt`, `updatePrompt`, `deletePrompt`, `getUserPrompts` | list_prompts, get_prompt, upsert_prompt, etc. |
| `modules.ts` | `listModulesWithTools`, `getModuleConfig`, `syncModules`, `upsertToolSettings`, `upsertModuleDescription` | list_modules_with_tools, sync_modules, etc. |
| `oauth.ts` | `getOAuthAppCredentials`, `upsertOAuthApp`, `listOAuthApps`, `deleteOAuthApp`, `listOAuthConsents`, `revokeOAuthConsent`, `listAllOAuthConsents` | get_oauth_app_credentials, etc. |
| `usage.ts` | `getUsage`, `recordUsage` | get_usage, record_usage |
| `stripe.ts` | `activateSubscription` | activate_subscription |

**重要な変更点:**
- `getMyProfile` / `getUserContext`: `auth.users` JOIN 不要 → `mcpist.users.email/role` を直接参照
- `getCredential` / `upsertCredential`: アプリ層暗号化/復号
- `getOAuthAppCredentials`: `vault.decrypted_secrets` → `oauth_apps.encrypted_credentials` から復号
- `generateApiKey`: `pgcrypto` → `crypto.getRandomValues` + `crypto.subtle.digest`

---

## Phase 2: ルート実装 + OpenAPI spec リライト

### 2A. OpenAPI spec 全面リライト

**ファイル:** `apps/worker/src/openapi.yaml`

dsn-restapi.md に準拠。主な変更:
- `/v1/user/*` → `/v1/me/*` (24 エンドポイント)
- 新規 `/v1/users/{user_id}/*` (5 エンドポイント)
- `/v1/credentials` → `/v1/me/credentials/{module}`
- `/v1/api-keys` → `/v1/me/apikeys`
- `PUT /v1/prompts` (upsert) → `POST /v1/me/prompts` + `PUT /v1/me/prompts/{id}`
- エラーレスポンス統一: `{ error: { code, message } }`

### 2B. 新規ルートファイル

**`apps/worker/src/v1/routes/me.ts`** — Console 用 (bearerAuth), 24 エンドポイント

```
GET    /me/profile
PUT    /me/settings
POST   /me/onboarding
GET    /me/usage
GET    /me/stripe
PUT    /me/stripe
GET    /me/credentials
PUT    /me/credentials/:module
DELETE /me/credentials/:module
GET    /me/apikeys
POST   /me/apikeys
DELETE /me/apikeys/:id
GET    /me/prompts
GET    /me/prompts/:id
POST   /me/prompts
PUT    /me/prompts/:id
DELETE /me/prompts/:id
GET    /me/modules/config
PUT    /me/modules/:name/tools
PUT    /me/modules/:name/description
GET    /me/oauth/consents
DELETE /me/oauth/consents/:id
```

**`apps/worker/src/v1/routes/users.ts`** — Go Server 用 (gatewaySecret), 5 エンドポイント

```
GET    /users/:userId/context
GET    /users/:userId/credentials/:module
PUT    /users/:userId/credentials/:module
POST   /users/:userId/usage
GET    /users/:userId/prompts
```

### 2C. 既存ルートファイル更新

| ファイル | 変更 |
|---------|------|
| `modules.ts` | `/modules/config`, `/modules/:name/*` 削除 (me.ts へ移動)。2 エンドポイントに縮小 |
| `oauth.ts` | `/oauth/consents` 削除 (me.ts へ移動)。1 エンドポイントに縮小 |
| `admin.ts` | Drizzle 化 + `PUT /admin/oauth/apps` → `PUT /admin/oauth/apps/:provider` (パスパラメータ化) |
| `stripe.ts` | Drizzle 化 (`callPostgRESTRpc` → クエリ関数) |

### 2D. ルートマウント更新

**ファイル:** `apps/worker/src/v1/index.ts`

```typescript
v1.route("/me", me);           // 新規
v1.route("/users", users);     // 新規
v1.route("/modules", modules); // 更新
v1.route("/oauth", oauth);     // 更新
v1.route("/admin", admin);     // 更新
v1.route("/stripe", stripe);   // 更新
```

### 2E. Auth モジュール更新

**ファイル:** `apps/worker/src/auth.ts`

- `verifyApiKey`: PostgREST RPC → Drizzle `lookupUserByKeyHash` クエリ
- KV キャッシュは維持

### 2F. 削除ファイル

- `apps/worker/src/v1/postgrest.ts`
- `apps/worker/src/v1/routes/user.ts`
- `apps/worker/src/v1/routes/credentials.ts`
- `apps/worker/src/v1/routes/api-keys.ts`
- `apps/worker/src/v1/routes/prompts.ts`

---

## Phase 3: Console クライアント更新

### 3A. 型再生成

```bash
npx openapi-typescript <worker-url>/openapi.json -o src/lib/worker/types.ts
```

### 3B. Consumer ファイル更新 (~25 ファイル)

| ファイル | 旧パス → 新パス |
|---------|---------------|
| `lib/auth/auth-context-actions.ts` | `/v1/user/context` → `/v1/me/profile` |
| `lib/billing/plan.ts` | `/v1/user/context`, `/v1/user/usage` → `/v1/me/profile`, `/v1/me/usage` |
| `lib/settings/user-settings.ts` | `/v1/user/settings` → `/v1/me/settings` |
| `lib/services/token-vault-actions.ts` | `/v1/credentials` → `/v1/me/credentials/{module}` |
| `lib/mcp/api-keys.ts` | `/v1/api-keys` → `/v1/me/apikeys` |
| `lib/mcp/prompts.ts` | `/v1/prompts` → `/v1/me/prompts` (create/update 分離) |
| `lib/mcp/tool-settings.ts` | `/v1/modules/{name}/*` → `/v1/me/modules/{name}/*` |
| `lib/oauth/consents.ts` | `/v1/oauth/consents` → `/v1/me/oauth/consents` |
| `app/api/stripe/{checkout,portal}/route.ts` | user context パス更新 |
| `app/auth/callback/route.ts` | onboarding パス更新 |
| `app/api/oauth/*/callback/route.ts` (11 files) | credential upsert パス更新 |

**スキーマ変更の注意点:**
- PostgREST TABLE 戻り値 (配列) → Drizzle (単一オブジェクト): `data![0]` → `data!`
- `upsertCredential`: body の `module` → path パラメータ
- `upsertPrompt` → `createMyPrompt` (POST) + `updateMyPrompt` (PUT)

---

## Phase 4: Go Server broker 書き換え

### 4A. ogen クライアント生成

```bash
mkdir -p apps/server/pkg/workerapi/gen
ogen --target pkg/workerapi/gen --clean --package gen ../../apps/worker/src/openapi.yaml
```

### 4B. 環境変数変更

| 旧 | 新 |
|----|-----|
| `POSTGREST_URL` | `WORKER_URL` |
| `POSTGREST_API_KEY` | `GATEWAY_SECRET` |

### 4C. broker 書き換え

**`internal/broker/user.go`:**

| 旧 PostgREST RPC | 新 Worker REST |
|------------------|---------------|
| `POST /rpc/get_user_context` | `GET /v1/users/{id}/context` |
| `POST /rpc/get_prompts` | `GET /v1/users/{id}/prompts` |
| `POST /rpc/record_usage` | `POST /v1/users/{id}/usage` |
| `POST /rpc/sync_modules` | `POST /v1/modules/sync` |
| `HEAD /` (health) | `GET /health` |

**`internal/broker/token.go`:**

| 旧 PostgREST RPC | 新 Worker REST |
|------------------|---------------|
| `POST /rpc/get_credential` | `GET /v1/users/{id}/credentials/{module}` |
| `POST /rpc/upsert_credential` | `PUT /v1/users/{id}/credentials/{module}` |
| `POST /rpc/get_oauth_app_credentials` | `GET /v1/oauth/apps/{provider}/credentials` |

**ヘッダー変更:**
- 旧: `apikey: <key>`, `Authorization: Bearer <key>`
- 新: `X-Gateway-Secret: <secret>`

---

## Phase 5: デプロイ + クリーンアップ

### 5A. デプロイ順序

1. DB マイグレーション (Phase 0) — 事前デプロイ
2. 暗号化移行スクリプト実行 (Phase 0D)
3. Hyperdrive 設定: `wrangler hyperdrive create mcpist-db --connection-string="..."`
4. Worker secrets: `wrangler secret put ENCRYPTION_KEY`
5. **同時デプロイ: Worker + Console + Go Server**

### 5B. デプロイ後検証

1. `GET /health` — ヘルスチェック
2. `GET /v1/modules` — パブリックエンドポイント
3. Console ログイン → `GET /v1/me/profile`
4. API キー作成 → 使用 (KV キャッシュ確認)
5. Credential 暗号化/復号サイクル
6. Stripe webhook テストイベント
7. Go Server 起動 → SyncModules ログ確認
8. Go Server ツール実行 → GetModuleToken + RecordUsage

### 5C. クリーンアップ (別 PR)

1. **DB クリーンアップマイグレーション:**
   - `user_credentials.credentials` カラム (平文) 削除
   - `oauth_apps.secret_id` カラム削除
   - vault.secrets 参照削除
   - 全 PostgREST RPC 関数削除
   - RLS ポリシー整理

2. **Worker クリーンアップ:**
   - `POSTGREST_URL`, `POSTGREST_API_KEY` 削除

3. **Go Server クリーンアップ:**
   - 旧環境変数ハンドリング削除

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| auth.users アクセス不可 | Phase 0A で email/role を mcpist.users に複製 + トリガー更新 |
| 暗号化キー喪失 | Cloudflare secret + セキュアバックアップに二重保管 |
| get_user_context の複雑なクエリ再現 | Drizzle クエリ関数に対する統合テスト |
| Hyperdrive 接続制限 | pg.Pool の max 設定、接続プール使用量モニタリング |
| vault.secrets 移行 | カットオーバー前に暗号化移行スクリプトを実行・検証 |
| レスポンス形式変更 | PostgREST TABLE (配列) → Drizzle (オブジェクト)。Console の `data![0]` パターンを全て更新 |
| API キー生成 | pgcrypto → Web Crypto API。SHA-256 ハッシュの一貫性を検証 |
| 同時デプロイ失敗 | ロールバック計画: Worker を前バージョンに戻す |

---

## ファイル一覧

### 新規作成 (19 ファイル)

| # | ファイル | 内容 |
|---|---------|------|
| 1 | `apps/worker/src/db/schema.ts` | Drizzle スキーマ (12 テーブル, introspect ベース) |
| 2 | `apps/worker/src/db/client.ts` | DB 接続ファクトリ |
| 3 | `apps/worker/src/crypto/encryption.ts` | AES-256-GCM モジュール |
| 4 | `apps/worker/src/db/queries/users.ts` | ユーザー関連クエリ |
| 5 | `apps/worker/src/db/queries/credentials.ts` | 認証情報クエリ (暗号化) |
| 6 | `apps/worker/src/db/queries/apikeys.ts` | API キークエリ |
| 7 | `apps/worker/src/db/queries/prompts.ts` | プロンプトクエリ |
| 8 | `apps/worker/src/db/queries/modules.ts` | モジュールクエリ |
| 9 | `apps/worker/src/db/queries/oauth.ts` | OAuth クエリ |
| 10 | `apps/worker/src/db/queries/usage.ts` | 使用量クエリ |
| 11 | `apps/worker/src/db/queries/stripe.ts` | Stripe クエリ |
| 12 | `apps/worker/src/v1/routes/me.ts` | Console 用 24 エンドポイント |
| 13 | `apps/worker/src/v1/routes/users.ts` | Go Server 用 5 エンドポイント |
| 14 | `apps/server/pkg/workerapi/client.go` | Worker API クライアントラッパー |
| 15 | `apps/server/pkg/workerapi/gen/` | ogen 生成クライアント |
| 16 | `supabase/migrations/2026MMDD000000_users_email_role.sql` | email/role 複製 |
| 17 | `supabase/migrations/2026MMDD000001_oauth_apps_encrypted.sql` | 暗号化カラム追加 |
| 18 | `supabase/migrations/2026MMDD000002_user_credentials_encrypted.sql` | 暗号化カラム追加 |
| 19 | `scripts/migrate-encryption.ts` | ワンタイム暗号化移行 |

### 変更 (~35 ファイル)

| # | ファイル | 変更内容 |
|---|---------|---------|
| 20 | `apps/worker/package.json` | drizzle-orm, pg 追加 |
| 21 | `apps/worker/wrangler.toml` | Hyperdrive バインディング |
| 22 | `apps/worker/src/types.ts` | HYPERDRIVE, ENCRYPTION_KEY |
| 23 | `apps/worker/src/openapi.yaml` | 全面リライト |
| 24 | `apps/worker/src/v1/index.ts` | ルートマウント更新 |
| 25 | `apps/worker/src/v1/routes/modules.ts` | 2 エンドポイントに縮小 |
| 26 | `apps/worker/src/v1/routes/oauth.ts` | 1 エンドポイントに縮小 |
| 27 | `apps/worker/src/v1/routes/admin.ts` | Drizzle 化 + パスパラメータ化 |
| 28 | `apps/worker/src/v1/routes/stripe.ts` | Drizzle 化 |
| 29 | `apps/worker/src/auth.ts` | Drizzle で API キー検証 |
| 30 | `apps/console/src/lib/worker/types.ts` | 再生成 |
| 31-41 | `apps/console/src/lib/**/*.ts` (11 files) | パス更新 |
| 42-52 | `apps/console/src/app/api/**/*.ts` (~11 files) | パス更新 |
| 53 | `apps/server/cmd/server/main.go` | 環境変数変更 |
| 54 | `apps/server/internal/broker/user.go` | Worker REST 呼び出し |
| 55 | `apps/server/internal/broker/token.go` | Worker REST 呼び出し |

### 削除 (5 ファイル)

| # | ファイル |
|---|---------|
| 56 | `apps/worker/src/v1/postgrest.ts` |
| 57 | `apps/worker/src/v1/routes/user.ts` |
| 58 | `apps/worker/src/v1/routes/credentials.ts` |
| 59 | `apps/worker/src/v1/routes/api-keys.ts` |
| 60 | `apps/worker/src/v1/routes/prompts.ts` |
