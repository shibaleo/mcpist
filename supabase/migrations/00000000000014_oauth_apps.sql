-- =============================================================================
-- MCPist: OAuth Apps Table and RPCs
-- =============================================================================
-- OAuth プロバイダー（Google, Microsoft等）のクライアント認証情報を管理
-- クライアントID/Secretは Vault に暗号化保存
-- =============================================================================

-- -----------------------------------------------------------------------------
-- oauth_apps テーブル
-- プロバイダー別のOAuthアプリケーション設定
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS mcpist.oauth_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL UNIQUE,  -- 'google', 'microsoft', etc.
    secret_id UUID REFERENCES vault.secrets(id) ON DELETE SET NULL,
    redirect_uri TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE mcpist.oauth_apps IS 'OAuth プロバイダーのクライアント認証情報';
COMMENT ON COLUMN mcpist.oauth_apps.provider IS 'プロバイダー識別子 (google, microsoft)';
COMMENT ON COLUMN mcpist.oauth_apps.secret_id IS 'Vault secrets への参照 (client_id, client_secret を暗号化保存)';
COMMENT ON COLUMN mcpist.oauth_apps.redirect_uri IS 'OAuth コールバック URI';
COMMENT ON COLUMN mcpist.oauth_apps.enabled IS 'このプロバイダーが有効かどうか';

-- RLS Policy (service_role のみアクセス可能)
ALTER TABLE mcpist.oauth_apps ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role can manage oauth_apps"
    ON mcpist.oauth_apps
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- -----------------------------------------------------------------------------
-- get_oauth_app_credentials RPC
-- Go Server から呼び出し: プロバイダーのクライアント認証情報を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_oauth_app_credentials(p_provider TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_secret_id UUID;
    v_redirect_uri TEXT;
    v_enabled BOOLEAN;
    v_credentials JSONB;
BEGIN
    -- oauth_apps から設定を取得
    SELECT oa.secret_id, oa.redirect_uri, oa.enabled
    INTO v_secret_id, v_redirect_uri, v_enabled
    FROM mcpist.oauth_apps oa
    WHERE oa.provider = p_provider;

    IF v_secret_id IS NULL THEN
        RETURN jsonb_build_object(
            'error', 'oauth_app_not_configured',
            'message', 'OAuth app not configured for provider: ' || p_provider
        );
    END IF;

    IF NOT v_enabled THEN
        RETURN jsonb_build_object(
            'error', 'oauth_app_disabled',
            'message', 'OAuth app is disabled for provider: ' || p_provider
        );
    END IF;

    -- Vault から復号されたシークレットを取得
    SELECT decrypted_secret::JSONB INTO v_credentials
    FROM vault.decrypted_secrets
    WHERE id = v_secret_id;

    IF v_credentials IS NULL THEN
        RETURN jsonb_build_object(
            'error', 'secret_not_found',
            'message', 'Credentials not found in vault for provider: ' || p_provider
        );
    END IF;

    -- client_id, client_secret, redirect_uri を返す
    RETURN jsonb_build_object(
        'provider', p_provider,
        'client_id', v_credentials->>'client_id',
        'client_secret', v_credentials->>'client_secret',
        'redirect_uri', v_redirect_uri
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_oauth_app_credentials(p_provider TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_oauth_app_credentials(p_provider);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_oauth_app_credentials(TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_oauth_app_credentials(TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- upsert_oauth_app RPC
-- Admin Console から呼び出し: OAuthアプリ設定を作成/更新
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

    IF v_existing_secret_id IS NOT NULL THEN
        -- 既存のシークレットを更新
        UPDATE vault.secrets
        SET secret = v_credentials::TEXT,
            updated_at = NOW()
        WHERE id = v_existing_secret_id;

        -- oauth_apps を更新
        UPDATE mcpist.oauth_apps
        SET redirect_uri = p_redirect_uri,
            enabled = p_enabled,
            updated_at = NOW()
        WHERE provider = p_provider;

        RETURN jsonb_build_object(
            'success', true,
            'action', 'updated',
            'provider', p_provider
        );
    ELSE
        -- 新しいシークレットを作成
        INSERT INTO vault.secrets (secret, name, description)
        VALUES (
            v_credentials::TEXT,
            'oauth_app_' || p_provider,
            'OAuth client credentials for ' || p_provider
        )
        RETURNING id INTO v_new_secret_id;

        -- oauth_apps に新規レコード作成
        INSERT INTO mcpist.oauth_apps (provider, secret_id, redirect_uri, enabled)
        VALUES (p_provider, v_new_secret_id, p_redirect_uri, p_enabled);

        RETURN jsonb_build_object(
            'success', true,
            'action', 'created',
            'provider', p_provider
        );
    END IF;
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

GRANT EXECUTE ON FUNCTION mcpist.upsert_oauth_app(TEXT, TEXT, TEXT, TEXT, BOOLEAN) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_oauth_app(TEXT, TEXT, TEXT, TEXT, BOOLEAN) TO service_role;

-- -----------------------------------------------------------------------------
-- list_oauth_apps RPC
-- Admin Console から呼び出し: OAuthアプリ設定一覧を取得（シークレットはマスク）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.list_oauth_apps()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_result JSONB;
BEGIN
    SELECT COALESCE(jsonb_agg(
        jsonb_build_object(
            'provider', oa.provider,
            'redirect_uri', oa.redirect_uri,
            'enabled', oa.enabled,
            'has_credentials', oa.secret_id IS NOT NULL,
            'client_id', CASE
                WHEN ds.decrypted_secret IS NOT NULL
                THEN (ds.decrypted_secret::JSONB)->>'client_id'
                ELSE NULL
            END,
            'created_at', oa.created_at,
            'updated_at', oa.updated_at
        ) ORDER BY oa.provider
    ), '[]'::JSONB) INTO v_result
    FROM mcpist.oauth_apps oa
    LEFT JOIN vault.decrypted_secrets ds ON ds.id = oa.secret_id;

    RETURN v_result;
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.list_oauth_apps()
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.list_oauth_apps();
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_oauth_apps() TO service_role;
GRANT EXECUTE ON FUNCTION public.list_oauth_apps() TO service_role;

-- -----------------------------------------------------------------------------
-- delete_oauth_app RPC
-- Admin Console から呼び出し: OAuthアプリ設定を削除
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.delete_oauth_app(p_provider TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_secret_id UUID;
BEGIN
    -- 既存の設定を確認
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

    -- oauth_apps から削除（Vault secrets は CASCADE で削除されない）
    DELETE FROM mcpist.oauth_apps WHERE provider = p_provider;

    -- Vault secrets から削除
    DELETE FROM vault.secrets WHERE id = v_secret_id;

    RETURN jsonb_build_object(
        'success', true,
        'provider', p_provider
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.delete_oauth_app(p_provider TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_oauth_app(p_provider);
$$;

GRANT EXECUTE ON FUNCTION mcpist.delete_oauth_app(TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.delete_oauth_app(TEXT) TO service_role;
