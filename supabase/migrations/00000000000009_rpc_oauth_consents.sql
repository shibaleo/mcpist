-- =============================================================================
-- MCPist RPC Functions for OAuth Consents
-- =============================================================================
-- This migration creates RPC functions for OAuth consent management:
-- 1. list_oauth_consents - ユーザーのOAuthコンセント一覧を取得
-- 2. revoke_oauth_consent - OAuthコンセントを取り消し
-- 3. list_all_oauth_consents - 全ユーザーのOAuthコンセント一覧を取得（管理者用）
-- =============================================================================

-- -----------------------------------------------------------------------------
-- list_oauth_consents
-- ユーザーのOAuthコンセント一覧を取得（auth.oauth_consentsから）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_oauth_consents()
RETURNS TABLE (
    id UUID,
    client_id UUID,
    client_name TEXT,
    scopes TEXT,
    granted_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.client_id,
        cl.client_name,
        c.scopes,
        c.granted_at
    FROM auth.oauth_consents c
    LEFT JOIN auth.oauth_clients cl ON c.client_id = cl.id
    WHERE c.user_id = auth.uid()
      AND c.revoked_at IS NULL
    ORDER BY c.granted_at DESC;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_oauth_consents() TO authenticated;

-- -----------------------------------------------------------------------------
-- revoke_oauth_consent
-- OAuthコンセントを取り消し（論理削除）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.revoke_oauth_consent(p_consent_id UUID)
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

    -- ユーザー自身のコンセントのみ取り消し可能
    UPDATE auth.oauth_consents
    SET revoked_at = NOW()
    WHERE id = p_consent_id
      AND user_id = v_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('revoked', v_affected > 0);
END;
$$;

GRANT EXECUTE ON FUNCTION public.revoke_oauth_consent(UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_all_oauth_consents (Admin only)
-- 全ユーザーのOAuthコンセント一覧を取得（管理者用）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_all_oauth_consents()
RETURNS TABLE (
    id UUID,
    user_id UUID,
    user_email TEXT,
    client_id UUID,
    client_name TEXT,
    scopes TEXT,
    granted_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_role TEXT;
BEGIN
    -- 管理者権限チェック
    SELECT COALESCE(raw_app_meta_data->>'role', 'user')
    INTO v_role
    FROM auth.users
    WHERE auth.users.id = auth.uid();

    IF v_role != 'admin' THEN
        RAISE EXCEPTION 'Admin access required';
    END IF;

    RETURN QUERY
    SELECT
        c.id,
        c.user_id,
        u.email::TEXT AS user_email,
        c.client_id,
        cl.client_name,
        c.scopes,
        c.granted_at
    FROM auth.oauth_consents c
    LEFT JOIN auth.oauth_clients cl ON c.client_id = cl.id
    LEFT JOIN auth.users u ON c.user_id = u.id
    WHERE c.revoked_at IS NULL
    ORDER BY c.granted_at DESC;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_all_oauth_consents() TO authenticated;
