-- =============================================================================
-- Migration: Stripe Integration
-- Description: Add stripe_customer_id to users table for Stripe integration
-- =============================================================================

-- Add stripe_customer_id column to users table
ALTER TABLE mcpist.users
ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT UNIQUE;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id
ON mcpist.users(stripe_customer_id)
WHERE stripe_customer_id IS NOT NULL;

-- Add comment
COMMENT ON COLUMN mcpist.users.stripe_customer_id IS 'Stripe Customer ID (cus_xxx) for payment integration';

-- =============================================================================
-- RPC: Add paid credits (for webhook processing)
-- =============================================================================
CREATE OR REPLACE FUNCTION mcpist.add_paid_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_result JSONB;
    v_new_paid_credits INTEGER;
BEGIN
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

    -- Add credits with row lock
    UPDATE mcpist.credits
    SET
        paid_credits = paid_credits + p_amount,
        updated_at = NOW()
    WHERE user_id = p_user_id
    RETURNING paid_credits INTO v_new_paid_credits;

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
        'purchase',
        p_amount,
        'paid',
        p_event_id
    );

    -- Mark event as processed
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW());

    RETURN jsonb_build_object(
        'success', true,
        'paid_credits', v_new_paid_credits,
        'added', p_amount
    );
END;
$$;

-- Grant execute permission to service_role
GRANT EXECUTE ON FUNCTION mcpist.add_paid_credits(UUID, INTEGER, TEXT) TO service_role;

-- =============================================================================
-- RPC: Link Stripe Customer to User
-- =============================================================================
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

-- Grant execute permission to service_role
GRANT EXECUTE ON FUNCTION mcpist.link_stripe_customer(UUID, TEXT) TO service_role;

-- =============================================================================
-- RPC: Get User by Stripe Customer ID
-- =============================================================================
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

-- Grant execute permission to service_role
GRANT EXECUTE ON FUNCTION mcpist.get_user_by_stripe_customer(TEXT) TO service_role;
