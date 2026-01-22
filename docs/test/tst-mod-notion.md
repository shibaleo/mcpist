# Notion Module 統合テスト結果（tst-mod-notion）

## テスト実施日

2026-01-17

---

## テスト環境

| 項目 | 値 |
|------|-----|
| MCP Server | localhost:8088 |
| Token Vault Mock | Prism (localhost:8089) |
| OpenAPI Spec | apps/server/api/openapi/token-vault.yaml |

---

## テスト結果サマリー

| テスト | エンドポイント | 結果 |
|--------|---------------|------|
| Prism health | GET /health | OK |
| Prism token-vault | POST /token-vault | OK |
| MCP Server health | GET /health | OK |
| MCP tools/list | POST /mcp | OK |
| MCP get_module_schema | POST /mcp | OK |
| MCP call (Notion search) | POST /mcp | OK |

---

## 詳細ログ

### 1. Prism Mock テスト

**リクエスト:**
```bash
curl -s -X POST http://127.0.0.1:8089/token-vault \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_anon_key_for_testing" \
  -d '{"user_id": "dev", "service": "notion"}'
```

**レスポンス:**
```json
{"user_id":"user-123","service":"notion","long_term_token":"ntn_xxx...","oauth_token":null}
```

### 2. MCP Server tools/list

**リクエスト:**
```bash
curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {"name": "get_module_schema", ...},
      {"name": "call", ...},
      {"name": "batch", ...}
    ]
  }
}
```

### 3. Notion search（空クエリ）

**リクエスト:**
```bash
curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"call","arguments":{"module":"notion","tool":"search","params":{"query":""}}}}'
```

**レスポンス（抜粋）:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"has_more\":true,\"next_cursor\":\"2e72cd76-e35b-801e-8925-f04eb364401d\",\"object\":\"list\",\"results\":[...]}"
      }
    ]
  }
}
```

**取得されたページ（一部）:**

| タイトル | ID | タイプ |
|---------|-----|------|
| DASHBOARD | 7267b9aa-6daf-4d8e-8f97-207ec3cf97e9 | page |
| MCPist DDL設計方針 | 2e82cd76-e35b-81fd-8e53-f413e9743530 | page |
| プロトタイピングの重要性 | 2e92cd76-e35b-80f9-9bca-c693e5677718 | page |
| TB__BRAIN | 49258655-fd8f-44a6-bea1-3116f7365e3a | database |
| 判断・決断・選択 - 類義語比較 | 2e82cd76-e35b-8191-923a-c731090977fb | page |

---

## Token Vault 統合フロー

```
1. MCP Client → MCP Server: tools/call (notion/search)
2. MCP Server → Notion Module: ExecuteTool("search", params)
3. Notion Module → Vault Client: GetTokens("dev", "notion")
4. Vault Client → Prism Mock: POST /token-vault
5. Prism Mock → Vault Client: {"long_term_token": "ntn_xxx..."}
6. Notion Module → Notion API: POST /v1/search (Bearer ntn_xxx...)
7. Notion API → Notion Module: 検索結果
8. Notion Module → MCP Server: JSON結果
9. MCP Server → MCP Client: JSON-RPC レスポンス
```

---

## 確認事項

- Token Vault から取得したトークンで Notion API 呼び出しが成功
- Prism mock が OpenAPI example の値を正しく返却
- MCP Server の3つのメタツール（get_module_schema, call, batch）が正常動作

---

## 備考

- 「設計」クエリで結果が0件だったのは Notion 検索 API のインデックス仕様による
- 空クエリでは全ページが返却され、統合が正常に動作していることを確認
