# User Console インタラクション仕様書（itr-con）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | User Console Interaction Specification |

---

## 概要

User Console（CON）は、ユーザーが自分の設定を管理するWebアプリケーション。

主な機能：
- ユーザー認証（ログイン/ログアウト）
- OAuth同意画面の提供（MCP Client認可時）
- 外部サービス連携（OAuth認可フロー）
- 権限設定（モジュール有効/無効）
- 課金管理

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| Payment Service Provider | CON → PSP | 決済リクエスト |
| Token Vault | CON → TVL | トークン登録 |
| Data Store | CON → DST | ツール設定登録 |
| External Auth Server | CON → EAS | 認可フロー |
| Identity Provider | CON → IDP | ソーシャルログイン |

---

## 連携詳細

### CON → PSP（決済リクエスト）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 用途 | プラン購入、課金管理 |

**フロー:**
```mermaid
sequenceDiagram
    participant User as ユーザー
    participant CON as User Console
    participant PSP as Payment Service Provider
    participant DST as Data Store

    User->>CON: プラン購入ボタンクリック
    CON->>PSP: Checkout Session作成
    PSP-->>CON: checkout_url
    CON->>User: PSP決済ページへリダイレクト
    User->>PSP: 決済情報入力・完了
    PSP-->>DST: Webhook（checkout.session.completed）
    DST->>DST: プラン情報更新
    PSP-->>CON: success_urlへリダイレクト
    CON->>User: 購入完了表示
```

---

### CON → TVL（トークン登録）

| 項目 | 内容 |
|------|------|
| 用途 | 外部サービスのOAuthトークン保存 |
| タイミング | 外部サービス連携完了時 |

**登録する情報:**
- user_id
- service_id（notion, google_calendar, microsoft_todo）
- access_token
- refresh_token
- token_type
- expires_at
- scope

---

### CON → DST（ツール設定登録）

| 項目 | 内容 |
|------|------|
| 用途 | ユーザー設定の管理 |
| 操作 | モジュール有効/無効、課金状態確認 |

**管理対象の設定：**
- 有効なモジュール一覧
- 各モジュールの個別設定
- 課金プラン
- アカウント状態

---

### CON → EAS（認可フロー）

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 |
| 用途 | 外部サービスへのアクセス権限取得 |

**フロー:**
```mermaid
sequenceDiagram
    participant User as ユーザー
    participant CON as User Console
    participant EAS as External Auth Server
    participant TVL as Token Vault

    User->>CON: 連携開始（/connect/:service）
    CON->>EAS: 認可リクエスト（OAuth）
    EAS->>User: 認証/同意画面
    User->>EAS: 同意
    EAS-->>CON: 認可コード（redirect）
    CON->>EAS: トークン交換
    EAS-->>CON: access_token, refresh_token
    CON->>TVL: トークン保存
    TVL-->>CON: 保存完了
    CON->>User: 連携完了表示
```

**対応サービス：**
- Notion
- Google Calendar
- Microsoft To Do

---

### CON → IDP（ソーシャルログイン）

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 / OpenID Connect |
| 用途 | ユーザーログイン |

**フロー:**
```mermaid
sequenceDiagram
    participant User as ユーザー
    participant CON as User Console
    participant SSM as Session Manager
    participant IDP as Identity Provider

    User->>CON: ソーシャルログインボタンクリック
    CON->>SSM: 認証開始リクエスト
    SSM->>IDP: OAuth認可リクエスト
    IDP->>User: ログイン画面
    User->>IDP: 認証情報入力
    IDP-->>SSM: ID Token
    SSM->>SSM: ユーザー情報作成/取得
    SSM-->>CON: セッション確立
    CON->>User: ダッシュボード表示
```

**対応プロバイダ:**
- Google
- GitHub

---

## CONが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | 別アプリケーション |
| MCP Client (API KEY) (CLK) | 別アプリケーション |
| API Gateway (GWY) | MCP通信専用 |
| Auth Server (AUS) | CLO向け認証（CONはSSM/IDP経由） |
| Session Manager (SSM) | CONのバックエンド（直接連携ではない） |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MCP Server内部 |
| Modules (MOD) | MCP Server内部 |
| External Service API (EXT) | EAS経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-psp.md](./itr-psp.md) | Payment Service Provider詳細仕様 |
| [itr-tvl.md](./itr-tvl.md) | Token Vault詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store詳細仕様 |
| [itr-eas.md](./itr-eas.md) | External Auth Server詳細仕様 |
| [itr-idp.md](./itr-idp.md) | Identity Provider詳細仕様 |
