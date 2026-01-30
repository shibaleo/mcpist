-- =============================================================================
-- MCPist RPC: update_module_token
-- =============================================================================
-- OAuth2トークンリフレッシュ後に、新しいトークンをVaultに保存する
-- 呼び出し元: MCP Server (Modules)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- update_module_token
-- モジュールがリフレッシュしたトークンをVaultに保存する
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.update_module_token(
    p_user_id UUID,
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
AS $$
DECLARE
    v_secret_id UUID;
BEGIN
    -- service_tokensからcredentials_secret_idを取得
    SELECT st.credentials_secret_id INTO v_secret_id
    FROM mcpist.service_tokens st
    WHERE st.user_id = p_user_id AND st.service = p_module;

    IF v_secret_id IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'token_not_found'
        );
    END IF;

    -- Vaultのシークレットを更新
    UPDATE vault.secrets
    SET secret = p_credentials::TEXT,
        updated_at = NOW()
    WHERE id = v_secret_id;

    -- service_tokensのupdated_atも更新
    UPDATE mcpist.service_tokens
    SET updated_at = NOW()
    WHERE user_id = p_user_id AND service = p_module;

    RETURN jsonb_build_object(
        'success', true
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.update_module_token(
    p_user_id UUID,
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.update_module_token(p_user_id, p_module, p_credentials);
$$;

GRANT EXECUTE ON FUNCTION mcpist.update_module_token(UUID, TEXT, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.update_module_token(UUID, TEXT, JSONB) TO service_role;
