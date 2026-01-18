-- Token Vault: Get Token RPC Function
-- For MCP Server to retrieve decrypted tokens

-- Function to get decrypted token for a service (for service role / server-side use)
CREATE OR REPLACE FUNCTION mcpist.get_service_token(
    p_user_id UUID,
    p_service TEXT
)
RETURNS TABLE (
    oauth_token TEXT,
    long_term_token TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_token_record RECORD;
    v_oauth_token TEXT;
    v_long_term_token TEXT;
BEGIN
    -- Get the token record
    SELECT
        ot.access_token_secret_id,
        ot.refresh_token_secret_id,
        ot.expires_at
    INTO v_token_record
    FROM mcpist.oauth_tokens ot
    WHERE ot.user_id = p_user_id AND ot.service = p_service;

    IF v_token_record IS NULL THEN
        -- No token found, return empty
        RETURN QUERY SELECT NULL::TEXT, NULL::TEXT;
        RETURN;
    END IF;

    -- Decrypt access token from vault
    IF v_token_record.access_token_secret_id IS NOT NULL THEN
        SELECT decrypted_secret INTO v_oauth_token
        FROM vault.decrypted_secrets
        WHERE id = v_token_record.access_token_secret_id;
    END IF;

    -- For now, long_term_token is same as oauth_token
    -- In future, may support separate long-term tokens
    v_long_term_token := v_oauth_token;

    RETURN QUERY SELECT v_oauth_token, v_long_term_token;
END;
$$;

-- Public schema wrapper (for API access)
CREATE OR REPLACE FUNCTION public.get_service_token(
    p_user_id UUID,
    p_service TEXT
)
RETURNS TABLE (
    oauth_token TEXT,
    long_term_token TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_service_token(p_user_id, p_service);
$$;

-- Grant execute to service_role only (not authenticated users)
-- This prevents users from accessing other users' tokens directly
GRANT EXECUTE ON FUNCTION mcpist.get_service_token TO service_role;
GRANT EXECUTE ON FUNCTION public.get_service_token TO service_role;
