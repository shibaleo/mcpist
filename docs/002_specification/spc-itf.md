# MCPist インターフェース仕様書（spc-itf）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v1.2 (2026-01-28) |
| Note | Interface Specification |

---

## 概要

本ドキュメントは、MCPistの各コンポーネント間のインターフェース（プロトコル、エンドポイント、認証、データ形式）を、**現行実装**に合わせて整理する。

**本ドキュメントの範囲:**
- MCPプロトコル（SSE/JSON-RPC）
- API Gateway/Server/Console間のHTTPインターフェース
- Token Vault（Supabase RPC）インターフェース
- 認証方式とエラー扱い

**関連ドキュメント:**
- [spc-sys.md](./spc-sys.md) - コンポーネント定義
- [spc-itr.md](spc-itr.md) - コンポーネント間のやり取り
- [spc-inf.md](./spc-inf.md) - インフラ構成
- [spc-dsn.md](./spc-dsn.md) - 実装構成

---

## コンポーネント略称

| 略称 | コンポーネント | 備考 |
|------|---------------|------|
| CLT | MCP Client | 実装範囲外 |
| GWY | API Gateway | Cloudflare Worker |
| AUS | Auth Server | Supabase Auth |
| SRV | MCP Server | Go |
| AMW | Auth Middleware | SRV内部 |
| HDL | MCP Handler | SRV内部 |
| REG | Module Registry | SRV内部 |
| MOD | Modules | SRV内部 |
| ENT | Entitlement Store | Supabase PostgreSQL |
| TVL | Token Vault | Supabase Vault |
| CON | User Console | Next.js |
| EXT | External API Server | 実装範囲外 |
| PSP | Payment Service Provider | 実装範囲外 |

---

## 1. MCP Protocol（CLT ↔ GWY ↔ SRV）

### 概要

LLMクライアントがMCPサーバーに接続するためのプロトコル。**API Gateway（Worker）** が認証とルーティングを担い、**MCP Server（Go）** がMCP処理を行う。

### プロトコル

| 項目 | 値 |
|------|-----|
| 形式 | JSON-RPC 2.0 + SSE（サーバーイベント） |
| MCP仕様バージョン | 2025-03-26 |
| トランスポート | HTTPS |
| 認証 | Bearer Token（JWT or API Key） |

### 公開エンドポイント（API Gateway）

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/health` | プライマリ/セカンダリのヘルス確認 |
| GET | `/mcp` | SSE接続（セッション生成） |
| POST | `/mcp` | JSON-RPC（インライン応答） |
| POST | `/mcp?sessionId=...` | JSON-RPC（SSE経由で応答） |
| GET | `/.well-known/oauth-protected-resource` | OAuth Protected Resource Metadata (RFC 9728) |
| GET | `/mcp/.well-known/oauth-protected-resource` | 上記と同一内容（/mcp配下） |
| GET | `/.well-known/oauth-authorization-server` | OAuth Authorization Server Metadata (RFC 8414) |
| GET | `/mcp/.well-known/oauth-authorization-server` | 上記と同一内容（/mcp配下） |

### 内部エンドポイント（API Gateway）

| メソッド | パス | 説明 |
|---------|------|------|
| POST | `/internal/invalidate-api-key` | API Keyキャッシュ無効化（X-Internal-Secret必須） |

### 認証ヘッダー

```
Authorization: Bearer <token>
```

- **JWT**: Supabase Auth発行（OAuth 2.1フロー経由）
- **API Key**: `mpt_` で始まる長期トークン

### 認証の実装（Gateway）

1. `Authorization: Bearer <token>` を検証
2. JWTは以下の順で検証
   - `/auth/v1/oauth/userinfo`
   - `/auth/v1/user`（publishable key併用）
   - JWKS署名検証（issuer: `${SUPABASE_URL}/auth/v1`）
3. API KeyはSHA-256でハッシュ化し、KVキャッシュ → Supabase RPC `lookup_user_by_key_hash` で検証
4. 認証OKならSRVへプロキシし、以下を付与
   - `X-User-ID`
   - `X-Auth-Type`（`jwt` / `api_key`）
   - `X-Request-ID`
   - `X-Gateway-Secret`

### SSE接続フロー

1. `GET /mcp` でSSE接続
2. 接続直後に `event: endpoint` が返る
   ```
   event: endpoint
   data: /mcp?sessionId=<id>
   ```
3. クライアントは `POST /mcp?sessionId=<id>` にJSON-RPCを送信
4. 結果はSSEの `event: message` で返却

### JSON-RPC（インライン）

- `POST /mcp` にJSON-RPCを送ると、その場でJSON-RPCレスポンスを返す

### MCP メソッド

| カテゴリ | メソッド | Phase |
|---------|---------|-------|
| 初期化 | `initialize`, `initialized` | 1 |
| Tools | `tools/list`, `tools/call` | 1 |

### tools/call パラメータ

```json
{
  "name": "get_module_schema",
  "arguments": {
    "module": ["notion", "github"]
  }
}
```

### ToolCallResult 形式

```json
{
  "content": [
    { "type": "text", "text": "..." }
  ],
  "isError": false
}
```

### tools/list レスポンス（メタツールのみ）

`tools/list` は常に**3つのメタツール**のみを返す。モジュール名はユーザーの有効設定に応じて動的に挿入される。

```json
{
  "tools": [
    {
      "name": "get_module_schema",
      "description": "Get tool definitions for modules. ...",
      "inputSchema": {
        "type": "object",
        "properties": {
          "module": {
            "type": "array",
            "description": "Array of module names",
            "items": { "type": "string" }
          }
        },
        "required": ["module"]
      }
    },
    {
      "name": "run",
      "description": "Execute a single module tool. ...",
      "inputSchema": {
        "type": "object",
        "properties": {
          "module": { "type": "string" },
          "tool": { "type": "string" },
          "params": { "type": "object" }
        },
        "required": ["module", "tool"]
      }
    },
    {
      "name": "batch",
      "description": "Execute multiple tools in batch (JSONL format). ...",
      "inputSchema": {
        "type": "object",
        "properties": {
          "commands": { "type": "string" }
        },
        "required": ["commands"]
      }
    }
  ]
}
```

### エラーコード（JSON-RPC）

| コード | 名前 | 説明 |
|--------|------|------|
| -32700 | PARSE_ERROR | JSON解析エラー |
| -32600 | INVALID_REQUEST | JSON-RPC形式エラー |
| -32601 | METHOD_NOT_FOUND | 未知のメソッド |
| -32602 | INVALID_PARAMS | パラメータエラー |
| -32603 | INTERNAL_ERROR | サーバー内部エラー |

**認証/認可系のHTTPエラー（JSON-RPCではなくHTTP応答）**

| HTTP | 例 | 説明 |
|------|----|------|
| 401 | `{"error":"Unauthorized"}` | Gateway認証失敗 |
| 401 | `{"error":"INVALID_GATEWAY_SECRET"}` | SRVでGateway Secret不一致 |
| 403 | `{"error":"ACCOUNT_NOT_ACTIVE"}` | アカウント停止 |
| 403 | `{"error":"MODULE_NOT_ENABLED"}` | モジュール無効 |
| 403 | `{"error":"TOOL_DISABLED"}` | ツール無効 |
| 402 | `{"error":"INSUFFICIENT_CREDITS"}` | クレジット不足 |

---

## 2. Module Registry メタツール（REG）

### get_module_schema

モジュールのツール定義・リソース・プロンプトを取得する。

**リクエスト:**
```json
{
  "module": ["notion", "jira"]
}
```

**補足:**
- `module` は配列が正規形（文字列1件でも後方互換で受理）
- 無効/未登録モジュールは警告として返し、他のモジュールは返却を継続

**レスポンス（content[0].textにJSON）:**
```json
[
  {
    "module": "notion",
    "description": "Notion API - ...",
    "api_version": "2022-06-28",
    "tools": [ ... ],
    "resources": [ ... ],
    "prompts": [ ... ]
  }
]
```

### run

モジュールのツールを単発実行する。

**リクエスト:**
```json
{
  "module": "notion",
  "tool": "search",
  "params": { "query": "設計" }
}
```

### batch

JSONL形式で複数ツールをDAG実行する。依存がなければ並列。

**リクエスト:**
```json
{
  "commands": "{\"id\":\"search\",\"module\":\"notion\",\"tool\":\"search\",\"params\":{\"query\":\"設計\"}}\n{\"id\":\"page\",\"module\":\"notion\",\"tool\":\"get_page_content\",\"params\":{\"page_id\":\"${search.results[0].id}\"},\"after\":[\"search\"],\"output\":true}"
}
```

**コマンドフィールド:**

| Field | Required | Description |
|-------|----------|-------------|
| id | Yes | タスク識別子 |
| module | Yes | モジュール名 |
| tool | Yes | ツール名 |
| params | No | ツールパラメータ |
| after | No | 依存タスクID配列 |
| output | No | trueでTOON/MD形式 |
| raw_output | No | trueでJSON形式（outputより優先） |

**変数参照:**
```
${taskId.results[index].field}
```

---

## 3. OAuth 2.1 認証（CLT ↔ AUS）

### 概要

Supabase AuthのOAuth 2.1 Authorization Code Flowを使用。メタデータはGatewayが `/auth/v1/.well-known/openid-configuration` を中継する。

### OAuth Metadata（例）

```json
{
  "issuer": "https://<project>.supabase.co/auth/v1",
  "authorization_endpoint": "https://<project>.supabase.co/auth/v1/authorize",
  "token_endpoint": "https://<project>.supabase.co/auth/v1/token",
  "jwks_uri": "https://<project>.supabase.co/auth/v1/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"]
}
```

### OAuth Protected Resource Metadata（RFC 9728）

```json
{
  "resource": "https://api.mcpist.app/mcp",
  "authorization_servers": ["https://<project>.supabase.co/auth/v1"],
  "scopes_supported": ["openid", "profile", "email"],
  "bearer_methods_supported": ["header"]
}
```

---

## 4. API Key（Long-lived Token）認証（CLT ↔ GWY）

### 概要

User Consoleで発行する長期API Key。Gatewayが検証し、SRVは`X-User-ID`のみを信頼する。

### トークン形式

```
mpt_<prefix>_<random>
```

- `mpt_`: 固定プレフィックス
- `<prefix>`: 8文字（高速検索/キャッシュ用）
- `<random>`: 24文字

### 検証フロー（Gateway）

1. API KeyをSHA-256ハッシュ化
2. KVキャッシュ照合（TTL: 24h / soft max-age: 1h）
3. キャッシュミス時はSupabase RPC `lookup_user_by_key_hash` を実行

---

## 5. User Console API（CON ↔ ENT/TVL）

### 認証

Supabase Auth（Cookie/Session）

### Next.js API Routes

| メソッド | パス | 説明 |
|---------|------|------|
| POST | `/api/validate-token` | 外部サービスTokenの事前検証 |
| POST | `/api/token-vault` | Token Vault取得（内部向け） |
| GET | `/api/oauth/google/authorize` | Google OAuth開始 |
| GET | `/api/oauth/google/callback` | Google OAuthコールバック |
| GET | `/api/oauth/microsoft/authorize` | Microsoft OAuth開始 |
| GET | `/api/oauth/microsoft/callback` | Microsoft OAuthコールバック |
| GET | `/api/admin/oauth-apps` | OAuth App一覧（Admin） |
| POST | `/api/admin/oauth-apps` | OAuth App登録/更新（Admin） |
| DELETE | `/api/admin/oauth-apps?provider=...` | OAuth App削除（Admin） |

### `/api/validate-token`（例）

**リクエスト:**
```json
{ "service": "notion", "token": "secret" }
```

**Jira/Confluenceの場合:**
```json
{ "service": "jira", "token": "...", "email": "...", "domain": "xxx.atlassian.net" }
```

**レスポンス:**
```json
{ "valid": true, "details": { ... } }
```

### `/api/token-vault`

- **Authorization:** `Bearer <INTERNAL_SERVICE_KEY>`
- **Body:** `{ "user_id": "...", "service": "notion" }`
- **Response:** `{ "oauth_token": "...", "long_term_token": "..." }`

### User Consoleが直接利用するSupabase RPC（代表）

| 用途 | RPC |
|------|-----|
| API Key | `list_api_keys`, `generate_api_key`, `revoke_api_key` |
| クレジット/権限 | `get_user_context` |
| 接続管理 | `list_service_connections`, `upsert_service_token`, `delete_service_token` |
| ツール設定 | `get_my_tool_settings`, `upsert_my_tool_settings` |
| OAuth Consent | `list_oauth_consents`, `revoke_oauth_consent`, `list_all_oauth_consents` |
| 管理者判定 | `get_user_role` |

---

## 6. Token Vault（MOD ↔ TVL）

### 概要

Supabase RPCを介して外部サービス用トークンを取得・更新する。

### RPC関数（SRVから利用）

| 関数 | 権限 | 用途 |
|------|------|------|
| `get_module_token` | anon/authenticated | トークン取得 |
| `update_module_token` | service_role | トークン更新 |
| `get_oauth_app_credentials` | service_role | OAuth App認証情報取得 |

### 返却されるCredentials（主要フィールド）

| フィールド | 説明 |
|-----------|------|
| `auth_type` | `oauth2` / `oauth1` / `api_key` / `basic` / `custom_header` |
| `access_token` | OAuth2/API Key |
| `refresh_token` | OAuth2（更新用） |
| `expires_at` | Unix timestamp |
| `username` / `password` | Basic認証 |
| `metadata` | Atlassian domain等 |

---

## 7. External API呼び出し（MOD ↔ EXT）

### 実装の前提

| 項目 | 値 |
|------|-----|
| HTTPクライアント | Go標準 `net/http` |
| タイムアウト | 30秒 |
| リトライ | **共通実装なし**（各モジュールの責務） |

### 認証方式

- OAuth2 / API Key: `Authorization: Bearer <access_token>`
- Basic（Jira/Confluence等）: `Authorization: Basic <base64(username:token)>`
- Custom Header: module実装に依存

### トークンリフレッシュ（実装済み）

| モジュール | リフレッシュ条件 | 更新先 |
|-----------|------------------|--------|
| `google_calendar` | `expires_at` が現在時刻 - 5分以内 | `update_module_token` |
| `microsoft_todo` | 同上 | `update_module_token` |

---

## 8. PSP Webhook（PSP → ENT）

**現行実装ではWebhookエンドポイントは未実装。**

---

## 9. 内部コンポーネント間（SRV内部）

### AMW → HDL

認証済みリクエストをContextで伝搬。

```go
type AuthContext struct {
    UserID         string
    AuthType       string // "jwt" | "api_key"
    AccountStatus  string
    FreeCredits    int
    PaidCredits    int
    EnabledModules []string
    DisabledTools  map[string][]string
}
```

### HDL → REG（Module Registry）

```go
// スキーマ取得
modules.GetModuleSchemas(moduleNames, enabledModules, disabledTools)

// ツール実行
modules.Run(ctx, module, tool, params)

// バッチ実行
modules.Batch(ctx, commands)
```

### REG → MOD

```go
type Module interface {
    Name() string
    Description() string
    APIVersion() string

    Tools() []Tool
    ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error)

    Resources() []Resource
    ReadResource(ctx context.Context, uri string) (string, error)

    Prompts() []Prompt
    GetPrompt(ctx context.Context, name string, args map[string]any) (string, error)
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [spc-inf.md](./spc-inf.md) | インフラ仕様書 |
| [spc-dsn.md](./spc-dsn.md) | 設計仕様書 |
