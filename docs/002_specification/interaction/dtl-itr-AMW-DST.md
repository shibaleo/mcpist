# AMW - DST インタラクション詳細（dtl-itr-AMW-DST）

## ドキュメント管理情報

| 項目      | 値                                               |
| ------- | ----------------------------------------------- |
| Status  | `reviewed`                                      |
| Version | v2.0                                            |
| Note    | Auth Middleware - Data Store Interaction Detail |

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

| 項目   | 内容                                           |
| ---- | -------------------------------------------- |
| トリガー | GWYからリクエスト受信時                                |
| 操作   | ユーザーコンテキスト（アカウント状態・クレジット残高・有効モジュール・無効ツール）の取得 |

### 取得する情報

| フィールド           | 型                       | 説明                                          |
| --------------- | ----------------------- | ------------------------------------------- |
| account_status  | string                  | アカウント状態（active / suspended / disabled）      |
| free_credits    | integer                 | 無料クレジット残高                                   |
| paid_credits    | integer                 | 有料クレジット残高                                   |
| enabled_modules | string[]                | 有効なモジュール一覧                                  |
| disabled_tools  | map\<string, string[]\> | モジュール別の無効ツール一覧（key: モジュール名, value: ツール名の配列） |

### 期待する振る舞い

- GWY で認証済みのため、有効な user_id が必ず渡される前提とする
- enabled_modules にはシステム上 active または beta のモジュールのうち、ユーザーが無効化していないものが含まれる
- あるモジュールの全ツールが disabled_tools に含まれている場合、そのモジュールは enabled_modules に含めない
- disabled_tools には enabled_modules に含まれるモジュールの無効ツールのみが含まれる

### チェック項目

- account_status が active 以外 → 403 ACCOUNT_NOT_ACTIVE

---

## 関連ドキュメント

| ドキュメント                             | 内容                 |
| ---------------------------------- | ------------------ |
| [itr-AMW.md](./itr-AMW.md)         | Auth Middleware 仕様 |
| [itr-DST.md](./itr-DST.md)         | Data Store 仕様      |

