-- Token Vault RPC Functions
-- Securely store and retrieve OAuth tokens using Supabase Vault

-- Function to store a token in vault and oauth_tokens table
CREATE OR REPLACE FUNCTION mcpist.upsert_oauth_token(
    p_service TEXT,
    p_access_token TEXT,
    p_refresh_token TEXT DEFAULT NULL,
    p_token_type TEXT DEFAULT 'Bearer',
    p_scope TEXT DEFAULT NULL,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_user_id UUID;
    v_token_id UUID;
    v_access_secret_id UUID;
    v_refresh_secret_id UUID;
    v_existing_token RECORD;
BEGIN
    -- Get the current authenticated user (mcpist.users.id = auth.users.id)
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify user exists in mcpist.users
    IF NOT EXISTS (SELECT 1 FROM mcpist.users u WHERE u.id = v_user_id) THEN
        RAISE EXCEPTION 'User not found in mcpist.users';
    END IF;

    -- Check for existing token
    SELECT * INTO v_existing_token
    FROM mcpist.oauth_tokens
    WHERE user_id = v_user_id AND service = p_service;

    -- If exists, delete old secrets from vault
    IF v_existing_token IS NOT NULL THEN
        IF v_existing_token.access_token_secret_id IS NOT NULL THEN
            DELETE FROM vault.secrets WHERE id = v_existing_token.access_token_secret_id;
        END IF;
        IF v_existing_token.refresh_token_secret_id IS NOT NULL THEN
            DELETE FROM vault.secrets WHERE id = v_existing_token.refresh_token_secret_id;
        END IF;
    END IF;

    -- Store access token in vault using vault.create_secret
    SELECT vault.create_secret(
        p_access_token,
        'oauth_access_' || p_service || '_' || v_user_id::TEXT,
        'OAuth access token for ' || p_service
    ) INTO v_access_secret_id;

    -- Store refresh token in vault (if provided)
    IF p_refresh_token IS NOT NULL THEN
        SELECT vault.create_secret(
            p_refresh_token,
            'oauth_refresh_' || p_service || '_' || v_user_id::TEXT,
            'OAuth refresh token for ' || p_service
        ) INTO v_refresh_secret_id;
    END IF;

    -- Upsert oauth_tokens record
    INSERT INTO mcpist.oauth_tokens (
        user_id,
        service,
        access_token_secret_id,
        refresh_token_secret_id,
        token_type,
        scope,
        expires_at
    )
    VALUES (
        v_user_id,
        p_service,
        v_access_secret_id,
        v_refresh_secret_id,
        p_token_type,
        p_scope,
        p_expires_at
    )
    ON CONFLICT (user_id, service)
    DO UPDATE SET
        access_token_secret_id = v_access_secret_id,
        refresh_token_secret_id = v_refresh_secret_id,
        token_type = p_token_type,
        scope = p_scope,
        expires_at = p_expires_at,
        updated_at = NOW()
    RETURNING id INTO v_token_id;

    RETURN v_token_id;
END;
$$;

-- Function to get user's connected services (without exposing tokens)
CREATE OR REPLACE FUNCTION mcpist.get_my_oauth_connections()
RETURNS TABLE (
    id UUID,
    service TEXT,
    token_type TEXT,
    scope TEXT,
    expires_at TIMESTAMPTZ,
    is_expired BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify user exists in mcpist.users
    IF NOT EXISTS (SELECT 1 FROM mcpist.users u WHERE u.id = v_user_id) THEN
        RAISE EXCEPTION 'User not found in mcpist.users';
    END IF;

    RETURN QUERY
    SELECT
        ot.id,
        ot.service,
        ot.token_type,
        ot.scope,
        ot.expires_at,
        CASE
            WHEN ot.expires_at IS NULL THEN FALSE
            WHEN ot.expires_at < NOW() THEN TRUE
            ELSE FALSE
        END AS is_expired,
        ot.created_at,
        ot.updated_at
    FROM mcpist.oauth_tokens ot
    WHERE ot.user_id = v_user_id
    ORDER BY ot.service;
END;
$$;

-- Function to delete a token
CREATE OR REPLACE FUNCTION mcpist.delete_oauth_token(p_service TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_user_id UUID;
    v_existing_token RECORD;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify user exists in mcpist.users
    IF NOT EXISTS (SELECT 1 FROM mcpist.users u WHERE u.id = v_user_id) THEN
        RAISE EXCEPTION 'User not found in mcpist.users';
    END IF;

    -- Get existing token
    SELECT * INTO v_existing_token
    FROM mcpist.oauth_tokens
    WHERE user_id = v_user_id AND service = p_service;

    IF v_existing_token IS NULL THEN
        RETURN FALSE;
    END IF;

    -- Delete secrets from vault
    IF v_existing_token.access_token_secret_id IS NOT NULL THEN
        DELETE FROM vault.secrets WHERE id = v_existing_token.access_token_secret_id;
    END IF;
    IF v_existing_token.refresh_token_secret_id IS NOT NULL THEN
        DELETE FROM vault.secrets WHERE id = v_existing_token.refresh_token_secret_id;
    END IF;

    -- Delete oauth_tokens record
    DELETE FROM mcpist.oauth_tokens WHERE id = v_existing_token.id;

    RETURN TRUE;
END;
$$;

-- Grant execute permissions to authenticated users
GRANT EXECUTE ON FUNCTION mcpist.upsert_oauth_token TO authenticated;
GRANT EXECUTE ON FUNCTION mcpist.get_my_oauth_connections TO authenticated;
GRANT EXECUTE ON FUNCTION mcpist.delete_oauth_token TO authenticated;

-- Public schema wrapper functions (for Supabase client compatibility)
-- These wrappers allow the Supabase JS client to call the functions without schema prefix

CREATE OR REPLACE FUNCTION public.upsert_oauth_token(
    p_service TEXT,
    p_access_token TEXT,
    p_refresh_token TEXT DEFAULT NULL,
    p_token_type TEXT DEFAULT 'Bearer',
    p_scope TEXT DEFAULT NULL,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
)
RETURNS UUID
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_oauth_token(p_service, p_access_token, p_refresh_token, p_token_type, p_scope, p_expires_at);
$$;

CREATE OR REPLACE FUNCTION public.get_my_oauth_connections()
RETURNS TABLE (
    id UUID,
    service TEXT,
    token_type TEXT,
    scope TEXT,
    expires_at TIMESTAMPTZ,
    is_expired BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_oauth_connections();
$$;

CREATE OR REPLACE FUNCTION public.delete_oauth_token(p_service TEXT)
RETURNS BOOLEAN
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_oauth_token(p_service);
$$;

-- Grant execute permissions on public wrappers
GRANT EXECUTE ON FUNCTION public.upsert_oauth_token TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_oauth_connections TO authenticated;
GRANT EXECUTE ON FUNCTION public.delete_oauth_token TO authenticated;
