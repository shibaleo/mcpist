# Auth Middleware インタラクション仕様書（itr-amw）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.0 |
| Note | Auth Middleware Interaction Specification (MCP Server内部) |

---

## 概要

Auth Middleware（AMW）は、MCP Server内部でリクエスト認証を担当するコンポーネント。

主な責務：
- API Gatewayからのリクエスト受信
- X-Gateway-Secret検証
- 認証済みリクエストのMCP Handlerへの転送

**認可（Authorization）は担当しない。** ツール実行権限の判定はMOD層で行う。

**位置づけ:** MCP Server内部コンポーネント

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| API Gateway | AMW ← GWY | リクエスト受信 |
| MCP Handler | AMW → HDL | 認証済みリクエスト転送 |

---

## 連携詳細

### GWY → AMW（リクエスト受信）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTP（内部通信） |
| 認証 | X-Gateway-Secret ヘッダー検証 |

**受信するヘッダー:**
```
X-User-Id: {user_id}
X-Gateway-Secret: {shared_secret}
X-Request-Id: {request_id}
X-Forwarded-For: {client_ip}
Content-Type: application/json
```

**AMWの処理:**
1. X-Gateway-Secretを検証
2. 検証失敗時：403 Forbidden返却
3. 検証成功時：リクエストをHDLへ転送

**X-Gateway-Secret検証:**
```
expected_secret = env.GATEWAY_SECRET
actual_secret = request.headers["X-Gateway-Secret"]

if actual_secret != expected_secret:
    return 403 Forbidden
```

**セキュリティ考慮:**
- X-Gateway-Secretは環境変数で管理
- デプロイ時にローテーション（GWYとMCP Serverは同一デプロイで更新）
- ログにシークレットを出力しない

---

### AMW → HDL（認証済みリクエスト転送）

| 項目 | 内容 |
|------|------|
| 方向 | AMW → HDL |
| 用途 | 認証済みリクエストの処理委譲 |

**転送する情報:**
- user_id（X-User-Idから抽出）
- リクエストボディ（JSON-RPC）
- リクエストメタデータ

**リクエストコンテキスト（HDLへ転送する情報）:**

| フィールド | 説明 |
|-----------|------|
| user_id | 認証済みユーザーID |
| request_id | リクエスト追跡用ID |
| client_ip | クライアントIPアドレス |

AMWは認証処理のみを担当し、MCPプロトコルの解釈はHDLに委譲する。

---

## エラーレスポンス

### 403 Forbidden（認証失敗）

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32600,
    "message": "Invalid request: authentication failed"
  },
  "id": 1
}
```

### 401 Unauthorized（ヘッダー欠落）

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32600,
    "message": "Invalid request: missing authentication headers"
  },
  "id": 1
}
```

**注:** `id`はリクエストの`id`を返す。リクエストが解析不能な場合のみ`null`。

---

## AMWが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | GWY経由 |
| MCP Client (API KEY) (CLK) | GWY経由 |
| Auth Server (AUS) | GWYがJWT検証を実行 |
| Session Manager (SSM) | ユーザー認証はGWY担当 |
| Data Store (DST) | HDL経由 |
| Token Vault (TVL) | GWYがAPI KEY検証を実行 |
| Modules (MOD) | HDL経由 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-gwy.md](./itr-gwy.md) | API Gateway詳細仕様 |
| [itr-hdl.md](./itr-hdl.md) | MCP Handler詳細仕様 |
| [itr-srv.md](./itr-srv.md) | MCP Server詳細仕様 |
