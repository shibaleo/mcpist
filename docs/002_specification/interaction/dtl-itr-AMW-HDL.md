# AMW - HDL インタラクション詳細（dtl-itr-AMW-HDL）

## ドキュメント管理情報

| 項目      | 値                                                |
| ------- | ------------------------------------------------ |
| Status  | `reviewed`                                       |
| Version | v2.0                                             |
| Note    | Auth Middleware - MCP Handler Interaction Detail |

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

### AMW が HDL へ渡す情報

リクエストコンテキストに以下を格納し、HDL へ委譲する。

**AuthContext:**

| フィールド | 型 | 説明 |
|-----------|------|------|
| user_id | string (UUID) | 認証済みユーザーID |
| auth_type | string | 認証方式（`jwt` / `api_key`） |
| free_credits | integer | 無料クレジット残高 |
| paid_credits | integer | 有料クレジット残高 |
| enabled_modules | string[] | 有効なモジュール一覧 |
| disabled_tools | map\<string, string[]\> | モジュール別の無効ツール一覧 |

**RequestID:**

| フィールド | 型 | 説明 |
|-----------|------|------|
| request_id | string (UUID) | リクエストトレース用ID（GWY が発行） |

### 期待する振る舞い

- AMW は認証・認可処理のみを担当し、MCP プロトコルの解釈は HDL に委譲する
- HDL に到達するリクエストは account_status が active であることが保証されている
- HDL はクレジット残高・モジュール有効性・ツール有効性の判定に AuthContext を使用する

---

## 関連ドキュメント

| ドキュメント                     | 内容                 |
| -------------------------- | ------------------ |
| [itr-AMW.md](./itr-AMW.md) | Auth Middleware 仕様 |
| [itr-HDL.md](./itr-HDL.md) | MCP Handler 仕様     |

