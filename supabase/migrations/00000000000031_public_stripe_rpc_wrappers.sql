-- =============================================================================
-- Migration: Public schema wrappers for Stripe RPC functions
-- Description: Create wrapper functions in public schema to call mcpist functions
-- =============================================================================

-- Wrapper for get_stripe_customer_id
CREATE OR REPLACE FUNCTION public.get_stripe_customer_id(
    p_user_id UUID
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN mcpist.get_stripe_customer_id(p_user_id);
END;
$$;

GRANT EXECUTE ON FUNCTION public.get_stripe_customer_id(UUID) TO service_role;

-- Wrapper for link_stripe_customer
CREATE OR REPLACE FUNCTION public.link_stripe_customer(
    p_user_id UUID,
    p_stripe_customer_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN mcpist.link_stripe_customer(p_user_id, p_stripe_customer_id);
END;
$$;

GRANT EXECUTE ON FUNCTION public.link_stripe_customer(UUID, TEXT) TO service_role;

-- Wrapper for add_paid_credits
CREATE OR REPLACE FUNCTION public.add_paid_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN mcpist.add_paid_credits(p_user_id, p_amount, p_event_id);
END;
$$;

GRANT EXECUTE ON FUNCTION public.add_paid_credits(UUID, INTEGER, TEXT) TO service_role;

-- Wrapper for get_user_by_stripe_customer
CREATE OR REPLACE FUNCTION public.get_user_by_stripe_customer(
    p_stripe_customer_id TEXT
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN mcpist.get_user_by_stripe_customer(p_stripe_customer_id);
END;
$$;

GRANT EXECUTE ON FUNCTION public.get_user_by_stripe_customer(TEXT) TO service_role;
