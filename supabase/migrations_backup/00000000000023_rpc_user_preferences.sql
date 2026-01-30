-- =============================================================================
-- User Preferences RPC Functions
-- =============================================================================
-- Functions for managing user preferences (language, etc.)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- get_my_preferences
-- Get current user's preferences
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_preferences()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_preferences JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT preferences INTO v_preferences
    FROM mcpist.users
    WHERE id = v_user_id;

    RETURN COALESCE(v_preferences, '{}'::JSONB);
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_my_preferences()
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_preferences();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_preferences() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_preferences() TO authenticated;

-- -----------------------------------------------------------------------------
-- update_my_preferences
-- Update current user's preferences (merge with existing)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.update_my_preferences(p_preferences JSONB)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_current JSONB;
    v_updated JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Get current preferences
    SELECT COALESCE(preferences, '{}'::JSONB) INTO v_current
    FROM mcpist.users
    WHERE id = v_user_id;

    -- Merge with new preferences
    v_updated := v_current || p_preferences;

    -- Update
    UPDATE mcpist.users
    SET preferences = v_updated
    WHERE id = v_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'preferences', v_updated
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.update_my_preferences(p_preferences JSONB)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.update_my_preferences(p_preferences);
$$;

GRANT EXECUTE ON FUNCTION mcpist.update_my_preferences(JSONB) TO authenticated;
GRANT EXECUTE ON FUNCTION public.update_my_preferences(JSONB) TO authenticated;
