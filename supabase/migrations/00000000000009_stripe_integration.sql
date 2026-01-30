-- =============================================================================
-- Migration: Stripe Integration
-- =============================================================================
-- Stripe連携機能:
-- 1. stripe_customer_id カラム追加
-- 2. add_user_credits - 統合クレジット追加RPC
-- 3. complete_user_onboarding - オンボーディング完了RPC
-- 4. link_stripe_customer - StripeカスタマーID紐付け
-- 5. get_user_by_stripe_customer - StripeカスタマーIDからユーザー取得
-- 6. get_stripe_customer_id - ユーザーのStripeカスタマーID取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- stripe_customer_id カラム追加
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.users
ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id
ON mcpist.users(stripe_customer_id)
WHERE stripe_customer_id IS NOT NULL;

COMMENT ON COLUMN mcpist.users.stripe_customer_id IS 'Stripe Customer ID (cus_xxx) for payment integration';

-- -----------------------------------------------------------------------------
-- add_user_credits RPC
-- クレジット追加（free/paid統合、冪等性保証）
-- -----------------------------------------------------------------------------

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
    v_new_free_credits INTEGER;
    v_new_paid_credits INTEGER;
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

    -- Add credits based on type
    IF p_credit_type = 'free' THEN
        UPDATE mcpist.credits
        SET
            free_credits = free_credits + p_amount,
            updated_at = NOW()
        WHERE user_id = p_user_id
        RETURNING free_credits, paid_credits INTO v_new_free_credits, v_new_paid_credits;
    ELSE
        UPDATE mcpist.credits
        SET
            paid_credits = paid_credits + p_amount,
            updated_at = NOW()
        WHERE user_id = p_user_id
        RETURNING free_credits, paid_credits INTO v_new_free_credits, v_new_paid_credits;
    END IF;

    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found',
            'message', 'User not found'
        );
    END IF;

    -- Record transaction
    INSERT INTO mcpist.credit_transactions (
        user_id,
        type,
        amount,
        credit_type,
        request_id
    ) VALUES (
        p_user_id,
        CASE WHEN p_credit_type = 'free' THEN 'bonus' ELSE 'purchase' END,
        p_amount,
        p_credit_type,
        p_event_id
    );

    -- Mark event as processed
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW());

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free_credits,
        'paid_credits', v_new_paid_credits
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

GRANT EXECUTE ON FUNCTION mcpist.add_user_credits(UUID, INTEGER, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.add_user_credits(UUID, INTEGER, TEXT, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- complete_user_onboarding RPC
-- オンボーディング完了（pre_active → active + 初期クレジット付与）
-- -----------------------------------------------------------------------------

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
    v_result JSONB;
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

    -- Grant signup bonus credits using add_user_credits (handles idempotency)
    SELECT mcpist.add_user_credits(p_user_id, 100, 'free', p_event_id) INTO v_result;

    IF NOT (v_result->>'success')::boolean THEN
        IF v_result->>'error' = 'event_already_processed' THEN
            NULL;  -- Continue to update status
        ELSE
            RETURN v_result;
        END IF;
    END IF;

    -- Update status to active
    UPDATE mcpist.users
    SET account_status = 'active'::mcpist.account_status,
        updated_at = NOW()
    WHERE id = p_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', 100
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

GRANT EXECUTE ON FUNCTION mcpist.complete_user_onboarding(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.complete_user_onboarding(UUID, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- link_stripe_customer RPC
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.link_stripe_customer(
    p_user_id UUID,
    p_stripe_customer_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    UPDATE mcpist.users
    SET stripe_customer_id = p_stripe_customer_id
    WHERE id = p_user_id;

    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'stripe_customer_id', p_stripe_customer_id
    );
END;
$$;

GRANT EXECUTE ON FUNCTION mcpist.link_stripe_customer(UUID, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_user_by_stripe_customer RPC
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_user_by_stripe_customer(
    p_stripe_customer_id TEXT
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
BEGIN
    SELECT id INTO v_user_id
    FROM mcpist.users
    WHERE stripe_customer_id = p_stripe_customer_id;

    RETURN v_user_id;
END;
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_by_stripe_customer(TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_stripe_customer_id RPC
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_stripe_customer_id(p_user_id UUID)
RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_stripe_customer_id TEXT;
BEGIN
    SELECT stripe_customer_id INTO v_stripe_customer_id
    FROM mcpist.users
    WHERE id = p_user_id;

    RETURN v_stripe_customer_id;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_stripe_customer_id(p_user_id UUID)
RETURNS TEXT
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_stripe_customer_id(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_stripe_customer_id(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_stripe_customer_id(UUID) TO service_role;
