-- =============================================================================
-- Tool ID Migration + Module Description
-- =============================================================================
-- 1. tool_settings.tool_name -> tool_id (破壊的変更)
-- 2. module_settings.description 追加
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. tool_settings: tool_name -> tool_id
-- 既存データは互換性なし（新しいID形式に移行）
-- -----------------------------------------------------------------------------

-- 既存データを削除（tool_name -> tool_id への自動変換は複雑なため）
TRUNCATE TABLE mcpist.tool_settings;

-- カラム名変更
ALTER TABLE mcpist.tool_settings RENAME COLUMN tool_name TO tool_id;

-- コメント追加
COMMENT ON COLUMN mcpist.tool_settings.tool_id IS 'Tool ID in format: {module}:{tool_name} (e.g., notion:search)';

-- PRIMARY KEYの再作成（カラム名変更に伴い）
-- Note: PostgreSQLではRENAME COLUMNで自動的にPKも更新される

-- -----------------------------------------------------------------------------
-- 2. module_settings: description カラム追加
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.module_settings
    ADD COLUMN description TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN mcpist.module_settings.description IS 'User-defined additional description for the module';

-- -----------------------------------------------------------------------------
-- 3. RPC関数の更新
-- -----------------------------------------------------------------------------

-- get_tool_settings: tool_name -> tool_id
-- 戻り値型が変わるためDROPが必要
DROP FUNCTION IF EXISTS mcpist.get_tool_settings(UUID, TEXT);
DROP FUNCTION IF EXISTS public.get_tool_settings(UUID, TEXT);

CREATE FUNCTION mcpist.get_tool_settings(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
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
        ts.tool_id,
        ts.enabled
    FROM mcpist.tool_settings ts
    JOIN mcpist.modules m ON m.id = ts.module_id
    WHERE ts.user_id = p_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY m.name, ts.tool_id;
END;
$$;

-- public wrapper（既にDROP済み）
CREATE FUNCTION public.get_tool_settings(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_tool_settings(p_user_id, p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_tool_settings(UUID, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_tool_settings(UUID, TEXT) TO authenticated;

-- upsert_tool_settings: tool_name -> tool_id
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
    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    -- 有効ツールをUPSERT
    IF p_enabled_tools IS NOT NULL AND array_length(p_enabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT p_user_id, v_module_id, unnest(p_enabled_tools), true
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = true;
    END IF;

    -- 無効ツールをUPSERT
    IF p_disabled_tools IS NOT NULL AND array_length(p_disabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT p_user_id, v_module_id, unnest(p_disabled_tools), false
        ON CONFLICT (user_id, module_id, tool_id)
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

-- get_my_tool_settings: 戻り値型変更
DROP FUNCTION IF EXISTS mcpist.get_my_tool_settings(TEXT);
CREATE FUNCTION mcpist.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
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

-- Drop and recreate public wrapper with new signature
DROP FUNCTION IF EXISTS public.get_my_tool_settings(TEXT);
CREATE FUNCTION public.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
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
-- 4. モジュール説明取得/更新用RPC
-- -----------------------------------------------------------------------------

-- get_my_module_descriptions: ユーザーのモジュール説明を取得
CREATE OR REPLACE FUNCTION mcpist.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ms.description
    FROM mcpist.module_settings ms
    JOIN mcpist.modules m ON m.id = ms.module_id
    WHERE ms.user_id = auth.uid()
      AND ms.description IS NOT NULL
      AND ms.description != ''
    ORDER BY m.name;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_module_descriptions();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_module_descriptions() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_module_descriptions() TO authenticated;

-- upsert_my_module_description: モジュール説明を更新
CREATE OR REPLACE FUNCTION mcpist.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
BEGIN
    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    INSERT INTO mcpist.module_settings (user_id, module_id, enabled, description)
    VALUES (auth.uid(), v_module_id, true, p_description)
    ON CONFLICT (user_id, module_id)
    DO UPDATE SET description = p_description;

    RETURN jsonb_build_object('success', true, 'module', p_module_name);
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_module_description(p_module_name, p_description);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_module_description(TEXT, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_module_description(TEXT, TEXT) TO authenticated;
