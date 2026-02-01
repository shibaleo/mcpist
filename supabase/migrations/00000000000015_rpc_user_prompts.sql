-- =============================================================================
-- MCPist RPC Functions for User Prompts (MCP Server)
-- =============================================================================
-- This migration creates RPC functions for MCP Server to access user prompts:
-- 1. list_user_prompts - ユーザーの有効なプロンプト一覧を取得
-- 2. get_user_prompt_by_name - プロンプト名でプロンプトを取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- list_user_prompts
-- サーバーからユーザーIDを指定してプロンプト一覧を取得
-- p_enabled_only: trueの場合、有効なプロンプトのみ返す
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.list_user_prompts(
    p_user_id UUID,
    p_enabled_only BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
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
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_user_prompts(p_user_id, p_enabled_only);
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_user_prompts(UUID, BOOLEAN) TO service_role;
GRANT EXECUTE ON FUNCTION public.list_user_prompts(UUID, BOOLEAN) TO service_role;

-- -----------------------------------------------------------------------------
-- get_user_prompt_by_name
-- サーバーからユーザーIDとプロンプト名を指定してプロンプトを取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_user_prompt_by_name(
    p_user_id UUID,
    p_prompt_name TEXT
)
RETURNS TABLE (
    id UUID,
    name TEXT,
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
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_prompt_by_name(p_user_id, p_prompt_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_prompt_by_name(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_prompt_by_name(UUID, TEXT) TO service_role;
