# MCPist システム仕様書（spc-sys）

## ドキュメント管理情報

| 項目      | 値                       |
| ------- | ----------------------- |
| Status  | `reviewed`              |
| Version | v2.1 (Sprint-006)       |
| Note    | System Specification    |

---

## 概要

本ドキュメントは、MCPist のシステムアーキテクチャの骨格を定義する。

---

## コンポーネント一覧

### 実装コンポーネント

| 統一名称（英語）    | 統一名称（日本語）   | 役割                                                                                                  |
| ------------------- | -------------------- | ----------------------------------------------------------------------------------------------------- |
| **API Gateway**     | APIゲートウェイ      | JWT/APIキー検証、X-User-ID/X-Request-ID付与、Rate Limit/Burst制限、OAuth Discovery、バックエンドフェイルオーバー、CORS |
| **Auth Server**     | Authサーバー         | OAuth 2.1準拠、JWT発行、JWKS公開、MCPクライアントに認証情報を付与                                     |
| **Session Manager** | Sessionマネージャー  | ユーザーID発行、ソーシャルログイン連携、セッション管理                                                |
| **Data Store**      | データストア         | ユーザー情報、課金情報、クレジット残高情報、ツール有効/無効設定、モジュール有効設定                   |
| **Token Vault**     | トークン保管庫       | 外部サービスOAuthトークン暗号化保存、APIシークレット保存                                              |
| **MCP Server**      | MCPサーバー          | Auth Middleware, MCP Handler, Modules からなるAPIサーバー                                             |
| **Auth Middleware** | 認証ミドルウェア     | X-Gateway-Secret検証、ユーザーコンテキスト取得（アカウント状態・クレジット残高・有効モジュール・無効ツール）、アクセス制御（モジュール/ツール/クレジット） |
| **MCP Handler**     | MCPハンドラ          | MCPメソッド（tools/list, tools/call）のルーティング、メタツール（get_module_schema, run, batch）提供   |
| **Modules**         | モジュール群         | 外部サービスAPI呼び出し、トークン取得、サービス固有のビジネスロジック実装。Module interface（8メソッド）を実装し、Tool Annotations（ReadOnly/Create/Update/Delete/Destructive）に対応 |
| **Observability**   | オブザーバビリティ   | ツール実行ログ、HTTPリクエストログ、エラーログ、セキュリティイベント記録（権限外アクセス検知）         |
| **User Console**    | ユーザーコンソール   | 外部OAuth連携、外部シークレット保存、ツール有効/無効設定、クレジット課金                              |

### 外部依存（実装範囲外）

| 統一名称（英語）             | 統一名称（日本語）        | 役割                                                         |
| ---------------------------- | ------------------------- | ------------------------------------------------------------ |
| **MCP Client (OAuth2.0)**    | MCPクライアント(OAuth2.0) | LLM Host（Claude.ai, ChatGPT等）からのリクエスト受付、ツール呼び出し、OAuth 2.1認証 |
| **MCP Client (API KEY)**     | MCPクライアント(apikey)   | LLM Host（Claude Code, Cursor等）からのリクエスト受付、ツール呼び出し、API KEY認証  |
| **Identity Provider**        | 認証プロバイダ            | ソーシャルログイン情報の付与（Google, Apple, Microsoft, GitHub）、OAuth 2.0想定 |
| **External Auth Server**     | 外部Authサーバー          | 外部サービスの認可フロー（OAuth 2.0想定）、ユーザーコンソールからの認可リクエスト処理 |
| **External Service API**     | 外部サービスAPI           | 外部サービスのAPI提供、ユーザーごとのリソース（Notion, GitHub, Jira, Confluence, Supabase, Google Calendar, Microsoft To Do, Airtable） |
| **Payment Service Provider** | 決済代行サービス          | Checkout処理、Webhook通知、決済代行（Stripe）                |

---

## コンポーネント責務の概略

### API Gateway

MCPクライアントからのリクエストを最前段で受け付けるエッジコンポーネント。

- JWT / APIキー検証、認証結果に基づく `X-User-ID`・`X-Request-ID` ヘッダ付与
- Rate Limit / Burst制限によるトラフィック制御
- OAuth Discovery エンドポイント提供
- Primary（Koyeb）→ Secondary（Fly.io）のフェイルオーバー、CORS

### Auth Server

OAuth 2.1 準拠の認証基盤。MCPクライアントに認証情報を付与する。

- JWT発行・JWKS公開
- 認可コード発行、トークン交換、リフレッシュトークン管理

### Session Manager

ユーザーアカウント管理とソーシャルログイン連携を担当する。

- ユーザーID発行、セッション管理
- Identity Provider（Google, Apple, Microsoft, GitHub）との認証フロー処理

### Data Store

システム全体の永続データを管理する中央データストア。

- ユーザー情報、課金情報、クレジット残高
- ツール有効/無効設定、モジュール有効/無効設定

### Token Vault

外部サービスのアクセストークンとシークレットを暗号化して保管する。

- OAuthトークン・APIシークレットの暗号化保存
- Gateway/Modules からのリクエストに応じたユーザートークン復号化・提供

### MCP Server

Auth Middleware、MCP Handler、Modulesの3つの内部コンポーネントからなるAPIサーバー。API Gatewayからのリクエストを受け付け、MCPプロトコルに従いツール実行を行う。

### Auth Middleware

API Gatewayから転送されたリクエストの認証・認可を行う MCP Server 内部コンポーネント。

- `X-Gateway-Secret` 検証により API Gateway 経由のリクエストのみ許可
- Data Store からユーザーコンテキスト（アカウント状態・クレジット残高・有効モジュール・無効ツール）を取得
- モジュール/ツール/クレジットに基づくアクセス制御

### MCP Handler

MCPプロトコルのメソッドルーティングとメタツール提供を行う MCP Server 内部コンポーネント。

- `tools/list`、`tools/call` のルーティング
- メタツール: `get_module_schema`、`run`、`batch`
- ツール実行成功時のクレジット消費（冪等）

### Modules

外部サービスAPIとの統合を実現する MCP Server 内部コンポーネント群。

- Module interface（8メソッド）を実装し、Tool Annotations に対応
- Token Vault からトークンを取得し、外部サービスAPIを呼び出す
- 8モジュール: Notion, GitHub, Jira, Confluence, Supabase, Google Calendar, Microsoft To Do, Airtable

### Observability

システムの可観測性を提供するコンポーネント。

- ツール実行ログ、HTTPリクエストログ、エラーログの記録
- セキュリティイベント（権限外アクセス試行）の検知・記録
- `X-Request-ID` によるリクエストトレース

### User Console

ユーザーが自身のアカウント設定、外部サービス連携、課金を管理するWebアプリケーション。

- 外部OAuth連携・シークレット保存（Token Vault へ保存）
- ツール有効/無効設定（Data Store へ書き込み）
- Stripe を通じた Checkout 処理（クレジット課金）

---

## コンポーネント間の関係

コンポーネント間の連携詳細は [spc-itr.md](./spc-itr.md) を参照。

---

## アーキテクチャ図

```
  MCP Client (OAuth2.0)          MCP Client (API KEY)
  [Claude.ai, ChatGPT]           [Claude Code, Cursor]
         │                              │
         │ MCP通信                      │ MCP通信
         │         ┌───────────────┐    │
         │ OAuth認可│               │    │
         ▼         ▼               ▼    ▼           ┌─────────────────┐
  ┌────────────┐  ┌──────────────────────────┐      │  Observability  │
  │Auth Server │  │      API Gateway         │─────▶│  - ログ記録     │
  │- JWT発行   │  │  - JWT/APIキー検証       │      │  - セキュリティ │
  │- JWKS公開  │  │  - X-User-ID 付与       │      │    イベント     │
  └─────┬──────┘  │  - Rate Limit           │      └─────────────────┘
        │         └────────┬─────────────────┘              ▲
        │ トークン検証▲    │ リクエスト転送                 │ ログ送信
        │             │    ▼                                │
  ┌─────┴──────────────┼───────────────────────────┐  ┌────┴──────────────────────┐
  │ Backend Platform   │                           │  │ MCP Server                │
  │                    │                           │  │                           │
  │  ┌──────────────┐  │  ┌─────────────────────┐  │  │  ┌──────────────────────┐ │
  │  │Session Mgr   │  │  │    Data Store       │  │  │  │ Auth Middleware      │ │
  │  │- ユーザーID  │  │  │  - ユーザー情報     │◀─┼──┼──│ - Gateway Secret検証 │ │
  │  │  発行        │  │  │  - 課金・クレジット │  │  │  │ - アクセス制御      │ │
  │  │- ソーシャル  │  │  │  - ツール設定       │  │  │  └──────────┬───────────┘ │
  │  │  ログイン    │  │  └──────────┬──────────┘  │  │             │ 認証済み     │
  │  └──────────────┘  │             │              │  │             ▼              │
  │                    │  ┌──────────┴──────────┐  │  │  ┌──────────────────────┐ │
  │                    │  │   Token Vault       │  │  │  │ MCP Handler          │ │
  │                    │  │  - OAuthトークン    │◀─┼──┼──│ - get_module_schema  │ │
  │                    │  │    暗号化保存       │  │  │  │ - run / batch        │ │
  │                    │  └─────────────────────┘  │  │  └──────────┬───────────┘ │
  │                    │                           │  │             │ ツール実行   │
  │                    │                           │  │             ▼              │
  │                    │                           │  │  ┌──────────────────────┐ │
  │                    │                           │  │  │ Modules（8モジュール）│ │
  │                    │                           │  │  │ - 外部API呼び出し    │ │
  │                    │                           │  │  └──────────┬───────────┘ │
  └────────────────────┼───────────────────────────┘  └────────────┼──────────────┘
                       │                                           │
  ┌────────────────────┼───────────────────────────────────────────┼──────────────┐
  │ External           │                                           │              │
  │                    ▼              ▼              ▼              ▼              │
  │  ┌──────────┐ ┌────────┐ ┌────────────┐ ┌──────────────────────────────────┐ │
  │  │   IdP    │ │  PSP   │ │ Ext Auth   │ │ External Service API             │ │
  │  │ Google   │ │ Stripe │ │ Server     │ │ Notion, GitHub, Jira,            │ │
  │  │ Apple    │ │        │ │            │ │ Confluence, Supabase,            │ │
  │  │ MS / GH  │ │        │ │            │ │ Google Calendar, MS To Do,       │ │
  │  └──────────┘ └────────┘ └────────────┘ │ Airtable                         │ │
  │                                         └──────────────────────────────────┘ │
  └──────────────────────────────────────────────────────────────────────────────┘

  ┌──────────────┐
  │User Console  │──▶ Session Mgr / Data Store / Token Vault / PSP / Ext Auth
  │- OAuth連携   │
  │- ツール設定  │
  │- 課金        │
  └──────────────┘
```


---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-itr.md](./spc-itr.md) | インタラクション仕様書 |
| [spc-dsn.md](./spc-dsn.md) | 設計仕様書（技術スタック） |
| [spc-inf.md](./spc-inf.md) | インフラストラクチャ仕様書 |
| [spc-tbl.md](./spc-tbl.md) | テーブル仕様書 |
