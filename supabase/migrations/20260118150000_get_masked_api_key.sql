-- RPC function to get masked API Key
-- Returns first 6 + "****..." + last 6 characters

CREATE OR REPLACE FUNCTION mcpist.get_masked_api_key(p_service TEXT)
RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
  v_token TEXT;
  v_first TEXT;
  v_last TEXT;
  v_masked TEXT;
BEGIN
  -- Get token from vault for current user
  SELECT decrypted_secret INTO v_token
  FROM vault.decrypted_secrets ds
  JOIN mcpist.oauth_tokens ot ON ot.access_token_secret_id = ds.id
  WHERE ot.user_id = auth.uid()
    AND ot.service = p_service;

  IF v_token IS NULL THEN
    RETURN NULL;
  END IF;

  -- Create masked version: first 6 + "****..." + last 2
  IF LENGTH(v_token) <= 8 THEN
    RETURN v_token;
  END IF;

  v_first := LEFT(v_token, 6);
  v_last := RIGHT(v_token, 2);
  v_masked := v_first || '****...' || v_last;

  RETURN v_masked;
END;
$$;

-- Public wrapper
CREATE OR REPLACE FUNCTION public.get_masked_api_key(p_service TEXT)
RETURNS TEXT
LANGUAGE sql
SECURITY DEFINER
AS $$
  SELECT mcpist.get_masked_api_key(p_service);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_masked_api_key TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_masked_api_key TO authenticated;
