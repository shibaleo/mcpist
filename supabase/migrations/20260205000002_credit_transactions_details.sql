-- =============================================================================
-- Credit Transactions Schema Enhancement
-- =============================================================================
-- Add meta_tool and details columns for unified run/batch tracking
--
-- meta_tool: 'run' or 'batch' - identifies the MCP meta tool used
-- details: JSONB array of tool execution details
--
-- Format:
--   run:   [{"module": "notion", "tool": "search"}]
--   batch: [{"task_id": "t1", "module": "notion", "tool": "search"}, ...]
-- =============================================================================

-- Add new columns
ALTER TABLE mcpist.credit_transactions
  ADD COLUMN IF NOT EXISTS meta_tool TEXT CHECK (meta_tool IN ('run', 'batch')),
  ADD COLUMN IF NOT EXISTS details JSONB;

-- Create index for meta_tool queries
CREATE INDEX IF NOT EXISTS idx_credit_transactions_meta_tool
  ON mcpist.credit_transactions(meta_tool)
  WHERE meta_tool IS NOT NULL;

-- Create GIN index for details JSONB queries
CREATE INDEX IF NOT EXISTS idx_credit_transactions_details
  ON mcpist.credit_transactions USING GIN (details);

-- Comment on new columns
COMMENT ON COLUMN mcpist.credit_transactions.meta_tool IS 'MCP meta tool: run (single tool) or batch (multiple tools)';
COMMENT ON COLUMN mcpist.credit_transactions.details IS 'JSONB array of tool execution details: [{module, tool, task_id?}, ...]';

-- =============================================================================
-- Update consume_user_credits RPC
-- =============================================================================
-- New signature: consume_user_credits(user_id, meta_tool, amount, request_id, details)
-- Replaces: consume_user_credits(user_id, module, tool, amount, request_id, task_id)
-- =============================================================================

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
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_meta_tool'
        );
    END IF;

    -- Idempotency check (request_id based)
    SELECT id, type INTO v_existing_tx
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
      AND request_id = p_request_id;

    IF v_existing_tx IS NOT NULL THEN
        SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id;

        RETURN jsonb_build_object(
            'success', true,
            'free_credits', v_current_free,
            'paid_credits', v_current_paid,
            'already_processed', true
        );
    END IF;

    -- Get current balance with row lock
    SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
    FROM mcpist.credits
    WHERE user_id = p_user_id
    FOR UPDATE;

    IF v_current_free IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found'
        );
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

    -- Consume free credits first
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

    -- Update credits
    UPDATE mcpist.credits
    SET free_credits = v_new_free, paid_credits = v_new_paid, updated_at = NOW()
    WHERE user_id = p_user_id;

    -- Record transaction (free credits consumed)
    IF v_consumed_free > 0 THEN
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, meta_tool, details, request_id
        ) VALUES (
            p_user_id, 'consume', -v_consumed_free, 'free', p_meta_tool, p_details, p_request_id
        );
    END IF;

    -- Record transaction (paid credits consumed)
    IF v_consumed_paid > 0 THEN
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, meta_tool, details, request_id
        ) VALUES (
            p_user_id, 'consume', -v_consumed_paid, 'paid', p_meta_tool, p_details, p_request_id
        );
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free,
        'paid_credits', v_new_paid
    );
END;
$$;

-- Public schema wrapper (new signature)
CREATE OR REPLACE FUNCTION public.consume_user_credits(
    p_user_id UUID,
    p_meta_tool TEXT,
    p_amount INTEGER,
    p_request_id TEXT,
    p_details JSONB DEFAULT NULL
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.consume_user_credits(p_user_id, p_meta_tool, p_amount, p_request_id, p_details);
$$;

-- Grant permissions
GRANT EXECUTE ON FUNCTION mcpist.consume_user_credits(UUID, TEXT, INTEGER, TEXT, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.consume_user_credits(UUID, TEXT, INTEGER, TEXT, JSONB) TO service_role;

-- =============================================================================
-- Update get_my_usage RPC to aggregate from details
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

    -- Aggregate by module from details JSONB
    -- Handles both new format (details) and legacy format (module column)
    WITH tool_details AS (
        -- New format: extract from details JSONB array
        SELECT
            d->>'module' AS module_name,
            1 AS credit_count
        FROM mcpist.credit_transactions ct,
             jsonb_array_elements(ct.details) AS d
        WHERE ct.user_id = v_user_id
          AND ct.type = 'consume'
          AND ct.details IS NOT NULL
          AND ct.created_at >= p_start_date
          AND ct.created_at < p_end_date

        UNION ALL

        -- Legacy format: use module column directly
        SELECT
            ct.module AS module_name,
            ABS(ct.amount) AS credit_count
        FROM mcpist.credit_transactions ct
        WHERE ct.user_id = v_user_id
          AND ct.type = 'consume'
          AND ct.details IS NULL
          AND ct.module IS NOT NULL
          AND ct.module != 'batch'  -- Exclude old batch records with 'batch' as module
          AND ct.created_at >= p_start_date
          AND ct.created_at < p_end_date
    )
    SELECT COALESCE(
        jsonb_object_agg(module_name, usage),
        '{}'::JSONB
    )
    INTO v_module_usage
    FROM (
        SELECT
            module_name,
            SUM(credit_count)::INTEGER AS usage
        FROM tool_details
        WHERE module_name IS NOT NULL
        GROUP BY module_name
        ORDER BY usage DESC
    ) sub;

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

-- Public schema wrapper
CREATE OR REPLACE FUNCTION public.get_my_usage(
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_usage(p_start_date, p_end_date);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_usage(TIMESTAMPTZ, TIMESTAMPTZ) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_usage(TIMESTAMPTZ, TIMESTAMPTZ) TO authenticated;

-- =============================================================================
-- Migrate existing data to new format
-- =============================================================================
-- Convert existing records with module/tool columns to new details format
-- This preserves all existing data while enabling new queries
-- =============================================================================

-- Migrate 'run' style records (single tool, no task_id)
-- These have module != 'batch' and tool != 'batch'
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
  AND meta_tool IS NULL
  AND details IS NULL;

-- Migrate legacy 'batch' records (module='batch', tool='batch')
-- These cannot be fully reconstructed, but we mark them as batch
-- The details will remain NULL, and get_my_usage handles this case
UPDATE mcpist.credit_transactions
SET
    meta_tool = 'batch'
    -- details remains NULL - we cannot reconstruct individual tool info
WHERE type = 'consume'
  AND module = 'batch'
  AND tool = 'batch'
  AND meta_tool IS NULL;

-- Add comment about legacy batch records
COMMENT ON COLUMN mcpist.credit_transactions.module IS 'Legacy: module name. For new records, use details JSONB.';
COMMENT ON COLUMN mcpist.credit_transactions.tool IS 'Legacy: tool name. For new records, use details JSONB.';
COMMENT ON COLUMN mcpist.credit_transactions.task_id IS 'Legacy: task_id for batch. For new records, use details JSONB.';
