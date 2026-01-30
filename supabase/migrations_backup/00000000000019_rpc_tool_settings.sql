-- =============================================================================
-- Tool Settings RPC Functions
-- =============================================================================
-- Console用のツール設定取得・更新RPC
-- =============================================================================

-- -----------------------------------------------------------------------------
-- get_tool_settings
-- ユーザーのツール設定を取得（モジュール名で指定）
-- 設定が存在しない場合は空の配列を返す
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_tool_settings(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_name TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ts.tool_name,
        ts.enabled
    FROM mcpist.tool_settings ts
    JOIN mcpist.modules m ON m.id = ts.module_id
    WHERE ts.user_id = p_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY m.name, ts.tool_name;
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_tool_settings(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_name TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_tool_settings(p_user_id, p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_tool_settings(UUID, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_tool_settings(UUID, TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_tool_settings
-- ツール設定を一括更新（モジュール単位）
-- enabled_tools: 有効にするツール名の配列
-- disabled_tools: 無効にするツール名の配列
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_tool_settings(
    p_user_id UUID,
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
BEGIN
    -- モジュールIDを取得
    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    -- 有効ツールをUPSERT
    IF p_enabled_tools IS NOT NULL AND array_length(p_enabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_name, enabled)
        SELECT p_user_id, v_module_id, unnest(p_enabled_tools), true
        ON CONFLICT (user_id, module_id, tool_name)
        DO UPDATE SET enabled = true;
    END IF;

    -- 無効ツールをUPSERT
    IF p_disabled_tools IS NOT NULL AND array_length(p_disabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_name, enabled)
        SELECT p_user_id, v_module_id, unnest(p_disabled_tools), false
        ON CONFLICT (user_id, module_id, tool_name)
        DO UPDATE SET enabled = false;
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'module', p_module_name,
        'enabled_count', COALESCE(array_length(p_enabled_tools, 1), 0),
        'disabled_count', COALESCE(array_length(p_disabled_tools, 1), 0)
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.upsert_tool_settings(
    p_user_id UUID,
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_tool_settings(p_user_id, p_module_name, p_enabled_tools, p_disabled_tools);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_tool_settings(UUID, TEXT, TEXT[], TEXT[]) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_tool_settings(UUID, TEXT, TEXT[], TEXT[]) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_tool_settings
-- 認証済みユーザー自身のツール設定を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_name TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM mcpist.get_tool_settings(auth.uid(), p_module_name);
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_name TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_tool_settings(p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_tool_settings(TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_tool_settings(TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_tool_settings
-- 認証済みユーザー自身のツール設定を更新
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_tool_settings(
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN mcpist.upsert_tool_settings(auth.uid(), p_module_name, p_enabled_tools, p_disabled_tools);
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.upsert_my_tool_settings(
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_tool_settings(p_module_name, p_enabled_tools, p_disabled_tools);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]) TO authenticated;
