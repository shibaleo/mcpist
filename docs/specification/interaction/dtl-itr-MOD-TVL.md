# MOD - TVL インタラクション詳細（dtl-itr-MOD-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-010 |
| Note | Modules - Token Vault Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Modules (MOD) |
| 連携先 | Token Vault (TVL) |
| 内容 | トークン取得 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | 外部API呼び出し時 |
| 操作 | サービス別トークン取得、有効期限確認、リフレッシュ |

### TVLの責務

- トークン選択（OAuth 2.0優先、なければ長期トークン）
- トークンリフレッシュ（OAuth 2.0の場合）
- 有効なトークンを1つ返却

### フロー

1. MODが外部APIリクエストを処理
2. TVLにuser_id + service_idで問い合わせ
3. OAuth 2.0トークンが存在すれば優先使用（期限切れならリフレッシュ）
4. OAuth 2.0トークンがなければ長期トークンを使用
5. どちらもなければエラー返却

### 取得リクエスト

```json
{
  "user_id": "user-123",
  "service": "notion"
}
```

### 取得レスポンス

TVLは有効なトークンを1つ返す。トークン選択・リフレッシュはTVLの責務。

*Bearer Token形式（OAuth 2.0、長期トークン）:*
```json
{
  "user_id": "user-123",
  "service": "notion",
  "auth_type": "bearer",
  "credentials": {
    "access_token": "ntn_xxx"
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

### auth_type一覧（APIリクエスト時の認証方式）

| auth_type | 説明 | credentials | Authorizationヘッダー |
|-----------|------|-------------|----------------------|
| `bearer` | Bearer Token | `access_token` | `Authorization: Bearer {token}` |
| `oauth1` | OAuth 1.0a署名 | `consumer_key`, `consumer_secret`, `access_token`, `access_token_secret` | `Authorization: OAuth ...` |
| `basic` | Basic認証 | `username`, `password` | `Authorization: Basic {base64}` |
| `custom_header` | カスタムヘッダー | `token`, `header_name` | `{header_name}: {token}` |

**注:** OAuth 2.0で取得したトークンも長期トークン（APIキー）も、APIリクエスト時は `bearer` として扱う。

### トークン選択ロジック

1. OAuth 2.0トークンが存在すれば優先使用（期限切れならリフレッシュ）
2. OAuth 2.0トークンがなければ長期トークンを使用
3. どちらもなければエラー

### トークン未設定時

```json
{
  "error": "no token configured for user: user-123, service: notion"
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-mod.md](./itr-mod.md) | Modules 詳細仕様 |
| [itr-tvl.md](./itr-tvl.md) | Token Vault 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
