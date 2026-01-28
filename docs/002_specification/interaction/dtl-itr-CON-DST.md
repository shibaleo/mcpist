# CON - DST インタラクション詳細（dtl-itr-CON-DST）

## ドキュメント管理情報

| 項目      | 値                                            |
| ------- | -------------------------------------------- |
| Status  | `reviewed`                                   |
| Version | v2.0                                         |
| Note    | User Console - Data Store Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | User Console (CON) |
| 連携先 | Data Store (DST) |
| 内容 | ユーザー設定管理 |
| プロトコル | Supabase RPC |

---

## 詳細

| 項目 | 内容 |
|------|------|
| 方向 | CON → DST（単方向） |
| 用途 | ユーザー設定の参照・登録・更新 |

### 管理対象の設定

| 設定 | 説明 | 操作 |
|------|------|------|
| enabled_modules | ユーザーが有効化したモジュール一覧 | 参照/登録/更新 |
| tool_settings | モジュール内の個別ツールの有効/無効設定 | 参照/登録/更新 |
| user_prompts | ユーザー定義プロンプト | 参照/登録/更新/削除 |
| credit_balance | クレジット残高 | 参照/付与 |
| account_status | アカウント状態（active/suspended/disabled） | 参照のみ |
| usage_stats | 利用統計（モジュール別/期間別の消費量等） | 参照のみ |

### クレジット付与

CON からユーザーに任意整数分のクレジットを付与できる。

| 項目 | 内容 |
|------|------|
| トリガー | 管理者操作、キャンペーン適用、課金完了等 |
| 方向 | CON → DST（単方向） |
| 操作 | credits テーブルへのクレジット加算 |

**credits テーブル:**

| フィールド | 型 | 説明 |
|-----------|------|------|
| user_id | UUID | ユーザーID |
| free_credits | integer | 無料クレジット残高 |
| paid_credits | integer | 有料クレジット残高 |

### 期待する振る舞い

- CON は DST に対して Supabase RPC を介してユーザー設定を参照・更新する
- enabled_modules の参照・登録・更新は module_settings テーブルを使用する
- tool_settings の参照・登録・更新は `get_my_tool_settings` / `upsert_my_tool_settings` RPC を使用する
- user_prompts の参照・登録・更新・削除は prompts テーブルを使用する
- credit_balance は `get_user_context` RPC で参照し、クレジット付与は専用 RPC を使用する
- account_status は `get_user_context` RPC で参照する（更新は他コンポーネントの責務）
- usage_stats は専用 RPC で参照する（集計は DST 側で実施）
- すべての RPC は認証済みユーザー（`authenticated` role）のみ実行可能
- RPC 内で `auth.uid()` を使用して現在のユーザーを判定し、他ユーザーのデータにはアクセスできない

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-CON.md](./itr-CON.md) | User Console 詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [dtl-itr-AUS-DST.md](./dtl-itr-AUS-DST.md) | AUS→DST ユーザー初期化（トリガー） |
