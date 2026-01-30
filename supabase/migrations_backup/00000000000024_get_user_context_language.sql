-- =============================================================================
-- Add language to get_user_context
-- =============================================================================
-- Return user's preferred language from preferences JSONB
-- =============================================================================

-- Drop and recreate with new return type
DROP FUNCTION IF EXISTS mcpist.get_user_context(UUID);
DROP FUNCTION IF EXISTS public.get_user_context(UUID);

CREATE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT
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
BEGIN
    -- ユーザー状態と言語設定を取得
    SELECT u.account_status::TEXT, COALESCE(u.preferences->>'language', 'en-US')
    INTO v_account_status, v_language
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- ユーザーが存在しない場合は空
    END IF;

    -- クレジット残高を取得
    SELECT c.free_credits, c.paid_credits INTO v_free_credits, v_paid_credits
    FROM mcpist.credits c
    WHERE c.user_id = p_user_id;

    IF v_free_credits IS NULL THEN
        v_free_credits := 0;
        v_paid_credits := 0;
    END IF;

    -- 有効なモジュールを取得（module_settingsに存在しないモジュールはデフォルトで有効）
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

    -- 有効なツールを取得（モジュール別） - ホワイトリスト方式
    -- tool_settingsでenabled=trueのツールのみ返す
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

    RETURN QUERY SELECT v_account_status, v_free_credits, v_paid_credits, v_enabled_modules, v_enabled_tools, v_language;
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
    language TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;
