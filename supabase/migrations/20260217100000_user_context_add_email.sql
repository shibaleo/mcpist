-- =============================================================================
-- Add user_id and email to get_user_context
-- =============================================================================
-- checkout/portal routes need user identity (id, email) to create Stripe
-- customers. Adding these fields removes their Supabase auth dependency.
--
-- =============================================================================

DROP FUNCTION IF EXISTS public.get_user_context(UUID);
DROP FUNCTION IF EXISTS mcpist.get_user_context(UUID);

CREATE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    user_id UUID,
    email TEXT,
    account_status TEXT,
    plan_id TEXT,
    daily_used INTEGER,
    daily_limit INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB,
    role TEXT,
    settings JSONB,
    display_name TEXT,
    connected_count INTEGER
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_account_status TEXT;
    v_email TEXT;
    v_plan_id TEXT;
    v_daily_used INTEGER;
    v_daily_limit INTEGER;
    v_language TEXT;
    v_role TEXT;
    v_settings JSONB;
    v_display_name TEXT;
    v_connected_count INTEGER;
    v_module_data JSONB;
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_module_descriptions JSONB;
BEGIN
    -- 1. Get user status, plan, language, settings, display_name
    SELECT
        u.account_status::TEXT,
        u.plan_id,
        COALESCE(u.settings->>'language', 'en-US'),
        COALESCE(u.settings, '{}'::JSONB),
        u.display_name
    INTO v_account_status, v_plan_id, v_language, v_settings, v_display_name
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- 2. Get role and email from auth.users
    SELECT
        COALESCE(raw_app_meta_data->>'role', 'user'),
        auth.users.email
    INTO v_role, v_email
    FROM auth.users
    WHERE auth.users.id = p_user_id;

    -- 3. Get plan daily limit
    SELECT p.daily_limit INTO v_daily_limit
    FROM mcpist.plans p
    WHERE p.id = v_plan_id;

    IF v_daily_limit IS NULL THEN
        v_daily_limit := 100;
    END IF;

    -- 4. Count today's usage (UTC day boundary)
    SELECT COUNT(*)::INTEGER INTO v_daily_used
    FROM mcpist.usage_log
    WHERE user_id = p_user_id
      AND created_at >= (CURRENT_DATE AT TIME ZONE 'UTC');

    -- 5. Count connected services
    SELECT COUNT(*)::INTEGER INTO v_connected_count
    FROM mcpist.user_credentials
    WHERE user_id = p_user_id;

    -- 6. Get enabled tools grouped by module with descriptions
    SELECT
        COALESCE(jsonb_object_agg(
            subq.module_name,
            jsonb_build_object(
                'tools', subq.tools,
                'description', subq.description
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

    -- 7. Extract enabled_modules
    SELECT array_agg(key)
    INTO v_enabled_modules
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 8. Extract enabled_tools: {module: [tool_ids]}
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'tools'),
        '{}'::JSONB
    )
    INTO v_enabled_tools
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 9. Extract module_descriptions
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'description'),
        '{}'::JSONB
    )
    INTO v_module_descriptions
    FROM jsonb_object_keys(v_module_data) AS key
    WHERE v_module_data->key->>'description' IS NOT NULL
      AND v_module_data->key->>'description' != '';

    IF v_enabled_modules IS NULL THEN
        v_enabled_modules := ARRAY[]::TEXT[];
    END IF;

    RETURN QUERY SELECT
        p_user_id,
        v_email,
        v_account_status,
        v_plan_id,
        v_daily_used,
        v_daily_limit,
        v_enabled_modules,
        v_enabled_tools,
        v_language,
        v_module_descriptions,
        v_role,
        v_settings,
        v_display_name,
        v_connected_count;
END;
$$;

CREATE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    user_id UUID,
    email TEXT,
    account_status TEXT,
    plan_id TEXT,
    daily_used INTEGER,
    daily_limit INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB,
    role TEXT,
    settings JSONB,
    display_name TEXT,
    connected_count INTEGER
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;
