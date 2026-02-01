-- =============================================================================
-- Fix: Replace vault.update_secret with delete + create pattern
-- =============================================================================
-- vault.update_secret does not exist in Supabase Vault
-- Use delete + create pattern instead
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.upsert_oauth_app(
    p_provider TEXT,
    p_client_id TEXT,
    p_client_secret TEXT,
    p_redirect_uri TEXT,
    p_enabled BOOLEAN DEFAULT true
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_existing_secret_id UUID;
    v_new_secret_id UUID;
    v_credentials TEXT;
    v_secret_name TEXT;
    v_action TEXT;
BEGIN
    v_secret_name := 'oauth_app_' || p_provider;

    v_credentials := jsonb_build_object(
        'client_id', p_client_id,
        'client_secret', p_client_secret
    )::TEXT;

    -- oauth_apps テーブルから既存レコードを検索
    SELECT oa.secret_id INTO v_existing_secret_id
    FROM mcpist.oauth_apps oa
    WHERE oa.provider = p_provider;

    IF v_existing_secret_id IS NOT NULL THEN
        v_action := 'updated';

        -- 既存シークレットを削除
        DELETE FROM vault.secrets WHERE id = v_existing_secret_id;
    ELSE
        v_action := 'created';

        -- 孤立したシークレット（oauth_appsに紐づいていないが同名のもの）を削除
        DELETE FROM vault.secrets WHERE name = v_secret_name;
    END IF;

    -- 新規シークレットを作成
    v_new_secret_id := vault.create_secret(
        v_credentials,
        v_secret_name,
        'OAuth client credentials for ' || p_provider
    );

    -- oauth_apps テーブルを UPSERT
    INSERT INTO mcpist.oauth_apps (provider, secret_id, redirect_uri, enabled)
    VALUES (p_provider, v_new_secret_id, p_redirect_uri, p_enabled)
    ON CONFLICT (provider) DO UPDATE SET
        secret_id = v_new_secret_id,
        redirect_uri = p_redirect_uri,
        enabled = p_enabled,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true,
        'action', v_action,
        'provider', p_provider
    );
END;
$$;

-- public wrapper も更新
CREATE OR REPLACE FUNCTION public.upsert_oauth_app(
    p_provider TEXT,
    p_client_id TEXT,
    p_client_secret TEXT,
    p_redirect_uri TEXT,
    p_enabled BOOLEAN DEFAULT true
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_oauth_app(p_provider, p_client_id, p_client_secret, p_redirect_uri, p_enabled);
$$;

-- delete_oauth_app も修正（vault.delete_secret は存在しない）
CREATE OR REPLACE FUNCTION mcpist.delete_oauth_app(p_provider TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_secret_id UUID;
BEGIN
    SELECT oa.secret_id INTO v_secret_id
    FROM mcpist.oauth_apps oa
    WHERE oa.provider = p_provider;

    IF v_secret_id IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'not_found',
            'message', 'OAuth app not found for provider: ' || p_provider
        );
    END IF;

    -- oauth_apps レコードを削除（FK で secret_id は SET NULL になる）
    DELETE FROM mcpist.oauth_apps WHERE provider = p_provider;

    -- vault secret を直接削除
    DELETE FROM vault.secrets WHERE id = v_secret_id;

    RETURN jsonb_build_object(
        'success', true,
        'provider', p_provider
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.delete_oauth_app(p_provider TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_oauth_app(p_provider);
$$;
