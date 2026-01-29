-- =============================================================================
-- Migration: Optimize get_user_context RPC
-- Description:
--   - Remove redundant enabled_modules query (derive from enabled_tools keys)
--   - Single query to get enabled_tools with module_descriptions
--   - Eliminates 3 separate subqueries, uses 1 efficient JOIN
-- =============================================================================

-- Drop existing functions
DROP FUNCTION IF EXISTS mcpist.get_user_context(UUID);
DROP FUNCTION IF EXISTS public.get_user_context(UUID);

-- Optimized get_user_context: enabled_modules derived from enabled_tools keys
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
    v_language TEXT;
    v_module_data JSONB;  -- Combined: {module_name: {tools: [...], description: "..."}}
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_module_descriptions JSONB;
BEGIN
    -- 1. Get user status and language (single query)
    SELECT u.account_status::TEXT, COALESCE(u.preferences->>'language', 'en-US')
    INTO v_account_status, v_language
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- 2. Get credit balance
    SELECT c.free_credits, c.paid_credits
    INTO v_free_credits, v_paid_credits
    FROM mcpist.credits c
    WHERE c.user_id = p_user_id;

    IF v_free_credits IS NULL THEN
        v_free_credits := 0;
        v_paid_credits := 0;
    END IF;

    -- 3. Single query: Get enabled tools grouped by module with descriptions
    --    Only modules with at least one enabled tool are returned
    SELECT
        COALESCE(jsonb_object_agg(
            module_name,
            jsonb_build_object(
                'tools', tools,
                'description', description
            )
        ), '{}'::JSONB)
    INTO v_module_data
    FROM (
        SELECT
            m.name AS module_name,
            array_agg(ts.tool_id) AS tools,
            ms.description
        FROM mcpist.tool_settings ts
        JOIN mcpist.modules m ON m.id = ts.module_id
        LEFT JOIN mcpist.module_settings ms
            ON ms.user_id = ts.user_id AND ms.module_id = ts.module_id
        WHERE ts.user_id = p_user_id
          AND ts.enabled = true
          AND m.status IN ('active', 'beta')
        GROUP BY m.name, ms.description
    ) AS subq;

    -- 4. Extract enabled_modules (keys of v_module_data)
    SELECT array_agg(key)
    INTO v_enabled_modules
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 5. Extract enabled_tools: {module: [tool_ids]}
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'tools'),
        '{}'::JSONB
    )
    INTO v_enabled_tools
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 6. Extract module_descriptions: {module: description} (only non-null)
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'description'),
        '{}'::JSONB
    )
    INTO v_module_descriptions
    FROM jsonb_object_keys(v_module_data) AS key
    WHERE v_module_data->key->>'description' IS NOT NULL
      AND v_module_data->key->>'description' != '';

    -- Handle empty array
    IF v_enabled_modules IS NULL THEN
        v_enabled_modules := ARRAY[]::TEXT[];
    END IF;

    RETURN QUERY SELECT
        v_account_status,
        v_free_credits,
        v_paid_credits,
        v_enabled_modules,
        v_enabled_tools,
        v_language,
        v_module_descriptions;
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

COMMENT ON FUNCTION mcpist.get_user_context IS
'Optimized: Get user context with enabled_modules derived from enabled_tools keys.
Single JOIN query for tools + descriptions. Modules with 0 enabled tools are excluded.';
