# DST - HDL インタラクション詳細（dtl-itr-DST-HDL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-008 |
| Note | Data Store - MCP Handler Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | MCP Handler (HDL) |
| 連携先 | Data Store (DST) |
| 内容 | ユーザー設定取得 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | MCPメソッドリクエスト時 |
| 操作 | ユーザー設定・状態の取得 |

### 取得する情報

| フィールド | 説明 |
|-----------|------|
| account_status | アカウント状態（active/suspended/disabled） |
| credit_balance | クレジット残高 |
| enabled_modules | 有効なモジュール一覧 |
| tool_settings | ツール単位の有効/無効設定 |
| user_prompts | ユーザー定義プロンプト |

**注:** プランによるモジュール制限は行わない。

### チェック項目

- account_statusがactive以外 → エラー
- credit_balanceが0以下 → エラー
- モジュールが無効 → エラー
- ツールが無効 → エラー

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-hdl.md](./itr-hdl.md) | MCP Handler 詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
