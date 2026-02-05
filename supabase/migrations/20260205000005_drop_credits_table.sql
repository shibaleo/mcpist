-- =============================================================================
-- Drop Credits Table
-- =============================================================================
-- Running Balance pattern is now the source of truth.
-- The credits table is redundant and can be safely removed.
--
-- Changes:
-- 1. Update consume_user_credits: remove credits table read/write
-- 2. Update add_user_credits: remove credits table read/write
-- 3. Update get_user_context: remove credits table read
-- 4. Update handle_new_user trigger: remove credits INSERT
-- 5. Drop check_credit_integrity (no longer needed)
-- 6. Drop credits table (+ RLS policies, triggers)
-- =============================================================================

-- =============================================================================
-- 1. Update consume_user_credits
-- =============================================================================
-- Remove: credits table fallback read, credits table UPDATE
-- For users with no transactions, return 0/0 as initial balance.
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

    -- Get current balance from latest transaction (Running Balance)
    SELECT running_free, running_paid INTO v_current_free, v_current_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE;

    -- No transactions = initial balance (0, 0)
    IF v_current_free IS NULL THEN
        -- Verify user exists
        IF NOT EXISTS (SELECT 1 FROM mcpist.users WHERE id = p_user_id) THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'user_not_found'
            );
        END IF;
        v_current_free := 0;
        v_current_paid := 0;
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

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free,
        'paid_credits', v_new_paid
    );
END;
$$;

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
-- 2. Update add_user_credits
-- =============================================================================
-- Remove: credits table fallback read, credits table UPDATE
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.add_user_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,
    p_event_id TEXT
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

    -- Get current balance from latest transaction (Running Balance)
    SELECT running_free, running_paid INTO v_current_free, v_current_paid
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1
    FOR UPDATE;

    -- No transactions = initial balance (0, 0)
    IF v_current_free IS NULL THEN
        IF NOT EXISTS (SELECT 1 FROM mcpist.users WHERE id = p_user_id) THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'user_not_found',
                'message', 'User not found'
            );
        END IF;
        v_current_free := 0;
        v_current_paid := 0;
    END IF;

    -- Calculate new balance
    IF p_credit_type = 'free' THEN
        v_new_free := LEAST(1000, v_current_free + p_amount);
        v_new_paid := v_current_paid;
    ELSE
        v_new_free := v_current_free;
        v_new_paid := v_current_paid + p_amount;
    END IF;

    -- Record transaction with running balance
    INSERT INTO mcpist.credit_transactions (
        user_id, type, amount, credit_type, meta_tool, details, request_id,
        running_free, running_paid
    ) VALUES (
        p_user_id,
        (CASE WHEN p_credit_type = 'free' THEN 'bonus' ELSE 'purchase' END)::mcpist.credit_transaction_type,
        p_amount,
        p_credit_type,
        'run',
        '[]'::JSONB,
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
-- 3. Update get_user_context
-- =============================================================================
-- Remove: credits table fallback read
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

    -- No transactions = initial balance (0, 0)
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
-- 4. Update handle_new_user trigger
-- =============================================================================
-- Remove: credits INSERT (no longer needed)
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_is_admin BOOLEAN;
BEGIN
    -- Check if email is in admin_emails table
    SELECT EXISTS(
        SELECT 1 FROM mcpist.admin_emails WHERE email = NEW.email
    ) INTO v_is_admin;

    -- If admin email, update raw_app_meta_data to include admin role
    IF v_is_admin THEN
        UPDATE auth.users
        SET raw_app_meta_data = COALESCE(raw_app_meta_data, '{}'::jsonb) || '{"role": "admin"}'::jsonb
        WHERE id = NEW.id;
    END IF;

    -- Create user record with pre_active status
    INSERT INTO mcpist.users (id, account_status)
    VALUES (NEW.id, 'pre_active'::mcpist.account_status);

    RETURN NEW;
END;
$$;

-- =============================================================================
-- 5. Drop check_credit_integrity (references credits table)
-- =============================================================================

DROP FUNCTION IF EXISTS mcpist.check_credit_integrity();

-- =============================================================================
-- 6. Drop credits table
-- =============================================================================
-- This also drops: RLS policies, triggers, indexes on credits table
-- =============================================================================

DROP TABLE IF EXISTS mcpist.credits CASCADE;
