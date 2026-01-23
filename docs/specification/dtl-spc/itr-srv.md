# MCP Server インタラクション仕様書（itr-srv）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v3.0 |
| Note | MCP Server Interaction Specification - REG統合版 |

---

## 概要

MCP Server（SRV）は、MCP Clientからのリクエストを受け付けるサーバー。外部からは単一のコンポーネントとして見えるが、内部は複数のコンポーネントで構成される。

**内部コンポーネント:**
- Auth Middleware（AMW）
- MCP Handler（HDL）
- Modules（MOD）

主な責務：
- MCP Protocolリクエストの受付
- 認証・認可の実行
- ツール/リソース/プロンプトの提供
- 外部サービスとの連携

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| API Gateway | SRV ← GWY | リクエスト受付（Auth Middleware経由） |

外部コンポーネントからはSRVとして抽象化される。内部コンポーネント間の詳細は各仕様書を参照。

---

## 連携詳細

### GWY → SRV（リクエスト受付）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTP（内部通信） |
| 認証 | X-Gateway-Secret ヘッダー（GWYで検証済み） |
| エントリーポイント | Auth Middleware（AMW） |

**リクエストフロー:**
```mermaid
sequenceDiagram
    participant CLO as MCP Client
    participant GWY as API Gateway
    participant AMW as Auth Middleware
    participant HDL as MCP Handler
    participant DST as Data Store
    participant MOD as Modules
    participant EXT as External API

    CLO->>GWY: MCP Request (JWT)
    GWY->>GWY: JWT検証
    GWY->>AMW: Request (X-Gateway-Secret)
    AMW->>AMW: Secret検証
    AMW->>HDL: 認証済みRequest
    HDL->>HDL: JSON-RPC解析
    HDL->>DST: ユーザー設定取得
    DST-->>HDL: account_status, credit_balance, enabled_modules, tool_settings
    HDL->>HDL: アカウント状態・クレジット・モジュール・ツール確認
    HDL->>MOD: プリミティブ操作
    MOD->>EXT: API呼び出し
    EXT-->>MOD: Response
    MOD->>DST: クレジット消費
    DST-->>MOD: 消費完了
    MOD-->>HDL: Result
    HDL-->>AMW: JSON-RPC Response
    AMW-->>GWY: Response
    GWY-->>CLO: Response
```

---

## Protected Resource Metadata

MCP Clientが初回認可フローで参照するメタデータ。

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

## MCP Protocol

### サポートするメソッド

| メソッド | 説明 | 処理担当 |
|----------|------|----------|
| initialize | セッション初期化 | HDL |
| ping | ヘルスチェック | HDL |
| tools/list | ツール一覧取得 | HDL（メタツール返却） |
| tools/call | ツール実行 | HDL → MOD |
| resources/list | リソース一覧取得 | HDL → MOD |
| resources/read | リソース取得 | HDL → MOD |
| prompts/list | プロンプト一覧取得 | HDL → DST + MOD |
| prompts/get | プロンプト取得 | HDL → DST + MOD |

### リクエスト形式

```
POST https://mcp.mcpist.app/mcp
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json, text/event-stream
MCP-Protocol-Version: 2025-11-25
MCP-Session-Id: {session_id}
```

### レスポンス形式

```
Content-Type: application/json または text/event-stream
MCP-Session-Id: {session_id}
```

---

## 内部コンポーネント詳細

### Auth Middleware（AMW）

X-Gateway-Secretを検証し、認証済みリクエストをHDLへ転送する。

詳細: [itr-amw.md](./itr-amw.md)

### MCP Handler（HDL）

JSON-RPCリクエストを解析し、MCPプリミティブ（tools, resources, prompts）を統一的に管理・処理する。DSTから権限情報を取得し、MODにプリミティブ操作を委譲する。

詳細: [itr-hdl.md](./itr-hdl.md)

### Modules（MOD）

外部サービスとの連携を実装する個別モジュールの集合。tools, resources, promptsの全プリミティブを提供する。

詳細: [itr-mod.md](./itr-mod.md)

---

## SRVが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | GWY経由 |
| MCP Client (API KEY) (CLK) | GWY経由 |
| Auth Server (AUS) | GWY経由（JWKS取得） |
| Session Manager (SSM) | DST経由 |
| Token Vault (TVL) | MOD経由 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-gwy.md](./itr-gwy.md) | API Gateway詳細仕様 |
| [itr-amw.md](./itr-amw.md) | Auth Middleware詳細仕様 |
| [itr-hdl.md](./itr-hdl.md) | MCP Handler詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
| [itr-clo.md](./itr-clo.md) | MCP Client (OAuth2.0)詳細仕様 |
