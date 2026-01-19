# MCPist システム仕様書（spc-sys）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | 表記揺らぎを統一した準決定版 |

---

## 概要

本ドキュメントは、MCPist のシステムアーキテクチャの骨格を定義する。

### コンポーネント

| 統一名称（英語） | 統一名称（日本語） | 役割 |
|-----------------|------------------|------|
| **MCP Client** | MCPクライアント | LLMからMCPサーバーへの接続（実装範囲外） |
| **Auth Server** | Authサーバー | OAuth2.1に準拠したAuthサーバー．|
| **MCP Server** | MCPサーバー | Auth Middleware, MCPHandler, Module Registry からなるAPIサーバー |
| **Auth Middleware** | 認証ミドルウェア | MCPクライアントのJWT検証 |
| **Entitlement Store** | 権限ストア | ユーザーの課金状況・ツール利用可否を保持 |
| **MCP Handler** | MCPハンドラ | MCP Protocol処理（tools/list, tools/call） |
| **Module Registry** | モジュールレジストリ | モジュール管理、メタツール提供 |
| **Modules** | モジュール群 | 各外部サービス（Notion, GitHub等）へのアクセス実装 |
| **Token Vault** | トークン保管庫 | トークンの暗号化保存・取得・リフレッシュ |
| **User Console** | ユーザー管理画面 | 課金・ツールの有効/無効設定 |
| **External API Server** | 外部APIサーバー | 各モジュールがアクセスする外部サービスのAPIサーバー |
| **Payment Service Provider** | 決済代行 | 課金処理を行う外部サービス（Stripe） |


## システムアーキテクチャ

### 全体構成

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      MCP Client (Claude Code等)【実装範囲外】                  │
└─────────────────────────────────────────────────────────────────────────────┘
        │                                                      │
        │ OAuth 2.1                                            │ MCP Protocol
        ▼                                                      ▼
┌─────────────────┐                          ┌────────────────────────────────┐
│   Auth Server   │                          │           MCP Server           │
│                 │                          │                                │
│ • OAuth 2.1準拠  │                          │  ┌────────────────────────┐   │
│ • JWT発行        │ ◀─── JWT検証 ─────────── │  │   Auth Middleware      │   │
│ • JWKS公開       │                          │  └───────────┬────────────┘   │
└─────────────────┘                          │              ▼                │
                                             │  ┌────────────────────────┐   │
┌─────────────────┐                          │  │     MCP Handler        │   │
│       PSP       │                          │  │  • tools/list, call    │   │
│    (Stripe)     │                          │  └───────────┬────────────┘   │
│                 │                          │              ▼                │
│ • Webhook       │                          │  ┌────────────────────────┐   │
│ • Checkout      │                          │  │   Module Registry      │   │
└────────┬────────┘                          │  │  • get_module_schema   │   │
         │                                   │  │  • call / batch        │   │
         │ 課金同期                           │  └───────────┬────────────┘   │
         ▼                                   │              ▼                │
┌─────────────────┐                          │  ┌────────────────────────┐   │
│Entitlement Store│ ◀──────── 参照 ──────────│  │  Modules (Notion等)    │   │
│  (Supabase PG)  │                          │  └──────────┬─────────────┘   │
│                 │                          │             │                 │
│ • users         │                          └─────────────┼─────────────────┘
│ • subscriptions │                                        │
│ • plans         │                                        │ トークン取得
│ • user_tool_pref│                                        ▼
└───────▲─────────┘                          ┌─────────────────┐
        │                                    │   Token Vault   │
        │ 設定                               │ (Supabase Vault)│
        │                            ┌──────▶│                 │
┌───────┴─────────┐   OAuth連携      │       │ • oauth_tokens  │
│  User Console   │ ─────────────────┘       └─────────────────┘
│                 │                                    │
│ • 課金設定       │                                    │ HTTPS
│ • ツール有効/無効 │                                    ▼
│ • OAuth連携      │                          ┌────────────────────────────────┐
└─────────────────┘                          │    External API Server         │
                                             │                                │
                                             │  Notion │ Google Calendar │ Microsoft Todo  │
                                             └────────────────────────────────┘
```

---

## コンポーネント詳細

### MCP Client（MCPクライアント）

- LLM Host（Claude Code, Cursor等）からMCPサーバーへ接続するクライアント。
- OAuth 2.1でAuth Serverから認証を受け、MCP ProtocolでMCP Serverと通信する。
- **実装範囲外**。

### Auth Server（Authサーバー）

- OAuth 2.1に準拠した認証サーバー。
- JWTの発行とJWKS公開鍵の提供を行う。
- MCP ServerはJWKSを参照してJWTを検証する。

### MCP Server（MCPサーバー）

- MCPプロトコルを処理するAPIサーバー。以下の内部コンポーネントで構成される：
	- **Auth Middleware**: JWTを検証し、user_idをcontextに抽出する
	- **MCP Handler**: `tools/list`、`tools/call`等のMCPリクエストを処理する
	- **Module Registry**: モジュールを管理し、メタツール（`get_module_schema`、`call`、`batch`）を提供する
	- **Modules**: 各外部サービス（Notion, Google Calendar, Microsoft Todo）へのアクセスを実装する

### User Console（ユーザー管理画面）

- ユーザーが課金設定、ツールの有効/無効、外部サービスとのOAuth連携を行うWebアプリケーション。
- Entitlement Storeへの設定書き込みと、Token VaultへのOAuthトークン登録を行う。

### Entitlement Store（権限ストア）

- ユーザーの課金状況とツール利用可否を保持するデータストア。

**保持するデータ**:
- `users`: ユーザー情報、アカウント状態（active/suspended/disabled）
- `subscriptions`: 課金状態（プラン、有効期限）
- `plans`: プラン定義（Rate Limit値、Quota上限）
- `user_tool_preferences`: ユーザーごとのツール有効/無効設定

### Token Vault（トークン保管庫）

- 外部サービスのOAuthトークンを暗号化保存するデータストア。
- Modulesからの要求に応じてトークンを復号化して返す。
- 期限切れ時は自動リフレッシュを行う。

### External API Server（外部APIサーバー）

- 各モジュールがアクセスする外部サービスのAPIサーバー（Notion API, Google Calendar API, Microsoft Graph API等）。
- ModulesがToken Vaultから取得したトークンを使ってHTTPSでアクセスする。
- **実装範囲外**。

### Payment Service Provider（決済代行）

- 課金処理を行う外部サービス（Stripe）。
- Entitlement StoreとWebhook/APIで課金情報を同期する。
- **実装範囲外**。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dtl-core.md](../DAY5/dtl-core.md) | コア機能定義（COR-001〜009） |
| [dtl-sys-cor.md](../DAY5/dtl-sys-cor.md) | システム仕様サブコア定義 |
| [spec-ifc.md](../DAY6/spec-ifc.md) | インターフェース仕様（IFC-001〜043） |
| [dsn-module-registry.md](../DAY7/dsn-module-registry.md) | Module Registry設計 |
| [dsn-permission-system.md](../DAY7/dsn-permission-system.md) | 権限システム設計 |
| [dsn-subscription.md](../DAY7/dsn-subscription.md) | サブスクリプション設計 |
| [adr-usage-control-architecture.md](../DAY7/adr-usage-control-architecture.md) | 使用量制御ADR |

