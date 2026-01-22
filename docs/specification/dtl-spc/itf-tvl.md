# Token Vault API仕様書（itf-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.1 (DAY9) |
| Note | Token Vault HTTP API Specification |

---

## 概要

Token VaultのHTTP API仕様。MCP ServerがユーザーのOAuthトークン・長期トークンを取得するためのエンドポイント。

**実装:** Supabase Edge Functions + PostgreSQL + Vault（暗号化保存）

---

## 認証

Supabase Edge Functions へのアクセスには publishable key が必要。

```
Authorization: Bearer <SUPABASE_PUBLISHABLE_KEY>
```

---

## エンドポイント

### POST /token-vault

ユーザーとサービスに対応するトークンを取得する。

**リクエストヘッダー:**

| ヘッダー | 値 | 必須 |
|---------|-----|------|
| Content-Type | application/json | Yes |
| Authorization | Bearer \<anon_key\> | Yes |

**リクエストボディ:**

```json
{
  "user_id": "user-123",
  "service": "notion"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| user_id | string | Yes | ユーザー識別子 |
| service | string | Yes | サービス名（notion, google_calendar, microsoft_todo） |

**レスポンス（200 OK）:**

```json
{
  "user_id": "user-123",
  "service": "notion",
  "long_term_token": "ntn_xxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "oauth_token": null
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| user_id | string | リクエストしたユーザーID |
| service | string | リクエストしたサービス名 |
| long_term_token | string \| null | 長期トークン（Internal Integration Token, API Token等） |
| oauth_token | string \| null | OAuthアクセストークン |

- `long_term_token` と `oauth_token` の少なくとも1つは非null
- 両方 null の場合は 404 を返す（トークン未設定）

**エラーレスポンス（401 Unauthorized）:**

```json
{
  "error": "unauthorized"
}
```

**エラーレスポンス（404 Not Found）:**

```json
{
  "error": "no token configured for user: user-123, service: notion"
}
```

**エラーレスポンス（400 Bad Request）:**

```json
{
  "error": "invalid service: invalid"
}
```

---

## cURL例

```bash
curl -X POST http://localhost:8089/token-vault \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_anon_key_for_testing" \
  -d '{"user_id": "user-123", "service": "notion"}'
```

---

## トークン優先度

クライアント（モジュール）は以下の優先度でトークンを使用：

1. `oauth_token` が存在すれば使用（ユーザー固有の権限）
2. `oauth_token` がなければ `long_term_token` を使用（共有/固定権限）

---

## サービス別トークン形式

| Service | long_term_token | oauth_token |
|---------|-----------------|-------------|
| notion | Internal Integration Token (`ntn_xxx`) | OAuth Access Token |
| google_calendar | - | OAuth Access Token |
| microsoft_todo | - | OAuth Access Token |

---

## 開発環境

### 環境変数（.env.development）

```
VAULT_URL=http://localhost:8089
SUPABASE_PUBLISHABLE_KEY=dev_publishable_key_for_testing
```

### モック起動方法

```bash
# Prismでモックサーバー起動（リポジトリルートから）
npx @stoplight/prism-cli mock apps/server/api/openapi/token-vault.yaml --port 8089
```

**OpenAPI仕様ファイル:** `apps/server/api/openapi/token-vault.yaml`

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [ifr-tvl.md](./ifr-tvl.md) | Token Vault インフラ仕様 |
| [itr-srv.md](./itr-srv.md) | MCP Server詳細仕様 |
| [spc-inf.md](../spc-inf.md) | インフラストラクチャ仕様 |
| [tst-tvl.md](./tst-tvl.md) | テスト手順書 |
