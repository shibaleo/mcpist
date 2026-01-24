# MCPist インターフェース仕様書（spc-itf）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.1 (DAY9) |
| Note | Interface Specification |

---

## 概要

本ドキュメントは、MCPistの各コンポーネント間のインターフェースを定義する。

**本ドキュメントの範囲:**
- プロトコル、エンドポイント、データ形式
- 認証方式
- エラーコード

**関連ドキュメント:**
- [spc-sys.md](./spc-sys.md) - コンポーネント定義
- [spc-itr.md](./spc-itr.md) - コンポーネント間のやり取り（誰が誰と話すか）

---

## コンポーネント略称

| 略称 | コンポーネント | 備考 |
|------|---------------|------|
| CLT | MCP Client | 実装範囲外 |
| AUS | Auth Server | Supabase Auth |
| SRV | MCP Server | 外部向け抽象化 |
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

## 1. MCP Protocol（CLT ↔ SRV）

### 概要

LLMクライアント（Claude Code等）がMCPサーバーと通信するプロトコル。

### プロトコル

| 項目 | 値 |
|------|-----|
| 形式 | JSON-RPC 2.0 over Streamable HTTP |
| MCP仕様バージョン | 2025-03-26 |
| トランスポート | HTTPS |
| 認証 | Bearer Token（JWT or Long-lived Token） |

### エンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/health` | ヘルスチェック |
| POST | `/mcp` | MCP Protocol（SSE） |
| GET | `/.well-known/oauth-authorization-server` | OAuth 2.1 メタデータ |

### 認証ヘッダー

```
Authorization: Bearer <token>
```

- JWT: Supabase Auth発行（OAuth 2.1フロー経由）
- Long-lived Token: User Consoleで発行（SHA-256ハッシュで検証）

### MCP メソッド

| カテゴリ | メソッド | Phase |
|---------|---------|-------|
| 初期化 | `initialize`, `initialized` | 1 |
| Tools | `tools/list`, `tools/call` | 1 |

### tools/list レスポンス

MCPistはModule Registry経由でツールを提供するため、`tools/list`は3つのメタツールのみを返す。

```json
{
  "tools": [
    {
      "name": "get_module_schema",
      "description": "モジュールのツール定義を取得",
      "inputSchema": {
        "type": "object",
        "properties": {
          "module": { "type": "string", "description": "モジュール名" }
        },
        "required": ["module"]
      }
    },
    {
      "name": "call",
      "description": "モジュールのツールを実行",
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
      "description": "複数ツールの一括実行（JSONL形式）",
      "inputSchema": {
        "type": "object",
        "properties": {
          "commands": { "type": "string", "description": "JSONL形式のコマンド列" }
        },
        "required": ["commands"]
      }
    }
  ]
}
```

### エラーコード

| コード | 名前 | 説明 |
|--------|------|------|
| -32600 | INVALID_REQUEST | 不正なJSONまたはJSON-RPC形式 |
| -32601 | METHOD_NOT_FOUND | 未知のメソッド |
| -32602 | INVALID_PARAMS | パラメータエラー |
| 2001 | INVALID_MODULE | 存在しないモジュール |
| 2002 | INVALID_TOOL | 存在しないツール |
| 2003 | MODULE_DISABLED | 無効化されたモジュール |
| 2004 | TOKEN_NOT_FOUND | 外部サービス連携が必要 |
| 2005 | EXTERNAL_API_ERROR | 外部API呼び出し失敗 |

---

## 2. Module Registry メタツール（REG）

### get_module_schema

モジュールのツール定義を取得する。

**リクエスト:**
```json
{
  "module": "notion"
}
```

**レスポンス:**
```json
{
  "module": "notion",
  "description": "Notion API - ページ・データベース・ブロック操作",
  "api_version": "2022-06-28",
  "tools": [
    {
      "name": "search",
      "description": "ページ・データベースを検索",
      "inputSchema": { ... }
    },
    ...
  ]
}
```

### call

モジュールのツールを実行する。

**リクエスト:**
```json
{
  "module": "notion",
  "tool": "search",
  "params": { "query": "設計" }
}
```

**レスポンス（TOON形式）:**
```
pages[3]{id,title,type}:
  abc123,設計ドキュメント,page
  def456,API仕様,page
  ghi789,DB設計,database
```

### batch

複数ツールをJSONL形式で一括実行する。DAGベースの依存関係解決と並列実行をサポート。

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
| output | No | true でTOON/MD形式で結果を返却 |
| raw_output | No | true でJSON形式で結果を返却 |

**変数参照:**
```
${taskId.results[index].field}
```

---

## 3. OAuth 2.1 認証（CLT ↔ AUS）

### 概要

MCP仕様2025-06に準拠したOAuth 2.1 Authorization Code Flow。

### エンドポイント

| エンドポイント | URL |
|---------------|-----|
| Authorization | `https://<project>.supabase.co/auth/v1/authorize` |
| Token | `https://<project>.supabase.co/auth/v1/token` |
| JWKS | `https://<project>.supabase.co/.well-known/jwks.json` |
| Metadata | `https://api.mcpist.app/.well-known/oauth-authorization-server` |

### OAuth Metadata

```json
{
  "issuer": "https://api.mcpist.app",
  "authorization_endpoint": "https://<project>.supabase.co/auth/v1/authorize",
  "token_endpoint": "https://<project>.supabase.co/auth/v1/token",
  "jwks_uri": "https://<project>.supabase.co/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"]
}
```

### JWT Claims

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "aud": "authenticated",
  "role": "authenticated",
  "exp": 1234567890,
  "iat": 1234567800
}
```

### JWT検証項目

| 項目 | 検証内容 |
|------|---------|
| 署名 | RS256、JWKS公開鍵で検証 |
| aud | `authenticated` であること |
| exp | 有効期限内であること |
| iss | Supabase AuthのURLであること |

---

## 4. Long-lived Token認証（CLT ↔ SRV）

### 概要

API直接呼び出し用のLong-lived Token。User Consoleで発行し、SHA-256ハッシュで検証。

### 発行

User Console経由でトークンを発行。平文トークンはユーザーに1回のみ表示。

### 保存

| 項目 | 保存場所 |
|------|---------|
| ハッシュ値 | `mcpist.mcp_tokens`テーブル |
| 平文 | 保存しない（発行時のみ表示） |

### 検証

```go
storedHash := getHashFromDB(tokenPrefix)
if storedHash == sha256(providedToken) {
    // 認証成功
}
```

### ヘッダー形式

```
Authorization: Bearer mcpist_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

## 5. User Console API（CON ↔ ENT/TVL）

### 認証

Supabase Auth（ソーシャルログイン + セッション）

### エンドポイント

#### ダッシュボード

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/api/dashboard` | 使用量サマリー取得 |
| GET | `/api/usage` | Quota使用量詳細 |
| GET | `/api/credits` | Credit残高・履歴 |

#### モジュール設定

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/api/modules` | 利用可能モジュール一覧 |
| PUT | `/api/modules/:module/enabled` | モジュール有効/無効切替 |

#### OAuth連携

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/api/oauth/status` | 連携状況一覧 |
| GET | `/api/oauth/:service/connect` | OAuth認可フロー開始 |
| GET | `/api/oauth/:service/callback` | OAuth コールバック |
| DELETE | `/api/oauth/:service` | 連携解除 |

#### MCP Token管理

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/api/tokens` | トークン一覧 |
| POST | `/api/tokens` | トークン発行 |
| DELETE | `/api/tokens/:id` | トークン削除 |

### レスポンス形式

**成功時:**
```json
{
  "data": { ... }
}
```

**エラー時:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid module name"
  }
}
```

---

## 6. Token Vault（MOD ↔ TVL）

### 概要

モジュールが外部APIアクセス用のトークン（長期トークン or OAuthトークン）を取得する。

| 項目 | 値 |
|------|-----|
| プロトコル | HTTPS（Supabase Edge Function） |
| メソッド | POST |
| エンドポイント | `/functions/v1/token-vault` |
| 認証 | `Authorization: Bearer <SUPABASE_PUBLISHABLE_KEY>` |

### 詳細仕様

→ [itf-tvl.md](./dtl-spc/itf-tvl.md)

---

## 7. External API呼び出し（MOD ↔ EXT）

### 概要

モジュールがToken Vaultから取得したトークンで外部APIを呼び出す。

### 認証

```
Authorization: Bearer <access_token>
```

### タイムアウト

30秒

### リトライ

| 失敗種別 | リトライ | 間隔 |
|---------|---------|------|
| 一時的エラー（5xx） | 最大3回 | 指数バックオフ（1s, 2s, 4s） |
| レート制限（429） | 最大3回 | Retry-Afterヘッダーに従う |
| 認証エラー（401） | 1回 | トークンリフレッシュ後に再試行 |
| その他（4xx） | なし | 即エラー返却 |

---

## 8. PSP Webhook（PSP → ENT）

### 概要

StripeからのWebhookを受信し、課金情報を同期する。

### エンドポイント

```
POST https://api.mcpist.app/webhooks/stripe
```

### 認証

Stripe Webhook署名検証

```
Stripe-Signature: t=xxx,v1=xxx
```

### イベント

| イベント | 処理 |
|---------|------|
| `checkout.session.completed` | サブスクリプション開始 |
| `customer.subscription.updated` | プラン変更 |
| `customer.subscription.deleted` | サブスクリプション終了 |
| `invoice.payment_failed` | 支払い失敗（suspended状態へ） |

### 冪等性

`processed_webhook_events`テーブルで`event_id`の重複を防止。

---

## 9. 内部コンポーネント間（SRV内部）

### AMW → HDL

認証済みリクエストの転送。Go context経由。

```go
type AuthContext struct {
    UserID    string
    Email     string
    ExpiresAt time.Time
}

ctx = context.WithValue(ctx, UserContextKey, authCtx)
```

### HDL → REG

メタツール呼び出し。Go関数呼び出し。

```go
// スキーマ取得
func (r *Registry) GetModuleSchema(ctx context.Context, module string) (ModuleSchema, error)

// ツール実行
func (r *Registry) Call(ctx context.Context, module, tool string, params map[string]any) (string, error)

// バッチ実行
func (r *Registry) Batch(ctx context.Context, commands string) (string, error)
```

### REG → MOD

モジュール呼び出し。Go interface呼び出し。

```go
type Module interface {
    Name() string
    Description() string
    APIVersion() string
    Tools() []Tool
    ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error)
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書（コンポーネント定義） |
| [spc-itr.md](./spc-itr.md) | インタラクション仕様書 |
| [spc-sec.md](./spc-sec.md) | セキュリティ仕様書 |
