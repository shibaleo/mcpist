-- ============================================================================
-- Migration: Add 'pre_active' status to account_status enum
-- Purpose: Explicitly track onboarding state
-- Status flow: pre_active (signup) -> active (onboarding complete)
-- ============================================================================

-- Add 'pre_active' value to account_status enum
ALTER TYPE mcpist.account_status ADD VALUE IF NOT EXISTS 'pre_active' BEFORE 'active';

-- Existing users with credits are considered onboarded (status remains 'active')

-- New users created after this migration will have 'pre_active' status by default
-- The trigger in 00000000000004_user_triggers.sql creates records with 'active'
-- We need to update it to use 'pre_active' instead

-- Update the trigger function to use 'pre_active' for new users
CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    -- Create user_credits record with pre_active status (no credits until onboarding)
    INSERT INTO mcpist.user_credits (
        user_id,
        account_status,
        free_credits,
        paid_credits
    )
    VALUES (
        NEW.id,
        'pre_active'::mcpist.account_status,
        0,  -- No credits until onboarding complete
        0
    )
    ON CONFLICT (user_id) DO NOTHING;

    RETURN NEW;
END;
$$;

-- RPC: Complete onboarding (grant credits and set status to active)
-- This replaces the grant-signup-bonus API logic
CREATE OR REPLACE FUNCTION mcpist.complete_onboarding(
    p_user_id UUID,
    p_event_id TEXT  -- Idempotency key (e.g., 'onboarding:{user_id}')
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
    FROM mcpist.user_credits
    WHERE user_id = p_user_id;

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

    -- Update status to active
    UPDATE mcpist.user_credits
    SET account_status = 'active'::mcpist.account_status,
        updated_at = NOW()
    WHERE user_id = p_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'credits_granted', 100,
        'status', 'active'
    );
END;
$$;

-- Grant execute permission
GRANT EXECUTE ON FUNCTION mcpist.complete_onboarding(UUID, TEXT) TO service_role;

-- Public schema wrapper for console access
CREATE OR REPLACE FUNCTION public.complete_onboarding(
    p_user_id UUID,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN mcpist.complete_onboarding(p_user_id, p_event_id);
END;
$$;

GRANT EXECUTE ON FUNCTION public.complete_onboarding(UUID, TEXT) TO service_role;
