# MCPist インターフェース仕様書（spc-itf）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | Interface Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist の外部インターフェースを定義する。

**範囲:**
- MCP プロトコル (JSON-RPC 2.0)
- REST API エンドポイント
- Worker パブリックエンドポイント

---

## MCP プロトコル

### 基本仕様

| 項目 | 値 |
|---|---|
| プロトコルバージョン | 2025-03-26 |
| 形式 | JSON-RPC 2.0 |
| トランスポート | SSE / Inline |
| 認証 | Bearer Token (Clerk JWT or `mpt_*` API Key) |

### サーバー情報

| 項目 | 値 |
|---|---|
| name | `mcpist` |
| version | `0.1.0` |
| capabilities | tools (listChanged), prompts (listChanged) |

### エンドポイント

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/v1/mcp` | SSE 接続 (セッション生成) |
| POST | `/v1/mcp` | JSON-RPC インラインリクエスト |
| POST | `/v1/mcp?sessionId=...` | JSON-RPC SSE セッションリクエスト |

### トランスポート

**SSE:**

1. `GET /v1/mcp` で SSE 接続
2. `event: endpoint` でセッション URL を返却: `/mcp?sessionId=<id>`
3. クライアントが `POST /mcp?sessionId=<id>` に JSON-RPC を送信
4. 結果は `event: message` で返却

**Inline:**

1. `POST /v1/mcp` に JSON-RPC を送信
2. JSON-RPC レスポンスを直接返却

### JSON-RPC メソッド

| メソッド | 説明 |
|---|---|
| `initialize` | プロトコルバージョン、capabilities、サーバー情報を返却 |
| `initialized` | クライアントからの初期化完了通知 |
| `tools/list` | メタツール一覧を返却 (ユーザーの有効モジュールに応じて動的生成) |
| `tools/call` | メタツール実行 (get_module_schema / run / batch) |
| `prompts/list` | ユーザー定義プロンプト一覧 |
| `prompts/get` | プロンプト取得 |

### メタツール

`tools/list` は常に 3 つのメタツールのみを返す。

**get_module_schema:**

モジュールのツール定義を取得する。

```json
{
  "name": "get_module_schema",
  "arguments": {
    "module": ["notion", "jira"]
  }
}
```

- `module` は配列が正規形 (文字列 1 件でも後方互換で受理)
- 無効/未登録モジュールは警告として返し、他のモジュールは返却を継続

**run:**

単一ツール実行。

```json
{
  "name": "run",
  "arguments": {
    "module": "notion",
    "tool": "search",
    "params": { "query": "設計" }
  }
}
```

**batch:**

JSONL 形式で最大 10 コマンドを DAG 実行。依存がなければ並列。

```json
{
  "name": "batch",
  "arguments": {
    "commands": "{\"id\":\"a\",\"module\":\"notion\",\"tool\":\"search\",\"params\":{\"query\":\"設計\"}}\n{\"id\":\"b\",\"module\":\"notion\",\"tool\":\"get_page_content\",\"params\":{\"page_id\":\"${a.results[0].id}\"},\"after\":[\"a\"],\"output\":true}"
  }
}
```

| フィールド | 必須 | 説明 |
|---|---|---|
| id | Yes | タスク識別子 |
| module | Yes | モジュール名 |
| tool | Yes | ツール名 |
| params | No | ツールパラメータ |
| after | No | 依存タスク ID 配列 |
| output | No | true で結果をレスポンスに含む |

変数参照: `${taskId.results[index].field}`

### レスポンス形式

```json
{
  "content": [
    { "type": "text", "text": "..." }
  ],
  "isError": false
}
```

デフォルトはコンパクト形式 (CSV/MD)。`params` に `format: "json"` を指定すると JSON 形式。

### エラーコード

| コード | 名前 | 説明 |
|---|---|---|
| -32700 | ParseError | JSON パース失敗 |
| -32600 | InvalidRequest | 不正なリクエスト |
| -32601 | MethodNotFound | 未知のメソッド |
| -32602 | InvalidParams | パラメータ不備 |
| -32603 | InternalError | サーバー内部エラー |
| -32001 | PermissionDenied | モジュール/ツール未有効化 |
| -32002 | UsageLimitExceeded | 日次使用量上限超過 |

-32700 〜 -32603 は JSON-RPC 2.0 標準。-32001, -32002 は MCPist 独自定義。

### LLM エラーハンドリングポリシー

**カテゴリ 1: ユーザーに伝えて停止**

認証エラー、認可エラー、内部エラーなど LLM 側で解決できないもの。

**カテゴリ 2: LLM が対応を変えるべきエラー**

ToolCallResult の `isError: true` のうち:

| 外部 API ステータス | 対応 |
|---|---|
| 404 Not Found | パラメータを修正して再試行 |
| 409 Conflict | パラメータを変えて再試行 |
| 429 Too Many Requests | 時間を置いて再試行 |
| 401/403 (外部サービス) | ユーザーに再接続を案内 |

---

## REST API

Worker が全リクエストを認証し、Server にプロキシする。

### 認証不要

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/health` | ヘルスチェック |
| GET | `/openapi.json` | OpenAPI 仕様 |
| GET | `/.well-known/oauth-protected-resource` | OAuth Protected Resource Metadata (RFC 9728) |
| GET | `/.well-known/oauth-authorization-server` | OAuth Authorization Server Metadata (RFC 8414) |
| GET | `/.well-known/jwks.json` | Gateway 公開鍵 |
| GET | `/v1/modules` | モジュール一覧 |

### ユーザー操作 (`/v1/me/*`)

認証必須 (Clerk JWT or API Key)。

| メソッド | パス | 説明 |
|---|---|---|
| POST | `/v1/me/register` | ユーザー登録 / 既存ユーザー検索 |
| GET | `/v1/me/profile` | プロフィール取得 |
| PUT | `/v1/me/settings` | ユーザー設定更新 |
| POST | `/v1/me/onboarding` | オンボーディング完了 |
| GET | `/v1/me/usage` | 使用量統計 |
| GET | `/v1/me/stripe` | Stripe 顧客 ID 取得 |
| PUT | `/v1/me/stripe` | Stripe 顧客 ID 紐付け |
| GET | `/v1/me/credentials` | 資格情報一覧 |
| PUT | `/v1/me/credentials/{module}` | 資格情報登録/更新 |
| DELETE | `/v1/me/credentials/{module}` | 資格情報削除 |
| GET | `/v1/me/apikeys` | API キー一覧 |
| POST | `/v1/me/apikeys` | API キー発行 |
| DELETE | `/v1/me/apikeys/{id}` | API キー失効 |
| GET | `/v1/me/prompts` | プロンプト一覧 |
| GET | `/v1/me/prompts/{id}` | プロンプト取得 |
| POST | `/v1/me/prompts` | プロンプト作成 |
| PUT | `/v1/me/prompts/{id}` | プロンプト更新 |
| DELETE | `/v1/me/prompts/{id}` | プロンプト削除 |
| GET | `/v1/me/modules/config` | モジュール設定取得 |
| PUT | `/v1/me/modules/{name}/tools` | ツール有効/無効設定 |
| PUT | `/v1/me/modules/{name}/description` | モジュール説明文更新 |
| GET | `/v1/me/oauth/consents` | OAuth 同意一覧 |
| DELETE | `/v1/me/oauth/consents/{id}` | OAuth 同意取消 |

### OAuth (`/v1/oauth/*`)

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/v1/oauth/apps/{provider}/credentials` | OAuth アプリ資格情報取得 |

### 管理者操作 (`/v1/admin/*`)

認証必須 + 管理者権限。

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/v1/admin/oauth/apps` | OAuth アプリ一覧 |
| PUT | `/v1/admin/oauth/apps/{provider}` | OAuth アプリ登録/更新 |
| DELETE | `/v1/admin/oauth/apps/{provider}` | OAuth アプリ削除 |
| GET | `/v1/admin/oauth/consents` | 全ユーザーの OAuth 同意一覧 |

### 内部 API

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/v1/internal/apikeys/{id}/status` | API キー有効性確認 (Worker → Server) |

### Stripe Webhook

| メソッド | パス | 説明 |
|---|---|---|
| POST | `/v1/stripe/webhook` | Stripe Webhook 受信 |

---

## 関連ドキュメント

| ドキュメント                                             | 内容        |
| -------------------------------------------------- | --------- |
| [spec-systems.md](./spec-systems.md)               | システム仕様書   |
| [spec-design.md](./spec-design.md)                 | 設計仕様書     |
| [spec-infrastructure.md](./spec-infrastructure.md) | インフラ仕様書   |
| [spec-security.md](./spec-security.md)             | セキュリティ仕様書 |
