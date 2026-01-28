# HDL - MOD インタラクション詳細（dtl-itr-HDL-MOD）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-009 |
| Note | MCP Handler - Modules Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | MCP Handler (HDL) |
| 連携先 | Modules (MOD) |
| 内容 | プリミティブ操作委譲 |
| プロトコル | 内部関数呼び出し |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | 権限チェック完了後 |
| 操作 | tools/resources/promptsの取得・実行 |

### 実行コンテキスト（HDLからMODへ渡す情報）

| フィールド | 説明 |
|-----------|------|
| user_id | 認証済みユーザーID |
| module | 対象モジュール名 |
| primitive_type | プリミティブ種別（tool/resource/prompt） |
| primitive_name | プリミティブ名 |
| params | パラメータ |
| request_id | リクエスト追跡用ID |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-HDL.md](./itr-HDL.md) | MCP Handler 詳細仕様 |
| [itr-MOD.md](./itr-MOD.md) | Modules 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
