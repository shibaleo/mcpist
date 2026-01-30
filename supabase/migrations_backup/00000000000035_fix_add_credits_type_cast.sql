-- =============================================================================
-- Migration: Fix type cast in add_credits function
-- Description: Cast 'type' column value to credit_transaction_type enum
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.add_credits(
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

    -- Record transaction with proper type cast
    INSERT INTO mcpist.credit_transactions (
        user_id,
        type,
        amount,
        credit_type,
        request_id
    ) VALUES (
        p_user_id,
        (CASE WHEN p_credit_type = 'free' THEN 'bonus' ELSE 'purchase' END)::mcpist.credit_transaction_type,
        p_amount,
        p_credit_type,
        p_event_id
    );

    -- Mark event as processed
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW());

    RETURN jsonb_build_object(
        'success', true,
        'credit_type', p_credit_type,
        'free_credits', v_new_free_credits,
        'paid_credits', v_new_paid_credits,
        'added', p_amount
    );
END;
$$;
