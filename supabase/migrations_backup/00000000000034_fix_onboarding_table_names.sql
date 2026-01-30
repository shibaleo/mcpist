-- ============================================================================
-- Migration: Fix table names in onboarding functions
-- Purpose: Use correct table names (mcpist.users, mcpist.credits)
-- ============================================================================

-- Fix the trigger function to use correct table names
CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    -- Create user record with pre_active status (no credits until onboarding)
    INSERT INTO mcpist.users (id, account_status)
    VALUES (NEW.id, 'pre_active'::mcpist.account_status)
    ON CONFLICT (id) DO NOTHING;

    -- Create credits record with 0 credits (will be granted after onboarding)
    INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
    VALUES (NEW.id, 0, 0)
    ON CONFLICT (user_id) DO NOTHING;

    RETURN NEW;
END;
$$;

-- Fix the complete_onboarding function to use correct table names
CREATE OR REPLACE FUNCTION mcpist.complete_onboarding(
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
    -- Check current status from mcpist.users table
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

    -- Grant signup bonus credits using add_credits (handles idempotency)
    SELECT mcpist.add_credits(p_user_id, 100, 'free', p_event_id) INTO v_result;

    IF NOT (v_result->>'success')::boolean THEN
        -- If already processed (idempotent), still update status
        IF v_result->>'error' = 'event_already_processed' THEN
            -- Continue to update status
            NULL;
        ELSE
            RETURN v_result;
        END IF;
    END IF;

    -- Update status to active in mcpist.users table
    UPDATE mcpist.users
    SET account_status = 'active'::mcpist.account_status,
        updated_at = NOW()
    WHERE id = p_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'credits_granted', 100,
        'status', 'active'
    );
END;
$$;
