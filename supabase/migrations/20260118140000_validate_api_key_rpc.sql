-- RPC function to validate API Key
-- Called by Go Server to authenticate API Key tokens
-- API Key format: mpt_<32 hex chars>

-- Function to validate API key by comparing with stored value in vault
CREATE OR REPLACE FUNCTION mcpist.validate_api_key(
  p_api_key TEXT,
  p_service TEXT
)
RETURNS TABLE (user_id UUID)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
  v_record RECORD;
  v_stored_key TEXT;
BEGIN
  -- Find all tokens for the given service
  FOR v_record IN
    SELECT ot.user_id, ot.access_token_secret_id
    FROM mcpist.oauth_tokens ot
    WHERE ot.service = p_service
      AND ot.access_token_secret_id IS NOT NULL
  LOOP
    -- Get the decrypted token from vault
    SELECT decrypted_secret INTO v_stored_key
    FROM vault.decrypted_secrets
    WHERE id = v_record.access_token_secret_id;

    -- Compare with provided API key
    IF v_stored_key = p_api_key THEN
      RETURN QUERY SELECT v_record.user_id;
      RETURN;
    END IF;
  END LOOP;

  -- No match found
  RETURN;
END;
$$;

-- Grant execute to service_role
GRANT EXECUTE ON FUNCTION mcpist.validate_api_key TO service_role;

-- Public schema wrapper
CREATE OR REPLACE FUNCTION public.validate_api_key(
  p_api_key TEXT,
  p_service TEXT
)
RETURNS TABLE (user_id UUID)
LANGUAGE sql
SECURITY DEFINER
AS $$
  SELECT * FROM mcpist.validate_api_key(p_api_key, p_service);
$$;

GRANT EXECUTE ON FUNCTION public.validate_api_key TO service_role;
