# Token Vault API インターフェース仕様書（itf-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Token Vault API Interface Specification |

---

## 概要

Token Vault（TVL）が提供するAPIインターフェースを定義する。

---

## データベーススキーマ

Token Vault は Supabase Vault を使用してトークンを暗号化保存する。

### テーブル: `mcpist.oauth_tokens`

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | プライマリキー |
| user_id | UUID | ユーザーID (mcpist.users.id) |
| service | TEXT | サービス名 (notion, github等) |
| access_token_secret_id | UUID | vault.secrets への参照 |
| refresh_token_secret_id | UUID | vault.secrets への参照 (nullable) |
| token_type | TEXT | トークンタイプ (default: "Bearer") |
| scope | TEXT | スコープ (nullable) |
| expires_at | TIMESTAMPTZ | 有効期限 (nullable) |
| created_at | TIMESTAMPTZ | 作成日時 |
| updated_at | TIMESTAMPTZ | 更新日時 |

**ユニーク制約:** `(user_id, service)` - 1ユーザー1サービスにつき1トークン

---

## RPC Functions

### upsert_oauth_token

OAuthトークンを登録・更新する。

**シグネチャ:**
```sql
CREATE FUNCTION public.upsert_oauth_token(
    p_service TEXT,
    p_access_token TEXT,
    p_refresh_token TEXT DEFAULT NULL,
    p_token_type TEXT DEFAULT 'Bearer',
    p_scope TEXT DEFAULT NULL,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
) RETURNS UUID
```

**呼び出し元:** CON（User Console）

**権限:** `authenticated` ロールのみ

**動作:**
1. `auth.uid()` で認証ユーザーを取得
2. 既存トークンがあれば vault.secrets から古いシークレットを削除
3. `vault.create_secret()` で新しいトークンを暗号化保存
4. `oauth_tokens` テーブルを UPSERT
5. トークンIDを返却

---

### get_my_oauth_connections

ユーザーの接続済みサービス一覧を取得。トークン値は含まない。

**シグネチャ:**
```sql
CREATE FUNCTION public.get_my_oauth_connections()
RETURNS TABLE (
    id UUID,
    service TEXT,
    token_type TEXT,
    scope TEXT,
    expires_at TIMESTAMPTZ,
    is_expired BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
```

**呼び出し元:** CON（User Console）

**権限:** `authenticated` ロールのみ

---

### get_service_token

サービストークンを復号して取得。

**シグネチャ:**
```sql
CREATE FUNCTION public.get_service_token(
    p_user_id UUID,
    p_service TEXT
) RETURNS TABLE (
    oauth_token TEXT,
    long_term_token TEXT
)
```

**呼び出し元:** SRV（MCP Server）経由の Console API

**権限:** `service_role` のみ（セキュリティ上、一般ユーザーは他ユーザーのトークンを取得不可）

**動作:**
1. `oauth_tokens` から該当レコードを取得
2. `vault.decrypted_secrets` から復号済みトークンを取得
3. トークンを返却

---

### delete_oauth_token

トークンを削除。

**シグネチャ:**
```sql
CREATE FUNCTION public.delete_oauth_token(p_service TEXT)
RETURNS BOOLEAN
```

**呼び出し元:** CON（User Console）

**権限:** `authenticated` ロールのみ

---

## HTTP API

MCP Server (Go) が Token Vault にアクセスするための内部 API。

### POST /api/token-vault

**認証:** `Authorization: Bearer <INTERNAL_SERVICE_KEY>`

**Request Body:**
```json
{
  "user_id": "uuid-string",
  "service": "notion"
}
```

**Response (200 OK):**
```json
{
  "oauth_token": "secret_xxx...",
  "long_term_token": "secret_xxx..."
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "Unauthorized"
}
```

**Response (404 Not Found):**
```json
{
  "error": "Token not found"
}
```

**Response (500 Internal Server Error):**
```json
{
  "error": "エラーメッセージ"
}
```

---

## 環境変数

| 変数名 | コンポーネント | 説明 |
|--------|--------------|------|
| INTERNAL_SERVICE_KEY | Console, Server | サーバー間認証用の共有シークレット |
| VAULT_URL | Server | Console API のベースURL (e.g., `http://localhost:3000/api`) |
| SUPABASE_SERVICE_ROLE_KEY | Console | Supabase サービスロールキー |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [ifr-tvl.md](./ifr-tvl.md) | Token Vault インタラクション仕様 |
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-tbl.md](../spc-tbl.md) | テーブル仕様書 |
