-- =============================================================================
-- Fix update_module_token: Use vault.create_secret instead of direct INSERT
-- =============================================================================
-- upsert_service_token と同じ方法で vault.create_secret() を使用
-- =============================================================================

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
    v_new_secret_id UUID;
    v_secret_name TEXT;
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

    -- シークレット名を生成 (upsert_service_tokenと同じ形式)
    v_secret_name := p_user_id::TEXT || ':' || p_module;

    -- 古いシークレットを削除
    DELETE FROM vault.secrets WHERE id = v_secret_id;

    -- 新しいシークレットを作成 (vault.create_secret を使用)
    SELECT vault.create_secret(
        p_credentials::TEXT,
        v_secret_name,
        'Service credentials for ' || p_module
    ) INTO v_new_secret_id;

    -- service_tokensのcredentials_secret_idを更新
    UPDATE mcpist.service_tokens
    SET credentials_secret_id = v_new_secret_id,
        updated_at = NOW()
    WHERE user_id = p_user_id AND service = p_module;

    RETURN jsonb_build_object(
        'success', true
    );
END;
$$;

-- public schema wrapper (再作成)
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
