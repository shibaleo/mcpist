# MOD - TVL インタラクション詳細（dtl-itr-MOD-TVL）

## ドキュメント管理情報

| 項目      | 値                                        |
| ------- | ---------------------------------------- |
| Status  | `draft`                                  |
| Version | v2.2                                     |
| Note    | Modules - Token Vault Interaction Detail |

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
| 操作 | モジュール別トークン取得、リフレッシュ（MOD側で実行）、トークン更新 |

### 責務分担

| コンポーネント | 責務 |
|---------------|------|
| **TVL** | トークンの保存・取得のみ（ストレージ） |
| **MOD** | トークンの有効期限確認、リフレッシュ実行、認証ヘッダー生成 |

**設計理由**: OAuth Auth Server と Resource Server は同じサービスのAPIであり、バージョン情報やエンドポイント情報を共有する。そのため、認証関連のスクリプトはMOD（モジュール）側に配置する。

### フロー

1. MODが外部APIリクエストを処理
2. TVLにuser_id + moduleで問い合わせ
3. TVLがauth_type + credentialsを返却
4. MODがauth_typeに応じて認証ヘッダーを生成
5. OAuth 2.0の場合、MODが期限切れを検知しリフレッシュ→TVLに更新保存

### 取得リクエスト

```json
{
  "user_id": "user-123",
  "module": "notion"
}
```

### 取得レスポンス

TVLはauth_type + credentialsをそのまま返す。リフレッシュはMODの責務。

#### credentials 構造

| フィールド | 位置 | 説明 |
|-----------|------|------|
| `auth_type` | credentials 外 | API呼び出し方式の識別子（機密情報ではない） |
| `access_token` 等 | credentials 内 | 認証に必要なトークン・シークレット |
| `metadata` | credentials 内 | モジュール固有の追加情報（domain, workspace 等） |

**設計理由**:
- `auth_type` は機密情報ではないため、credentials 外に配置（復号化せずに参照可能）
- `auth_type` は「API呼び出し方式」を表す（認可方式ではない）
- `metadata` には機密情報を含む場合があるため、暗号化対象の credentials 内に格納

*OAuth 2.0形式:*
```json
{
  "user_id": "user-123",
  "module": "notion",
  "auth_type": "oauth2",
  "credentials": {
    "access_token": "ntn_xxx",
    "refresh_token": "ntn_refresh_xxx",
    "expires_at": 1706140800,
    "metadata": {
      "workspace_id": "xxx",
      "workspace_name": "My Workspace"
    }
  }
}
```

*API Key形式:*
```json
{
  "user_id": "user-123",
  "module": "github",
  "auth_type": "api_key",
  "credentials": {
    "access_token": "ghp_xxx"
  }
}
```

*API Key形式（クエリパラメータ方式 - Trello）:*

Trello は OAuth 1.0a で認可するが、API 呼び出しは API Key + Token のクエリパラメータ方式。

```json
{
  "user_id": "user-123",
  "module": "trello",
  "auth_type": "api_key",
  "credentials": {
    "access_token": "xxx",
    "api_key": "xxx"
  }
}
```

*Basic認証形式（Jira/Confluence等）:*
```json
{
  "user_id": "user-123",
  "module": "jira",
  "auth_type": "basic",
  "credentials": {
    "username": "user@example.com",
    "password": "ATATT3xFfGF0...",
    "metadata": {
      "domain": "mycompany.atlassian.net"
    }
  }
}
```

*API Key形式（metadata付き - Grafana）:*
```json
{
  "user_id": "user-123",
  "module": "grafana",
  "auth_type": "api_key",
  "credentials": {
    "access_token": "glsa_xxx",
    "metadata": {
      "base_url": "https://mycompany.grafana.net"
    }
  }
}
```

---

## auth_type一覧

`auth_type` は**API呼び出し方式**を表す（認可方式ではない）。

| auth_type | 説明 | credentials フィールド | API呼び出し方式 |
|-----------|------|----------------------|----------------|
| `oauth2` | Bearer トークン（リフレッシュ対応） | `access_token`, `refresh_token`, `expires_at` | `Authorization: Bearer {token}` |
| `api_key` | Bearer トークンまたはクエリパラメータ | `access_token`（+ `api_key` for Trello） | `Authorization: Bearer {token}` or `?key=&token=` |
| `basic` | Basic認証 | `username`, `password` | `Authorization: Basic {base64(user:pass)}` |

### モジュール別auth_type

| モジュール | auth_type | 認可方式 | metadata | 備考 |
|-----------|-----------|---------|----------|------|
| notion | `oauth2` | OAuth 2.0 | `workspace_id`, `workspace_name` | - |
| github | `oauth2` / `api_key` | OAuth 2.0 / PAT | - | OAuth App または Personal Access Token |
| google_calendar | `oauth2` | OAuth 2.0 | - | - |
| google_docs | `oauth2` | OAuth 2.0 | - | - |
| google_drive | `oauth2` | OAuth 2.0 | - | - |
| google_sheets | `oauth2` | OAuth 2.0 | - | - |
| google_tasks | `oauth2` | OAuth 2.0 | - | - |
| google_apps_script | `oauth2` | OAuth 2.0 | - | - |
| microsoft_todo | `oauth2` | OAuth 2.0 | - | Microsoft Graph API |
| todoist | `oauth2` | OAuth 2.0 | - | リフレッシュトークンなし |
| asana | `oauth2` | OAuth 2.0 | - | - |
| airtable | `oauth2` | OAuth 2.0 + PKCE | - | - |
| ticktick | `oauth2` | OAuth 2.0 | - | - |
| dropbox | `oauth2` | OAuth 2.0 | - | - |
| jira | `basic` | API Token | `domain`（必須） | email + API Token |
| confluence | `basic` | API Token | `domain`（必須） | email + API Token |
| trello | `api_key` | OAuth 1.0a | - | クエリパラメータ方式 |
| supabase | `api_key` | PAT | - | Management API Token |
| grafana | `api_key` | Service Account | `base_url`（必須） | - |
| postgresql | `basic` | 直接接続 | `host`, `port`, `database`（必須） | - |

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
  "module": "notion",
  "auth_type": "oauth2",
  "credentials": {
    "access_token": "new_access_token",
    "refresh_token": "new_refresh_token",
    "expires_at": 1706227200
  }
}
```

**注意**: リフレッシュ時は `metadata` を含めない（既存の metadata は保持される）。

---

## トークン未設定時

```json
{
  "error": "no token configured for user: user-123, module: notion"
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-MOD.md](./itr-MOD.md) | Modules 詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault 詳細仕様 |
