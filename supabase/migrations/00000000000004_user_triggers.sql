-- =============================================================================
-- MCPist User Management Triggers
-- =============================================================================
-- This migration creates triggers for user creation
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Handle New User (triggered when auth.users row is created)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
    -- Create user record
    INSERT INTO mcpist.users (id)
    VALUES (NEW.id);

    -- Create credits record with default free credits
    INSERT INTO mcpist.credits (user_id)
    VALUES (NEW.id);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Trigger on auth.users
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.handle_new_user();
