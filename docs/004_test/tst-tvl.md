# Token Vault テスト手順書（tst-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.1 (DAY9) |
| Note | Token Vault Integration Test Procedure |

---

## 概要

Token Vault API（Prism mock）と MCP Server の統合テスト手順。

---

## 前提条件

- Node.js（npx使用可能）
- Go 1.21+
- Prism CLI（`@stoplight/prism-cli`）

---

## テスト構成

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   curl/Client   │────▶│   MCP Server    │────▶│  Prism Mock     │
│                 │     │   (port 8088)   │     │  (port 8089)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                        ┌─────────────────┐
                        │  Vault Client   │
                        │  (POST request) │
                        └─────────────────┘
```

---

## 環境変数

`.env.development` に以下が設定されていること:

```
VAULT_URL=http://localhost:8089
SUPABASE_PUBLISHABLE_KEY=dev_anon_key_for_testing
```

---

## 手順

### Step 1: Prism Mock Server 起動

**ターミナル1:**

```bash
# リポジトリルートで実行
cd C:\Users\shiba\HOBBY\mcpist
npx @stoplight/prism-cli mock apps/server/api/openapi/token-vault.yaml --port 8089
```

**期待出力:**
```
[CLI] ►  start     Prism is listening on http://127.0.0.1:8089
```

### Step 2: Prism Mock 単体テスト

**ターミナル2:**

```bash
# ヘルスチェック
curl -s http://127.0.0.1:8089/health
# 期待: ok

# トークン取得（正常系）
curl -s -X POST http://127.0.0.1:8089/token-vault \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_anon_key_for_testing" \
  -d '{"user_id": "user-123", "service": "notion"}'
# 期待: {"user_id":"user-123","service":"notion","long_term_token":"ntn_xxx...","oauth_token":null}

# トークン取得（別サービス）
curl -s -X POST http://127.0.0.1:8089/token-vault \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_anon_key_for_testing" \
  -d '{"user_id": "user-123", "service": "github"}'
# 期待: TokenResponse JSON
```

### Step 3: MCP Server 起動

**ターミナル2:**

```bash
cd C:\Users\shiba\HOBBY\mcpist\apps\server

# 環境変数を設定して起動
set VAULT_URL=http://127.0.0.1:8089
set SUPABASE_PUBLISHABLE_KEY=dev_anon_key_for_testing
go run ./cmd/server/main.go
```

**期待出力:**
```
2026/01/17 XX:XX:XX Registered modules: [notion]
2026/01/17 XX:XX:XX Starting MCP server on port 8088
```

### Step 4: MCP Server 単体テスト

**ターミナル3:**

```bash
# ヘルスチェック
curl -s http://localhost:8088/health
# 期待: ok

# tools/list
curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
# 期待: {"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"get_module_schema",...},{"name":"call",...},{"name":"batch",...}]}}

# get_module_schema (notion)
curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_module_schema","arguments":{"module":"notion"}}}'
# 期待: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"module\":\"notion\",\"tools\":[...]}"}]}}
```

### Step 5: 統合テスト（Token Vault経由）

```bash
# Notion search（Token Vault経由でトークン取得）
curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"call","arguments":{"module":"notion","tool":"search","params":{"query":"test"}}}}'
```

**期待動作:**
1. MCP Server が `call` ツールを受信
2. Vault Client が `POST http://127.0.0.1:8089/token-vault` を呼び出し
   - Body: `{"user_id":"dev","service":"notion"}`
   - Header: `Authorization: Bearer dev_anon_key_for_testing`
3. Prism が `{"long_term_token":"ntn_xxx...","oauth_token":null}` を返す
4. Notion モジュールが取得したトークンで Notion API を呼び出し

**注意:** Prism が返すトークンはダミー値のため、Notion API への実際のリクエストは 401 エラーになる。Token Vault 統合の動作確認が目的。

---

## テスト結果チェックリスト

| テスト | エンドポイント | メソッド | 期待結果 | 結果 |
|--------|---------------|----------|----------|------|
| Prism health | `/health` | GET | `ok` | |
| Prism tokens | `/token-vault` | POST | TokenResponse JSON | |
| MCP health | `/health` | GET | `ok` | |
| MCP tools/list | `/mcp` | POST | tools配列 | |
| MCP get_module_schema | `/mcp` | POST | notion schema | |
| MCP call (統合) | `/mcp` | POST | Vault経由でトークン取得 | |

---

## トラブルシューティング

### ポート競合

```bash
# Windows: ポート使用プロセス確認
netstat -ano | findstr :8088
netstat -ano | findstr :8089

# プロセス終了
taskkill /F /PID <PID>

# Goプロセス一括終了
taskkill /F /IM go.exe
taskkill /F /IM main.exe
```

### Prism が起動しない

```bash
# npm キャッシュクリア
npm cache clean --force
npx @stoplight/prism-cli --version
```

### MCP Server がトークン取得に失敗

1. `VAULT_URL` 環境変数が正しく設定されているか確認
2. `SUPABASE_PUBLISHABLE_KEY` 環境変数が設定されているか確認
3. Prism が起動しているか確認
4. Prism のログでリクエストが来ているか確認

---

## クリーンアップ

```bash
# すべてのGoプロセスを終了
taskkill /F /IM go.exe
taskkill /F /IM main.exe

# Prism は Ctrl+C で終了
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itf-tvl.md](../002_specification/dtl-spc/itf-tvl.md) | Token Vault API仕様 |
| token-vault.yaml (`apps/server/api/openapi/`) | OpenAPI仕様 |
| client.go (`apps/server/internal/vault/`) | Vault クライアント実装 |
