# MCPist システム仕様書（spc-sys）

## 概要

本ドキュメントは、MCPist のシステムアーキテクチャの骨格を定義する。

---

## コンポーネント一覧

### 実装コンポーネント

| 統一名称（英語）    | 統一名称（日本語）   | 役割                                                                                                  |
| ------------------- | -------------------- | ----------------------------------------------------------------------------------------------------- |
| **API Gateway**     | APIゲートウェイ      | ロードバランシング、ユーザー認証（JWT/API KEY検証）、MCPサーバーへのリクエスト転送                    |
| **Auth Server**     | Authサーバー         | OAuth 2.1準拠、JWT発行、JWKS公開、MCPクライアントに認証情報を付与                                     |
| **Session Manager** | Sessionマネージャー  | ユーザーID発行、ソーシャルログイン連携、セッション管理                                                |
| **Data Store**      | データストア         | ユーザー情報、課金情報、クレジット残高情報、ツール有効/無効設定                                       |
| **Token Vault**     | トークン保管庫       | 外部サービスシークレット、OAuthリフレッシュトークン、MCPサーバー認証シークレット/リフレッシュトークン |
| **MCP Server**      | MCPサーバー          | Auth Middleware, MCP Handler, Modules からなるAPIサーバー                                             |
| **Auth Middleware** | 認証ミドルウェア     | X-Gateway-Secret検証、認証済みリクエストをMCP Handlerに転送                                           |
| **MCP Handler**     | MCPハンドラ          | MCPメソッド（tools, resources, prompts）のルーティング、モジュール管理、メタツール（get_module_schema, run, batch）提供 |
| **Modules**         | モジュール群         | 外部サービスAPI呼び出し、トークン取得、サービス固有のビジネスロジック実装                             |
| **User Console**    | ユーザーコンソール   | 外部OAuth連携、外部シークレット保存、ツール有効/無効設定、クレジット課金                              |

### 外部依存（実装範囲外）

| 統一名称（英語）             | 統一名称（日本語）        | 役割                                                         |
| ---------------------------- | ------------------------- | ------------------------------------------------------------ |
| **MCP Client (OAuth2.0)**    | MCPクライアント(OAuth2.0) | LLM Host（Claude Code, Cursor等）からのリクエスト受付、ツール呼び出し、OAuth 2.1認証 |
| **MCP Client (API KEY)**     | MCPクライアント(apikey)   | LLM Host（Claude Code, Cursor等）からのリクエスト受付、ツール呼び出し、API KEY認証  |
| **Identity Provider**        | 認証プロバイダ            | ソーシャルログイン情報の付与（Google, GitHub等）、OAuth 2.0想定 |
| **External Auth Server**     | 外部Authサーバー          | 外部サービスの認可フロー（OAuth 2.0想定）、ユーザーコンソールからの認可リクエスト処理 |
| **External Service API**     | 外部サービスAPI           | 外部サービスのAPI提供、ユーザーごとのリソース（Notion, GitHub, Google Calendar, Jira等） |
| **Payment Service Provider** | 決済代行サービス          | クレジットカード情報管理、プラン情報管理、Webhook通知、Checkout処理（Stripe） |

---

## コンポーネント間の関係

| From                     | To                       | ラベル             | 説明                                           |
| ------------------------ | ------------------------ | ------------------ | ---------------------------------------------- |
| MCP Client (OAuth2.0)    | API Gateway              | MCP通信            | JSON-RPC over SSE                              |
| MCP Client (OAuth2.0)    | Auth Server              | 認証               | OAuth 2.1認証フロー（認可コード取得、トークン交換） |
| MCP Client (API KEY)     | Token Vault              | 認証               | API KEY認証                                    |
| API Gateway              | Auth Middleware          | -                  | 認証済みリクエストの転送                       |
| Auth Server              | API Gateway              | JWT                | JWT提供・検証                                  |
| Token Vault              | API Gateway              | API KEY            | API KEY提供・検証                              |
| Identity Provider        | Session Manager          | ID連携             | ソーシャルログイン情報の付与                   |
| Session Manager          | Data Store               | ユーザーID共有     | ユーザー情報の登録・参照                       |
| Data Store               | Auth Server              | ユーザーID共有     | 認証時のユーザー情報参照                       |
| Data Store               | Token Vault              | ユーザーID共有     | トークン管理のためのユーザー紐付け             |
| Data Store               | MCP Handler              | ユーザー設定       | ツール有効/無効設定、利用可否判定、カスタムプロンプト取得 |
| Payment Service Provider | Data Store               | プラン情報         | 課金情報の同期（Webhook/API）                  |
| User Console             | Payment Service Provider | 決済               | 決済リクエスト、Checkout処理                   |
| User Console             | Token Vault              | トークン登録       | OAuth連携完了時のトークン保存                  |
| User Console             | Data Store               | ツール設定登録     | ツール有効/無効設定の書き込み                  |
| User Console             | External Auth Server     | 認可フロー         | 外部サービスOAuth認可                          |
| User Console             | Identity Provider        | -                  | ソーシャルログイン                             |
| External Auth Server     | Token Vault              | 認証               | 認可完了後のトークン受信                       |
| Token Vault              | Modules                  | トークン取得       | 外部サービスアクセス用トークンの復号化・提供   |
| Modules                  | External Service API     | リソースアクセス   | HTTPS + Bearer Token                           |
| Auth Middleware          | MCP Handler              | -                  | 認証済みリクエストの転送                       |
| MCP Handler              | Modules                  | -                  | プリミティブ操作委譲、スキーマ取得             |

---

## アーキテクチャ図

![[mcpist-system-architecture.canvas]]
