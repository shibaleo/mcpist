-- =============================================================================
-- Fix OAuth Apps RPC to use vault.create_secret() instead of direct INSERT
-- =============================================================================

-- Drop existing functions
DROP FUNCTION IF EXISTS public.upsert_oauth_app(TEXT, TEXT, TEXT, TEXT, BOOLEAN);
DROP FUNCTION IF EXISTS mcpist.upsert_oauth_app(TEXT, TEXT, TEXT, TEXT, BOOLEAN);

-- -----------------------------------------------------------------------------
-- upsert_oauth_app RPC (fixed to use vault.create_secret)
-- -----------------------------------------------------------------------------

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
    v_credentials JSONB;
    v_secret_name TEXT;
BEGIN
    -- 既存の設定を確認
    SELECT oa.secret_id INTO v_existing_secret_id
    FROM mcpist.oauth_apps oa
    WHERE oa.provider = p_provider;

    -- credentials JSON を構築
    v_credentials := jsonb_build_object(
        'client_id', p_client_id,
        'client_secret', p_client_secret
    );

    -- シークレット名を生成
    v_secret_name := 'oauth_app_' || p_provider;

    IF v_existing_secret_id IS NOT NULL THEN
        -- 既存のシークレットを削除
        DELETE FROM vault.secrets WHERE id = v_existing_secret_id;
    END IF;

    -- 新しいシークレットを作成 (vault.create_secret を使用)
    SELECT vault.create_secret(
        v_credentials::TEXT,
        v_secret_name,
        'OAuth client credentials for ' || p_provider
    ) INTO v_new_secret_id;

    -- oauth_apps を upsert
    INSERT INTO mcpist.oauth_apps (provider, secret_id, redirect_uri, enabled)
    VALUES (p_provider, v_new_secret_id, p_redirect_uri, p_enabled)
    ON CONFLICT (provider)
    DO UPDATE SET
        secret_id = v_new_secret_id,
        redirect_uri = p_redirect_uri,
        enabled = p_enabled,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true,
        'action', CASE WHEN v_existing_secret_id IS NOT NULL THEN 'updated' ELSE 'created' END,
        'provider', p_provider
    );
END;
$$;

-- public schema wrapper
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

-- Grant to service_role (Admin用なのでauthenticatedには付与しない)
GRANT EXECUTE ON FUNCTION mcpist.upsert_oauth_app(TEXT, TEXT, TEXT, TEXT, BOOLEAN) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_oauth_app(TEXT, TEXT, TEXT, TEXT, BOOLEAN) TO service_role;
