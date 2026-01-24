-- =============================================================================
-- MCPist RPC Functions for Console Frontend
-- =============================================================================
-- This migration creates RPC functions used by Console (authenticated):
-- 1. generate_api_key - APIキー生成
-- 2. list_api_keys - APIキー一覧取得
-- 3. revoke_api_key - APIキー削除（論理削除）
-- 4. list_service_connections - サービス接続一覧
-- 5. upsert_service_token - サービストークン登録/更新
-- 6. delete_service_token - サービストークン削除
-- =============================================================================

-- -----------------------------------------------------------------------------
-- generate_api_key
-- APIキーを生成（キーは生成時のみ返される）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.generate_api_key(
    p_name TEXT,
    p_expires_in_days INTEGER DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
DECLARE
    v_user_id UUID;
    v_key TEXT;
    v_key_hash TEXT;
    v_key_prefix TEXT;
    v_expires_at TIMESTAMPTZ;
    v_key_id UUID;
BEGIN
    -- 認証確認
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- キー生成（mpt_ + 32文字のランダム16進数）
    v_key := 'mpt_' || encode(gen_random_bytes(16), 'hex');
    v_key_prefix := substring(v_key from 1 for 8) || '...' || substring(v_key from length(v_key) - 3 for 4);
    v_key_hash := encode(sha256(v_key::bytea), 'hex');

    -- 有効期限設定
    IF p_expires_in_days IS NOT NULL THEN
        v_expires_at := NOW() + (p_expires_in_days || ' days')::INTERVAL;
    END IF;

    -- 挿入
    INSERT INTO mcpist.api_keys (user_id, name, key_hash, key_prefix, expires_at)
    VALUES (v_user_id, p_name, v_key_hash, v_key_prefix, v_expires_at)
    RETURNING id INTO v_key_id;

    RETURN jsonb_build_object(
        'id', v_key_id,
        'name', p_name,
        'key', v_key,
        'key_prefix', v_key_prefix,
        'expires_at', v_expires_at
    );
END;
$$;

GRANT EXECUTE ON FUNCTION public.generate_api_key(TEXT, INTEGER) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_api_keys
-- ユーザーのAPIキー一覧を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_api_keys()
RETURNS TABLE (
    id UUID,
    name TEXT,
    key_prefix TEXT,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    is_expired BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        k.id,
        k.name,
        k.key_prefix,
        k.last_used_at,
        k.expires_at,
        k.created_at,
        (k.expires_at IS NOT NULL AND k.expires_at < NOW()) AS is_expired
    FROM mcpist.api_keys k
    WHERE k.user_id = auth.uid()
      AND k.revoked_at IS NULL
    ORDER BY k.created_at DESC;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_api_keys() TO authenticated;

-- -----------------------------------------------------------------------------
-- revoke_api_key
-- APIキーを論理削除（key_hashをキャッシュ無効化用に返す）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.revoke_api_key(p_key_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id UUID;
    v_key_hash TEXT;
    v_affected INTEGER;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- 先にkey_hashを取得
    SELECT key_hash INTO v_key_hash
    FROM mcpist.api_keys
    WHERE id = p_key_id
      AND user_id = v_user_id
      AND revoked_at IS NULL;

    IF v_key_hash IS NULL THEN
        RETURN jsonb_build_object('revoked', false, 'key_hash', NULL);
    END IF;

    -- 論理削除
    UPDATE mcpist.api_keys
    SET revoked_at = NOW()
    WHERE id = p_key_id
      AND user_id = v_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('revoked', v_affected > 0, 'key_hash', v_key_hash);
END;
$$;

GRANT EXECUTE ON FUNCTION public.revoke_api_key(UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_service_connections
-- サービス接続一覧を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_service_connections()
RETURNS TABLE (
    id UUID,
    service TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        st.id,
        st.service,
        st.created_at,
        st.updated_at
    FROM mcpist.service_tokens st
    WHERE st.user_id = auth.uid()
    ORDER BY st.service;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_service_connections() TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_service_token
-- サービストークンを登録/更新（Vaultに保存）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.upsert_service_token(
    p_service TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_user_id UUID;
    v_existing_secret_id UUID;
    v_new_secret_id UUID;
    v_secret_name TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- 既存のsecret_idを取得
    SELECT st.credentials_secret_id INTO v_existing_secret_id
    FROM mcpist.service_tokens st
    WHERE st.user_id = v_user_id AND st.service = p_service;

    -- シークレット名を生成
    v_secret_name := v_user_id::TEXT || ':' || p_service;

    -- 既存のシークレットがあれば削除
    IF v_existing_secret_id IS NOT NULL THEN
        DELETE FROM vault.secrets WHERE id = v_existing_secret_id;
    END IF;

    -- 新しいシークレットを作成
    SELECT vault.create_secret(
        p_credentials::TEXT,
        v_secret_name,
        'Service credentials for ' || p_service
    ) INTO v_new_secret_id;

    -- service_tokensをupsert
    INSERT INTO mcpist.service_tokens (user_id, service, credentials_secret_id)
    VALUES (v_user_id, p_service, v_new_secret_id)
    ON CONFLICT (user_id, service)
    DO UPDATE SET
        credentials_secret_id = v_new_secret_id,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true,
        'service', p_service
    );
END;
$$;

GRANT EXECUTE ON FUNCTION public.upsert_service_token(TEXT, JSONB) TO authenticated;

-- -----------------------------------------------------------------------------
-- delete_service_token
-- サービストークンを削除
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.delete_service_token(p_service TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_user_id UUID;
    v_secret_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- secret_idを取得
    SELECT st.credentials_secret_id INTO v_secret_id
    FROM mcpist.service_tokens st
    WHERE st.user_id = v_user_id AND st.service = p_service;

    IF v_secret_id IS NULL THEN
        RETURN jsonb_build_object('deleted', false);
    END IF;

    -- シークレットを削除
    DELETE FROM vault.secrets WHERE id = v_secret_id;

    -- service_tokensを削除
    DELETE FROM mcpist.service_tokens
    WHERE user_id = v_user_id AND service = p_service;

    RETURN jsonb_build_object('deleted', true);
END;
$$;

GRANT EXECUTE ON FUNCTION public.delete_service_token(TEXT) TO authenticated;
