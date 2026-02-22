# MCPist システム仕様書（spc-sys）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | System Specification — 現行実装に基づく全面改訂 |

---

## 概要

MCPist は、MCP (Model Context Protocol) を通じて外部サービスへの接続を提供するプラットフォームである。
本ドキュメントは、MCPist を構成するコンポーネントとその責務を定義する。

---

## 実装コンポーネント

MCPist は 4 つの実装コンポーネントで構成される。

| コンポーネント | ディレクトリ | 言語 | デプロイ先 | 役割 |
|---|---|---|---|---|
| **Server** | `apps/server/` | Go | Render | MCP プロトコル処理、REST API、認可、モジュール実行 |
| **Worker** | `apps/worker/` | TypeScript | Cloudflare Workers | API ゲートウェイ、認証、リクエストプロキシ |
| **Console** | `apps/console/` | TypeScript (Next.js) | Vercel | ユーザー管理画面 |
| **Database** | `database/` | SQL | Neon PostgreSQL | データ永続化 |

---

## コンポーネント責務

### Server (`apps/server/`)

MCP プロトコルのハンドリングと REST API を提供する Go サーバー。

**内部パッケージ構成:**

| パッケージ | 責務 |
|---|---|
| `cmd/server/` | エントリーポイント。モジュール登録、DB 接続、ルーティング設定 |
| `internal/auth/` | Ed25519 鍵管理、API キー JWT 発行、Gateway JWT 検証 (JWKS) |
| `internal/broker/user.go` | ユーザーコンテキスト取得 (30 秒キャッシュ)、使用量記録 |
| `internal/broker/token.go` | OAuth2 トークン自動リフレッシュ (13 モジュール / 7 プロバイダ対応) |
| `internal/db/` | GORM モデル定義、リポジトリ、AES-256-GCM 暗号化 |
| `internal/middleware/` | Authorization、Rate Limit、SSE/Inline Transport、Recovery |
| `internal/mcp/` | MCP ハンドラ (initialize, tools/list, tools/call, prompts/list, prompts/get) |
| `internal/modules/` | モジュールの登録・実行。Module interface 実装 |
| `internal/observability/` | Grafana Loki へのログ送信 |
| `internal/ogenserver/` | OpenAPI (ogen) 生成の REST API ハンドラ |
| `internal/jsonrpc/` | JSON-RPC 2.0 型定義 |
| `pkg/` | ogen 生成 API クライアント (モジュール別) |

**HTTP ルーティング:**

| パス | メソッド | 説明 |
|---|---|---|
| `/health` | GET | ヘルスチェック |
| `/v1/mcp` | GET/POST | MCP エンドポイント — GET: SSE 接続、POST: インライン / セッション (Authorization → RateLimit → Transport → Recovery) |
| `/.well-known/jwks.json` | GET | API キー検証用公開鍵 |
| `/v1/*` | * | ogen 生成 REST API (Console 向け) |
| `/v1/stripe/webhook` | POST | Stripe Webhook 受信 |

**MCP メタツール:**

| ツール | 説明 |
|---|---|
| `get_module_schema` | モジュールのスキーマ (ツール定義) を返す |
| `run` | 単一ツール実行 |
| `batch` | 最大 10 コマンドの JSONL バッチ実行 |

**ミドルウェアスタック (MCP エンドポイント):**

```
Authorization → RateLimit → Transport → Recovery → MCP Handler
```

1. **Authorization** — Gateway JWT 検証、ユーザーコンテキスト取得、アカウント状態確認
2. **RateLimit** — インメモリ sliding window (10 req/sec per user)
3. **Transport** — SSE / Inline JSON-RPC トランスポート管理
4. **Recovery** — パニックキャッチ、Loki へログ送信

---

### Worker (`apps/worker/`)

Cloudflare Workers 上で動作する API ゲートウェイ。全リクエストの認証と Server へのプロキシを担当する。

**責務:**

| 機能 | 説明 |
|---|---|
| 認証 | Clerk JWT 検証 (JWKS)、API キー検証 (mpt_* 形式 JWT) |
| Gateway JWT 発行 | Worker → Server 間の Ed25519 署名 JWT (30 秒有効) |
| API キーキャッシュ | インメモリ、5 分 TTL。削除時に即時無効化 |
| リクエストプロキシ | `/v1/mcp/*`, `/v1/me/*`, `/v1/admin/*`, `/v1/oauth/*`, `/v1/stripe/webhook` |
| OAuth Discovery | RFC 8414 / 9728 メタデータ提供 |
| ヘルスチェック | 5 分間隔の cron トリガーで Server の死活監視 |
| ログ | リクエストログ・セキュリティイベントを Grafana Loki に送信 |

**ルーティング:**

| パス | 説明 |
|---|---|
| `/health` | Worker + Server ヘルスチェック |
| `/openapi.json` | OpenAPI 3.1 仕様 |
| `/.well-known/oauth-*` | OAuth Discovery メタデータ |
| `/.well-known/jwks.json` | Gateway 公開鍵 |
| `/v1/mcp/*` | MCP プロキシ (認証必須) |
| `/v1/me/*` | ユーザー操作 REST プロキシ (認証必須) |
| `/v1/admin/*` | 管理者操作 REST プロキシ (認証必須) |
| `/v1/modules` | モジュール一覧 (認証不要) |
| `/v1/stripe/webhook` | Stripe Webhook パススルー |

---

### Console (`apps/console/`)

ユーザーが自身のアカウント設定、外部サービス連携、課金を管理する Next.js Web アプリケーション。

**責務:**

| 機能 | 説明 |
|---|---|
| 認証 | Clerk によるソーシャルログイン (Google, Apple, Microsoft, GitHub) |
| サービス連携 | OAuth フロー・API キー入力による外部サービス接続 |
| 資格情報検証 | サービス固有のバリデータで接続テスト |
| ツール設定 | モジュール・ツールの有効/無効切り替え |
| API キー管理 | MCP クライアント用 API キーの発行・失効 |
| 使用量ダッシュボード | 日次・月次の使用量表示 |
| プラン管理 | Stripe 連携による課金・プランアップグレード |
| プロンプト管理 | ユーザー定義プロンプトの CRUD |
| 管理者画面 | OAuth アプリ登録・管理 |

**データアクセス:**

Console は DB に直接アクセスしない。すべてのデータ操作は Worker API (`/v1/*`) 経由で行う。

```
Console → Server Action → Worker API Client (openapi-fetch) → Worker → Server → DB
```

---

### Database (`database/`)

Neon PostgreSQL 上の `mcpist` スキーマ。GORM によるアクセス。

**テーブル一覧:**

| テーブル | 説明 |
|---|---|
| `users` | ユーザー (clerk_id, plan_id, account_status, stripe_customer_id) |
| `plans` | プランマスタ (free / plus / team) |
| `modules` | モジュールマスタ (Go Server から同期) |
| `module_settings` | ユーザーごとのモジュール有効/無効・説明文 |
| `tool_settings` | ユーザーごとのツール有効/無効 |
| `prompts` | ユーザー定義プロンプト |
| `api_keys` | JWT ベース API キー (jwt_kid, expires_at) |
| `user_credentials` | 外部サービス資格情報 (AES-256-GCM 暗号化) |
| `oauth_apps` | OAuth プロバイダ設定 (client_secret 暗号化) |
| `usage_log` | ツール実行ログ (非同期記録) |
| `processed_webhook_events` | Stripe Webhook 冪等性管理 |

**詳細は [spc-tbl.md](spec-tables.md) を参照。**

---

## 外部依存

| サービス | 役割 |
|---|---|
| **Clerk** | ユーザー認証、ソーシャルログイン、JWT 発行 |
| **Neon** | PostgreSQL ホスティング |
| **Stripe** | 決済代行、Webhook 通知 |
| **Grafana Cloud (Loki)** | ログ収集 |
| **Cloudflare** | Worker ホスティング、DNS |
| **Render** | Server ホスティング |
| **Vercel** | Console ホスティング |
| **GitHub** | ソースコード管理、CI/CD (Actions) |

---

## 対応モジュール

| カテゴリ | モジュール |
|---|---|
| プロジェクト管理 | Notion, Asana, Jira, Todoist, TickTick, Trello, Microsoft ToDo, Google Tasks |
| 開発 | GitHub |
| ドキュメント | Confluence, Google Docs |
| カレンダー | Google Calendar |
| ストレージ | Google Drive, Dropbox |
| スプレッドシート | Google Sheets, Airtable |
| スクリプト | Google Apps Script |
| クラウド | Supabase, PostgreSQL, Grafana |

---

## リクエストフロー

### MCP クライアントからのリクエスト

```
MCP Client (Claude Code, Cursor, Claude.ai 等)
    │
    │ Bearer JWT or mpt_* API Key
    ▼
┌──────────────────────────────────────────┐
│ Worker (Cloudflare Workers)              │
│  1. 認証 (Clerk JWT / API Key JWT)       │
│  2. Gateway JWT 発行 (Ed25519, 30s)      │
│  3. Server へプロキシ                     │
└──────────────┬───────────────────────────┘
               │ X-Gateway-Token
               ▼
┌──────────────────────────────────────────┐
│ Server (Go / Render)                     │
│  1. Gateway JWT 検証                     │
│  2. ユーザーコンテキスト取得 (DB/キャッシュ) │
│  3. 認可チェック (モジュール/ツール/上限)  │
│  4. MCP Handler → Module 実行            │
│  5. 使用量記録 (非同期)                   │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ External Service API                     │
│  (Notion, GitHub, Jira, Google 等)       │
└──────────────────────────────────────────┘
```

### Console からのリクエスト

```
Console (Next.js / Vercel)
    │
    │ Server Action → openapi-fetch
    │ Bearer Clerk JWT
    ▼
Worker → Server → Database (Neon)
```

---

## セキュリティ

**詳細は [spc-sec.md](spec-security.md) を参照。**

---

## 関連ドキュメント

| ドキュメント | 内容 |
|---|---|
| [spc-tbl.md](spec-tables.md) | テーブル仕様書 |
| [spc-dsn.md](spec-design.md) | 設計仕様書 (技術スタック) |
| [spc-inf.md](spec-infrastructure.md) | インフラ仕様書 |
| [spc-itf.md](spec-interface.md) | インターフェース仕様書 |
| [spc-sec.md](spec-security.md) | セキュリティ仕様書 |
