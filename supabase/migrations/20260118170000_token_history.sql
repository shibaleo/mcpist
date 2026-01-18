-- Token History Table for Audit Trail
-- Records who used which secret from when to when

CREATE TABLE IF NOT EXISTS mcpist.oauth_token_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    -- Vault secret IDs (kept for reference even after deletion)
    access_token_secret_id UUID,
    refresh_token_secret_id UUID,
    token_type TEXT DEFAULT 'Bearer',
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ,  -- When this token was rotated/revoked
    expired_reason TEXT,     -- 'rotated', 'revoked', 'expired'
    -- Metadata
    created_by_ip TEXT,
    expired_by_ip TEXT
);

-- Index for efficient queries
CREATE INDEX IF NOT EXISTS idx_oauth_token_history_user_service
    ON mcpist.oauth_token_history(user_id, service);
CREATE INDEX IF NOT EXISTS idx_oauth_token_history_created_at
    ON mcpist.oauth_token_history(created_at DESC);

-- Update upsert_oauth_token to record history
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
    -- Get the current authenticated user
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

    -- If exists, record history and delete old secrets
    IF v_existing_token IS NOT NULL THEN
        -- Record in history before deletion
        INSERT INTO mcpist.oauth_token_history (
            user_id,
            service,
            access_token_secret_id,
            refresh_token_secret_id,
            token_type,
            created_at,
            expired_at,
            expired_reason
        ) VALUES (
            v_user_id,
            p_service,
            v_existing_token.access_token_secret_id,
            v_existing_token.refresh_token_secret_id,
            v_existing_token.token_type,
            v_existing_token.created_at,
            NOW(),
            'rotated'
        );

        -- Delete old secrets from vault
        IF v_existing_token.access_token_secret_id IS NOT NULL THEN
            DELETE FROM vault.secrets WHERE id = v_existing_token.access_token_secret_id;
        END IF;
        IF v_existing_token.refresh_token_secret_id IS NOT NULL THEN
            DELETE FROM vault.secrets WHERE id = v_existing_token.refresh_token_secret_id;
        END IF;
    END IF;

    -- Store access token in vault
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

-- Update delete_oauth_token to record history
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

    -- Record in history before deletion
    INSERT INTO mcpist.oauth_token_history (
        user_id,
        service,
        access_token_secret_id,
        refresh_token_secret_id,
        token_type,
        created_at,
        expired_at,
        expired_reason
    ) VALUES (
        v_user_id,
        p_service,
        v_existing_token.access_token_secret_id,
        v_existing_token.refresh_token_secret_id,
        v_existing_token.token_type,
        v_existing_token.created_at,
        NOW(),
        'revoked'
    );

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

-- RPC to get token history for current user
CREATE OR REPLACE FUNCTION mcpist.get_my_token_history(p_service TEXT DEFAULT NULL)
RETURNS TABLE (
    id UUID,
    service TEXT,
    token_type TEXT,
    created_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    expired_reason TEXT
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

    RETURN QUERY
    SELECT
        h.id,
        h.service,
        h.token_type,
        h.created_at,
        h.expired_at,
        h.expired_reason
    FROM mcpist.oauth_token_history h
    WHERE h.user_id = v_user_id
      AND (p_service IS NULL OR h.service = p_service)
    ORDER BY h.created_at DESC
    LIMIT 100;
END;
$$;

-- Public wrapper
CREATE OR REPLACE FUNCTION public.get_my_token_history(p_service TEXT DEFAULT NULL)
RETURNS TABLE (
    id UUID,
    service TEXT,
    token_type TEXT,
    created_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    expired_reason TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_token_history(p_service);
$$;

-- Grant permissions
GRANT EXECUTE ON FUNCTION mcpist.get_my_token_history TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_token_history TO authenticated;

-- Update public wrapper for upsert
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

-- Update public wrapper for delete
CREATE OR REPLACE FUNCTION public.delete_oauth_token(p_service TEXT)
RETURNS BOOLEAN
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_oauth_token(p_service);
$$;
