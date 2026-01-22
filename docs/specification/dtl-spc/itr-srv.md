# MCP Server 詳細仕様書（itr-srv）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | MCP Server Interaction Specification |

---

## 概要

MCP Server（SRV）は、MCP Clientからのリクエストを受け付けるサーバー。

### 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| MCP Client | SRV ← CLT | MCP Protocolリクエスト受付 |
| Auth Server | SRV → AUS | JWKS取得（JWT検証用公開鍵） |
| Entitlement Store | SRV → ENT | 権限情報の参照 |
| Token Vault | SRV → TVL | トークンの取得 |
| User Console | - | 直接やり取りなし |
| External API Server | SRV → EXT | 外部API呼び出し |

---

## 連携詳細

### CLT → SRV（MCP Clientからのリクエスト受付）

| 項目 | 内容 |
|------|------|
| プロトコル | [MCP Protocol 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)（JSON-RPC 2.0 over Streamable HTTP） |
| 認証 | Bearer Token（JWT） |
| エンドポイント | `https://mcp.mcpist.app/mcp` |

**リクエストヘッダー:**
```
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json, text/event-stream
MCP-Protocol-Version: 2025-11-25
MCP-Session-Id: {session_id}
```

**レスポンスヘッダー:**
```
Content-Type: application/json または text/event-stream
MCP-Session-Id: {session_id}
```

---

### SRV → AUS（JWKS取得）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| エンドポイント | `https://auth.mcpist.app/.well-known/jwks.json` |
| 用途 | JWT署名検証用公開鍵の取得 |
| キャッシュ | 必須（Cache-Controlヘッダーに従う） |

詳細は [itr-aus.md](./itr-aus.md) を参照。

---

### SRV → ENT（権限情報の参照）

| 項目 | 内容 |
|------|------|
| 用途 | ユーザーの権限情報（許可モジュール、ツール）の取得 |
| タイミング | リクエスト処理時 |

---

### SRV → TVL（トークンの取得）

| 項目 | 内容 |
|------|------|
| 用途 | 外部サービスのOAuthトークン取得 |
| タイミング | 外部API呼び出し時 |

---

### SRV → EXT（外部API呼び出し）

| 項目 | 内容 |
|------|------|
| 用途 | 外部サービスAPI呼び出し |
| 認証 | TVLから取得したBearer Token |

---

## Protected Resource Metadata

CLTが初回認可フローで参照するメタデータ。

**エンドポイント:** `https://mcp.mcpist.app/.well-known/oauth-protected-resource`

**レスポンス:**
```json
{
  "resource": "https://mcp.mcpist.app",
  "authorization_servers": ["https://auth.mcpist.app"],
  "scopes_supported": ["openid", "profile", "email"],
  "bearer_methods_supported": ["header"]
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-clt.md](./itr-clt.md) | MCP Client詳細仕様 |
| [itr-aus.md](./itr-aus.md) | Auth Server詳細仕様 |
| [idx-ept.md](./idx-ept.md) | エンドポイント一覧 |
