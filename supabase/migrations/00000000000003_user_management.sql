-- =============================================================================
-- MCPist User Management
-- =============================================================================
-- This migration creates:
-- 1. Trigger to auto-create mcpist.users when auth.users is created
-- 2. RPC to get current user's role
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Auth User Trigger
-- Auto-create mcpist.users record when auth.users is created
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO mcpist.users (id, display_name, status, role)
    VALUES (
        NEW.id,
        COALESCE(NEW.raw_user_meta_data->>'name', NEW.email),
        'active',
        'user'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION mcpist.handle_new_user();

COMMENT ON FUNCTION mcpist.handle_new_user() IS 'Auto-create mcpist.users record when auth.users is created';

-- -----------------------------------------------------------------------------
-- Get My Role RPC
-- Returns the role of the current authenticated user
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_role()
RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_role TEXT;
BEGIN
    SELECT role INTO v_role
    FROM mcpist.users
    WHERE id = auth.uid();

    RETURN COALESCE(v_role, 'user');
END;
$$;

-- Public wrapper
CREATE OR REPLACE FUNCTION public.get_my_role()
RETURNS TEXT
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_role();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_role TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_role TO authenticated;
