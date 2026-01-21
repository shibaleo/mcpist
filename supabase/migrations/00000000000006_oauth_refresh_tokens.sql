-- =============================================================================
-- OAuth Refresh Tokens Table and RPCs
-- =============================================================================
-- This migration adds support for OAuth 2.1 refresh tokens.
-- Refresh tokens are rotated on each use (one-time use).
-- =============================================================================

-- -----------------------------------------------------------------------------
-- OAuth Refresh Tokens Table
-- Stores refresh tokens for token refresh flow
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.oauth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    scope TEXT,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_refresh_tokens_token ON mcpist.oauth_refresh_tokens(token);
CREATE INDEX idx_oauth_refresh_tokens_user_id ON mcpist.oauth_refresh_tokens(user_id);
CREATE INDEX idx_oauth_refresh_tokens_expires_at ON mcpist.oauth_refresh_tokens(expires_at);

-- -----------------------------------------------------------------------------
-- Store OAuth Refresh Token
-- Called when issuing a new refresh token
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.store_oauth_refresh_token(
    p_token TEXT,
    p_user_id UUID,
    p_client_id TEXT,
    p_scope TEXT,
    p_expires_at TIMESTAMPTZ
)
RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    INSERT INTO mcpist.oauth_refresh_tokens (
        token, user_id, client_id, scope, expires_at
    ) VALUES (
        p_token, p_user_id, p_client_id, p_scope, p_expires_at
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.store_oauth_refresh_token(
    p_token TEXT,
    p_user_id UUID,
    p_client_id TEXT,
    p_scope TEXT,
    p_expires_at TIMESTAMPTZ
)
RETURNS VOID
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.store_oauth_refresh_token(
        p_token, p_user_id, p_client_id, p_scope, p_expires_at
    );
$$;

GRANT EXECUTE ON FUNCTION mcpist.store_oauth_refresh_token TO service_role;
GRANT EXECUTE ON FUNCTION public.store_oauth_refresh_token TO service_role;

-- -----------------------------------------------------------------------------
-- Consume OAuth Refresh Token
-- Validates and consumes (marks as used) a refresh token
-- Returns token data if valid, empty if invalid/expired/already used
-- Implements token rotation: each refresh token can only be used once
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.consume_oauth_refresh_token(p_token TEXT)
RETURNS TABLE (
    token TEXT,
    user_id UUID,
    client_id TEXT,
    scope TEXT,
    expires_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_token RECORD;
BEGIN
    -- Get and lock the token for update
    SELECT * INTO v_token
    FROM mcpist.oauth_refresh_tokens t
    WHERE t.token = p_token
      AND t.used = FALSE
      AND t.expires_at > now()
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Mark token as used (token rotation)
    UPDATE mcpist.oauth_refresh_tokens
    SET used = TRUE
    WHERE mcpist.oauth_refresh_tokens.token = p_token;

    RETURN QUERY SELECT
        v_token.token,
        v_token.user_id,
        v_token.client_id,
        v_token.scope,
        v_token.expires_at;
END;
$$;

CREATE OR REPLACE FUNCTION public.consume_oauth_refresh_token(p_token TEXT)
RETURNS TABLE (
    token TEXT,
    user_id UUID,
    client_id TEXT,
    scope TEXT,
    expires_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.consume_oauth_refresh_token(p_token);
$$;

GRANT EXECUTE ON FUNCTION mcpist.consume_oauth_refresh_token TO service_role;
GRANT EXECUTE ON FUNCTION public.consume_oauth_refresh_token TO service_role;

-- -----------------------------------------------------------------------------
-- Revoke OAuth Refresh Tokens for User
-- Called when user explicitly revokes access or changes password
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.revoke_oauth_refresh_tokens(
    p_user_id UUID,
    p_client_id TEXT DEFAULT NULL
)
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    IF p_client_id IS NOT NULL THEN
        -- Revoke tokens for specific client
        DELETE FROM mcpist.oauth_refresh_tokens
        WHERE user_id = p_user_id AND client_id = p_client_id;
    ELSE
        -- Revoke all tokens for user
        DELETE FROM mcpist.oauth_refresh_tokens
        WHERE user_id = p_user_id;
    END IF;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$;

CREATE OR REPLACE FUNCTION public.revoke_oauth_refresh_tokens(
    p_user_id UUID,
    p_client_id TEXT DEFAULT NULL
)
RETURNS INTEGER
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.revoke_oauth_refresh_tokens(p_user_id, p_client_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.revoke_oauth_refresh_tokens TO service_role;
GRANT EXECUTE ON FUNCTION public.revoke_oauth_refresh_tokens TO service_role;

-- -----------------------------------------------------------------------------
-- Cleanup Expired/Used Refresh Tokens
-- Should be called periodically to clean up old tokens
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.cleanup_expired_oauth_refresh_tokens()
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired or used tokens (keep used tokens for 1 day for audit)
    DELETE FROM mcpist.oauth_refresh_tokens
    WHERE expires_at < now()
       OR (used = TRUE AND created_at < now() - interval '1 day');

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$;

GRANT EXECUTE ON FUNCTION mcpist.cleanup_expired_oauth_refresh_tokens() TO service_role;
