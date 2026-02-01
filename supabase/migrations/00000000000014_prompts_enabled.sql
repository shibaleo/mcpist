-- =============================================================================
-- Add enabled column to prompts table
-- =============================================================================
-- プロンプトの有効/無効を切り替えられるようにする
-- =============================================================================

-- enabled カラムを追加（デフォルト true）
ALTER TABLE mcpist.prompts ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;

-- -----------------------------------------------------------------------------
-- list_my_prompts を更新（enabled カラムを含める）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.list_my_prompts(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    content TEXT,
    enabled BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    RETURN QUERY
    SELECT
        p.id,
        m.name AS module_name,
        p.name,
        p.content,
        p.enabled,
        p.created_at,
        p.updated_at
    FROM mcpist.prompts p
    LEFT JOIN mcpist.modules m ON m.id = p.module_id
    WHERE p.user_id = v_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY p.updated_at DESC;
END;
$$;

CREATE OR REPLACE FUNCTION public.list_my_prompts(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    content TEXT,
    enabled BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_my_prompts(p_module_name);
$$;

-- -----------------------------------------------------------------------------
-- get_my_prompt を更新（enabled カラムを含める）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_prompt RECORD;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT
        p.id,
        m.name AS module_name,
        p.name,
        p.content,
        p.enabled,
        p.created_at,
        p.updated_at
    INTO v_prompt
    FROM mcpist.prompts p
    LEFT JOIN mcpist.modules m ON m.id = p.module_id
    WHERE p.id = p_prompt_id AND p.user_id = v_user_id;

    IF v_prompt IS NULL THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'prompt_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'found', true,
        'id', v_prompt.id,
        'module_name', v_prompt.module_name,
        'name', v_prompt.name,
        'content', v_prompt.content,
        'enabled', v_prompt.enabled,
        'created_at', v_prompt.created_at,
        'updated_at', v_prompt.updated_at
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_prompt(p_prompt_id);
$$;

-- -----------------------------------------------------------------------------
-- upsert_my_prompt を更新（enabled パラメータを追加）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_prompt(
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL,
    p_enabled BOOLEAN DEFAULT true
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_module_id UUID;
    v_result_id UUID;
    v_action TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- モジュール名からIDを取得（指定された場合）
    IF p_module_name IS NOT NULL THEN
        SELECT id INTO v_module_id
        FROM mcpist.modules
        WHERE name = p_module_name AND status IN ('active', 'beta');

        IF v_module_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'module_not_found'
            );
        END IF;
    END IF;

    -- 更新の場合
    IF p_prompt_id IS NOT NULL THEN
        UPDATE mcpist.prompts
        SET
            name = p_name,
            content = p_content,
            module_id = v_module_id,
            enabled = p_enabled
        WHERE id = p_prompt_id AND user_id = v_user_id
        RETURNING id INTO v_result_id;

        IF v_result_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'prompt_not_found'
            );
        END IF;

        v_action := 'updated';
    ELSE
        -- 新規作成
        INSERT INTO mcpist.prompts (user_id, module_id, name, content, enabled)
        VALUES (v_user_id, v_module_id, p_name, p_content, p_enabled)
        ON CONFLICT (user_id, module_id, name) DO UPDATE
        SET content = EXCLUDED.content, enabled = EXCLUDED.enabled
        RETURNING id INTO v_result_id;

        v_action := 'created';
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'id', v_result_id,
        'action', v_action
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_prompt(
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL,
    p_enabled BOOLEAN DEFAULT true
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_prompt(p_name, p_content, p_module_name, p_prompt_id, p_enabled);
$$;

-- 古い関数シグネチャを削除して新しいものに置き換え
DROP FUNCTION IF EXISTS mcpist.upsert_my_prompt(TEXT, TEXT, TEXT, UUID);
DROP FUNCTION IF EXISTS public.upsert_my_prompt(TEXT, TEXT, TEXT, UUID);

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_prompt(TEXT, TEXT, TEXT, UUID, BOOLEAN) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_prompt(TEXT, TEXT, TEXT, UUID, BOOLEAN) TO authenticated;
