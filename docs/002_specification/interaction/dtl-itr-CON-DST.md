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

### クレジット初期化

サインアップ完了時に、アプリケーション層からクレジットの初期レコードを生成する。

| 項目 | 内容 |
|------|------|
| トリガー | ユーザーのサインアップ完了（初回ログイン検知） |
| 方向 | CON → DST（単方向） |
| 操作 | credits テーブルへの初期レコード生成 |

**credits テーブル:**

| フィールド | 型 | 初期値 |
|-----------|------|--------|
| user_id | UUID | ユーザーID |
| free_credits | integer | 1000 |
| paid_credits | integer | 0 |

### 期待する振る舞い

- ユーザーの初回サインアップ時に、CON が DST に credits レコードを作成する
- 無料クレジット（free_credits = 1000）が初期付与される
- credits レコードが既に存在する場合は重複作成しない（冪等）
- module_settings / tool_settings / api_keys 等はユーザー操作時にオンデマンドで作成される
- クレジット残高・アカウント状態・利用統計は参照のみ（更新は他コンポーネントの責務）

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-CON.md](./itr-CON.md) | User Console 詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
