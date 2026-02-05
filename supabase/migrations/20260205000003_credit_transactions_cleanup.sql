-- =============================================================================
-- Credit Transactions Cleanup Migration
-- =============================================================================
-- Complete the migration by:
-- 1. Migrating remaining legacy records to new format
-- 2. Deleting legacy batch records
-- 3. Adding NOT NULL constraints
-- 4. Dropping legacy columns and index
-- =============================================================================

-- Migrate remaining 'run' style records (single tool)
-- These have module/tool but no details yet
UPDATE mcpist.credit_transactions
SET
    meta_tool = 'run',
    details = jsonb_build_array(
        jsonb_build_object('module', module, 'tool', tool)
    )
WHERE type = 'consume'
  AND module IS NOT NULL
  AND tool IS NOT NULL
  AND module != 'batch'
  AND tool != 'batch'
  AND details IS NULL;

-- Delete legacy 'batch' records (module='batch', tool='batch')
-- These cannot be reconstructed, so we remove them
-- Credit totals are already reflected in the credits table
DELETE FROM mcpist.credit_transactions
WHERE type = 'consume'
  AND module = 'batch'
  AND tool = 'batch';

-- Set default empty array for non-consume records (grant, purchase, etc.)
UPDATE mcpist.credit_transactions
SET
    meta_tool = 'run',
    details = '[]'::JSONB
WHERE details IS NULL;

-- Add NOT NULL constraint to details
ALTER TABLE mcpist.credit_transactions
  ALTER COLUMN details SET NOT NULL;

-- Add NOT NULL constraint to meta_tool
ALTER TABLE mcpist.credit_transactions
  ALTER COLUMN meta_tool SET NOT NULL;

-- Drop legacy idempotency index that references task_id
DROP INDEX IF EXISTS mcpist.idx_credit_transactions_idempotency;

-- Create new idempotency index (one record per request_id)
CREATE UNIQUE INDEX idx_credit_transactions_idempotency
  ON mcpist.credit_transactions(user_id, request_id)
  WHERE request_id IS NOT NULL;

-- Drop legacy columns
ALTER TABLE mcpist.credit_transactions
  DROP COLUMN IF EXISTS module,
  DROP COLUMN IF EXISTS tool,
  DROP COLUMN IF EXISTS task_id;

-- =============================================================================
-- Update get_my_usage RPC to use only new details format
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.get_my_usage(
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_total_consumed INTEGER;
    v_module_usage JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Total consumption (type = 'consume')
    SELECT COALESCE(SUM(ABS(amount)), 0)::INTEGER
    INTO v_total_consumed
    FROM mcpist.credit_transactions
    WHERE user_id = v_user_id
      AND type = 'consume'
      AND created_at >= p_start_date
      AND created_at < p_end_date;

    -- Aggregate by module from details JSONB array
    SELECT COALESCE(
        jsonb_object_agg(module_name, usage),
        '{}'::JSONB
    )
    INTO v_module_usage
    FROM (
        SELECT
            d->>'module' AS module_name,
            COUNT(*)::INTEGER AS usage
        FROM mcpist.credit_transactions ct,
             jsonb_array_elements(ct.details) AS d
        WHERE ct.user_id = v_user_id
          AND ct.type = 'consume'
          AND ct.created_at >= p_start_date
          AND ct.created_at < p_end_date
        GROUP BY d->>'module'
        ORDER BY usage DESC
    ) sub
    WHERE module_name IS NOT NULL;

    RETURN jsonb_build_object(
        'total_consumed', v_total_consumed,
        'by_module', v_module_usage,
        'period', jsonb_build_object(
            'start', p_start_date,
            'end', p_end_date
        )
    );
END;
$$;
