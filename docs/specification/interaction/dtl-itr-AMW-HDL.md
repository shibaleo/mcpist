# AMW - HDL インタラクション詳細（dtl-itr-AMW-HDL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-007 |
| Note | Auth Middleware - MCP Handler Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Auth Middleware (AMW) |
| 連携先 | MCP Handler (HDL) |
| 内容 | 認証済みリクエスト |
| プロトコル | 内部関数呼び出し |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | 内部関数呼び出し |
| 入力 | JSON-RPC 2.0リクエスト + ユーザーコンテキスト |

### ユーザーコンテキスト（AMWからHDLへ渡す情報）

| フィールド | 説明 |
|-----------|------|
| user_id | 認証済みユーザーID |
| request_id | リクエスト追跡用ID |
| client_ip | クライアントIPアドレス |

### 転送する情報

- user_id（X-User-Idから抽出）
- リクエストボディ（JSON-RPC）
- リクエストメタデータ

AMWは認証処理のみを担当し、MCPプロトコルの解釈はHDLに委譲する。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-amw.md](./itr-amw.md) | Auth Middleware 詳細仕様 |
| [itr-hdl.md](./itr-hdl.md) | MCP Handler 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
