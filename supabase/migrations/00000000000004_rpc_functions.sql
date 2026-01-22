-- =============================================================================
-- MCPist RPC Functions
-- =============================================================================
-- This migration creates all RPC functions:
-- 1. Token Vault RPCs (upsert, get connections, delete, get masked, get service token)
-- 2. OAuth Authorization Codes RPCs
-- 3. API Key Validation RPC
-- 4. Entitlement RPCs (get user entitlement, increment usage, deduct credits, get tool cost)
-- =============================================================================

-- =============================================================================
-- Token Vault RPCs
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Upsert OAuth Token (with history tracking)
-- -----------------------------------------------------------------------------

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
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM mcpist.users u WHERE u.id = v_user_id) THEN
        RAISE EXCEPTION 'User not found in mcpist.users';
    END IF;

    v_timestamp := EXTRACT(EPOCH FROM NOW())::TEXT;

    SELECT * INTO v_existing_token
    FROM mcpist.oauth_tokens
    WHERE user_id = v_user_id AND service = p_service;

    -- Record history and delete old secrets if exists
    IF v_existing_token IS NOT NULL THEN
        INSERT INTO mcpist.oauth_token_history (
            user_id, service, access_token_secret_id, refresh_token_secret_id,
            token_type, created_at, expired_at, expired_reason
        ) VALUES (
            v_user_id, p_service, v_existing_token.access_token_secret_id,
            v_existing_token.refresh_token_secret_id, v_existing_token.token_type,
            v_existing_token.created_at, NOW(), 'rotated'
        );

        IF v_existing_token.access_token_secret_id IS NOT NULL THEN
            DELETE FROM vault.secrets WHERE id = v_existing_token.access_token_secret_id;
        END IF;
        IF v_existing_token.refresh_token_secret_id IS NOT NULL THEN
            DELETE FROM vault.secrets WHERE id = v_existing_token.refresh_token_secret_id;
        END IF;
    END IF;

    SELECT vault.create_secret(
        p_access_token,
        'oauth_access_' || p_service || '_' || v_user_id::TEXT || '_' || v_timestamp,
        'OAuth access token for ' || p_service
    ) INTO v_access_secret_id;

    IF p_refresh_token IS NOT NULL THEN
        SELECT vault.create_secret(
            p_refresh_token,
            'oauth_refresh_' || p_service || '_' || v_user_id::TEXT || '_' || v_timestamp,
            'OAuth refresh token for ' || p_service
        ) INTO v_refresh_secret_id;
    END IF;

    INSERT INTO mcpist.oauth_tokens (
        user_id, service, access_token_secret_id, refresh_token_secret_id,
        token_type, scope, expires_at
    ) VALUES (
        v_user_id, p_service, v_access_secret_id, v_refresh_secret_id,
        p_token_type, p_scope, p_expires_at
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

GRANT EXECUTE ON FUNCTION mcpist.upsert_oauth_token TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_oauth_token TO authenticated;

-- -----------------------------------------------------------------------------
-- Get My OAuth Connections
-- -----------------------------------------------------------------------------

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

GRANT EXECUTE ON FUNCTION mcpist.get_my_oauth_connections TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_oauth_connections TO authenticated;

-- -----------------------------------------------------------------------------
-- Delete OAuth Token (with history tracking)
-- -----------------------------------------------------------------------------

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

    IF NOT EXISTS (SELECT 1 FROM mcpist.users u WHERE u.id = v_user_id) THEN
        RAISE EXCEPTION 'User not found in mcpist.users';
    END IF;

    SELECT * INTO v_existing_token
    FROM mcpist.oauth_tokens
    WHERE user_id = v_user_id AND service = p_service;

    IF v_existing_token IS NULL THEN
        RETURN FALSE;
    END IF;

    -- Record in history before deletion
    INSERT INTO mcpist.oauth_token_history (
        user_id, service, access_token_secret_id, refresh_token_secret_id,
        token_type, created_at, expired_at, expired_reason
    ) VALUES (
        v_user_id, p_service, v_existing_token.access_token_secret_id,
        v_existing_token.refresh_token_secret_id, v_existing_token.token_type,
        v_existing_token.created_at, NOW(), 'revoked'
    );

    IF v_existing_token.access_token_secret_id IS NOT NULL THEN
        DELETE FROM vault.secrets WHERE id = v_existing_token.access_token_secret_id;
    END IF;
    IF v_existing_token.refresh_token_secret_id IS NOT NULL THEN
        DELETE FROM vault.secrets WHERE id = v_existing_token.refresh_token_secret_id;
    END IF;

    DELETE FROM mcpist.oauth_tokens WHERE id = v_existing_token.id;

    RETURN TRUE;
END;
$$;

CREATE OR REPLACE FUNCTION public.delete_oauth_token(p_service TEXT)
RETURNS BOOLEAN
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_oauth_token(p_service);
$$;

GRANT EXECUTE ON FUNCTION mcpist.delete_oauth_token TO authenticated;
GRANT EXECUTE ON FUNCTION public.delete_oauth_token TO authenticated;

-- -----------------------------------------------------------------------------
-- Get Masked API Key
-- -----------------------------------------------------------------------------

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
    SELECT decrypted_secret INTO v_token
    FROM vault.decrypted_secrets ds
    JOIN mcpist.oauth_tokens ot ON ot.access_token_secret_id = ds.id
    WHERE ot.user_id = auth.uid()
      AND ot.service = p_service;

    IF v_token IS NULL THEN
        RETURN NULL;
    END IF;

    IF LENGTH(v_token) <= 8 THEN
        RETURN v_token;
    END IF;

    v_first := LEFT(v_token, 6);
    v_last := RIGHT(v_token, 2);
    v_masked := v_first || '****...' || v_last;

    RETURN v_masked;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_masked_api_key(p_service TEXT)
RETURNS TEXT
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_masked_api_key(p_service);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_masked_api_key TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_masked_api_key TO authenticated;

-- -----------------------------------------------------------------------------
-- Get My Token History
-- -----------------------------------------------------------------------------

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

GRANT EXECUTE ON FUNCTION mcpist.get_my_token_history TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_token_history TO authenticated;

-- -----------------------------------------------------------------------------
-- Get Service Token (for MCP Server)
-- -----------------------------------------------------------------------------

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
    SELECT
        ot.access_token_secret_id,
        ot.refresh_token_secret_id,
        ot.expires_at
    INTO v_token_record
    FROM mcpist.oauth_tokens ot
    WHERE ot.user_id = p_user_id AND ot.service = p_service;

    IF v_token_record IS NULL THEN
        RETURN QUERY SELECT NULL::TEXT, NULL::TEXT;
        RETURN;
    END IF;

    IF v_token_record.access_token_secret_id IS NOT NULL THEN
        SELECT decrypted_secret INTO v_oauth_token
        FROM vault.decrypted_secrets
        WHERE id = v_token_record.access_token_secret_id;
    END IF;

    v_long_term_token := v_oauth_token;

    RETURN QUERY SELECT v_oauth_token, v_long_term_token;
END;
$$;

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

GRANT EXECUTE ON FUNCTION mcpist.get_service_token TO service_role;
GRANT EXECUTE ON FUNCTION public.get_service_token TO service_role;

-- =============================================================================
-- OAuth Authorization Codes RPCs
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Store OAuth Code
-- -----------------------------------------------------------------------------

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
        code, user_id, client_id, redirect_uri, code_challenge,
        code_challenge_method, scope, state, expires_at
    ) VALUES (
        p_code, p_user_id, p_client_id, p_redirect_uri, p_code_challenge,
        p_code_challenge_method, p_scope, p_state, p_expires_at
    );
END;
$$;

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

GRANT EXECUTE ON FUNCTION mcpist.store_oauth_code TO service_role;
GRANT EXECUTE ON FUNCTION public.store_oauth_code TO service_role;

-- -----------------------------------------------------------------------------
-- Consume OAuth Code
-- -----------------------------------------------------------------------------

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
    SELECT * INTO v_record
    FROM mcpist.oauth_authorization_codes oc
    WHERE oc.code = p_code
      AND oc.used_at IS NULL
      AND oc.expires_at > now();

    IF NOT FOUND THEN
        RETURN;
    END IF;

    UPDATE mcpist.oauth_authorization_codes
    SET used_at = now()
    WHERE oauth_authorization_codes.code = p_code;

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

GRANT EXECUTE ON FUNCTION mcpist.consume_oauth_code TO service_role;
GRANT EXECUTE ON FUNCTION public.consume_oauth_code TO service_role;

-- -----------------------------------------------------------------------------
-- Cleanup Expired OAuth Codes
-- -----------------------------------------------------------------------------

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

GRANT EXECUTE ON FUNCTION mcpist.cleanup_expired_oauth_codes() TO service_role;

-- =============================================================================
-- API Key Validation RPC
-- =============================================================================

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
    FOR v_record IN
        SELECT ot.user_id, ot.access_token_secret_id
        FROM mcpist.oauth_tokens ot
        WHERE ot.service = p_service
          AND ot.access_token_secret_id IS NOT NULL
    LOOP
        SELECT decrypted_secret INTO v_stored_key
        FROM vault.decrypted_secrets
        WHERE id = v_record.access_token_secret_id;

        IF v_stored_key = p_api_key THEN
            RETURN QUERY SELECT v_record.user_id;
            RETURN;
        END IF;
    END LOOP;

    RETURN;
END;
$$;

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

GRANT EXECUTE ON FUNCTION mcpist.validate_api_key TO service_role;
GRANT EXECUTE ON FUNCTION mcpist.validate_api_key TO anon;
GRANT EXECUTE ON FUNCTION public.validate_api_key TO service_role;
GRANT EXECUTE ON FUNCTION public.validate_api_key TO anon;

-- =============================================================================
-- Entitlement RPCs
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Get User Entitlement
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_user_entitlement(p_user_id UUID)
RETURNS TABLE (
    user_status TEXT,
    plan_name TEXT,
    rate_limit_rpm INTEGER,
    rate_limit_burst INTEGER,
    quota_monthly INTEGER,
    credit_enabled BOOLEAN,
    credit_balance INTEGER,
    usage_current_month INTEGER,
    enabled_modules TEXT[]
) AS $$
DECLARE
    v_period_start DATE;
BEGIN
    v_period_start := DATE_TRUNC('month', NOW())::DATE;

    RETURN QUERY
    SELECT
        COALESCE(u.status, 'active')::TEXT AS user_status,
        COALESCE(p.name, 'free')::TEXT AS plan_name,
        COALESCE(p.rate_limit_rpm, 10) AS rate_limit_rpm,
        COALESCE(p.rate_limit_burst, 5) AS rate_limit_burst,
        p.quota_monthly AS quota_monthly,
        COALESCE(p.credit_enabled, false) AS credit_enabled,
        COALESCE(c.balance, 0) AS credit_balance,
        COALESCE(us.request_count, 0) AS usage_current_month,
        COALESCE(
            ARRAY(
                SELECT m.name
                FROM mcpist.user_module_preferences ump
                JOIN mcpist.modules m ON m.id = ump.module_id
                WHERE ump.user_id = p_user_id AND ump.is_enabled = true
            ),
            ARRAY[]::TEXT[]
        ) AS enabled_modules
    FROM (SELECT p_user_id AS id) AS input
    LEFT JOIN mcpist.users u ON u.id = input.id
    LEFT JOIN mcpist.subscriptions s ON s.user_id = input.id AND s.status = 'active'
    LEFT JOIN mcpist.plans p ON p.id = s.plan_id
    LEFT JOIN mcpist.credits c ON c.user_id = input.id
    LEFT JOIN mcpist.usage us ON us.user_id = input.id AND us.period_start = v_period_start;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION public.get_user_entitlement(p_user_id UUID)
RETURNS TABLE (
    user_status TEXT,
    plan_name TEXT,
    rate_limit_rpm INTEGER,
    rate_limit_burst INTEGER,
    quota_monthly INTEGER,
    credit_enabled BOOLEAN,
    credit_balance INTEGER,
    usage_current_month INTEGER,
    enabled_modules TEXT[]
) AS $$
BEGIN
    RETURN QUERY SELECT * FROM mcpist.get_user_entitlement(p_user_id);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION mcpist.get_user_entitlement(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_entitlement(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_entitlement(UUID) TO anon;

-- -----------------------------------------------------------------------------
-- Increment Usage
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.increment_usage(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_period_start DATE;
    v_new_count INTEGER;
BEGIN
    v_period_start := DATE_TRUNC('month', NOW())::DATE;

    INSERT INTO mcpist.usage (user_id, period_start, request_count)
    VALUES (p_user_id, v_period_start, 1)
    ON CONFLICT (user_id, period_start)
    DO UPDATE SET
        request_count = mcpist.usage.request_count + 1,
        updated_at = NOW()
    RETURNING request_count INTO v_new_count;

    RETURN v_new_count;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION public.increment_usage(p_user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN mcpist.increment_usage(p_user_id);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION mcpist.increment_usage(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.increment_usage(UUID) TO service_role;

-- -----------------------------------------------------------------------------
-- Deduct Credits
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.deduct_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_description TEXT DEFAULT NULL,
    p_reference_id TEXT DEFAULT NULL
)
RETURNS INTEGER AS $$
DECLARE
    v_current_balance INTEGER;
    v_new_balance INTEGER;
BEGIN
    SELECT balance INTO v_current_balance
    FROM mcpist.credits
    WHERE user_id = p_user_id
    FOR UPDATE;

    IF NOT FOUND THEN
        INSERT INTO mcpist.credits (user_id, balance)
        VALUES (p_user_id, 0);
        v_current_balance := 0;
    END IF;

    IF v_current_balance < p_amount THEN
        RETURN -1;
    END IF;

    v_new_balance := v_current_balance - p_amount;

    UPDATE mcpist.credits
    SET balance = v_new_balance, updated_at = NOW()
    WHERE user_id = p_user_id;

    INSERT INTO mcpist.credit_transactions (
        user_id, amount, balance_after, transaction_type, description, reference_id
    ) VALUES (
        p_user_id, -p_amount, v_new_balance, 'consume', p_description, p_reference_id
    );

    RETURN v_new_balance;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION public.deduct_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_description TEXT DEFAULT NULL,
    p_reference_id TEXT DEFAULT NULL
)
RETURNS INTEGER AS $$
BEGIN
    RETURN mcpist.deduct_credits(p_user_id, p_amount, p_description, p_reference_id);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION mcpist.deduct_credits(UUID, INTEGER, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.deduct_credits(UUID, INTEGER, TEXT, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- Get Tool Cost
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_tool_cost(
    p_module_name TEXT,
    p_tool_name TEXT
)
RETURNS INTEGER AS $$
DECLARE
    v_cost INTEGER;
BEGIN
    SELECT tc.credit_cost INTO v_cost
    FROM mcpist.tool_costs tc
    JOIN mcpist.modules m ON m.id = tc.module_id
    WHERE m.name = p_module_name AND tc.tool_name = p_tool_name;

    RETURN COALESCE(v_cost, 1);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION public.get_tool_cost(
    p_module_name TEXT,
    p_tool_name TEXT
)
RETURNS INTEGER AS $$
BEGIN
    RETURN mcpist.get_tool_cost(p_module_name, p_tool_name);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION mcpist.get_tool_cost(TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_tool_cost(TEXT, TEXT) TO service_role;
