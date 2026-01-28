# AMW - DST インタラクション詳細（dtl-itr-AMW-DST）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-022 |
| Note | Auth Middleware - Data Store Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Auth Middleware (AMW) |
| 連携先 | Data Store (DST) |
| 内容 | ユーザーコンテキスト取得 |
| プロトコル | Supabase RPC |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | GWYからリクエスト受信時 |
| 操作 | ユーザーコンテキスト（アカウント状態・クレジット残高・有効モジュール・無効ツール）の取得 |

### 取得する情報

| フィールド | 説明 |
|-----------|------|
| account_status | アカウント状態（active/suspended/disabled） |
| free_credits | 無料クレジット残高 |
| paid_credits | 有料クレジット残高 |
| enabled_modules | 有効なモジュール一覧 |
| disabled_tools | モジュール別の無効ツール一覧 |

### チェック項目

- account_statusがactive以外 → 403 ACCOUNT_NOT_ACTIVE
- クレジット残高が0以下 → 402 INSUFFICIENT_CREDITS

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-AMW.md](./itr-AMW.md) | Auth Middleware 詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
