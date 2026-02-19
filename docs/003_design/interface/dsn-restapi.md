# REST API 設計書 (dsn-restapi)

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Worker REST API Design — PostgREST 廃止・Drizzle ORM 導入に向けた API 再設計 |

---

## 1. 概要

本ドキュメントは、MCPist Worker REST API の設計を定義する。
PostgREST RPC プロキシから Drizzle ORM ベースのアプリケーションサーバーへの移行に伴い、
API の URL 設計・認証方式・レスポンス構造を整理する。

### 1.1 設計方針

1. **クライアント分離**: Console (エンドユーザー) と Go Server (サーバー間) の API を URL パターンで明確に分離する
2. **RESTful**: リソース中心の URL 設計。RPC 的な命名を避ける
3. **OpenAPI 駆動**: openapi.yaml を正とし、Console (openapi-fetch) と Go Server (ogen) の型を自動生成する
4. **単一バージョン**: `/v1/` のまま改修。同時リリースで破壊的変更を許容する

### 1.2 クライアント

| クライアント | 認証方式 | URL パターン | 用途 |
|-------------|---------|------------|------|
| Console (Next.js) | Bearer JWT / API Key | `/v1/me/*` | ユーザー自身のリソース管理 |
| Go Server (MCP) | X-Gateway-Secret | `/v1/users/{user_id}/*` | 指定ユーザーのリソース参照 |
| Stripe | Stripe-Signature | `/v1/stripe/*` | Webhook 受信 |
| Public | なし | `/v1/modules`, `/v1/oauth/apps/*/credentials` | カタログ・OAuth 設定 |

### 1.3 URL 設計原則

```
/v1/me/*                     — Console: 「自分自身」のリソース (bearerAuth)
/v1/users/{user_id}/*        — Go Server: 指定ユーザーのリソース (gatewaySecret)
/v1/modules/*                — モジュールカタログ (public / bearerAuth / gatewaySecret)
/v1/oauth/*                  — OAuth 関連 (public / bearerAuth)
/v1/admin/*                  — 管理者専用 (bearerAuth + admin role)
/v1/stripe/*                 — Stripe Webhook (Stripe 署名検証)
```

**`/v1/me/` vs `/v1/users/{user_id}/` の使い分け:**

- `/v1/me/` — 認証トークンから user_id を解決。ブラウザ/CLI からのアクセス
- `/v1/users/{user_id}/` — パスパラメータで user_id を明示。サーバー間通信専用
- Gateway auth (`X-Gateway-Secret`) がない限り `/v1/users/*` は 401 を返す

### 1.4 認証方式

| 認証方式 | ヘッダー | 対象 |
|---------|--------|------|
| bearerAuth | `Authorization: Bearer <jwt \| mpt_*>` | Console, MCP クライアント |
| gatewaySecret | `X-Gateway-Secret: <secret>` | Go Server → Worker |
| stripe | `Stripe-Signature: <sig>` | Stripe → Worker |
| (なし) | — | Public エンドポイント |

---

## 2. エンドポイント一覧

### 2.1 Console 向け — `/v1/me/*` (bearerAuth)

#### プロフィール・設定

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/profile` | getMyProfile | ユーザープロフィール (旧 get_user_context の Console 部分) |
| PUT | `/v1/me/settings` | updateMySettings | ユーザー設定更新 |
| POST | `/v1/me/onboarding` | completeOnboarding | オンボーディング完了 |

#### 使用量・課金

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/usage` | getMyUsage | 使用量統計 (`?start=&end=`) |
| GET | `/v1/me/stripe` | getMyStripeCustomerId | Stripe 顧客 ID 取得 |
| PUT | `/v1/me/stripe` | linkMyStripeCustomer | Stripe 顧客 ID 紐付け |

#### 認証情報 (Credentials)

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/credentials` | listMyCredentials | 接続済みサービス一覧 |
| PUT | `/v1/me/credentials/{module}` | upsertMyCredential | 認証情報の登録/更新 |
| DELETE | `/v1/me/credentials/{module}` | deleteMyCredential | 認証情報の削除 |

#### API キー

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/apikeys` | listMyApikeys | API キー一覧 |
| POST | `/v1/me/apikeys` | createMyApikey | API キー生成 |
| DELETE | `/v1/me/apikeys/{id}` | revokeMyApikey | API キー失効 |

#### プロンプト

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/prompts` | listMyPrompts | プロンプト一覧 (`?module=`) |
| GET | `/v1/me/prompts/{id}` | getMyPrompt | プロンプト詳細 |
| POST | `/v1/me/prompts` | createMyPrompt | プロンプト作成 |
| PUT | `/v1/me/prompts/{id}` | updateMyPrompt | プロンプト更新 |
| DELETE | `/v1/me/prompts/{id}` | deleteMyPrompt | プロンプト削除 |

#### モジュール設定

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/modules/config` | getMyModuleConfig | モジュール設定 (`?module=`) |
| PUT | `/v1/me/modules/{name}/tools` | updateMyToolSettings | ツール有効/無効設定 |
| PUT | `/v1/me/modules/{name}/description` | updateMyModuleDescription | モジュール説明更新 |

#### OAuth 同意

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/me/oauth/consents` | listMyOAuthConsents | 同意一覧 |
| DELETE | `/v1/me/oauth/consents/{id}` | revokeMyOAuthConsent | 同意取消 |

### 2.2 Go Server 向け — `/v1/users/{user_id}/*` (gatewaySecret)

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| GET | `/v1/users/{user_id}/context` | getUserContext | ユーザー認可コンテキスト (軽量) |
| GET | `/v1/users/{user_id}/credentials/{module}` | getUserCredential | 認証情報取得 (復号済み) |
| PUT | `/v1/users/{user_id}/credentials/{module}` | upsertUserCredential | 認証情報更新 (トークンリフレッシュ後) |
| POST | `/v1/users/{user_id}/usage` | recordUserUsage | 使用量記録 |
| GET | `/v1/users/{user_id}/prompts` | getUserPrompts | 有効プロンプト取得 (`?name=`) |

### 2.3 モジュール — `/v1/modules/*`

| Method | Path | Auth | operationId | 説明 |
|--------|------|------|-------------|------|
| GET | `/v1/modules` | public | listModules | モジュールカタログ (ツール定義含む) |
| POST | `/v1/modules/sync` | gatewaySecret | syncModules | モジュール定義同期 (Go Server 起動時) |

### 2.4 OAuth — `/v1/oauth/*`

| Method | Path | Auth | operationId | 説明 |
|--------|------|------|-------------|------|
| GET | `/v1/oauth/apps/{provider}/credentials` | public | getOAuthAppCredentials | OAuth アプリ設定 (client_id 等) |

### 2.5 管理者 — `/v1/admin/*` (bearerAuth + admin role)

| Method | Path                              | operationId          | 説明             |
| ------ | --------------------------------- | -------------------- | -------------- |
| GET    | `/v1/admin/oauth/apps`            | listOAuthApps        | OAuth アプリ一覧    |
| PUT    | `/v1/admin/oauth/apps/{provider}` | upsertOAuthApp       | OAuth アプリ登録/更新 |
| DELETE | `/v1/admin/oauth/apps/{provider}` | deleteOAuthApp       | OAuth アプリ削除    |
| GET    | `/v1/admin/oauth/consents`        | listAllOAuthConsents | 全ユーザーの同意一覧     |

### 2.6 Webhook — `/v1/stripe/*` (Stripe 署名検証)

| Method | Path | operationId | 説明 |
|--------|------|-------------|------|
| POST | `/v1/stripe/webhook` | handleStripeWebhook | Stripe イベント処理 |

### 2.7 システム (バージョンなし)

| Method | Path | 説明 |
|--------|------|------|
| GET | `/health` | ヘルスチェック |
| GET | `/openapi.json` | OpenAPI 仕様 |
| GET | `/.well-known/oauth-protected-resource` | OAuth リソースメタデータ (RFC 9728) |
| GET | `/.well-known/oauth-authorization-server` | OAuth 認可サーバーメタデータ (RFC 8414) |
| POST | `/v1/mcp/{path}` | MCP トランスポート (Go Server プロキシ) |

---

## 3. スキーマ概要

### 3.1 ユーザーコンテキストの分割

**旧:** `get_user_context` (Console と Go Server が同じデータを取得)

**新:** 2つに分割

#### GET /v1/me/profile — Console 用 (フルプロフィール)

```typescript
{
  user_id: string           // UUID
  email: string
  role: "user" | "admin"
  display_name: string | null
  account_status: string    // "active" | "suspended" | ...
  plan_id: string           // "free" | "plus"
  daily_used: number
  daily_limit: number
  connected_count: number   // 接続済みサービス数
  settings: object | null   // ユーザー設定 JSON
  module_descriptions: Record<string, string>  // カスタム説明
}
```

#### GET /v1/users/{user_id}/context — Go Server 用 (認可コンテキスト)

```typescript
{
  account_status: string
  plan_id: string
  daily_used: number
  daily_limit: number
  enabled_modules: string[]              // ["github", "notion", ...]
  enabled_tools: Record<string, string[]> // {"github": ["list_repos", ...]}
  language: string                        // BCP47 ("en-US")
  module_descriptions: Record<string, string>
}
```

### 3.2 認証情報 (Credentials)

#### GET /v1/me/credentials — Console 用 (メタデータのみ)

```typescript
[{
  module: string
  created_at: string   // ISO 8601
  updated_at: string
}]
```

#### GET /v1/users/{user_id}/credentials/{module} — Go Server 用 (復号済み)

```typescript
{
  found: boolean
  user_id: string
  service: string
  auth_type: string     // "oauth2" | "api_key" | "basic" | ...
  credentials: {
    access_token?: string
    refresh_token?: string
    expires_at?: string | number
    api_key?: string
    // ... 認証方式に依存するフィールド
  }
  metadata?: object
  error?: string
}
```

### 3.3 プロンプト

#### GET /v1/me/prompts — Console 用 (管理画面用、全件)

```typescript
[{
  id: string            // UUID
  module_name: string | null
  name: string
  description: string | null
  content: string
  enabled: boolean
  created_at: string
  updated_at: string
}]
```

#### GET /v1/users/{user_id}/prompts — Go Server 用 (有効のみ)

```typescript
[{
  id: string
  name: string
  description: string | null
  content: string
  enabled: boolean       // 常に true
}]
```

### 3.4 使用量記録

#### POST /v1/users/{user_id}/usage — Go Server → Worker

```typescript
// Request
{
  meta_tool: string           // "run" | "batch"
  request_id: string
  details: [{
    task_id?: string
    module: string
    tool: string
  }]
}
```

### 3.5 モジュール同期

#### POST /v1/modules/sync — Go Server → Worker

```typescript
// Request
{
  modules: [{
    name: string
    status: string
    descriptions: Record<string, string>
    tools: [{
      name: string
      description: string
      input_schema: object
    }]
  }]
}

// Response
{
  success: boolean
  upserted: number
  total: number
}
```

### 3.6 プロンプトの CRUD 分離

**旧:** `PUT /v1/prompts` (body の `prompt_id` 有無で create/update を分岐)

**新:**
- `POST /v1/me/prompts` — 作成 (prompt_id なし)
- `PUT /v1/me/prompts/{id}` — 更新 (prompt_id は path)

### 3.7 Credentials の module パス化

**旧:** `PUT /v1/credentials` (body に `module` フィールド)

**新:** `PUT /v1/me/credentials/{module}` (path パラメータ)

### 3.8 Admin OAuth Apps のパス化

**旧:** `PUT /v1/admin/oauth/apps` (body に `provider` フィールド)

**新:** `PUT /v1/admin/oauth/apps/{provider}` (path パラメータ)

---

## 4. RPC 関数対応表

### Console (bearerAuth) → Worker

| 旧 RPC | 旧エンドポイント | 新エンドポイント |
|--------|----------------|----------------|
| get_user_context | GET /v1/user/context | GET /v1/me/profile |
| get_usage | GET /v1/user/usage | GET /v1/me/usage |
| get_stripe_customer_id | GET /v1/user/stripe | GET /v1/me/stripe |
| link_stripe_customer | PUT /v1/user/stripe | PUT /v1/me/stripe |
| update_settings | PUT /v1/user/settings | PUT /v1/me/settings |
| complete_user_onboarding | POST /v1/user/onboarding | POST /v1/me/onboarding |
| list_credentials | GET /v1/credentials | GET /v1/me/credentials |
| upsert_credential | PUT /v1/credentials | PUT /v1/me/credentials/{module} |
| delete_credential | DELETE /v1/credentials/{module} | DELETE /v1/me/credentials/{module} |
| list_api_keys | GET /v1/api-keys | GET /v1/me/apikeys |
| generate_api_key | POST /v1/api-keys | POST /v1/me/apikeys |
| revoke_api_key | DELETE /v1/api-keys/{id} | DELETE /v1/me/apikeys/{id} |
| list_prompts | GET /v1/prompts | GET /v1/me/prompts |
| get_prompt | GET /v1/prompts/{id} | GET /v1/me/prompts/{id} |
| upsert_prompt (create) | PUT /v1/prompts | POST /v1/me/prompts |
| upsert_prompt (update) | PUT /v1/prompts | PUT /v1/me/prompts/{id} |
| delete_prompt | DELETE /v1/prompts/{id} | DELETE /v1/me/prompts/{id} |
| get_module_config | GET /v1/modules/config | GET /v1/me/modules/config |
| upsert_tool_settings | PUT /v1/modules/{name}/tools | PUT /v1/me/modules/{name}/tools |
| upsert_module_description | PUT /v1/modules/{name}/description | PUT /v1/me/modules/{name}/description |
| list_oauth_consents | GET /v1/oauth/consents | GET /v1/me/oauth/consents |
| revoke_oauth_consent | DELETE /v1/oauth/consents/{id} | DELETE /v1/me/oauth/consents/{id} |

### Go Server (gatewaySecret) → Worker

| 旧 RPC | 旧エンドポイント (PostgREST 直接) | 新エンドポイント |
|--------|-------------------------------|----------------|
| get_user_context | POST /rpc/get_user_context | GET /v1/users/{user_id}/context |
| get_credential | POST /rpc/get_credential | GET /v1/users/{user_id}/credentials/{module} |
| upsert_credential | POST /rpc/upsert_credential | PUT /v1/users/{user_id}/credentials/{module} |
| record_usage | POST /rpc/record_usage | POST /v1/users/{user_id}/usage |
| get_prompts | POST /rpc/get_prompts | GET /v1/users/{user_id}/prompts |
| sync_modules | POST /rpc/sync_modules | POST /v1/modules/sync |
| get_oauth_app_credentials | POST /rpc/get_oauth_app_credentials | GET /v1/oauth/apps/{provider}/credentials |

### 変更なし

| エンドポイント | 備考 |
|--------------|------|
| GET /v1/modules | public — 変更なし |
| POST /v1/stripe/webhook | Stripe 署名 — 変更なし |
| GET /v1/admin/oauth/apps | admin — 変更なし |
| DELETE /v1/admin/oauth/apps/{provider} | admin — 変更なし |
| GET /v1/admin/oauth/consents | admin — 変更なし |

---

## 5. 移行影響

### 5.1 Console (Next.js) — URL 変更

| 変更種別 | 件数 | 内容 |
|---------|------|------|
| `/v1/user/*` → `/v1/me/*` | 7 | profile, usage, stripe, settings, onboarding |
| `/v1/credentials` → `/v1/me/credentials` | 3 | list, upsert (+ module パス化), delete |
| `/v1/api-keys` → `/v1/me/apikeys` | 3 | list, create, revoke |
| `/v1/prompts` → `/v1/me/prompts` | 5 | list, get, create/update 分離, delete |
| `/v1/modules/config` → `/v1/me/modules/config` | 1 | config |
| `/v1/modules/{name}/*` → `/v1/me/modules/{name}/*` | 2 | tools, description |
| `/v1/oauth/consents` → `/v1/me/oauth/consents` | 2 | list, revoke |

**影響ファイル:**
- `apps/console/src/lib/auth/auth-context-actions.ts`
- `apps/console/src/lib/billing/plan.ts`
- `apps/console/src/lib/settings/user-settings.ts`
- `apps/console/src/lib/services/token-vault-actions.ts`
- `apps/console/src/lib/mcp/api-keys.ts`
- `apps/console/src/lib/mcp/prompts.ts`
- `apps/console/src/lib/mcp/tool-settings.ts`
- `apps/console/src/lib/oauth/consents.ts`
- `apps/console/src/app/api/stripe/checkout/route.ts`
- `apps/console/src/app/api/stripe/portal/route.ts`
- `apps/console/src/app/api/user/preferences/route.ts`
- `apps/console/src/app/api/credits/grant-signup-bonus/route.ts`
- `apps/console/src/app/auth/callback/route.ts`
- `apps/console/src/app/api/oauth/*/callback/route.ts` (11 files)

### 5.2 Go Server — PostgREST → Worker REST

| 影響ファイル | 変更内容 |
|-------------|---------|
| `apps/server/internal/broker/user.go` | fetchUserContext → GET /v1/users/{id}/context |
| | RecordUsage → POST /v1/users/{id}/usage |
| | SyncModules → POST /v1/modules/sync |
| | fetchUserPrompts → GET /v1/users/{id}/prompts |
| `apps/server/internal/broker/token.go` | fetchCredentials → GET /v1/users/{id}/credentials/{module} |
| | UpdateModuleToken → PUT /v1/users/{id}/credentials/{module} |
| | GetOAuthAppCredentials → GET /v1/oauth/apps/{provider}/credentials |
| `apps/server/internal/broker/retry.go` | HTTP クライアント設定更新 (URL ベース変更) |
| `apps/server/cmd/main.go` | 環境変数: POSTGREST_URL → WORKER_URL + GATEWAY_SECRET |

### 5.3 Worker — OpenAPI spec + ルート実装

| 変更種別 | 内容 |
|---------|------|
| 新ルートファイル | `apps/worker/src/v1/routes/me.ts` (Console 用の統合ルーター) |
| 新ルートファイル | `apps/worker/src/v1/routes/users.ts` (Go Server 用) |
| 廃止 | `apps/worker/src/v1/routes/user.ts` → `me.ts` に統合 |
| 廃止 | `apps/worker/src/v1/routes/credentials.ts` → `me.ts` に統合 |
| 廃止 | `apps/worker/src/v1/routes/apikeys.ts` → `me.ts` に統合 |
| 廃止 | `apps/worker/src/v1/routes/prompts.ts` → `me.ts` に統合 |
| 更新 | `apps/worker/src/v1/routes/modules.ts` (sync は残る) |
| 更新 | `apps/worker/src/openapi.yaml` (全パス更新) |

### 5.4 エンドポイント数比較

| カテゴリ | 旧 | 新 | 差分 |
|---------|-----|-----|------|
| Console (me) | 23 | 24 | +1 (prompts create/update 分離) |
| Go Server (users) | 0 (PostgREST直接) | 5 | +5 |
| Public | 2 | 2 | 0 |
| Modules (共通) | 3 | 2 | -1 (config は me 配下に移動) |
| OAuth (共通) | 3 | 1 | -2 (consents は me 配下に移動) |
| Admin | 4 | 4 | 0 |
| Webhook | 1 | 1 | 0 |
| **合計** | **36** | **39** | **+3** |

---

## 6. エラーレスポンス

全エンドポイント共通のエラー形式:

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Prompt not found"
  }
}
```

| HTTP Status | code | 用途 |
|------------|------|------|
| 400 | BAD_REQUEST | バリデーションエラー |
| 401 | UNAUTHORIZED | 認証なし・無効 |
| 403 | FORBIDDEN | 権限不足 (admin でない等) |
| 404 | NOT_FOUND | リソース不存在 |
| 409 | CONFLICT | 重複 (API キー名等) |
| 429 | USAGE_LIMIT_EXCEEDED | 使用量上限 |
| 500 | INTERNAL_ERROR | サーバーエラー |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dsn-rpc.md](./dsn-rpc.md) | RPC 関数設計書 (旧設計、PostgREST 時代) |
| [dsn-route.md](./dsn-route.md) | Console フロントエンドルーティング設計 |
