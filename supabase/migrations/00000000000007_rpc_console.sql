-- =============================================================================
-- MCPist RPC Functions for Console Frontend
-- =============================================================================
-- This migration creates RPC functions used by Console (authenticated):
-- 1. generate_my_api_key - APIキー生成
-- 2. list_my_api_keys - APIキー一覧取得
-- 3. revoke_my_api_key - APIキー削除（論理削除）
-- 4. list_my_credentials - サービス接続一覧
-- 5. upsert_my_credential - サービストークン登録/更新
-- 6. delete_my_credential - サービストークン削除
-- 7. get_my_role - ユーザーロール取得
-- 8. get_my_preferences - ユーザー設定取得
-- 9. update_my_preferences - ユーザー設定更新
-- 10. get_my_tool_settings - ツール設定取得
-- 11. upsert_my_tool_settings - ツール設定更新
-- 12. get_my_module_descriptions - モジュール説明取得
-- 13. upsert_my_module_description - モジュール説明更新
-- 14. list_modules - モジュール一覧取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- generate_my_api_key
-- APIキーを生成（キーは生成時のみ返される）
-- 旧名: generate_api_key
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.generate_my_api_key(
    p_display_name TEXT,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
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
    v_key_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- キー生成（mpt_ + 32文字のランダム16進数）
    v_key := 'mpt_' || encode(gen_random_bytes(16), 'hex');
    v_key_prefix := substring(v_key from 1 for 8) || '...' || substring(v_key from length(v_key) - 3 for 4);
    v_key_hash := encode(sha256(v_key::bytea), 'hex');

    -- 挿入
    INSERT INTO mcpist.api_keys (user_id, name, key_hash, key_prefix, expires_at)
    VALUES (v_user_id, p_display_name, v_key_hash, v_key_prefix, p_expires_at)
    RETURNING id INTO v_key_id;

    RETURN jsonb_build_object(
        'api_key', v_key,
        'key_prefix', v_key_prefix
    );
END;
$$;

GRANT EXECUTE ON FUNCTION public.generate_my_api_key(TEXT, TIMESTAMPTZ) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_my_api_keys
-- ユーザーのAPIキー一覧を取得
-- 旧名: list_api_keys
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_my_api_keys()
RETURNS TABLE (
    id UUID,
    key_prefix TEXT,
    display_name TEXT,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        k.id,
        k.key_prefix,
        k.name AS display_name,
        k.expires_at,
        k.last_used_at,
        k.revoked_at
    FROM mcpist.api_keys k
    WHERE k.user_id = auth.uid()
      AND k.revoked_at IS NULL
    ORDER BY k.created_at DESC;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_my_api_keys() TO authenticated;

-- -----------------------------------------------------------------------------
-- revoke_my_api_key
-- APIキーを論理削除
-- 旧名: revoke_api_key
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.revoke_my_api_key(p_key_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id UUID;
    v_affected INTEGER;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    UPDATE mcpist.api_keys
    SET revoked_at = NOW()
    WHERE id = p_key_id
      AND user_id = v_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('success', v_affected > 0);
END;
$$;

GRANT EXECUTE ON FUNCTION public.revoke_my_api_key(UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_my_credentials
-- サービス接続一覧を取得
-- 旧名: list_service_connections
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_my_credentials()
RETURNS TABLE (
    module TEXT,
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
        uc.module,
        uc.created_at,
        uc.updated_at
    FROM mcpist.user_credentials uc
    WHERE uc.user_id = auth.uid()
    ORDER BY uc.module;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_my_credentials() TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_credential
-- サービストークンを登録/更新
-- 旧名: upsert_service_token
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.upsert_my_credential(
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
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

    -- UPSERT
    INSERT INTO mcpist.user_credentials (user_id, module, credentials)
    VALUES (v_user_id, p_module, p_credentials::TEXT)
    ON CONFLICT (user_id, module)
    DO UPDATE SET
        credentials = p_credentials::TEXT,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true,
        'module', p_module
    );
END;
$$;

GRANT EXECUTE ON FUNCTION public.upsert_my_credential(TEXT, JSONB) TO authenticated;

-- -----------------------------------------------------------------------------
-- delete_my_credential
-- サービストークンを削除
-- 旧名: delete_service_token
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.delete_my_credential(p_module TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_affected INTEGER;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    DELETE FROM mcpist.user_credentials
    WHERE user_id = v_user_id AND module = p_module;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('success', v_affected > 0);
END;
$$;

GRANT EXECUTE ON FUNCTION public.delete_my_credential(TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_role
-- 現在ログインしているユーザーのロールを取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.get_my_role()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id UUID;
    v_role TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RETURN jsonb_build_object('role', NULL);
    END IF;

    SELECT COALESCE(
        raw_app_meta_data->>'role',
        'user'
    ) INTO v_role
    FROM auth.users
    WHERE id = v_user_id;

    RETURN jsonb_build_object('role', v_role);
END;
$$;

GRANT EXECUTE ON FUNCTION public.get_my_role() TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_settings
-- Get current user's settings
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_settings()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_settings JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT settings INTO v_settings
    FROM mcpist.users
    WHERE id = v_user_id;

    RETURN COALESCE(v_settings, '{}'::JSONB);
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_settings()
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_settings();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_settings() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_settings() TO authenticated;

-- -----------------------------------------------------------------------------
-- update_my_settings
-- Update current user's settings (merge with existing)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.update_my_settings(p_settings JSONB)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_current JSONB;
    v_updated JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT COALESCE(settings, '{}'::JSONB) INTO v_current
    FROM mcpist.users
    WHERE id = v_user_id;

    v_updated := v_current || p_settings;

    UPDATE mcpist.users
    SET settings = v_updated
    WHERE id = v_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'settings', v_updated
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.update_my_settings(p_settings JSONB)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.update_my_settings(p_settings);
$$;

GRANT EXECUTE ON FUNCTION mcpist.update_my_settings(JSONB) TO authenticated;
GRANT EXECUTE ON FUNCTION public.update_my_settings(JSONB) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_tool_settings
-- ユーザーのツール設定を取得
-- -----------------------------------------------------------------------------

CREATE FUNCTION mcpist.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ts.tool_id,
        ts.enabled
    FROM mcpist.tool_settings ts
    JOIN mcpist.modules m ON m.id = ts.module_id
    WHERE ts.user_id = auth.uid()
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY m.name, ts.tool_id;
END;
$$;

CREATE FUNCTION public.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_tool_settings(p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_tool_settings(TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_tool_settings(TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_tool_settings
-- ツール設定を一括更新（モジュール単位）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_tool_settings(
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    -- 有効ツールをUPSERT
    IF p_enabled_tools IS NOT NULL AND array_length(p_enabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT v_user_id, v_module_id, unnest(p_enabled_tools), true
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = true;
    END IF;

    -- 無効ツールをUPSERT
    IF p_disabled_tools IS NOT NULL AND array_length(p_disabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT v_user_id, v_module_id, unnest(p_disabled_tools), false
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = false;
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'enabled_count', COALESCE(array_length(p_enabled_tools, 1), 0),
        'disabled_count', COALESCE(array_length(p_disabled_tools, 1), 0)
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_tool_settings(
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_tool_settings(p_module_name, p_enabled_tools, p_disabled_tools);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_module_descriptions
-- ユーザーのモジュール説明を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ms.description
    FROM mcpist.module_settings ms
    JOIN mcpist.modules m ON m.id = ms.module_id
    WHERE ms.user_id = auth.uid()
      AND ms.description IS NOT NULL
      AND ms.description != ''
    ORDER BY m.name;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_module_descriptions();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_module_descriptions() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_module_descriptions() TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_module_description
-- モジュール説明を更新
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    INSERT INTO mcpist.module_settings (user_id, module_id, enabled, description)
    VALUES (v_user_id, v_module_id, true, p_description)
    ON CONFLICT (user_id, module_id)
    DO UPDATE SET description = p_description;

    RETURN jsonb_build_object('success', true, 'module', p_module_name);
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_module_description(p_module_name, p_description);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_module_description(TEXT, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_module_description(TEXT, TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_modules
-- 利用可能なモジュール一覧を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_modules()
RETURNS TABLE (
    id UUID,
    name TEXT,
    status TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.id,
        m.name,
        m.status::TEXT,
        m.created_at
    FROM mcpist.modules m
    WHERE m.status IN ('active', 'beta')
    ORDER BY m.name;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_modules() TO authenticated;
GRANT EXECUTE ON FUNCTION public.list_modules() TO anon;
