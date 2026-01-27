# Token Vault API仕様書（itf-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `active` |
| Version | v2.0 (DAY16) |
| Note | Token Vault RPC Specification |

---

## 概要

Token VaultのRPC API仕様。MCP Server（Go）がユーザーのOAuthトークン・長期トークンを取得・更新するためのSupabase RPC関数。

**実装:** Supabase PostgreSQL RPC + Vault（暗号化保存）

---

## アーキテクチャ

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Go Server  │────▶│  Supabase RPC    │────▶│  vault.secrets  │
│  (Module)   │     │  (PostgreSQL)    │     │  (暗号化保存)    │
└─────────────┘     └──────────────────┘     └─────────────────┘
```

Go ServerがSupabase PostgREST経由でRPC関数を呼び出し、Vaultからトークンを取得する。

---

## RPC関数

### get_module_token

ユーザーとモジュールに対応するトークンを取得する。

**シグネチャ:**
```sql
get_module_token(p_user_id UUID, p_module TEXT)
RETURNS JSON
```

**パラメータ:**

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | Yes | ユーザーID |
| p_module | TEXT | Yes | モジュール名（notion, google_calendar, microsoft_todo） |

**レスポンス:**

```json
{
  "auth_type": "oauth2",
  "access_token": "ya29.xxx",
  "refresh_token": "1//xxx",
  "expires_at": 1706180400
}
```

または（長期トークン）:

```json
{
  "auth_type": "long_term",
  "access_token": "ntn_xxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

**エラー:**
- トークン未設定の場合、空のJSONを返す

**権限:** `anon`, `authenticated`

---

### update_module_token

OAuthトークンリフレッシュ後に新しいトークンを保存する。

**シグネチャ:**
```sql
update_module_token(p_user_id UUID, p_module TEXT, p_credentials JSON)
RETURNS VOID
```

**パラメータ:**

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | Yes | ユーザーID |
| p_module | TEXT | Yes | モジュール名 |
| p_credentials | JSON | Yes | 新しいトークン情報 |

**p_credentials形式:**

```json
{
  "auth_type": "oauth2",
  "access_token": "ya29.new_xxx",
  "refresh_token": "1//new_xxx",
  "expires_at": 1706266800
}
```

**権限:** `service_role`（Go Serverのみ）

---

### get_oauth_app_credentials

OAuth App（client_id/client_secret）の認証情報を取得する。トークンリフレッシュ時に使用。

**シグネチャ:**
```sql
get_oauth_app_credentials(p_provider TEXT)
RETURNS JSON
```

**パラメータ:**

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_provider | TEXT | Yes | プロバイダ名（google, microsoft） |

**レスポンス:**

```json
{
  "client_id": "xxx.apps.googleusercontent.com",
  "client_secret": "GOCSPX-xxx",
  "redirect_uri": "https://dev.mcpist.app/api/oauth/google/callback"
}
```

**権限:** `service_role`（Go Serverのみ）

---

## Go Server実装例

```go
// トークン取得
func (s *Store) GetModuleToken(ctx context.Context, userID, module string) (*Credentials, error) {
    var result json.RawMessage
    err := s.client.Rpc("get_module_token", "", map[string]interface{}{
        "p_user_id": userID,
        "p_module":  module,
    }).Execute(&result)
    // ...
}

// トークン更新
func (s *Store) UpdateModuleToken(ctx context.Context, userID, module string, creds *Credentials) error {
    return s.serviceClient.Rpc("update_module_token", "", map[string]interface{}{
        "p_user_id":      userID,
        "p_module":       module,
        "p_credentials":  creds,
    }).Execute(nil)
}
```

---

## トークン優先度

モジュールは以下の優先度でトークンを使用：

1. `auth_type: oauth2` のトークンが存在すれば使用（ユーザー固有の権限）
2. `auth_type: long_term` のトークンを使用（共有/固定権限）

---

## サービス別トークン形式

| Service | auth_type | 説明 |
|---------|-----------|------|
| notion | long_term | Internal Integration Token (`ntn_xxx`) |
| notion | oauth2 | OAuth Access Token（オプション） |
| google_calendar | oauth2 | OAuth Access Token |
| microsoft_todo | oauth2 | OAuth Access Token |
| github | long_term | Personal Access Token |
| jira | long_term | API Token |
| confluence | long_term | API Token |

---

## データベース構造

### service_tokens テーブル

```sql
CREATE TABLE mcpist.service_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    auth_type TEXT NOT NULL DEFAULT 'long_term',
    credentials_secret_id UUID, -- vault.secrets への参照
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, service)
);
```

### vault.secrets（Supabase Vault）

トークンは暗号化されて保存される。

**命名規則:**
```
{user_id}:{service}
```

**保存形式:**
```json
{
  "auth_type": "oauth2",
  "access_token": "xxx",
  "refresh_token": "yyy",
  "expires_at": 1706180400
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-tvl.md](../interaction/itr-tvl.md) | Token Vault インタラクション仕様 |
| [itr-srv.md](../interaction/itr-srv.md) | MCP Server詳細仕様 |
| [spc-inf.md](../spc-inf.md) | インフラストラクチャ仕様 |
| [tst-tvl.md](../../004_test/tst-tvl.md) | テスト手順書 |
