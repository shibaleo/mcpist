-- =============================================================================
-- Add description column to prompts table
-- =============================================================================
-- MCP仕様に合わせてdescriptionとcontentを分離:
-- - description: prompts/listで返される短い説明文
-- - content: prompts/getで返される実際のプロンプト内容
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Add description column to prompts table
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.prompts
ADD COLUMN IF NOT EXISTS description TEXT;

COMMENT ON COLUMN mcpist.prompts.description IS 'Short description shown in prompts/list (MCP spec)';

-- -----------------------------------------------------------------------------
-- 2. Update list_my_prompts to include description
-- Drop and recreate because return type is changing
-- -----------------------------------------------------------------------------

DROP FUNCTION IF EXISTS public.list_my_prompts(TEXT);
DROP FUNCTION IF EXISTS mcpist.list_my_prompts(TEXT);

CREATE OR REPLACE FUNCTION mcpist.list_my_prompts(p_module_name TEXT DEFAULT NULL)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    description TEXT,
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
        p.description,
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

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.list_my_prompts(p_module_name TEXT DEFAULT NULL)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    description TEXT,
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
-- 3. Update get_my_prompt to include description
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
        p.description,
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
        'description', v_prompt.description,
        'content', v_prompt.content,
        'enabled', v_prompt.enabled,
        'created_at', v_prompt.created_at,
        'updated_at', v_prompt.updated_at
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_prompt(p_prompt_id);
$$;

-- -----------------------------------------------------------------------------
-- 4. Update upsert_my_prompt to accept description
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_prompt(
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL,
    p_enabled BOOLEAN DEFAULT TRUE,
    p_description TEXT DEFAULT NULL
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

    -- Get module_id if module_name is provided
    IF p_module_name IS NOT NULL THEN
        SELECT id INTO v_module_id
        FROM mcpist.modules
        WHERE name = p_module_name;

        IF v_module_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'module_not_found'
            );
        END IF;
    END IF;

    IF p_prompt_id IS NOT NULL THEN
        -- Update existing prompt
        UPDATE mcpist.prompts
        SET
            name = p_name,
            description = p_description,
            content = p_content,
            module_id = v_module_id,
            enabled = p_enabled,
            updated_at = NOW()
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
        -- Create new prompt
        INSERT INTO mcpist.prompts (user_id, module_id, name, description, content, enabled)
        VALUES (v_user_id, v_module_id, p_name, p_description, p_content, p_enabled)
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

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.upsert_my_prompt(
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL,
    p_enabled BOOLEAN DEFAULT TRUE,
    p_description TEXT DEFAULT NULL
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_prompt(p_name, p_content, p_module_name, p_prompt_id, p_enabled, p_description);
$$;

-- -----------------------------------------------------------------------------
-- 5. Update list_user_prompts (server RPC) to return description
-- Drop and recreate because return type is changing
-- -----------------------------------------------------------------------------

DROP FUNCTION IF EXISTS public.list_user_prompts(UUID, BOOLEAN);
DROP FUNCTION IF EXISTS mcpist.list_user_prompts(UUID, BOOLEAN);

CREATE OR REPLACE FUNCTION mcpist.list_user_prompts(
    p_user_id UUID,
    p_enabled_only BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.id,
        p.name,
        p.description,
        p.content,
        p.enabled
    FROM mcpist.prompts p
    WHERE p.user_id = p_user_id
      AND (NOT p_enabled_only OR p.enabled = TRUE)
    ORDER BY p.updated_at DESC;
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.list_user_prompts(
    p_user_id UUID,
    p_enabled_only BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_user_prompts(p_user_id, p_enabled_only);
$$;

-- -----------------------------------------------------------------------------
-- 6. Update get_user_prompt_by_name (server RPC) to return description
-- Drop and recreate because return type is changing
-- -----------------------------------------------------------------------------

DROP FUNCTION IF EXISTS public.get_user_prompt_by_name(UUID, TEXT);
DROP FUNCTION IF EXISTS mcpist.get_user_prompt_by_name(UUID, TEXT);

CREATE OR REPLACE FUNCTION mcpist.get_user_prompt_by_name(
    p_user_id UUID,
    p_prompt_name TEXT
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.id,
        p.name,
        p.description,
        p.content,
        p.enabled
    FROM mcpist.prompts p
    WHERE p.user_id = p_user_id
      AND p.name = p_prompt_name;
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_user_prompt_by_name(
    p_user_id UUID,
    p_prompt_name TEXT
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_prompt_by_name(p_user_id, p_prompt_name);
$$;
