-- Grant execute permission on validate_api_key to anon role
-- This allows the Worker to validate API keys without using service_role key

GRANT EXECUTE ON FUNCTION mcpist.validate_api_key TO anon;
GRANT EXECUTE ON FUNCTION public.validate_api_key TO anon;
