-- =============================================================================
-- Subscription Model Migration
-- =============================================================================
-- Migrate from credit-based billing to subscription-based daily usage limits.
--
-- Changes:
-- 1. Create plans table (free/plus)
-- 2. Add plan_id to users table
-- 3. Rename credit_transactions → usage_log
-- 4. Drop credit-related RPCs (consume_user_credits, add_user_credits)
-- 5. Create record_usage RPC (fire-and-forget)
-- 6. Update get_user_context to return plan + daily usage
-- 7. Update get_my_usage to query usage_log
-- 8. Update complete_user_onboarding (no credit grant, just activate)
-- =============================================================================

-- =============================================================================
-- 1. Create plans table
-- =============================================================================

CREATE TABLE mcpist.plans (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    daily_limit     INTEGER NOT NULL,
    price_monthly   INTEGER DEFAULT 0,
    stripe_price_id TEXT,
    features        JSONB DEFAULT '{}'
);

COMMENT ON TABLE mcpist.plans IS 'Subscription plans with daily usage limits';
COMMENT ON COLUMN mcpist.plans.daily_limit IS 'Maximum tool executions per day (UTC 0:00 reset)';
COMMENT ON COLUMN mcpist.plans.price_monthly IS 'Monthly price in JPY (0 = free)';
COMMENT ON COLUMN mcpist.plans.stripe_price_id IS 'Stripe Price ID (NULL = free plan)';

-- Seed initial plans
INSERT INTO mcpist.plans (id, name, daily_limit, price_monthly, stripe_price_id, features) VALUES
    ('free', 'Free', 100, 0, NULL, '{}'),
    ('plus', 'Plus', 500, 980, NULL, '{}');

GRANT SELECT ON mcpist.plans TO anon, authenticated, service_role;

-- =============================================================================
-- 2. Add plan_id to users table
-- =============================================================================

ALTER TABLE mcpist.users
    ADD COLUMN IF NOT EXISTS plan_id TEXT DEFAULT 'free' REFERENCES mcpist.plans(id);

COMMENT ON COLUMN mcpist.users.plan_id IS 'Current subscription plan';

-- Set all existing users to free plan
UPDATE mcpist.users SET plan_id = 'free' WHERE plan_id IS NULL;

ALTER TABLE mcpist.users ALTER COLUMN plan_id SET NOT NULL;

-- =============================================================================
-- 3. Rename credit_transactions → usage_log
-- =============================================================================
-- Repurpose the table: drop credit-specific columns, keep usage tracking.
-- The table already has: user_id, meta_tool, details, request_id, created_at
-- Drop: type, amount, credit_type, running_free, running_paid
-- =============================================================================

ALTER TABLE mcpist.credit_transactions RENAME TO usage_log;

-- Drop credit-specific columns
ALTER TABLE mcpist.usage_log
    DROP COLUMN IF EXISTS type,
    DROP COLUMN IF EXISTS amount,
    DROP COLUMN IF EXISTS credit_type,
    DROP COLUMN IF EXISTS running_free,
    DROP COLUMN IF EXISTS running_paid;

-- Drop the credit_transaction_type enum (no longer needed)
DROP TYPE IF EXISTS mcpist.credit_transaction_type;

-- Add index for daily usage count queries
CREATE INDEX IF NOT EXISTS idx_usage_log_daily
    ON mcpist.usage_log(user_id, created_at DESC);

-- Drop the old idempotency index (no longer enforce uniqueness on request_id)
DROP INDEX IF EXISTS mcpist.idx_credit_transactions_idempotency;

COMMENT ON TABLE mcpist.usage_log IS 'Tool usage records for daily limit tracking and analytics';

-- =============================================================================
-- 4. Drop credit-related RPCs
-- =============================================================================

-- Drop consume_user_credits (both schemas)
DROP FUNCTION IF EXISTS public.consume_user_credits(UUID, TEXT, INTEGER, TEXT, JSONB);
DROP FUNCTION IF EXISTS mcpist.consume_user_credits(UUID, TEXT, INTEGER, TEXT, JSONB);
-- Also drop old signature if exists
DROP FUNCTION IF EXISTS public.consume_user_credits(UUID, TEXT, TEXT, INTEGER, TEXT, TEXT);
DROP FUNCTION IF EXISTS mcpist.consume_user_credits(UUID, TEXT, TEXT, INTEGER, TEXT, TEXT);

-- Drop add_user_credits (both schemas)
DROP FUNCTION IF EXISTS public.add_user_credits(UUID, INTEGER, TEXT, TEXT);
DROP FUNCTION IF EXISTS mcpist.add_user_credits(UUID, INTEGER, TEXT, TEXT);

-- =============================================================================
-- 5. Create record_usage RPC
-- =============================================================================
-- Fire-and-forget: records usage for analytics. No balance check, no failure propagation.
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.record_usage(
    p_user_id UUID,
    p_meta_tool TEXT,        -- 'run' or 'batch'
    p_request_id TEXT,
    p_details JSONB          -- [{module, tool, task_id?}, ...]
)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    INSERT INTO mcpist.usage_log (user_id, meta_tool, request_id, details)
    VALUES (p_user_id, p_meta_tool, p_request_id, p_details);
END;
$$;

CREATE OR REPLACE FUNCTION public.record_usage(
    p_user_id UUID,
    p_meta_tool TEXT,
    p_request_id TEXT,
    p_details JSONB
)
RETURNS VOID
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.record_usage(p_user_id, p_meta_tool, p_request_id, p_details);
$$;

GRANT EXECUTE ON FUNCTION mcpist.record_usage(UUID, TEXT, TEXT, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.record_usage(UUID, TEXT, TEXT, JSONB) TO service_role;

-- =============================================================================
-- 6. Update get_user_context
-- =============================================================================
-- Replace free_credits/paid_credits with plan_id, daily_used, daily_limit.
-- Return type changes, so DROP first then CREATE.
-- =============================================================================

DROP FUNCTION IF EXISTS public.get_user_context(UUID);
DROP FUNCTION IF EXISTS mcpist.get_user_context(UUID);

CREATE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    plan_id TEXT,
    daily_used INTEGER,
    daily_limit INTEGER,
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
    v_plan_id TEXT;
    v_daily_used INTEGER;
    v_daily_limit INTEGER;
    v_language TEXT;
    v_module_data JSONB;
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_module_descriptions JSONB;
BEGIN
    -- 1. Get user status, plan, and language
    SELECT u.account_status::TEXT, u.plan_id, COALESCE(u.settings->>'language', 'en-US')
    INTO v_account_status, v_plan_id, v_language
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- 2. Get plan daily limit
    SELECT p.daily_limit INTO v_daily_limit
    FROM mcpist.plans p
    WHERE p.id = v_plan_id;

    IF v_daily_limit IS NULL THEN
        v_daily_limit := 100;  -- Fallback to free plan limit
    END IF;

    -- 3. Count today's usage (UTC day boundary)
    SELECT COUNT(*)::INTEGER INTO v_daily_used
    FROM mcpist.usage_log
    WHERE user_id = p_user_id
      AND created_at >= (CURRENT_DATE AT TIME ZONE 'UTC');

    -- 4. Get enabled tools grouped by module with descriptions
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

    -- 5. Extract enabled_modules
    SELECT array_agg(key)
    INTO v_enabled_modules
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 6. Extract enabled_tools: {module: [tool_ids]}
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'tools'),
        '{}'::JSONB
    )
    INTO v_enabled_tools
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 7. Extract module_descriptions
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
        v_plan_id,
        v_daily_used,
        v_daily_limit,
        v_enabled_modules,
        v_enabled_tools,
        v_language,
        v_module_descriptions;
END;
$$;

CREATE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    plan_id TEXT,
    daily_used INTEGER,
    daily_limit INTEGER,
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

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;

-- =============================================================================
-- 7. Update get_my_usage
-- =============================================================================
-- Now queries usage_log instead of credit_transactions.
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
    v_total_used INTEGER;
    v_module_usage JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Total usage count
    SELECT COUNT(*)::INTEGER
    INTO v_total_used
    FROM mcpist.usage_log
    WHERE user_id = v_user_id
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
        FROM mcpist.usage_log ul,
             jsonb_array_elements(ul.details) AS d
        WHERE ul.user_id = v_user_id
          AND ul.created_at >= p_start_date
          AND ul.created_at < p_end_date
        GROUP BY d->>'module'
        ORDER BY usage DESC
    ) sub
    WHERE module_name IS NOT NULL;

    RETURN jsonb_build_object(
        'total_used', v_total_used,
        'by_module', v_module_usage,
        'period', jsonb_build_object(
            'start', p_start_date,
            'end', p_end_date
        )
    );
END;
$$;

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
-- 8. Update complete_user_onboarding
-- =============================================================================
-- No credit grant. Just activate and assign free plan.
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.complete_user_onboarding(
    p_user_id UUID,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_current_status mcpist.account_status;
BEGIN
    -- Check current status
    SELECT account_status INTO v_current_status
    FROM mcpist.users
    WHERE id = p_user_id;

    -- If already active, return success (idempotent)
    IF v_current_status = 'active' THEN
        RETURN jsonb_build_object(
            'success', true,
            'already_completed', true,
            'message', 'Onboarding already completed'
        );
    END IF;

    -- If not pre_active, something is wrong
    IF v_current_status IS NULL OR v_current_status != 'pre_active' THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_status',
            'message', 'User is not in pre_active status'
        );
    END IF;

    -- Activate user with free plan
    UPDATE mcpist.users
    SET account_status = 'active'::mcpist.account_status,
        plan_id = 'free',
        updated_at = NOW()
    WHERE id = p_user_id;

    -- Mark event as processed (idempotency)
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW())
    ON CONFLICT DO NOTHING;

    RETURN jsonb_build_object(
        'success', true,
        'plan_id', 'free'
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.complete_user_onboarding(
    p_user_id UUID,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.complete_user_onboarding(p_user_id, p_event_id);
$$;

-- =============================================================================
-- 9. Create activate_subscription RPC
-- =============================================================================
-- Called by Stripe webhook on invoice.paid to upgrade/maintain plan.
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.activate_subscription(
    p_user_id UUID,
    p_plan_id TEXT,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    -- Idempotency check
    IF EXISTS (
        SELECT 1 FROM mcpist.processed_webhook_events
        WHERE event_id = p_event_id
    ) THEN
        RETURN jsonb_build_object(
            'success', true,
            'already_processed', true
        );
    END IF;

    -- Validate plan exists
    IF NOT EXISTS (SELECT 1 FROM mcpist.plans WHERE id = p_plan_id) THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_plan',
            'message', 'Plan not found: ' || p_plan_id
        );
    END IF;

    -- Update user's plan
    UPDATE mcpist.users
    SET plan_id = p_plan_id,
        updated_at = NOW()
    WHERE id = p_user_id;

    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found'
        );
    END IF;

    -- Mark event as processed
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW());

    RETURN jsonb_build_object(
        'success', true,
        'plan_id', p_plan_id
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.activate_subscription(
    p_user_id UUID,
    p_plan_id TEXT,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.activate_subscription(p_user_id, p_plan_id, p_event_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.activate_subscription(UUID, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.activate_subscription(UUID, TEXT, TEXT) TO service_role;
