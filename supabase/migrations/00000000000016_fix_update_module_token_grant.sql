-- Fix update_module_token RPC permissions
-- The function needs to be accessible via service_role key for token refresh

-- Grant execute permissions to service_role
GRANT EXECUTE ON FUNCTION mcpist.update_module_token(UUID, TEXT, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.update_module_token(UUID, TEXT, JSONB) TO service_role;

-- Grant UPDATE permission on vault.secrets to postgres
-- The SECURITY DEFINER function runs as postgres, but postgres doesn't have UPDATE permission by default
GRANT UPDATE ON vault.secrets TO postgres;
