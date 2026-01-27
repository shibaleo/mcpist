# CON - DST インタラクション詳細（dtl-itr-CON-DST）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-016 |
| Note | User Console - Data Store Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | User Console (CON) |
| 連携先 | Data Store (DST) |
| 内容 | ツール設定登録 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| 用途 | ユーザー設定の管理 |
| 操作 | モジュール有効/無効、ツール設定変更 |

### 管理対象の設定

| 設定 | 説明 | 操作 |
|------|------|------|
| enabled_modules | ユーザーが有効化したモジュール一覧 | 登録/更新 |
| tool_settings | モジュール内の個別ツールの有効/無効設定 | 登録/更新 |
| user_prompts | ユーザー定義プロンプト | 登録/更新/削除 |
| credit_balance | クレジット残高 | 参照のみ |
| account_status | アカウント状態（active/suspended/disabled） | 参照のみ |
| usage_stats | 利用統計（モジュール別/期間別の消費量等） | 参照のみ |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-con.md](./itr-con.md) | User Console 詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
