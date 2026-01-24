# 監査・請求・分析設計書（dsn-adt）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Audit, Billing, Analytics Design |

---

## 概要

本ドキュメントは、MCPistにおける監査ログ、請求、利用分析の設計を定義する。

**設計原則:**
- 冗長性による整合性検証
- 監査ログと残高の独立管理
- 分析クエリの容易さ

---

## データモデル

### 監査ログテーブル（credit_usage）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | uuid | 主キー |
| user_id | uuid | ユーザーID |
| module | string | モジュール名 |
| tool | string | ツール名 |
| amount | integer | 消費量（現時点では固定1） |
| request_id | string | リクエスト追跡ID |
| task_id | string | batch内タスクID（runの場合はnull） |
| created_at | timestamp | 記録時刻（DST側で付与） |

**インデックス:**
- `user_id` + `created_at`（ユーザー別時系列クエリ）
- `request_id`（リクエスト追跡）
- `task_id`（run/batch判別）

### ユーザー残高（users.credit_balance）

| カラム | 型 | 説明 |
|--------|-----|------|
| credit_balance | integer | 現在のクレジット残高 |
| initial_credits | integer | 初期付与クレジット（検証用） |

---

## run / batch の識別

| 呼び出し方式 | task_id | 説明 |
|-------------|---------|------|
| run | `null` | 単発ツール実行 |
| batch | `"task_name"` | batch内の各ツール実行 |

**設計意図:**
- `task_id IS NULL` の件数 = run呼び出し回数
- `task_id IS NOT NULL` の件数 = batch内ツール実行回数
- 両者の合計 = 全ツール呼び出し回数

---

## 整合性検証

### 基本検証式

```sql
-- 監査ログの合計 = 累計クレジット消費高
SUM(amount) FROM credit_usage WHERE user_id = ?
  =
initial_credits - credit_balance FROM users WHERE id = ?
```

### 検証クエリ

```sql
-- 不整合検出
SELECT
  u.id,
  u.initial_credits - u.credit_balance AS balance_consumed,
  COALESCE(SUM(c.amount), 0) AS log_consumed,
  (u.initial_credits - u.credit_balance) - COALESCE(SUM(c.amount), 0) AS diff
FROM users u
LEFT JOIN credit_usage c ON u.id = c.user_id
GROUP BY u.id
HAVING diff != 0;
```

**不整合の原因候補:**
- バグ（二重消費、消費漏れ）
- 手動調整（管理者による残高変更）
- データ破損

---

## 分析クエリ

### ユーザー別利用状況

```sql
SELECT
  user_id,
  COUNT(*) AS total_calls,
  COUNT(*) FILTER (WHERE task_id IS NULL) AS run_calls,
  COUNT(*) FILTER (WHERE task_id IS NOT NULL) AS batch_tool_calls,
  COUNT(DISTINCT request_id) FILTER (WHERE task_id IS NOT NULL) AS batch_requests,
  SUM(amount) AS total_credits
FROM credit_usage
GROUP BY user_id;
```

### モジュール別利用状況

```sql
SELECT
  module,
  tool,
  COUNT(*) AS call_count,
  SUM(amount) AS total_credits
FROM credit_usage
GROUP BY module, tool
ORDER BY call_count DESC;
```

### 時系列利用状況

```sql
SELECT
  DATE_TRUNC('day', created_at) AS date,
  COUNT(*) AS daily_calls,
  SUM(amount) AS daily_credits
FROM credit_usage
GROUP BY date
ORDER BY date;
```

---

## 請求フロー

### クレジット消費記録

1. MODがツール実行成功
2. MOD → DST に消費リクエスト送信
3. DST が credit_usage に INSERT
4. DST が users.credit_balance を UPDATE（減算）
5. DST が新しい残高を返却

**トランザクション:**
- INSERT と UPDATE は同一トランザクションで実行
- 失敗時はロールバック

### 消費リクエスト形式

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
| amount | ✅ | 消費量 |
| request_id | ✅ | リクエスト追跡ID |
| task_id | ✅ | batch内タスクID（runはnull） |

### 消費レスポンス形式

```json
{
  "success": true
}
```

---

## メトリクス定義

| メトリクス | 計算式 | 用途 |
|-----------|--------|------|
| 全ツール呼び出し回数 | `COUNT(*)` | 利用量把握 |
| run呼び出し回数 | `COUNT(*) WHERE task_id IS NULL` | run利用分析 |
| batch内ツール実行回数 | `COUNT(*) WHERE task_id IS NOT NULL` | batch利用分析 |
| batch呼び出し回数 | `COUNT(DISTINCT request_id) WHERE task_id IS NOT NULL` | batch頻度分析 |
| 累計クレジット消費高 | `SUM(amount)` | 請求根拠 |
| ユーザー残高 | `users.credit_balance` | 残高表示 |

---

## 将来の拡張

### ツール別コスト設定

```sql
-- 将来追加予定
CREATE TABLE tool_costs (
  module VARCHAR NOT NULL,
  tool VARCHAR NOT NULL,
  cost INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (module, tool)
);
```

消費時にルックアップしてamountを決定。

### クレジット購入履歴

```sql
-- 将来追加予定
CREATE TABLE credit_purchases (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  amount INTEGER NOT NULL,
  payment_id VARCHAR,
  created_at TIMESTAMP DEFAULT NOW()
);
```

検証式の拡張:
```sql
initial_credits + SUM(purchases) - SUM(usage) = credit_balance
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-mod.md](../specification/interaction/itr-mod.md) | Modules詳細仕様（消費リクエスト形式） |
| [itr-dst.md](../specification/interaction/itr-dst.md) | Data Store詳細仕様 |
| [spc-tbl.md](../specification/spc-tbl.md) | テーブル仕様書 |
