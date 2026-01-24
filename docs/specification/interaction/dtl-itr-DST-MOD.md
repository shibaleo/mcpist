# DST - MOD インタラクション詳細（dtl-itr-DST-MOD）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-011 |
| Note | Data Store - Modules Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Modules (MOD) |
| 連携先 | Data Store (DST) |
| 内容 | クレジット消費 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | ツール実行成功時 |
| 操作 | クレジット残高の減算、消費記録の保存 |
| 対象 | 外部API呼び出しを伴うツール（メタツールを除く） |
| 除外 | get_module_schema, run, batch（メタツール）、リソース取得 |

### 消費リクエスト

```json
{
  "user_id": "user-123",
  "module": "notion",
  "tool": "search",
  "amount": 1,
  "request_id": "req-456",
  "task_id": null
}
```

| フィールド | 必須 | 説明 |
|-----------|------|------|
| user_id | ✅ | ユーザーID |
| module | ✅ | モジュール名 |
| tool | ✅ | ツール名 |
| amount | ✅ | 消費量（現時点では固定1） |
| request_id | ✅ | リクエスト追跡用ID |
| task_id | ✅ | batch内タスクID（runの場合はnull） |

### run/batch の識別

| 呼び出し方式 | task_id | 説明 |
|-------------|---------|------|
| run | `null` | 単発ツール実行 |
| batch | `"task_name"` | batch内の各ツール実行 |

**設計意図:** task_idにより run/batch の利用状況を分析可能。

### 消費レスポンス

```json
{
  "success": true
}
```

**注:** 監査・請求・分析の詳細設計は[dsn-adt.md](../../design/dsn-adt.md)を参照。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-mod.md](./itr-mod.md) | Modules 詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
