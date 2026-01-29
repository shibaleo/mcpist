-- =============================================================================
-- Migration: Add module_descriptions to get_user_context
-- Description: Consolidate get_module_descriptions into get_user_context to reduce RPC calls
--              Server now gets all user context in a single RPC call
-- =============================================================================

-- Drop existing functions
DROP FUNCTION IF EXISTS mcpist.get_user_context(UUID);
DROP FUNCTION IF EXISTS public.get_user_context(UUID);

-- Recreate with module_descriptions included
CREATE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_account_status TEXT;
    v_free_credits INTEGER;
    v_paid_credits INTEGER;
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_language TEXT;
    v_module_descriptions JSONB;
BEGIN
    -- Get user status and language setting
    SELECT u.account_status::TEXT, COALESCE(u.preferences->>'language', 'en-US')
    INTO v_account_status, v_language
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- Get credit balance
    SELECT c.free_credits, c.paid_credits INTO v_free_credits, v_paid_credits
    FROM mcpist.credits c
    WHERE c.user_id = p_user_id;

    IF v_free_credits IS NULL THEN
        v_free_credits := 0;
        v_paid_credits := 0;
    END IF;

    -- Get enabled modules (modules not explicitly disabled)
    SELECT ARRAY(
        SELECT m.name
        FROM mcpist.modules m
        WHERE m.status IN ('active', 'beta')
          AND NOT EXISTS (
              SELECT 1 FROM mcpist.module_settings ms
              WHERE ms.user_id = p_user_id
                AND ms.module_id = m.id
                AND ms.enabled = false
          )
    ) INTO v_enabled_modules;

    -- Get enabled tools (whitelist approach)
    SELECT COALESCE(
        jsonb_object_agg(module_name, tool_list),
        '{}'::JSONB
    ) INTO v_enabled_tools
    FROM (
        SELECT m.name AS module_name, array_agg(ts.tool_id) AS tool_list
        FROM mcpist.tool_settings ts
        JOIN mcpist.modules m ON m.id = ts.module_id
        WHERE ts.user_id = p_user_id AND ts.enabled = true
        GROUP BY m.name
    ) AS subq;

    -- Get module descriptions (custom user descriptions)
    SELECT COALESCE(
        jsonb_object_agg(m.name, ms.description),
        '{}'::JSONB
    ) INTO v_module_descriptions
    FROM mcpist.module_settings ms
    JOIN mcpist.modules m ON m.id = ms.module_id
    WHERE ms.user_id = p_user_id
      AND ms.description IS NOT NULL
      AND ms.description != '';

    RETURN QUERY SELECT v_account_status, v_free_credits, v_paid_credits,
                        v_enabled_modules, v_enabled_tools, v_language, v_module_descriptions;
END;
$$;

-- public schema wrapper
CREATE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;

-- Drop the separate get_module_descriptions function (no longer needed)
DROP FUNCTION IF EXISTS public.get_module_descriptions(UUID);

COMMENT ON FUNCTION mcpist.get_user_context IS
'Get complete user context including account status, credits, enabled modules/tools, language, and module descriptions. Called by Go server with service_role key.';
