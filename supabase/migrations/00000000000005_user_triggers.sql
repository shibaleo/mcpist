-- =============================================================================
-- MCPist User Management Triggers
-- =============================================================================
-- This migration creates triggers for user creation
-- New users are created with 'pre_active' status and 0 credits.
-- Onboarding completion grants credits and sets status to 'active'.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Handle New User (triggered when auth.users row is created)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    -- Create user record with pre_active status
    INSERT INTO mcpist.users (id, account_status)
    VALUES (NEW.id, 'pre_active'::mcpist.account_status);

    -- Create credits record with 0 credits (granted on onboarding completion)
    INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
    VALUES (NEW.id, 0, 0);

    RETURN NEW;
END;
$$;

-- Trigger on auth.users
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.handle_new_user();
