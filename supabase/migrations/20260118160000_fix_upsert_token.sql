-- Fix upsert_oauth_token to handle vault secret name uniqueness
-- Use timestamp in secret name to avoid conflicts

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
    v_timestamp TEXT;
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

    -- Generate timestamp for unique secret name
    v_timestamp := EXTRACT(EPOCH FROM NOW())::TEXT;

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
    -- Include timestamp to ensure unique name
    SELECT vault.create_secret(
        p_access_token,
        'oauth_access_' || p_service || '_' || v_user_id::TEXT || '_' || v_timestamp,
        'OAuth access token for ' || p_service
    ) INTO v_access_secret_id;

    -- Store refresh token in vault (if provided)
    IF p_refresh_token IS NOT NULL THEN
        SELECT vault.create_secret(
            p_refresh_token,
            'oauth_refresh_' || p_service || '_' || v_user_id::TEXT || '_' || v_timestamp,
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

-- Update public wrapper to match
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
