# CLK - GWY インタラクション詳細（dtl-itr-CLK-GWY）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-003 |
| Note | MCP Client (API KEY) - API Gateway Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | MCP Client (API KEY) (CLK) |
| 連携先 | API Gateway (GWY) |
| 内容 | MCP通信 |
| プロトコル | MCP over SSE (HTTPS) |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | [MCP Protocol 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)（JSON-RPC 2.0 over Streamable HTTP） |
| 認証方式 | API KEY（Bearer Token形式） |
| エンドポイント | `https://mcp.mcpist.app/mcp` |

### API KEY形式

```
mcpist_{random_string_32chars}
```

### リクエストヘッダー（MCP仕様準拠）

[Transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports):
```
Accept: application/json, text/event-stream
MCP-Protocol-Version: 2025-11-25
MCP-Session-Id: {session_id}
```

[Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization):
```
Authorization: Bearer mcpist_xxx
```

HTTP標準:
```
Content-Type: application/json
```

CLKはMCPプリミティブ（Tools, Resources, Prompts）をJSON-RPCリクエストとして送信する。詳細は [spc-itf.md](../spc-itf.md) を参照。

### 認証フロー

```mermaid
sequenceDiagram
    participant CLK as MCP Client (API KEY)
    participant GWY as API Gateway
    participant TVL as Token Vault
    participant AMW as Auth Middleware

    CLK->>GWY: MCPリクエスト（API KEY）
    Note over GWY: SHA256(api_key)を計算
    GWY->>TVL: ハッシュで検証リクエスト
    TVL-->>GWY: ユーザーID返却
    GWY->>AMW: リクエスト転送（X-User-Id付与）
    AMW-->>GWY: レスポンス
    GWY-->>CLK: 正常レスポンス
```

### API KEY検証

1. CLKがAPI KEYでリクエスト
2. GWYがAPI KEYのSHA256ハッシュを計算（平文は即破棄）
3. GWYがTVLにハッシュで検証を依頼
4. TVLがハッシュからユーザーIDを特定
5. 検証成功時：GWYがX-User-Idヘッダーを付与してAMWへ転送
6. 検証失敗時：401 Unauthorized返却

### API KEY取得方法

API KEYはUser Console（CON）で発行する。ユーザーは発行されたAPI KEYをCLKの設定（環境変数やconfig等）に設定し、リクエスト時にAuthorizationヘッダーに付与する。

**注意事項:**
- API KEYは発行時に一度だけ表示される（再表示不可）
- 紛失した場合は再発行が必要
- 発行フローの詳細は [itr-con.md](./itr-con.md) を参照

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-clk.md](./itr-clk.md) | MCP Client (API KEY) 詳細仕様 |
| [itr-gwy.md](./itr-gwy.md) | API Gateway 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
