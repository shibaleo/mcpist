-- =============================================================================
-- Running Balance Migration
-- =============================================================================
-- Implement Running Balance pattern:
-- Each credit_transactions row stores the balance at that point in time.
--
-- Changes:
-- 1. Add running_free, running_paid columns (NULLABLE first)
-- 2. Backfill existing data
-- 3. Add NOT NULL constraints
-- 4. Update consume_user_credits to write running balance
-- 5. Update add_user_credits to write running balance
-- 6. Update get_user_context to read running balance
-- 7. Add integrity check function
--
-- The credits table is kept for backward compatibility but is now
-- always in sync via running balance writes.
-- =============================================================================

-- =============================================================================
-- Phase 1: Add running balance columns
-- =============================================================================

ALTER TABLE mcpist.credit_transactions
  ADD COLUMN IF NOT EXISTS running_free INTEGER,
  ADD COLUMN IF NOT EXISTS running_paid INTEGER;

-- Index for efficient "latest transaction" query
CREATE INDEX IF NOT EXISTS idx_credit_transactions_user_latest
  ON mcpist.credit_transactions(user_id, created_at DESC);

COMMENT ON COLUMN mcpist.credit_transactions.running_free IS 'Balance of free credits after this transaction';
COMMENT ON COLUMN mcpist.credit_transactions.running_paid IS 'Balance of paid credits after this transaction';

-- =============================================================================
-- Phase 2: Backfill existing data
-- =============================================================================
-- Strategy: Use the CURRENT credits table values as the "final" balance,
-- then walk backwards to compute each row's running balance.
--
-- For each user, the latest transaction should match credits table.
-- Earlier transactions are computed by reversing the amount.
-- =============================================================================

-- Step 1: Set the latest transaction per user to match credits table
WITH latest_tx AS (
  SELECT DISTINCT ON (ct.user_id) ct.id, c.free_credits, c.paid_credits
  FROM mcpist.credit_transactions ct
  JOIN mcpist.credits c ON ct.user_id = c.user_id
  ORDER BY ct.user_id, ct.created_at DESC
)
UPDATE mcpist.credit_transactions ct
SET
  running_free = lt.free_credits,
  running_paid = lt.paid_credits
FROM latest_tx lt
WHERE ct.id = lt.id;

-- Step 2: Walk backwards from the latest transaction to fill in earlier ones.
-- For each transaction, running balance = next transaction's running balance - next transaction's effect.
-- We use a reverse cumulative approach:
--   running_balance[i] = running_balance[latest] - SUM(amounts from i+1 to latest)
--
-- More precisely, for each row we compute:
--   running_free = (latest running_free) - SUM(free amounts after this row)
--   running_paid = (latest running_paid) - SUM(paid amounts after this row)
WITH user_balances AS (
  -- Get each user's final balance from credits table
  SELECT user_id, free_credits, paid_credits
  FROM mcpist.credits
),
ordered_tx AS (
  SELECT
    ct.id,
    ct.user_id,
    ct.amount,
    ct.credit_type,
    ct.created_at,
    -- Cumulative sum of amounts AFTER this row (exclusive) to the end
    -- We use: total_sum_for_user - cumulative_sum_up_to_and_including_this_row
    SUM(CASE WHEN ct.credit_type = 'free' THEN ct.amount ELSE 0 END)
      OVER (PARTITION BY ct.user_id ORDER BY ct.created_at
            ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING)
      - (CASE WHEN ct.credit_type = 'free' THEN ct.amount ELSE 0 END) AS free_after,
    SUM(CASE WHEN ct.credit_type = 'paid' THEN ct.amount ELSE 0 END)
      OVER (PARTITION BY ct.user_id ORDER BY ct.created_at
            ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING)
      - (CASE WHEN ct.credit_type = 'paid' THEN ct.amount ELSE 0 END) AS paid_after
  FROM mcpist.credit_transactions ct
  WHERE ct.running_free IS NULL  -- Only fill unfilled rows
)
UPDATE mcpist.credit_transactions ct
SET
  running_free = GREATEST(0, ub.free_credits - ot.free_after),
  running_paid = GREATEST(0, ub.paid_credits - ot.paid_after)
FROM ordered_tx ot
JOIN user_balances ub ON ot.user_id = ub.user_id
WHERE ct.id = ot.id;

-- Step 3: Handle any remaining NULL rows (users with no credits record)
UPDATE mcpist.credit_transactions
SET running_free = 0, running_paid = 0
WHERE running_free IS NULL OR running_paid IS NULL;

-- =============================================================================
-- Phase 3: Add NOT NULL constraints
-- =============================================================================

ALTER TABLE mcpist.credit_transactions
  ALTER COLUMN running_free SET NOT NULL,
  ALTER COLUMN running_paid SET NOT NULL;

-- =============================================================================
-- Phase 4: Update consume_user_credits RPC
-- =============================================================================
-- Now reads current balance from latest transaction (Running Balance),
-- and writes running_free/running_paid on new transactions.
-- Also updates credits table for backward compatibility.
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
    SELECT id, running_free, running_paid INTO v_existing_tx
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
      AND request_id = p_request_id;

    IF v_existing_tx IS NOT NULL THEN
        RETURN jsonb_build_object(
            'success', true,
            'free_credits', v_existing_tx.running_free,
            'paid_credits', v_existing_tx.running_paid,
            'already_processed', true
        );
    END IF;

    -- Get current balance from latest transaction (Running Balance pattern)
    SELECT running_free, running_paid INTO v_current_free, v_current_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE;

    -- Fallback to credits table if no transactions exist yet
    IF v_current_free IS NULL THEN
        SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id
        FOR UPDATE;
    END IF;

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

    -- Consume free credits first, then paid
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

    -- Record transaction with running balance (free credits consumed)
    IF v_consumed_free > 0 THEN
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, meta_tool, details, request_id,
            running_free, running_paid
        ) VALUES (
            p_user_id, 'consume', -v_consumed_free, 'free', p_meta_tool, p_details, p_request_id,
            v_new_free, v_new_paid
        );
    END IF;

    -- Record transaction with running balance (paid credits consumed)
    IF v_consumed_paid > 0 THEN
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

-- Public schema wrapper (unchanged signature)
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

-- =============================================================================
-- Phase 5: Update add_user_credits RPC
-- =============================================================================
-- Now writes running_free/running_paid on grant/purchase transactions.
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.add_user_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,  -- 'free' or 'paid'
    p_event_id TEXT      -- idempotency key
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
    -- Validate credit_type
    IF p_credit_type NOT IN ('free', 'paid') THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_credit_type',
            'message', 'credit_type must be "free" or "paid"'
        );
    END IF;

    -- Check if event already processed (idempotency)
    IF EXISTS (
        SELECT 1 FROM mcpist.processed_webhook_events
        WHERE event_id = p_event_id
    ) THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'event_already_processed',
            'message', 'This event has already been processed'
        );
    END IF;

    -- Validate amount
    IF p_amount <= 0 THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_amount',
            'message', 'Amount must be positive'
        );
    END IF;

    -- Get current balance from latest transaction (Running Balance pattern)
    SELECT running_free, running_paid INTO v_current_free, v_current_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE;

    -- Fallback to credits table if no transactions
    IF v_current_free IS NULL THEN
        SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id
        FOR UPDATE;
    END IF;

    IF v_current_free IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found',
            'message', 'User not found'
        );
    END IF;

    -- Calculate new balance
    IF p_credit_type = 'free' THEN
        v_new_free := LEAST(1000, v_current_free + p_amount);  -- Cap at 1000
        v_new_paid := v_current_paid;
    ELSE
        v_new_free := v_current_free;
        v_new_paid := v_current_paid + p_amount;
    END IF;

    -- Update credits table (backward compatibility)
    UPDATE mcpist.credits
    SET free_credits = v_new_free, paid_credits = v_new_paid, updated_at = NOW()
    WHERE user_id = p_user_id;

    -- Record transaction with running balance
    INSERT INTO mcpist.credit_transactions (
        user_id, type, amount, credit_type, meta_tool, details, request_id,
        running_free, running_paid
    ) VALUES (
        p_user_id,
        (CASE WHEN p_credit_type = 'free' THEN 'bonus' ELSE 'purchase' END)::mcpist.credit_transaction_type,
        p_amount,
        p_credit_type,
        'run',          -- meta_tool: grant/purchase use 'run' as default
        '[]'::JSONB,    -- details: empty for non-tool transactions
        p_event_id,
        v_new_free,
        v_new_paid
    );

    -- Mark event as processed
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW());

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free,
        'paid_credits', v_new_paid
    );
END;
$$;

-- Public schema wrapper (unchanged signature)
CREATE OR REPLACE FUNCTION public.add_user_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.add_user_credits(p_user_id, p_amount, p_credit_type, p_event_id);
$$;

-- =============================================================================
-- Phase 6: Update get_user_context RPC
-- =============================================================================
-- Now reads balance from latest transaction instead of credits table.
-- Falls back to credits table for users with no transactions.
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_account_status TEXT;
    v_free_credits INTEGER;
    v_paid_credits INTEGER;
    v_language TEXT;
    v_module_data JSONB;
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_module_descriptions JSONB;
BEGIN
    -- 1. Get user status and language
    SELECT u.account_status::TEXT, COALESCE(u.settings->>'language', 'en-US')
    INTO v_account_status, v_language
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- 2. Get credit balance from latest transaction (Running Balance)
    SELECT ct.running_free, ct.running_paid
    INTO v_free_credits, v_paid_credits
    FROM mcpist.credit_transactions ct
    WHERE ct.user_id = p_user_id
    ORDER BY ct.created_at DESC
    LIMIT 1;

    -- Fallback to credits table if no transactions
    IF v_free_credits IS NULL THEN
        SELECT c.free_credits, c.paid_credits
        INTO v_free_credits, v_paid_credits
        FROM mcpist.credits c
        WHERE c.user_id = p_user_id;
    END IF;

    IF v_free_credits IS NULL THEN
        v_free_credits := 0;
        v_paid_credits := 0;
    END IF;

    -- 3. Get enabled tools grouped by module with descriptions
    SELECT
        COALESCE(jsonb_object_agg(
            module_name,
            jsonb_build_object(
                'tools', tools,
                'description', description
            )
        ), '{}'::JSONB)
    INTO v_module_data
    FROM (
        SELECT
            m.name AS module_name,
            array_agg(ts.tool_id) AS tools,
            ms.description
        FROM mcpist.tool_settings ts
        JOIN mcpist.modules m ON m.id = ts.module_id
        LEFT JOIN mcpist.module_settings ms
            ON ms.user_id = ts.user_id AND ms.module_id = ts.module_id
        WHERE ts.user_id = p_user_id
          AND ts.enabled = true
          AND m.status IN ('active', 'beta')
        GROUP BY m.name, ms.description
    ) AS subq;

    -- 4. Extract enabled_modules
    SELECT array_agg(key)
    INTO v_enabled_modules
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 5. Extract enabled_tools: {module: [tool_ids]}
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'tools'),
        '{}'::JSONB
    )
    INTO v_enabled_tools
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 6. Extract module_descriptions
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'description'),
        '{}'::JSONB
    )
    INTO v_module_descriptions
    FROM jsonb_object_keys(v_module_data) AS key
    WHERE v_module_data->key->>'description' IS NOT NULL
      AND v_module_data->key->>'description' != '';

    IF v_enabled_modules IS NULL THEN
        v_enabled_modules := ARRAY[]::TEXT[];
    END IF;

    RETURN QUERY SELECT
        v_account_status,
        v_free_credits,
        v_paid_credits,
        v_enabled_modules,
        v_enabled_tools,
        v_language,
        v_module_descriptions;
END;
$$;

-- Public schema wrapper (unchanged signature)
CREATE OR REPLACE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

-- =============================================================================
-- Phase 7: Integrity check function
-- =============================================================================
-- Detects discrepancies between credits table and running balance.
-- Can be called periodically for monitoring.
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.check_credit_integrity()
RETURNS TABLE (
    user_id UUID,
    credits_free INTEGER,
    credits_paid INTEGER,
    running_free INTEGER,
    running_paid INTEGER,
    diff_free INTEGER,
    diff_paid INTEGER
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
    SELECT
        c.user_id,
        c.free_credits AS credits_free,
        c.paid_credits AS credits_paid,
        t.running_free,
        t.running_paid,
        c.free_credits - COALESCE(t.running_free, c.free_credits) AS diff_free,
        c.paid_credits - COALESCE(t.running_paid, c.paid_credits) AS diff_paid
    FROM mcpist.credits c
    LEFT JOIN LATERAL (
        SELECT ct.running_free, ct.running_paid
        FROM mcpist.credit_transactions ct
        WHERE ct.user_id = c.user_id
        ORDER BY ct.created_at DESC
        LIMIT 1
    ) t ON true
    WHERE c.free_credits != COALESCE(t.running_free, c.free_credits)
       OR c.paid_credits != COALESCE(t.running_paid, c.paid_credits);
$$;

GRANT EXECUTE ON FUNCTION mcpist.check_credit_integrity() TO service_role;
