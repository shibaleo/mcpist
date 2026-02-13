# ランニングバランス方式 実装計画書

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Created | 2026-02-05 |
| Note | Running Balance Pattern Implementation Plan |

---

## 概要

### 背景

現在の MCPist クレジットシステムは以下の2テーブル構成:

| テーブル | 役割 | 更新タイミング |
|----------|------|----------------|
| `credits` | 現在残高（キャッシュ） | 毎回消費/付与時に更新 |
| `credit_transactions` | 取引履歴（監査ログ） | INSERT のみ |

**問題点:**
- `credits` と `credit_transactions` の整合性が保証されない
- `credit_transactions` から残高を再計算する場合 O(n)
- 取引履歴の削除が残高に影響しない（イベントソーシングではない）

### 提案: ランニングバランス方式

各 `credit_transactions` レコードに **その時点の残高** を記録する方式。

```
取引1: amount=-10, running_free=990, running_paid=0
取引2: amount=-5,  running_free=985, running_paid=0
取引3: amount=+100, running_free=985, running_paid=100  (購入)
取引4: amount=-20, running_free=965, running_paid=100
```

**メリット:**
- 残高取得: O(1) - 最新レコードを1件取得
- 履歴整合性: 各レコードが正確な残高を保持
- イベントソーシング準拠: 履歴から状態を再構築可能
- `credits` テーブル廃止可能

---

## 設計

### 1. スキーマ変更

#### Phase 1: カラム追加（後方互換）

```sql
-- credit_transactions に running balance カラム追加
ALTER TABLE mcpist.credit_transactions
  ADD COLUMN running_free INTEGER,
  ADD COLUMN running_paid INTEGER;

-- インデックス: 最新レコード取得用
CREATE INDEX idx_credit_transactions_user_latest
  ON mcpist.credit_transactions(user_id, created_at DESC);

COMMENT ON COLUMN mcpist.credit_transactions.running_free IS 'Balance of free credits after this transaction';
COMMENT ON COLUMN mcpist.credit_transactions.running_paid IS 'Balance of paid credits after this transaction';
```

#### Phase 2: データバックフィル

```sql
-- 既存レコードに running balance を設定
-- created_at 順に累積計算
WITH ordered_transactions AS (
  SELECT
    id,
    user_id,
    type,
    amount,
    credit_type,
    created_at,
    -- 累積計算 (free credits)
    SUM(CASE WHEN credit_type = 'free' THEN amount ELSE 0 END)
      OVER (PARTITION BY user_id ORDER BY created_at
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) + 1000 AS calc_free,
    -- 累積計算 (paid credits)
    SUM(CASE WHEN credit_type = 'paid' THEN amount ELSE 0 END)
      OVER (PARTITION BY user_id ORDER BY created_at
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS calc_paid
  FROM mcpist.credit_transactions
),
validated AS (
  SELECT
    id,
    GREATEST(0, LEAST(1000, calc_free))::INTEGER AS running_free,
    GREATEST(0, calc_paid)::INTEGER AS running_paid
  FROM ordered_transactions
)
UPDATE mcpist.credit_transactions ct
SET
  running_free = v.running_free,
  running_paid = v.running_paid
FROM validated v
WHERE ct.id = v.id
  AND (ct.running_free IS NULL OR ct.running_paid IS NULL);
```

#### Phase 3: NOT NULL 制約追加

```sql
-- 全レコードに値が設定された後
ALTER TABLE mcpist.credit_transactions
  ALTER COLUMN running_free SET NOT NULL,
  ALTER COLUMN running_paid SET NOT NULL;
```

#### Phase 4: credits テーブル非推奨化（オプション）

```sql
-- credits テーブルは残すが、RPC では credit_transactions から残高取得
-- 後方互換性のため credits テーブルも同期更新を継続
```

---

### 2. RPC 変更

#### consume_user_credits の更新

```sql
CREATE OR REPLACE FUNCTION mcpist.consume_user_credits(
    p_user_id UUID,
    p_meta_tool TEXT,
    p_amount INTEGER,
    p_request_id TEXT,
    p_details JSONB DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_current_free INTEGER;
    v_current_paid INTEGER;
    v_new_free INTEGER;
    v_new_paid INTEGER;
    v_consumed_free INTEGER;
    v_consumed_paid INTEGER;
    v_existing_tx RECORD;
BEGIN
    -- Validate meta_tool
    IF p_meta_tool NOT IN ('run', 'batch') THEN
        RETURN jsonb_build_object('success', false, 'error', 'invalid_meta_tool');
    END IF;

    -- Idempotency check
    SELECT id, running_free, running_paid INTO v_existing_tx
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id AND request_id = p_request_id;

    IF v_existing_tx IS NOT NULL THEN
        RETURN jsonb_build_object(
            'success', true,
            'free_credits', v_existing_tx.running_free,
            'paid_credits', v_existing_tx.running_paid,
            'already_processed', true
        );
    END IF;

    -- Get current balance from latest transaction (Running Balance)
    SELECT running_free, running_paid INTO v_current_free, v_current_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE;  -- Lock to prevent concurrent modifications

    -- If no transactions exist, get from credits table (initial state)
    IF v_current_free IS NULL THEN
        SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id
        FOR UPDATE;
    END IF;

    IF v_current_free IS NULL THEN
        RETURN jsonb_build_object('success', false, 'error', 'user_not_found');
    END IF;

    -- Check sufficient balance
    IF (v_current_free + v_current_paid) < p_amount THEN
        RETURN jsonb_build_object(
            'success', false,
            'free_credits', v_current_free,
            'paid_credits', v_current_paid,
            'error', 'insufficient_credits'
        );
    END IF;

    -- Calculate consumption (free first)
    IF v_current_free >= p_amount THEN
        v_consumed_free := p_amount;
        v_consumed_paid := 0;
        v_new_free := v_current_free - p_amount;
        v_new_paid := v_current_paid;
    ELSE
        v_consumed_free := v_current_free;
        v_consumed_paid := p_amount - v_current_free;
        v_new_free := 0;
        v_new_paid := v_current_paid - v_consumed_paid;
    END IF;

    -- Record transaction with running balance (free credits)
    IF v_consumed_free > 0 THEN
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, meta_tool, details, request_id,
            running_free, running_paid
        ) VALUES (
            p_user_id, 'consume', -v_consumed_free, 'free', p_meta_tool, p_details, p_request_id,
            v_new_free, v_new_paid
        );
    END IF;

    -- Record transaction with running balance (paid credits)
    IF v_consumed_paid > 0 THEN
        -- Note: running balance is the same since both happen in same request
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, meta_tool, details, request_id,
            running_free, running_paid
        ) VALUES (
            p_user_id, 'consume', -v_consumed_paid, 'paid', p_meta_tool, p_details, p_request_id,
            v_new_free, v_new_paid
        );
    END IF;

    -- Update credits table (backward compatibility)
    UPDATE mcpist.credits
    SET free_credits = v_new_free, paid_credits = v_new_paid, updated_at = NOW()
    WHERE user_id = p_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free,
        'paid_credits', v_new_paid
    );
END;
$$;
```

#### get_user_balance の新規作成

```sql
-- Running Balance から現在残高を取得（O(1)）
CREATE OR REPLACE FUNCTION mcpist.get_user_balance(p_user_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_free INTEGER;
    v_paid INTEGER;
BEGIN
    -- Get balance from latest transaction
    SELECT running_free, running_paid INTO v_free, v_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1;

    -- Fallback to credits table if no transactions
    IF v_free IS NULL THEN
        SELECT free_credits, paid_credits INTO v_free, v_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id;
    END IF;

    IF v_free IS NULL THEN
        RETURN jsonb_build_object('error', 'user_not_found');
    END IF;

    RETURN jsonb_build_object(
        'free_credits', v_free,
        'paid_credits', v_paid,
        'total_credits', v_free + v_paid
    );
END;
$$;
```

#### grant_user_credits の更新（購入・補充）

```sql
CREATE OR REPLACE FUNCTION mcpist.grant_user_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,  -- 'free' or 'paid'
    p_reason TEXT DEFAULT 'grant'
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_current_free INTEGER;
    v_current_paid INTEGER;
    v_new_free INTEGER;
    v_new_paid INTEGER;
BEGIN
    -- Get current balance
    SELECT running_free, running_paid INTO v_current_free, v_current_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE;

    IF v_current_free IS NULL THEN
        SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id
        FOR UPDATE;
    END IF;

    IF v_current_free IS NULL THEN
        RETURN jsonb_build_object('error', 'user_not_found');
    END IF;

    -- Calculate new balance
    IF p_credit_type = 'free' THEN
        v_new_free := LEAST(1000, v_current_free + p_amount);  -- Cap at 1000
        v_new_paid := v_current_paid;
    ELSE
        v_new_free := v_current_free;
        v_new_paid := v_current_paid + p_amount;
    END IF;

    -- Record transaction with running balance
    INSERT INTO mcpist.credit_transactions (
        user_id, type, amount, credit_type, meta_tool, details,
        running_free, running_paid
    ) VALUES (
        p_user_id, p_reason::mcpist.credit_transaction_type, p_amount, p_credit_type, 'run', '[]'::JSONB,
        v_new_free, v_new_paid
    );

    -- Update credits table (backward compatibility)
    UPDATE mcpist.credits
    SET free_credits = v_new_free, paid_credits = v_new_paid, updated_at = NOW()
    WHERE user_id = p_user_id;

    RETURN jsonb_build_object(
        'free_credits', v_new_free,
        'paid_credits', v_new_paid
    );
END;
$$;
```

---

### 3. アプリケーション層変更

#### Go Server (store/user.go)

```go
// ConsumeResult に RunningFree/RunningPaid を追加
type ConsumeResult struct {
    Success          bool   `json:"success"`
    FreeCredits      int    `json:"free_credits"`      // = running_free
    PaidCredits      int    `json:"paid_credits"`      // = running_paid
    AlreadyProcessed bool   `json:"already_processed,omitempty"`
    Error            string `json:"error,omitempty"`
}

// GetUserBalance - 新規関数（credits テーブルではなく RPC 経由）
func (s *UserStore) GetUserBalance(userID string) (int, int, error) {
    // RPC call to get_user_balance
}
```

#### Console (credits.ts)

```typescript
// getUserContext は変更なし（RPC が内部で running balance を使用）
// 表示も変更なし
```

---

### 4. マイグレーション戦略

| Phase | 内容 | ダウンタイム | ロールバック |
|-------|------|-------------|-------------|
| 1 | カラム追加 (NULLABLE) | なし | DROP COLUMN |
| 2 | バックフィル（既存データ） | なし | - |
| 3 | RPC 更新（両方書き込み） | なし | 旧 RPC |
| 4 | NOT NULL 制約追加 | なし | DROP CONSTRAINT |
| 5 | credits テーブル非推奨化 | なし | 再有効化 |

**ゼロダウンタイム移行:**
- Phase 3 で RPC は `credits` と `credit_transactions` 両方を更新
- 移行完了後、`credits` テーブルは読み取り用として残す（廃止はオプション）

---

### 5. 整合性チェック

#### 定期整合性確認クエリ

```sql
-- credits テーブルと running balance の差分検出
SELECT
    c.user_id,
    c.free_credits AS credits_free,
    c.paid_credits AS credits_paid,
    t.running_free AS tx_free,
    t.running_paid AS tx_paid,
    c.free_credits - t.running_free AS diff_free,
    c.paid_credits - t.running_paid AS diff_paid
FROM mcpist.credits c
LEFT JOIN LATERAL (
    SELECT running_free, running_paid
    FROM mcpist.credit_transactions
    WHERE user_id = c.user_id
    ORDER BY created_at DESC
    LIMIT 1
) t ON true
WHERE c.free_credits != COALESCE(t.running_free, c.free_credits)
   OR c.paid_credits != COALESCE(t.running_paid, c.paid_credits);
```

#### 自動修復（必要時）

```sql
-- credits テーブルを running balance から修復
UPDATE mcpist.credits c
SET
    free_credits = t.running_free,
    paid_credits = t.running_paid,
    updated_at = NOW()
FROM (
    SELECT DISTINCT ON (user_id) user_id, running_free, running_paid
    FROM mcpist.credit_transactions
    ORDER BY user_id, created_at DESC
) t
WHERE c.user_id = t.user_id
  AND (c.free_credits != t.running_free OR c.paid_credits != t.running_paid);
```

---

### 6. パフォーマンス考慮

| 操作 | Before | After |
|------|--------|-------|
| 残高取得 | `SELECT * FROM credits` O(1) | `SELECT ... ORDER BY created_at DESC LIMIT 1` O(1) |
| クレジット消費 | `UPDATE credits` + `INSERT transactions` | `INSERT transactions` + `UPDATE credits` |
| 履歴集計 | O(n) | O(n) (変更なし) |

**インデックス:**
```sql
-- 最新レコード取得の高速化（すでに存在）
CREATE INDEX idx_credit_transactions_user_latest
  ON mcpist.credit_transactions(user_id, created_at DESC);
```

---

### 7. 実装スケジュール

| タスク | 見積もり | 依存 |
|--------|---------|------|
| Phase 1: カラム追加マイグレーション | 小 | - |
| Phase 2: バックフィルマイグレーション | 小 | Phase 1 |
| Phase 3: consume_user_credits RPC 更新 | 中 | Phase 2 |
| Phase 3: grant_user_credits RPC 更新 | 中 | Phase 2 |
| Phase 3: get_user_balance RPC 新規作成 | 小 | Phase 2 |
| Phase 4: NOT NULL 制約追加 | 小 | Phase 3 完了確認 |
| 整合性チェッククエリ作成 | 小 | Phase 3 |
| テスト（本番前確認） | 中 | 全 Phase |

---

### 8. リスクと対策

| リスク | 対策 |
|--------|------|
| 並行実行時の競合 | `FOR UPDATE` ロックで排他制御 |
| バックフィル失敗 | トランザクション内で実行、失敗時ロールバック |
| 移行中の不整合 | Phase 3 で両テーブル更新により整合性維持 |
| パフォーマンス劣化 | インデックス `(user_id, created_at DESC)` で O(1) 維持 |

---

## 結論

ランニングバランス方式により:

1. **データ整合性**: 各取引レコードが正確な残高を保持
2. **監査可能性**: 任意時点の残高を履歴から確認可能
3. **パフォーマンス**: O(1) での残高取得を維持
4. **後方互換性**: `credits` テーブルを併用し段階的移行

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dtl-spc-credit-model.md](../../docs/002_specification/details/dtl-spc-credit-model.md) | クレジットモデル仕様 |
| [20260205000002_credit_transactions_details.sql](../../supabase/migrations/20260205000002_credit_transactions_details.sql) | details カラム追加 |
| [20260205000003_credit_transactions_cleanup.sql](../../supabase/migrations/20260205000003_credit_transactions_cleanup.sql) | レガシーカラム削除 |
