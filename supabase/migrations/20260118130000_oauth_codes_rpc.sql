-- RPC functions for OAuth Authorization Codes
-- These functions allow service_role to manage authorization codes

-- Store authorization code
CREATE OR REPLACE FUNCTION mcpist.store_oauth_code(
  p_code TEXT,
  p_user_id UUID,
  p_client_id TEXT,
  p_redirect_uri TEXT,
  p_code_challenge TEXT,
  p_code_challenge_method TEXT,
  p_scope TEXT,
  p_state TEXT,
  p_expires_at TIMESTAMPTZ
)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
  INSERT INTO mcpist.oauth_authorization_codes (
    code,
    user_id,
    client_id,
    redirect_uri,
    code_challenge,
    code_challenge_method,
    scope,
    state,
    expires_at
  ) VALUES (
    p_code,
    p_user_id,
    p_client_id,
    p_redirect_uri,
    p_code_challenge,
    p_code_challenge_method,
    p_scope,
    p_state,
    p_expires_at
  );
END;
$$;

-- Consume authorization code (get and mark as used)
CREATE OR REPLACE FUNCTION mcpist.consume_oauth_code(p_code TEXT)
RETURNS TABLE (
  code TEXT,
  user_id UUID,
  client_id TEXT,
  redirect_uri TEXT,
  code_challenge TEXT,
  code_challenge_method TEXT,
  scope TEXT,
  state TEXT,
  expires_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
  v_record RECORD;
BEGIN
  -- Get the code if valid
  SELECT * INTO v_record
  FROM mcpist.oauth_authorization_codes oc
  WHERE oc.code = p_code
    AND oc.used_at IS NULL
    AND oc.expires_at > now();

  IF NOT FOUND THEN
    RETURN;
  END IF;

  -- Mark as used
  UPDATE mcpist.oauth_authorization_codes
  SET used_at = now()
  WHERE oauth_authorization_codes.code = p_code;

  -- Return the record
  RETURN QUERY SELECT
    v_record.code,
    v_record.user_id,
    v_record.client_id,
    v_record.redirect_uri,
    v_record.code_challenge,
    v_record.code_challenge_method,
    v_record.scope,
    v_record.state,
    v_record.expires_at;
END;
$$;

-- Grant execute to service_role
GRANT EXECUTE ON FUNCTION mcpist.store_oauth_code TO service_role;
GRANT EXECUTE ON FUNCTION mcpist.consume_oauth_code TO service_role;

-- Public schema wrapper functions (for Supabase client compatibility)
-- Supabase JS client calls RPC on public schema by default

CREATE OR REPLACE FUNCTION public.store_oauth_code(
  p_code TEXT,
  p_user_id UUID,
  p_client_id TEXT,
  p_redirect_uri TEXT,
  p_code_challenge TEXT,
  p_code_challenge_method TEXT,
  p_scope TEXT,
  p_state TEXT,
  p_expires_at TIMESTAMPTZ
)
RETURNS VOID
LANGUAGE sql
SECURITY DEFINER
AS $$
  SELECT mcpist.store_oauth_code(
    p_code, p_user_id, p_client_id, p_redirect_uri,
    p_code_challenge, p_code_challenge_method, p_scope, p_state, p_expires_at
  );
$$;

CREATE OR REPLACE FUNCTION public.consume_oauth_code(p_code TEXT)
RETURNS TABLE (
  code TEXT,
  user_id UUID,
  client_id TEXT,
  redirect_uri TEXT,
  code_challenge TEXT,
  code_challenge_method TEXT,
  scope TEXT,
  state TEXT,
  expires_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
  SELECT * FROM mcpist.consume_oauth_code(p_code);
$$;

-- Grant execute to service_role on public wrappers
GRANT EXECUTE ON FUNCTION public.store_oauth_code TO service_role;
GRANT EXECUTE ON FUNCTION public.consume_oauth_code TO service_role;
