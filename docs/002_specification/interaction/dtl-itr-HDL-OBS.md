# HDL - OBS インタラクション詳細（dtl-itr-HDL-OBS）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-028 |
| Note | MCP Handler - Observability Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | MCP Handler (HDL) |
| 連携先 | Observability (OBS) |
| 内容 | ツール実行ログ・セキュリティイベント送信 |
| プロトコル | HTTP（Loki Push API） |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | ツール実行完了時、セキュリティイベント発生時 |
| 操作 | 構造化ログの送信 |

### ツール実行ログ

| フィールド | 説明 |
|-----------|------|
| user_id | 実行ユーザーID |
| module | モジュール名 |
| tool | ツール名 |
| duration | 実行時間 |
| success | 成功/失敗 |
| error | エラー内容（失敗時） |
| request_id | リクエスト追跡用ID |

### セキュリティイベント

| フィールド | 説明 |
|-----------|------|
| event_type | イベント種別（unauthorized_access, permission_denied等） |
| user_id | 対象ユーザーID |
| detail | イベント詳細 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-HDL.md](./itr-HDL.md) | MCP Handler 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
