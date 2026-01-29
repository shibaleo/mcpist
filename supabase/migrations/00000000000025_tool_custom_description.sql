-- =============================================================================
-- Module Custom Description - Go Server RPC
-- =============================================================================
-- Add RPC for Go server to fetch module descriptions by user_id
-- (module_settings.description already exists from migration #20)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- get_module_descriptions RPC (for service_role / Go server)
-- Get module descriptions for a specific user
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_module_descriptions(p_user_id UUID)
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
    WHERE ms.user_id = p_user_id
      AND ms.description IS NOT NULL
      AND ms.description != ''
    ORDER BY m.name;
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_module_descriptions(p_user_id UUID)
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_module_descriptions(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_module_descriptions(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_module_descriptions(UUID) TO service_role;
