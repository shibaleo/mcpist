-- OAuth Authorization Codes Table
-- Stores temporary authorization codes for OAuth 2.1 PKCE flow

CREATE TABLE mcpist.oauth_authorization_codes (
  code TEXT PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  client_id TEXT NOT NULL,
  redirect_uri TEXT NOT NULL,
  code_challenge TEXT NOT NULL,
  code_challenge_method TEXT NOT NULL DEFAULT 'S256',
  scope TEXT,
  state TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for cleanup of expired codes
CREATE INDEX idx_oauth_codes_expires_at ON mcpist.oauth_authorization_codes(expires_at);

-- Index for user lookup
CREATE INDEX idx_oauth_codes_user_id ON mcpist.oauth_authorization_codes(user_id);

-- RLS Policy: Only service_role can access (internal use only)
ALTER TABLE mcpist.oauth_authorization_codes ENABLE ROW LEVEL SECURITY;

-- No policies for anon/authenticated - this table is service_role only
-- Service role bypasses RLS by default

-- Function to clean up expired codes (run via cron)
CREATE OR REPLACE FUNCTION mcpist.cleanup_expired_oauth_codes()
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
  deleted_count INTEGER;
BEGIN
  DELETE FROM mcpist.oauth_authorization_codes
  WHERE expires_at < now() OR used_at IS NOT NULL;

  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RETURN deleted_count;
END;
$$;

-- Grant execute to service_role
GRANT EXECUTE ON FUNCTION mcpist.cleanup_expired_oauth_codes() TO service_role;
