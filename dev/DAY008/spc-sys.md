# MCPist システム仕様書（spc-sys）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.3 (Sprint-002) |
| Note | 画像ベースのアーキテクチャ図に合わせて更新 |

---

## 概要

本ドキュメントは、MCPist のシステムアーキテクチャの骨格を定義する。

### コンポーネント

#### 実装コンポーネント

| 統一名称（英語） | 統一名称（日本語） | 役割 |
|-----------------|------------------|------|
| **Auth Server** | Authサーバー | OAuth 2.1準拠、JWT発行、JWKS公開、MCPクライアントに認証情報を与える |
| **MCP Server** | MCPサーバー | Auth Middleware, MCP Handler, Module Registry, Modules からなるAPIサーバー |
| **Auth Middleware** | 認証ミドルウェア | JWT検証 |
| **MCP Handler** | MCPハンドラ | tools/list, call, resources, prompts 処理 |
| **Module Registry** | モジュールレジストリ | get_module_schema, run/batch 提供 |
| **Modules** | モジュール群 | MCPリソース、MCPツール、MCPプロンプト、外部サービスへのリクエスト |
| **Entitlement Store** | 権限ストア | ユーザー情報、課金情報、クレジット残高情報、ツール有効/無効設定情報 |
| **Token Vault** | トークン保管庫 | 外部サービスシークレット、OAuthリフレッシュトークン、MCPサーバー認証シークレット/リフレッシュトークン |
| **User Console** | ユーザー管理画面 | 外部OAuth連携、外部シークレット保存、ツール有効/無効設定、クレジット課金 |

#### 外部依存（実装範囲外）

| 統一名称（英語） | 統一名称（日本語） | 役割 | 依存元 |
|-----------------|------------------|------|--------|
| **MCP Client** | MCPクライアント | LLMからリクエスト受付、ツール呼び出し | - |
| **Identity Provider** | 認証プロバイダ | ソーシャルログイン情報の付与（Google, GitHub等） | Auth Server |
| **External Web Service** | 外部WEBサービス | 外部Authサーバー + 外部サービスAPI | User Console, Modules |
| **Payment Service Provider** | 決済代行サービス | クレジットカード情報、プラン情報、Webhook | Entitlement Store |

---

## システムアーキテクチャ

### 全体構成

![[mcpist-systems.jpg]]

---

## コンポーネント詳細

### Authサーバー（Auth Server）

- OAuth 2.1に準拠した認証サーバー。
- JWTの発行とJWKS公開鍵の提供を行う。
- MCPクライアントに認証情報を与える。
- MCP ServerはJWKSを参照してJWTを検証する。
- **Entitlement Store連携**: 認証成功時にユーザー情報をEntitlement Storeに登録・参照する。
- **外部依存**: 認証プロバイダ（Identity Provider）を利用してソーシャルログインを実現。

### MCPサーバー（MCP Server）

MCPプロトコルを処理するAPIサーバー。以下の内部コンポーネントで構成される：

#### Authミドルウェア（Auth Middleware）
- JWTを検証し、user_idをcontextに抽出する。

#### MCPハンドラ（MCP Handler）
- `tools/list`、`tools/call` のMCPリクエストを処理する。
- `resources`、`prompts` のMCPリクエストを処理する。

#### モジュールレジストリ（Module Registry）
- モジュールを管理し、メタツールを提供する。
- `get_module_schema`: モジュールのスキーマ取得
- `run` / `batch`: モジュールツールの実行
- **Entitlement Store参照**: ツール設定と統計情報を取得し、利用可否の判定やルーティングを行う。

#### モジュール群（Modules）
- 各外部サービス（Notion, GitHub等）へのアクセスを実装する。
- **MCPリソース**: 外部サービスのリソース提供
- **MCPツール**: 外部サービスの操作機能
- **MCPプロンプト**: 定義済みプロンプト
- **Token Vault参照**: 外部サービスアクセス時にトークンを取得する。
- **外部サービスへのリクエスト**: 取得したトークンを使用してHTTPアクセスを行う。

### ユーザーコンソール（User Console）

ユーザーが各種設定を行うWebアプリケーション。

**機能:**
- 外部OAuth連携
- 外部シークレット保存
- ツール有効/無効設定
- クレジット課金・決済

**連携先:**
- Entitlement Storeへの設定書き込み
- Token VaultへのOAuthトークン登録
- 外部WEBサービスへの認可フロー
- 決済代行サービス（PSP）への決済リクエスト

### Entitlement Store（権限ストア）

ユーザーの課金状況とツール利用可否を保持するデータストア。

**保持するデータ:**
- ユーザー情報
- 課金情報
- クレジット残高情報
- ツール有効/無効設定情報
- 使用統計情報

**連携:**
- Auth Serverからユーザー情報の登録・参照を受ける
- Module Registryにツール設定・統計情報を提供する
- 決済代行サービス（PSP）からプラン情報を参照し、課金情報の登録を受ける

### Token Vault（トークン保管庫）

外部サービスのOAuthトークンを暗号化保存するデータストア。

**保持するデータ:**
- 外部サービスシークレット
- OAuthリフレッシュトークン
- MCPサーバー認証シークレット
- MCPサーバー認証リフレッシュトークン

**機能:**
- Modulesからの要求に応じてトークンを復号化して返す
- トークン期限切れ時は外部WEBサービスのAuthサーバーに対して自動リフレッシュを行う

**連携:**
- User Consoleからトークン登録を受ける
- Modulesにトークンを提供する
- 外部WEBサービス（外部Authサーバー）に対してトークンリフレッシュを要求する

---

## 外部依存（実装範囲外）

以下のコンポーネントはMCPistの動作に必要だが、実装範囲外である。

### MCPクライアント（MCP Client）

- LLM Host（Claude Code, Cursor等）からMCPサーバーへ接続するクライアント。
- LLMからリクエストを受付し、ツール呼び出しを行う。
- OAuth 2.1でAuth Serverから認証を受け、MCP ProtocolでMCP Serverと通信する。

### 認証プロバイダ（Identity Provider）

- ソーシャルログイン情報の付与を行う。
- OAuth 2.0想定（Google, GitHub等）。
- **依存元**: Auth Server（ユーザー登録・ログイン時のみ）
- **用途**: 初回認証のみ。JWT発行後はAuth Serverが独立して動作。

### 外部WEBサービス（External Web Service）

各モジュールがアクセスする外部サービス。2つのコンポーネントで構成される。

#### 外部Authサーバー
- 認可フロー（OAuth 2.0想定）
- ユーザーコンソールからの認可リクエストを処理

#### 外部サービスAPI
- API提供
- ユーザーごとのリソース
- モジュールからHTTPアクセスを受ける

**対象サービス**: Notion API, GitHub API, Google Calendar API, Jira API等

### 決済代行サービス（Payment Service Provider）

課金処理を行う外部サービス（Stripe）。

**機能:**
- クレジットカード情報管理
- プラン情報管理
- Webhook通知
- Checkout処理

**連携:**
- User Consoleから決済リクエストを受ける
- Entitlement Storeにプラン情報を提供する
- 課金完了時にEntitlement Storeへ課金情報を登録（Webhook経由）

---

## データフロー

### 認証フロー

```
MCPクライアント → Authサーバー ─→ 認証プロバイダ [外部依存]
                     │              (Google, GitHub等)
                     ├─→ Entitlement Store（ユーザー情報登録/参照）
                     ▼ JWT発行
               MCPクライアント → MCPサーバー（JWT検証）
```

**補足**: 認証プロバイダはユーザー登録・ログイン時のみ使用。通常のAPI呼び出しでは関与しない。Auth Serverは認証成功時にEntitlement Storeへユーザー情報を登録・参照する。

### ツール呼び出しフロー

```
MCPクライアント → MCPサーバー
                     │
                     ├─→ Authミドルウェア（JWT検証 → Authサーバー参照）
                     ├─→ MCPハンドラ（tools/list, call）
                     ├─→ モジュールレジストリ
                     │        └─→ Entitlement Store（ツール設定/統計情報）
                     └─→ モジュール
                            ├─→ Token Vault（トークン取得）
                            └─→ 外部サービスAPI（HTTPアクセス）
```

### OAuth連携フロー（ユーザーコンソール経由）

```
ユーザーコンソール → 外部Authサーバー（認可フロー）
        │
        ▼
  Token Vault（トークン登録）
```

### 課金フロー

```
ユーザーコンソール → 決済代行サービス（決済）
                           │
                           ├─→ Entitlement Store（プラン情報参照）
                           └─→ Entitlement Store（課金情報登録 / Webhook）
```

### トークンリフレッシュフロー

```
Token Vault → 外部WEBサービス（外部Authサーバー）
                 │
                 ▼ リフレッシュトークン交換
           Token Vault（新トークン保存）
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-inf.md](../../mcpist/docs/specification/spc-inf.md) | インフラストラクチャ仕様書 |
| [dtl-core.md](../DAY5/dtl-core.md) | コア機能定義（COR-001〜009） |
| [dtl-sys-cor.md](../DAY5/dtl-sys-cor.md) | システム仕様サブコア定義 |
| [spec-ifc.md](../DAY6/spec-ifc.md) | インターフェース仕様（IFC-001〜043） |
| [dsn-module-registry.md](../DAY7/dsn-module-registry.md) | Module Registry設計 |
| [dsn-permission-system.md](../DAY7/dsn-permission-system.md) | 権限システム設計 |
| [dsn-subscription.md](../DAY7/dsn-subscription.md) | サブスクリプション設計 |
| [adr-usage-control-architecture.md](../DAY7/adr-usage-control-architecture.md) | 使用量制御ADR |
