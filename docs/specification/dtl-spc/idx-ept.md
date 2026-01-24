# エンドポイント一覧（idx-ept）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Endpoint Index |

---

## 概要

MCPistシステムが公開するエンドポイントの一覧。各エンドポイントの詳細は参照先ドキュメントを参照。

---

## MCP Server（SRV）

| エンドポイント | メソッド | 用途 | 参照 |
|---------------|--------|------|------|
| `https://mcp.mcpist.app/mcp` | POST/GET | MCP Protocol (Streamable HTTP) | [itr-clo.md](../interaction/itr-clo.md#clo--srvmcp-server) |
| `https://mcp.mcpist.app/.well-known/oauth-protected-resource` | GET | Protected Resource Metadata (RFC 9728) | [itr-clo.md](../interaction/itr-clo.md#初回認可フローclo--srv--aus) |

---

## Auth Server（AUS）

| エンドポイント | メソッド | 用途 | 参照 |
|---------------|--------|------|------|
| `https://auth.mcpist.app/.well-known/openid-configuration` | GET | OpenID Connect Discovery 1.0 | [itr-aus.md](../interaction/itr-aus.md#メタデータエンドポイント) |
| `https://auth.mcpist.app/.well-known/oauth-authorization-server` | GET | OAuth 2.0 Authorization Server Metadata (RFC 8414) | [itr-aus.md](../interaction/itr-aus.md#メタデータエンドポイント) |
| `https://auth.mcpist.app/.well-known/jwks.json` | GET | JWT検証用公開鍵 (JWKS) | [itr-aus.md](../interaction/itr-aus.md#amw--ausjwks取得) |
| `https://auth.mcpist.app/authorize` | GET | OAuth 2.1 認可リクエスト | [itr-aus.md](../interaction/itr-aus.md#clt--ausmcp-client-からの認可リクエスト) |
| `https://auth.mcpist.app/token` | POST | トークン交換・リフレッシュ | [itr-aus.md](../interaction/itr-aus.md#clt--ausmcp-client-からの認可リクエスト) |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [itr-clo.md](../interaction/itr-clo.md) | MCP Client (OAuth) 詳細仕様 |
| [itr-aus.md](../interaction/itr-aus.md) | Auth Server詳細仕様 |
| [itr-srv.md](../interaction/itr-srv.md) | MCP Server詳細仕様 |
| [itr-con.md](../interaction/itr-con.md) | User Console詳細仕様 |
