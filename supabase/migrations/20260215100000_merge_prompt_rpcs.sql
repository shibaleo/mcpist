-- Merge list_user_prompts + get_user_prompt_by_name into a single RPC: get_user_prompts
-- When p_prompt_name is NULL: returns all prompts (replaces list_user_prompts)
-- When p_prompt_name is specified: returns single matching prompt (replaces get_user_prompt_by_name)

-- Drop old functions
DROP FUNCTION IF EXISTS public.list_user_prompts(UUID, BOOLEAN);
DROP FUNCTION IF EXISTS mcpist.list_user_prompts(UUID, BOOLEAN);
DROP FUNCTION IF EXISTS public.get_user_prompt_by_name(UUID, TEXT);
DROP FUNCTION IF EXISTS mcpist.get_user_prompt_by_name(UUID, TEXT);

-- Unified function
CREATE OR REPLACE FUNCTION mcpist.get_user_prompts(
    p_user_id UUID,
    p_prompt_name TEXT DEFAULT NULL,
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
      AND (p_prompt_name IS NULL OR p.name = p_prompt_name)
      AND (NOT p_enabled_only OR p.enabled = TRUE)
    ORDER BY p.updated_at DESC;
END;
$$;

-- Public schema wrapper
CREATE OR REPLACE FUNCTION public.get_user_prompts(
    p_user_id UUID,
    p_prompt_name TEXT DEFAULT NULL,
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
    SELECT * FROM mcpist.get_user_prompts(p_user_id, p_prompt_name, p_enabled_only);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_prompts(UUID, TEXT, BOOLEAN) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_prompts(UUID, TEXT, BOOLEAN) TO service_role;
