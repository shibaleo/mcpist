-- =============================================================================
-- OAuth Authorization Requests Table and RPCs
-- =============================================================================
-- This migration adds support for the Supabase OAuth Server-compatible
-- authorization flow using authorization_id.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- OAuth Authorization Requests Table
-- Stores pending authorization requests (before user consent)
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.oauth_authorization_requests (
    id TEXT PRIMARY KEY,  -- authorization_id
    client_id TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    scope TEXT NOT NULL DEFAULT 'openid profile email',
    state TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    user_id UUID,  -- Set when approved
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_auth_requests_expires_at ON mcpist.oauth_authorization_requests(expires_at);
CREATE INDEX idx_oauth_auth_requests_status ON mcpist.oauth_authorization_requests(status);

-- -----------------------------------------------------------------------------
-- Store OAuth Authorization Request
-- Called by OAuth Server when /authorize is invoked
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.store_oauth_authorization_request(
    p_id TEXT,
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
    INSERT INTO mcpist.oauth_authorization_requests (
        id, client_id, redirect_uri, code_challenge,
        code_challenge_method, scope, state, expires_at
    ) VALUES (
        p_id, p_client_id, p_redirect_uri, p_code_challenge,
        p_code_challenge_method, p_scope, p_state, p_expires_at
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.store_oauth_authorization_request(
    p_id TEXT,
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
    SELECT mcpist.store_oauth_authorization_request(
        p_id, p_client_id, p_redirect_uri, p_code_challenge,
        p_code_challenge_method, p_scope, p_state, p_expires_at
    );
$$;

GRANT EXECUTE ON FUNCTION mcpist.store_oauth_authorization_request TO service_role;
GRANT EXECUTE ON FUNCTION public.store_oauth_authorization_request TO service_role;

-- -----------------------------------------------------------------------------
-- Get OAuth Authorization Request
-- Called by consent page to display authorization details
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_oauth_authorization_request(p_id TEXT)
RETURNS TABLE (
    id TEXT,
    client_id TEXT,
    redirect_uri TEXT,
    code_challenge TEXT,
    code_challenge_method TEXT,
    scope TEXT,
    state TEXT,
    status TEXT,
    created_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT
        r.id,
        r.client_id,
        r.redirect_uri,
        r.code_challenge,
        r.code_challenge_method,
        r.scope,
        r.state,
        r.status,
        r.created_at,
        r.expires_at
    FROM mcpist.oauth_authorization_requests r
    WHERE r.id = p_id
      AND r.status = 'pending'
      AND r.expires_at > now();
END;
$$;

CREATE OR REPLACE FUNCTION public.get_oauth_authorization_request(p_id TEXT)
RETURNS TABLE (
    id TEXT,
    client_id TEXT,
    redirect_uri TEXT,
    code_challenge TEXT,
    code_challenge_method TEXT,
    scope TEXT,
    state TEXT,
    status TEXT,
    created_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_oauth_authorization_request(p_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_oauth_authorization_request TO service_role;
GRANT EXECUTE ON FUNCTION public.get_oauth_authorization_request TO service_role;

-- -----------------------------------------------------------------------------
-- Approve OAuth Authorization
-- Called when user approves the authorization request
-- Generates authorization code and returns redirect info
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.approve_oauth_authorization(
    p_authorization_id TEXT,
    p_user_id UUID
)
RETURNS TABLE (
    code TEXT,
    redirect_uri TEXT,
    state TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_request RECORD;
    v_code TEXT;
    v_expires_at TIMESTAMPTZ;
BEGIN
    -- Get and lock the request
    SELECT * INTO v_request
    FROM mcpist.oauth_authorization_requests r
    WHERE r.id = p_authorization_id
      AND r.status = 'pending'
      AND r.expires_at > now()
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Generate authorization code (64 hex chars)
    v_code := encode(gen_random_bytes(32), 'hex');

    -- Code expires in 10 minutes
    v_expires_at := now() + interval '10 minutes';

    -- Mark request as approved
    UPDATE mcpist.oauth_authorization_requests
    SET status = 'approved', user_id = p_user_id
    WHERE id = p_authorization_id;

    -- Store the authorization code
    INSERT INTO mcpist.oauth_authorization_codes (
        code, user_id, client_id, redirect_uri, code_challenge,
        code_challenge_method, scope, state, expires_at
    ) VALUES (
        v_code, p_user_id, v_request.client_id, v_request.redirect_uri,
        v_request.code_challenge, v_request.code_challenge_method,
        v_request.scope, v_request.state, v_expires_at
    );

    RETURN QUERY SELECT v_code, v_request.redirect_uri, v_request.state;
END;
$$;

CREATE OR REPLACE FUNCTION public.approve_oauth_authorization(
    p_authorization_id TEXT,
    p_user_id UUID
)
RETURNS TABLE (
    code TEXT,
    redirect_uri TEXT,
    state TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.approve_oauth_authorization(p_authorization_id, p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.approve_oauth_authorization TO service_role;
GRANT EXECUTE ON FUNCTION public.approve_oauth_authorization TO service_role;

-- -----------------------------------------------------------------------------
-- Deny OAuth Authorization
-- Called when user denies the authorization request
-- Returns redirect info for error redirect
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.deny_oauth_authorization(p_authorization_id TEXT)
RETURNS TABLE (
    redirect_uri TEXT,
    state TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_request RECORD;
BEGIN
    -- Get and lock the request
    SELECT * INTO v_request
    FROM mcpist.oauth_authorization_requests r
    WHERE r.id = p_authorization_id
      AND r.status = 'pending'
      AND r.expires_at > now()
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Mark request as denied
    UPDATE mcpist.oauth_authorization_requests
    SET status = 'denied'
    WHERE id = p_authorization_id;

    RETURN QUERY SELECT v_request.redirect_uri, v_request.state;
END;
$$;

CREATE OR REPLACE FUNCTION public.deny_oauth_authorization(p_authorization_id TEXT)
RETURNS TABLE (
    redirect_uri TEXT,
    state TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.deny_oauth_authorization(p_authorization_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.deny_oauth_authorization TO service_role;
GRANT EXECUTE ON FUNCTION public.deny_oauth_authorization TO service_role;

-- -----------------------------------------------------------------------------
-- Cleanup Expired Authorization Requests
-- Should be called periodically to clean up old requests
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.cleanup_expired_oauth_authorization_requests()
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Mark expired requests
    UPDATE mcpist.oauth_authorization_requests
    SET status = 'expired'
    WHERE status = 'pending' AND expires_at < now();

    -- Delete old requests (older than 1 day)
    DELETE FROM mcpist.oauth_authorization_requests
    WHERE created_at < now() - interval '1 day';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$;

GRANT EXECUTE ON FUNCTION mcpist.cleanup_expired_oauth_authorization_requests() TO service_role;
