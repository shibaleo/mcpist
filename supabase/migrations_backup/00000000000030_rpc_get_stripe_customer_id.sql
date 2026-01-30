-- =============================================================================
-- Migration: RPC function to get stripe_customer_id
-- Description: Add RPC function to retrieve stripe_customer_id for a user
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.get_stripe_customer_id(
    p_user_id UUID
)
RETURNS JSONB
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

    RETURN jsonb_build_object(
        'stripe_customer_id', v_stripe_customer_id
    );
END;
$$;

-- Grant execute permission to service_role
GRANT EXECUTE ON FUNCTION mcpist.get_stripe_customer_id(UUID) TO service_role;
