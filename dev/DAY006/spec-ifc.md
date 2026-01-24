---
title: MCPist インターフェース仕様書（spec-ifc）
aliases:
  - spec-ifc
  - MCPist-interface-specification
tags:
  - MCPist
  - specification
  - interface
document-type:
  - specification
document-class: specification
created: 2026-01-15T00:00:00+09:00
updated: 2026-01-15T00:00:00+09:00
---
# MCPist インターフェース仕様書（spec-ifc）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY6) |
| Note | コンポーネント間インターフェース定義 |

---

本ドキュメントは、MCPistの各コンポーネント間で発生する情報のやり取り（インターフェース）を定義する。

---

## 用語変更

| 旧名称 | 新名称 | 理由 |
|--------|--------|------|
| Token Broker | Token Vault | 業界標準の名称に統一。Auth0等が同名で提供するパターンと同義。 |

**Token Vaultパターンについて:**

Token Vaultは、外部APIトークンを安全に保管・管理するための業界標準パターン。主な責務：
- トークンの暗号化保存
- 自動リフレッシュ
- user_idに紐づくトークンの取得
- 監査ログ

Auth0がAIエージェント向けに「Token Vault」として提供しているが、2025年1月時点でEarly Access（早期アクセス）段階であり、正式リリース後の価格体系は未定。

**IDaaS各社の対応状況:**

| サービス | Token Vault相当機能 | 状態 | 備考 |
|----------|---------------------|------|------|
| Auth0 | Token Vault | Early Access | AIエージェント向け、CIBA対応 |
| Clerk | OAuth Access Token | 正式機能 | `getUserOauthAccessToken()`で取得可能 |
| Supabase Auth | なし | - | 自前実装が必要 |

ClerkはToken Vaultという名称は使っていないが、ユーザーがOAuth認証した外部サービス（Google等）のアクセストークンを保存し、APIから取得する機能を標準提供している。

MCPistでは、Supabase Edge Function + Supabase Vaultで同等機能を自前実装する。将来的にClerk採用も選択肢となる。

**参考:**
- [Token Vault - Auth0](https://auth0.com/features/token-vault)
- [Configure Token Vault - Auth0 Docs](https://auth0.com/docs/ja-jp/secure/tokens/token-vault/configure-token-vault)
- [getUserOauthAccessToken() - Clerk Docs](https://clerk.com/docs/references/backend/user/get-user-oauth-access-token)

---

## 認証アーキテクチャの位置づけ

### MCP仕様における認証

MCP仕様では認証は**必須ではない**（SHOULD/MAY）。認証なしで動作するMCPサーバーも仕様準拠である。

MCPistは認証を採用するが、これはMCPistの設計判断であり、MCP仕様の要求ではない。

### OAuth 2.1エコシステムにおける役割

MCP仕様 2025-06以降、認証を採用する場合のMCPサーバーの役割が明確化された：

```
【OAuth 2.1エコシステム】

Authorization Server（認可サーバー）
  └─ トークン発行

Resource Server（MCPサーバー）
  └─ トークン検証、Protected Resourceの提供
```

MCPサーバーは**OAuth 2.1 Resource Server**として位置づけられる。Authorization Serverが発行したトークンを検証するだけであり、トークン発行の責務は持たない。

### MCPistにおける実装

MCPistでは：
- **Authorization Server**: Supabase Auth（外部コンポーネント）
- **Resource Server**: MCPサーバー（Koyeb）

AuthサーバーはMCPサーバーとは論理的に分離された外部コンポーネントである。物理的な分離（別プロセス・別サーバー）は実装の選択による。

### Token Brokerの役割：2つのOAuthコンテキストの橋渡し

MCPistは2つのOAuthコンテキストに関わる：

```
【コンテキスト1】LLMクライアント → MCPist
  MCPistの役割: Resource Server
  トークン: Supabase Auth発行のJWT
  目的: user_idを特定

        ↓ Token Broker（橋渡し）

【コンテキスト2】MCPist → 外部API（Google等）
  MCPistの役割: Client
  トークン: 外部サービスが発行したOAuthトークン
  目的: user_idに紐づくトークンを取得・使用
```

Token Brokerは、コンテキスト1で特定したuser_idを使って、コンテキスト2で使用する外部サービスのトークンを安全に取得・管理する。この橋渡しがなければ、「このユーザーのGoogleトークンはどれか」を解決できない。

**参考:**
- [MCP Authorization Specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/authorization)
- [MCP Specs Update June 2025 - Auth0](https://auth0.com/blog/mcp-specs-update-all-about-auth/)

---

## インターフェース一覧

### 外部アクター ↔ MCPistシステム

| ID | From | To | 説明 |
|----|------|-----|------|
| IFC-001 | LLMユーザー | Authサーバー | ソーシャルログイン認証 |
| IFC-002 | LLMクライアント | MCPサーバー | MCP Protocol通信 |
| IFC-003 | 管理者 | 管理UI | 管理操作（Web） |
| IFC-004 | ユーザー | 管理UI | プロファイル・連携操作（Web） |

### MCPサーバー内部

| ID | From | To | 説明 |
|----|------|-----|------|
| IFC-010 | 認証ミドルウェア | Authサーバー | JWT検証 |
| IFC-011 | 認証ミドルウェア | Tool Sieve | user_accountID受け渡し |
| IFC-012 | Tool Sieve | MCPプロトコルハンドラ | フィルタ済みリクエスト転送 |
| IFC-013 | MCPプロトコルハンドラ | モジュールレジストリ | ツール実行要求 |
| IFC-014 | モジュールレジストリ | 各モジュール | モジュール呼び出し |

### Token Broker連携

| ID      | From         | To           | 説明         |
| ------- | ------------ | ------------ | ---------- |
| IFC-020 | 各モジュール       | Token Broker | トークン取得要求   |
| IFC-021 | Token Broker | Vault        | トークン照会・保存  |
| IFC-022 | Token Broker | 外部OAuthサーバー  | トークンリフレッシュ |

### 外部API連携

| ID | From | To | 説明 |
|----|------|-----|------|
| IFC-030 | 各モジュール | 外部API | API呼び出し |

### 管理UI連携

| ID | From | To | 説明 |
|----|------|-----|------|
| IFC-040 | 管理UI | Authサーバー | ユーザー認証 |
| IFC-041 | 管理UI | Tool Sieve（DB） | ユーザー・ロール・権限管理 |
| IFC-042 | 管理UI | Token Broker | トークン登録・管理 |
| IFC-043 | 管理UI | 外部OAuthサーバー | OAuth認可フロー |

---

## IFC-001: LLMユーザー ↔ Authサーバー

### 概要

LLMユーザーがMCPistにログインする際の認証フロー。

### 方向

```
LLMユーザー → Authサーバー（Supabase Auth）
```

### プロトコル

OAuth 2.0 / OIDC（ソーシャルログイン）

### データフロー

```
1. ユーザーがログインボタンをクリック
2. Supabase Authがソーシャルプロバイダへリダイレクト
3. ユーザーがソーシャルプロバイダで認証・同意
4. コールバックでAuthサーバーに戻る
5. AuthサーバーがJWTを発行
6. セッション開始
```

### データ形式

**認証成功時のJWT Claims:**
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

### 対応プロバイダ

| プロバイダ | プロトコル |
|-----------|-----------|
| Google | OIDC |
| GitHub | OAuth 2.0 |
| Microsoft | OIDC |

---

## IFC-002: LLMクライアント ↔ MCPサーバー

### 概要

LLMクライアント（Claude Code, Cursor等）がMCPサーバーにツール実行を要求する。

### 方向

```
LLMクライアント ↔ MCPサーバー（双方向）
```

### プロトコル

MCP Protocol（JSON-RPC 2.0 over SSE）

### エンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/health` | ヘルスチェック |
| POST | `/mcp` | MCP Protocol（SSE） |

### 認証

```
Authorization: Bearer <JWT or API_TOKEN>
```

### リクエスト形式

**initialize:**
```json
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {},
    "clientInfo": { "name": "claude-code", "version": "1.0.0" }
  },
  "id": 1
}
```

**tools/list:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "id": 2
}
```

**tools/call:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "call",
    "arguments": {
      "module": "notion",
      "tool_name": "search_pages",
      "params": { "query": "プロジェクト" }
    }
  },
  "id": 3
}
```

### レスポンス形式

**成功時（TOON形式）:**
```
items[3]{id,title,status}:
  task1,買い物,notStarted
  task2,掃除,completed
  task3,料理,inProgress
```

**エラー時:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": 2001,
    "message": "INVALID_MODULE",
    "data": { "module": "unknown" }
  }
}
```

### 対応メソッド

| カテゴリ | メソッド | Phase |
|---------|---------|-------|
| 初期化 | `initialize`, `initialized` | 1 |
| Tools | `tools/list`, `tools/call` | 1 |
| Tasks | `tasks/get`, `tasks/result`, `tasks/list`, `tasks/cancel` | 1 |
| Elicitation | `elicitation/create` | 1 |
| 通知 | `notifications/elicitation/complete`, `notifications/tasks/status` | 1 |

---

## IFC-003: 管理者 ↔ 管理UI

### 概要

管理者（admin）が管理UIを通じてシステム設定を行う。

### 方向

```
管理者（ブラウザ） ↔ 管理UI（Vercel）
```

### プロトコル

HTTPS / REST API

### 認証

Supabase Auth（ソーシャルログイン + セッション）

### 管理者向けAPI

| メソッド | パス | 説明 |
|---------|------|------|
| GET/POST/PUT/DELETE | `/api/users/*` | ユーザー管理 |
| GET/POST/PUT/DELETE | `/api/roles/*` | ロール管理 |
| GET/PUT | `/api/roles/:id/permissions` | 権限管理 |
| GET/PUT/DELETE | `/api/roles/:id/services/*` | サービス・トークン管理 |
| GET | `/api/logs` | 監査ログ |

### 主要操作

| 操作 | API | 説明 |
|------|-----|------|
| ユーザー作成 | POST `/api/users` | 新規ユーザー追加 |
| ロール作成 | POST `/api/roles` | 権限パターン作成 |
| ロール割当 | POST `/api/users/:id/roles` | ユーザーにロール付与 |
| 権限設定 | PUT `/api/roles/:id/permissions` | ツール権限設定 |
| OAuthアプリ登録 | PUT `/api/roles/:id/services/:service` | Client ID/Secret登録 |
| 共有トークン設定 | POST `/api/roles/:id/services/:service/oauth/start` | OAuth認可実行 |

---

## IFC-004: ユーザー ↔ 管理UI

### 概要

一般ユーザー（user）が管理UIを通じてプロファイル確認・サービス連携を行う。

### 方向

```
ユーザー（ブラウザ） ↔ 管理UI（Vercel）
```

### プロトコル

HTTPS / REST API

### ユーザー向けAPI

| メソッド | パス | 説明 |
|---------|------|------|
| GET/PUT | `/api/profile` | プロファイル |
| GET | `/api/profile/roles` | 自分のロール一覧 |
| GET | `/api/profile/tools` | 使えるツール一覧 |
| GET | `/api/profile/services` | 連携状況一覧 |
| POST | `/api/profile/services/:service/oauth/start` | 個人アカウント連携 |
| DELETE | `/api/profile/services/:service/token` | 個人トークン削除 |
| POST | `/api/profile/tools/request` | ツール利用申請 |

### 主要操作

| 操作 | API | 説明 |
|------|-----|------|
| プロファイル確認 | GET `/api/profile` | 自分の情報取得 |
| ツール確認 | GET `/api/profile/tools` | 利用可能ツール一覧 |
| 個人連携開始 | POST `/api/profile/services/:service/oauth/start` | OAuth認可開始 |
| ツール申請 | POST `/api/profile/tools/request` | adminへ利用申請 |

---

## IFC-010: 認証ミドルウェア ↔ Authサーバー

### 概要

認証ミドルウェアがJWTをAuthサーバーに検証依頼する。

### 方向

```
認証ミドルウェア → Authサーバー（Supabase Auth）
```

### プロトコル

JWKS（JSON Web Key Set）による署名検証

### データフロー

```
1. 認証ミドルウェアがリクエストからJWTを抽出
2. Supabase AuthのJWKSエンドポイントから公開鍵取得（キャッシュ）
3. JWT署名を検証
4. Claims（sub, email, exp等）を抽出
5. 有効期限を確認
```

### JWKS URL

```
https://<project>.supabase.co/.well-known/jwks.json
```

### 検証項目

| 項目 | 説明 |
|------|------|
| 署名 | RS256署名の検証 |
| aud | `authenticated` であること |
| exp | 有効期限内であること |
| iss | Supabase AuthのURLであること |

### 出力

```go
type AuthContext struct {
    UserID    string // sub claim
    Email     string // email claim
    ExpiresAt time.Time
}
```

---

## IFC-011: 認証ミドルウェア → Tool Sieve

### 概要

認証済みのuser_accountIDをTool Sieveに渡す。

### 方向

```
認証ミドルウェア → Tool Sieve（同一プロセス内）
```

### プロトコル

Go関数呼び出し（context.Context経由）

### データ形式

```go
// Context Key
type contextKey string
const UserContextKey contextKey = "user"

// AuthContext
type AuthContext struct {
    UserID    string
    Email     string
    TenantID  string
    Roles     []string
}

// 設定
ctx = context.WithValue(ctx, UserContextKey, authCtx)

// 取得
authCtx := ctx.Value(UserContextKey).(*AuthContext)
```

---

## IFC-012: Tool Sieve → MCPプロトコルハンドラ

### 概要

Tool Sieveがロール権限に基づきフィルタリングした後、リクエストをハンドラに転送。

### 方向

```
Tool Sieve → MCPプロトコルハンドラ（同一プロセス内）
```

### プロトコル

Go関数呼び出し

### 処理フロー

```
1. Tool SieveがDB（role_permissions）から権限情報取得
2. リクエストされたモジュール/ツールが許可されているか確認
3. 許可されていればハンドラに転送
4. 不許可なら403エラーを返却
```

### 権限チェック

| チェック項目 | 説明 |
|-------------|------|
| enabled_modules | ユーザーのロールで有効なモジュール一覧 |
| tool_masks | ツール単位の有効/無効設定 |

### データ形式

```go
type FilteredRequest struct {
    OriginalRequest *JSONRPCRequest
    AuthContext     *AuthContext
    AllowedModules  []string
    AllowedTools    map[string][]string // module -> tools
}
```

---

## IFC-013: MCPプロトコルハンドラ → モジュールレジストリ

### 概要

ハンドラがモジュールレジストリにツール実行を依頼する。

### 方向

```
MCPプロトコルハンドラ → モジュールレジストリ
```

### プロトコル

Go関数呼び出し

### メソッド

```go
// スキーマ取得
func (r *Registry) GetModuleSchema(ctx context.Context, modules []string) ([]ModuleSchema, error)

// ツール実行
func (r *Registry) ExecuteTool(ctx context.Context, req ExecuteRequest) (string, error)

// バッチ実行
func (r *Registry) ExecuteBatch(ctx context.Context, jsonl string) (BatchResult, error)
```

### リクエスト形式

```go
type ExecuteRequest struct {
    Module   string
    Tool     string
    Params   map[string]interface{}
    UserID   string
    RoleID   string
}
```

### レスポンス形式

TOON形式の文字列:
```
items[3]{id,title,status}:
  task1,買い物,notStarted
  task2,掃除,completed
  task3,料理,inProgress
```

---

## IFC-014: モジュールレジストリ → 各モジュール

### 概要

レジストリが個別モジュール（notion, github等）を呼び出す。

### 方向

```
モジュールレジストリ → 各モジュール（同一プロセス内）
```

### プロトコル

Go interface呼び出し

### インターフェース定義

```go
type Module interface {
    Name() string
    Description() string
    APIVersion() string
    Tools() []ToolDefinition
    Execute(ctx context.Context, tool string, params map[string]interface{}) (string, error)
}

type ToolDefinition struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
    OutputSchema OutputSchema
    Dangerous   bool
}

type OutputSchema struct {
    Format string   // "toon"
    Fields []string // ["id", "title", "status"]
}
```

### 登録モジュール

| モジュール | ツール数 |
|-----------|----------|
| notion | 20 |
| github | 3 |
| jira | 6 |
| confluence | 5 |
| supabase | 30 |
| google_calendar | 8 |
| microsoft_todo | 9 |
| rag | 3 |

---

## IFC-020: 各モジュール → Token Broker

### 概要

モジュールが外部APIアクセス用のトークンをToken Brokerから取得する。

### 方向

```
各モジュール → Token Broker（Edge Function）
```

### プロトコル

HTTPS / REST API

### エンドポイント

```
POST https://<project>.supabase.co/functions/v1/token-broker
```

### リクエスト

```json
{
  "user_id": "user-uuid",
  "role_id": "role-uuid",
  "service": "notion"
}
```

### レスポンス

**成功時:**
```json
{
  "access_token": "secret-token-xxx",
  "expires_at": "2026-01-15T12:00:00Z"
}
```

**未連携時:**
```json
{
  "error": "TOKEN_NOT_FOUND",
  "message": "サービス連携が必要です",
  "elicitation_url": "https://mcpist.app/oauth/notion/authorize"
}
```

### 認証

MCPサーバーからの呼び出しは内部サービス間通信として、環境変数の共有シークレットで認証。

```
Authorization: Bearer <INTERNAL_SECRET>
```

### リトライ戦略

| 失敗種別 | リトライ | 間隔 |
|---------|---------|------|
| 一時的エラー（5xx） | 最大3回 | 指数バックオフ（1s, 2s, 4s） |
| 認証エラー（401） | なし | 即エラー返却 |
| 未連携（404） | なし | Elicitation URLを返却 |

---

## IFC-021: Token Broker ↔ Vault

### 概要

Token BrokerがSupabase Vaultからトークンを照会・保存する。

### 方向

```
Token Broker ↔ Vault（双方向）
```

### プロトコル

Supabase Vault API（Edge Function内から呼び出し）

### 操作

| 操作 | 説明 |
|------|------|
| 照会 | oauth_tokensテーブルからトークン取得（Vaultが復号化） |
| 保存 | 新トークンを保存（Vaultが暗号化） |
| 更新 | リフレッシュ後のトークンを上書き |

### トークン解決順序

```
1. user_id + role_id + service で個人トークン検索
2. なければ role_id + service で共有トークン検索
3. どちらもなければエラー（要連携）
```

### データ構造

```sql
-- oauth_tokens テーブル
| カラム | 説明 |
|--------|------|
| role_id | ロールID（必須） |
| user_id | ユーザーID（NULL=共有トークン） |
| service | サービス名 |
| access_token | アクセストークン（暗号化） |
| refresh_token | リフレッシュトークン（暗号化） |
| expires_at | 有効期限 |
```

---

## IFC-022: Token Broker → 外部OAuthサーバー

### 概要

Token Brokerがトークンリフレッシュを実行する。

### 方向

```
Token Broker → 外部OAuthサーバー（Google, Microsoft等）
```

### プロトコル

OAuth 2.0 Token Refresh

### リクエスト

```http
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=<refresh_token>
&client_id=<client_id>
&client_secret=<client_secret>
```

### レスポンス

```json
{
  "access_token": "new-access-token",
  "refresh_token": "new-refresh-token",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

### 処理フロー

```
1. Vaultから期限切れトークンを取得
2. refresh_tokenで外部OAuthサーバーにリフレッシュ要求
3. 新トークンをVaultに保存
4. 新access_tokenをMCPサーバーに返却
```

### サービス別エンドポイント

| サービス | Token URL |
|---------|-----------|
| Google | `https://oauth2.googleapis.com/token` |
| Microsoft | `https://login.microsoftonline.com/common/oauth2/v2.0/token` |
| GitHub | `https://github.com/login/oauth/access_token` |
| Notion | `https://api.notion.com/v1/oauth/token` |

---

## IFC-030: 各モジュール → 外部API

### 概要

モジュールがToken Brokerから取得したトークンで外部APIを呼び出す。

### 方向

```
各モジュール → 外部API（Notion, GitHub等）
```

### プロトコル

HTTPS / REST API（各サービス固有）

### 認証

```
Authorization: Bearer <access_token>
```

### 処理フロー

```
1. Token Brokerからaccess_token取得（IFC-020）
2. 外部APIにリクエスト
3. レスポンスをTOON形式に変換
4. MCPプロトコルハンドラに返却
```

### レートリミット

各モジュールがサービス固有のレートリミットを管理:

| サービス | 制限 |
|---------|------|
| Notion | 3 req/sec |
| GitHub | 5000 req/hour |
| Jira | 50 req/sec |
| Google | 100 req/sec |

### タイムアウト

デフォルト: 30秒

### ページネーション

各モジュールが内部で全件取得（上限500件）してからTOON形式で返却。

---

## IFC-040: 管理UI → Authサーバー

### 概要

管理UIがユーザー認証をAuthサーバーに依頼する。

### 方向

```
管理UI（Next.js） → Authサーバー（Supabase Auth）
```

### プロトコル

Supabase Auth SDK

### 認証フロー

```typescript
// ログイン
const { data, error } = await supabase.auth.signInWithOAuth({
  provider: 'google',
  options: { redirectTo: '/callback' }
});

// セッション確認
const { data: { session } } = await supabase.auth.getSession();

// ログアウト
await supabase.auth.signOut();
```

### セッション管理

| 項目 | 値 |
|------|-----|
| セッション保存 | Cookie（httpOnly） |
| 有効期限 | 1時間（自動リフレッシュ） |
| リフレッシュトークン有効期限 | 7日 |

---

## IFC-041: 管理UI → Tool Sieve（DB）

### 概要

管理UIがユーザー・ロール・権限情報を読み書きする。

### 方向

```
管理UI → Supabase DB（Tool Sieveテーブル群）
```

### プロトコル

Supabase Client SDK（PostgreSQL）

### 対象テーブル

| テーブル | 操作 |
|---------|------|
| tenants | R（読み取りのみ、Phase 1） |
| users | CRUD |
| auth_accounts | R |
| roles | CRUD |
| user_roles | CRUD |
| role_permissions | CRUD |
| module_registry | RU |

### RLS（Row Level Security）

補助的なセキュリティ層として設定:

```sql
-- usersテーブル: 同一テナント内のみ
CREATE POLICY "Users can view same tenant" ON users
  FOR SELECT USING (tenant_id = current_tenant_id());

-- oauth_tokens: 自分のトークンのみ
CREATE POLICY "Users can view own tokens" ON oauth_tokens
  FOR SELECT USING (user_id = auth.uid() OR user_id IS NULL);
```

---

## IFC-042: 管理UI → Token Broker

### 概要

管理UIがトークン登録・管理をToken Brokerに依頼する。

### 方向

```
管理UI → Token Broker（Edge Function）
```

### プロトコル

HTTPS / REST API

### 操作

| 操作 | メソッド | パス |
|------|---------|------|
| トークン状態確認 | GET | `/api/roles/:id/services/:service` |
| APIトークン登録（タイプA） | PUT | `/api/roles/:id/services/:service` |
| OAuth認可開始（タイプB） | POST | `/api/roles/:id/services/:service/oauth/start` |
| トークン削除 | DELETE | `/api/roles/:id/services/:service/token` |

### リクエスト例（タイプA）

```json
// PUT /api/roles/:id/services/notion
{
  "auth_type": "api_key",
  "api_token": "secret_xxx..."
}
```

### リクエスト例（タイプB）

```json
// PUT /api/roles/:id/services/google
{
  "auth_type": "oauth",
  "client_id": "xxx.apps.googleusercontent.com",
  "client_secret": "GOCSPX-xxx"
}
```

---

## IFC-043: 管理UI → 外部OAuthサーバー

### 概要

管理UIがOAuth認可フローを実行する。

### 方向

```
管理UI → 外部OAuthサーバー（リダイレクト）
```

### プロトコル

OAuth 2.0 Authorization Code Flow

### 認可フロー

```
1. 管理UIが認可URLにリダイレクト
2. ユーザーが外部サービスで同意
3. コールバックURLにリダイレクト（認可コード付き）
4. 認可コードでトークン取得
5. Token Broker経由でVaultに保存
```

### コールバックURL

```
https://mcpist.app/api/oauth/:service/callback
```

### Stateパラメータ

```json
{
  "role_id": "role-uuid",      // 共有トークンの場合
  "user_id": "user-uuid",      // 個人トークンの場合（任意）
  "return_url": "/roles?id=xxx"
}
```

Base64エンコードしてstateパラメータに設定。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [システム仕様書](../DAY5/002-spec-sys/spec-sys.md) | システム全体像 |
| [設計仕様書](../DAY5/004-spec-dsn/spec-dsn.md) | 詳細設計 |
| [dtl-sys-cor.md](../DAY5/dtl-sys-cor.md) | システム仕様サブコア |
