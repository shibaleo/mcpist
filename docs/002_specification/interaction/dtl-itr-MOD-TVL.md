# MOD - TVL インタラクション詳細（dtl-itr-MOD-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.0 |
| Note | Modules - Token Vault Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Modules (MOD) |
| 連携先 | Token Vault (TVL) |
| 内容 | トークン取得・保存 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | 外部API呼び出し時 |
| 操作 | サービス別トークン取得、リフレッシュ（MOD側で実行）、トークン更新 |

### 責務分担

| コンポーネント | 責務 |
|---------------|------|
| **TVL** | トークンの保存・取得のみ（ストレージ） |
| **MOD** | トークンの有効期限確認、リフレッシュ実行、認証ヘッダー生成 |

**設計理由**: OAuth Auth Server と Resource Server は同じサービスのAPIであり、バージョン情報やエンドポイント情報を共有する。そのため、認証関連のスクリプトはMOD（モジュール）側に配置する。

### フロー

1. MODが外部APIリクエストを処理
2. TVLにuser_id + serviceで問い合わせ
3. TVLがcredentials（auth_type含む）を返却
4. MODがauth_typeに応じて認証ヘッダーを生成
5. OAuth 2.0の場合、MODが期限切れを検知しリフレッシュ→TVLに更新保存

### 取得リクエスト

```json
{
  "user_id": "user-123",
  "service": "notion"
}
```

### 取得レスポンス

TVLはcredentialsをそのまま返す。リフレッシュはMODの責務。

*OAuth 2.0形式:*
```json
{
  "user_id": "user-123",
  "service": "notion",
  "auth_type": "oauth2",
  "credentials": {
    "access_token": "ntn_xxx",
    "refresh_token": "ntn_refresh_xxx",
    "expires_at": 1706140800
  }
}
```

*API Key形式:*
```json
{
  "user_id": "user-123",
  "service": "github",
  "auth_type": "api_key",
  "credentials": {
    "access_token": "ghp_xxx"
  }
}
```

*OAuth 1.0a形式:*
```json
{
  "user_id": "user-123",
  "service": "zaim",
  "auth_type": "oauth1",
  "credentials": {
    "consumer_key": "xxx",
    "consumer_secret": "xxx",
    "access_token": "xxx",
    "access_token_secret": "xxx"
  }
}
```

*Basic認証形式（Jira/Confluence等）:*
```json
{
  "user_id": "user-123",
  "service": "jira",
  "auth_type": "basic",
  "credentials": {
    "username": "user@example.com",
    "password": "ATATT3xFfGF0..."
  },
  "metadata": {
    "domain": "mycompany.atlassian.net"
  }
}
```

---

## auth_type一覧

| auth_type | 説明 | credentials | Authorizationヘッダー |
|-----------|------|-------------|----------------------|
| `oauth2` | OAuth 2.0（リフレッシュ対応） | `access_token`, `refresh_token`, `expires_at` | `Authorization: Bearer {token}` |
| `api_key` | API Key（リフレッシュなし） | `access_token` | `Authorization: Bearer {token}` |
| `oauth1` | OAuth 1.0a署名 | `consumer_key`, `consumer_secret`, `access_token`, `access_token_secret` | `Authorization: OAuth ...` |
| `basic` | Basic認証 | `username`, `password` | `Authorization: Basic {base64(user:pass)}` |
| `custom_header` | カスタムヘッダー | `token`, `header_name` | `{header_name}: {token}` |

### サービス別auth_type例

| サービス | auth_type | 備考 |
|---------|-----------|------|
| Notion | `oauth2` | OAuth 2.0 Public Integration |
| GitHub | `oauth2` / `api_key` | OAuth App または Personal Access Token |
| Jira | `basic` | email + API Token、`metadata.domain`必須 |
| Confluence | `basic` | email + API Token、`metadata.domain`必須 |
| Supabase | `api_key` | Management API Token |
| Zaim | `oauth1` | OAuth 1.0a |

---

## OAuth 2.0 リフレッシュフロー（MOD側で実行）

```
┌─────────────────────┐
│ MOD (client.go)     │
│                     │
│ 1. TVLからtoken取得  │
│ 2. expires_at確認   │
│ 3. 期限切れなら:     │
│    - Auth Serverに  │
│      リフレッシュ    │
│    - TVLに更新保存   │
│ 4. APIリクエスト実行 │
└─────────────────────┘
```

### リフレッシュ後のトークン更新リクエスト

```json
{
  "user_id": "user-123",
  "service": "notion",
  "credentials": {
    "access_token": "new_access_token",
    "refresh_token": "new_refresh_token",
    "expires_at": 1706227200
  }
}
```

---

## トークン未設定時

```json
{
  "error": "no token configured for user: user-123, service: notion"
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-MOD.md](./itr-MOD.md) | Modules 詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
