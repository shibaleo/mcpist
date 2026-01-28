# CLO - GWY インタラクション詳細（dtl-itr-CLO-GWY）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-001 |
| Note | MCP Client (OAuth2.0) - API Gateway Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | MCP Client (OAuth2.0) (CLO) |
| 連携先 | API Gateway (GWY) |
| 内容 | MCP通信 |
| プロトコル | MCP over SSE (HTTPS) |

---

## 詳細

| 項目      | 内容                                                                                                                     |
| ------- | ---------------------------------------------------------------------------------------------------------------------- |
| プロトコル   | [MCP Protocol 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)（JSON-RPC 2.0 over Streamable HTTP） |
| 認証      | Bearer Token（JWT）                                                                                                      |
| エンドポイント | `https://mcp.mcpist.app/mcp`                                                                                           |

### リクエストヘッダー（MCP仕様準拠）

[Transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports):
```
Accept: application/json, text/event-stream
MCP-Protocol-Version: 2025-11-25
MCP-Session-Id: {session_id}
```

[Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization):
```
Authorization: Bearer {access_token}
```

HTTP標準:
```
Content-Type: application/json
```

CLOはMCPプリミティブ（Tools, Resources, Prompts）をJSON-RPCリクエストとして送信する。詳細は [spc-itf.md](../spc-itf.md) を参照。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-CLO.md](./itr-CLO.md) | MCP Client (OAuth2.0) 詳細仕様 |
| [itr-GWY.md](./itr-GWY.md) | API Gateway 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
