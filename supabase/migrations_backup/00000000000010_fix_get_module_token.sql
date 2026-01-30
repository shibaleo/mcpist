-- Fix get_module_token RPC to return response format matching specification (dtl-itr-MOD-TVL.md)
-- Old format: {"found": true, "credentials": {...}}
-- New format: {"user_id": "...", "service": "...", "auth_type": "...", "credentials": {...}}

CREATE OR REPLACE FUNCTION mcpist.get_module_token(p_user_id UUID, p_module TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_secret_id UUID;
    v_credentials JSONB;
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

    -- 仕様通りのレスポンス形式で返す (dtl-itr-MOD-TVL.md)
    RETURN jsonb_build_object(
        'user_id', p_user_id,
        'service', p_module,
        'auth_type', v_credentials->>'_auth_type',
        'credentials', v_credentials - '_auth_type'
    );
END;
$$;
