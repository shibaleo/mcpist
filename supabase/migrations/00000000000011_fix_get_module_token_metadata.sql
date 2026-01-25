-- Fix get_module_token RPC to handle _metadata for Basic auth (Jira/Confluence)
-- Extracts _metadata to top-level metadata field

CREATE OR REPLACE FUNCTION mcpist.get_module_token(p_user_id UUID, p_module TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_secret_id UUID;
    v_credentials JSONB;
    v_auth_type TEXT;
    v_metadata JSONB;
    v_clean_credentials JSONB;
    v_result JSONB;
BEGIN
    -- service_tokensからcredentials_secret_idを取得
    SELECT st.credentials_secret_id INTO v_secret_id
    FROM mcpist.service_tokens st
    WHERE st.user_id = p_user_id AND st.service = p_module;

    IF v_secret_id IS NULL THEN
        RETURN jsonb_build_object(
            'error', 'no token configured for user: ' || p_user_id || ', service: ' || p_module
        );
    END IF;

    -- Vaultから復号されたシークレットを取得
    SELECT decrypted_secret::JSONB INTO v_credentials
    FROM vault.decrypted_secrets
    WHERE id = v_secret_id;

    IF v_credentials IS NULL THEN
        RETURN jsonb_build_object(
            'error', 'secret not found in vault'
        );
    END IF;

    -- _auth_typeと_metadataを抽出
    v_auth_type := v_credentials->>'_auth_type';
    v_metadata := v_credentials->'_metadata';

    -- _auth_typeと_metadataを除外したcredentialsを作成
    v_clean_credentials := v_credentials - '_auth_type' - '_metadata';

    -- 基本レスポンスを構築
    v_result := jsonb_build_object(
        'user_id', p_user_id,
        'service', p_module,
        'auth_type', v_auth_type,
        'credentials', v_clean_credentials
    );

    -- _metadataがある場合はトップレベルのmetadataとして追加
    IF v_metadata IS NOT NULL THEN
        v_result := v_result || jsonb_build_object('metadata', v_metadata);
    END IF;

    RETURN v_result;
END;
$$;
