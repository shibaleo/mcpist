-- =============================================================================
-- MCPist RPC Functions for MCP Server
-- =============================================================================
-- This migration creates RPC functions used by MCP Server (service_role):
-- 1. lookup_user_by_key_hash - APIキーハッシュからuser_idを取得
-- 2. get_user_context - ユーザー情報取得
-- 3. consume_credit - クレジット消費
-- 4. get_module_token - モジュールトークン取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- lookup_user_by_key_hash
-- APIキーのハッシュからuser_idを取得する
-- Worker側でSHA-256ハッシュを計算し、キャッシュと組み合わせて使用
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.lookup_user_by_key_hash(p_key_hash TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_key_record RECORD;
BEGIN
    -- キー検索
    SELECT
        k.id,
        k.user_id,
        k.name,
        k.expires_at,
        k.revoked_at
    INTO v_key_record
    FROM mcpist.api_keys k
    WHERE k.key_hash = p_key_hash;

    -- キーが存在しない
    IF v_key_record IS NULL THEN
        RETURN jsonb_build_object('valid', false, 'error', 'invalid_key');
    END IF;

    -- 削除済み
    IF v_key_record.revoked_at IS NOT NULL THEN
        RETURN jsonb_build_object('valid', false, 'error', 'revoked');
    END IF;

    -- 有効期限切れ
    IF v_key_record.expires_at IS NOT NULL AND v_key_record.expires_at < NOW() THEN
        RETURN jsonb_build_object('valid', false, 'error', 'expired');
    END IF;

    -- 最終使用日時を更新
    UPDATE mcpist.api_keys
    SET last_used_at = NOW()
    WHERE id = v_key_record.id;

    RETURN jsonb_build_object(
        'valid', true,
        'user_id', v_key_record.user_id
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.lookup_user_by_key_hash(p_key_hash TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.lookup_user_by_key_hash(p_key_hash);
$$;

GRANT EXECUTE ON FUNCTION mcpist.lookup_user_by_key_hash(TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.lookup_user_by_key_hash(TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_user_context
-- ツール実行に必要なユーザー情報を一括取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    disabled_tools JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_account_status TEXT;
    v_free_credits INTEGER;
    v_paid_credits INTEGER;
    v_enabled_modules TEXT[];
    v_disabled_tools JSONB;
BEGIN
    -- ユーザー状態を取得
    SELECT u.account_status::TEXT INTO v_account_status
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- ユーザーが存在しない場合は空
    END IF;

    -- クレジット残高を取得
    SELECT c.free_credits, c.paid_credits INTO v_free_credits, v_paid_credits
    FROM mcpist.credits c
    WHERE c.user_id = p_user_id;

    IF v_free_credits IS NULL THEN
        v_free_credits := 0;
        v_paid_credits := 0;
    END IF;

    -- 有効なモジュールを取得（module_settingsに存在しないモジュールはデフォルトで有効）
    SELECT ARRAY(
        SELECT m.name
        FROM mcpist.modules m
        WHERE m.status IN ('active', 'beta')
          AND NOT EXISTS (
              SELECT 1 FROM mcpist.module_settings ms
              WHERE ms.user_id = p_user_id
                AND ms.module_id = m.id
                AND ms.enabled = false
          )
    ) INTO v_enabled_modules;

    -- 無効なツールを取得（モジュール別）
    SELECT COALESCE(
        jsonb_object_agg(m.name, tool_list),
        '{}'::JSONB
    ) INTO v_disabled_tools
    FROM (
        SELECT m.name, array_agg(ts.tool_name) AS tool_list
        FROM mcpist.tool_settings ts
        JOIN mcpist.modules m ON m.id = ts.module_id
        WHERE ts.user_id = p_user_id AND ts.enabled = false
        GROUP BY m.name
    ) AS subq
    JOIN mcpist.modules m ON m.name = subq.name;

    RETURN QUERY SELECT v_account_status, v_free_credits, v_paid_credits, v_enabled_modules, v_disabled_tools;
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    disabled_tools JSONB
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;

-- -----------------------------------------------------------------------------
-- consume_credit
-- クレジットを消費し、履歴を記録する（冪等性対応）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.consume_credit(
    p_user_id UUID,
    p_module TEXT,
    p_tool TEXT,
    p_amount INTEGER,
    p_request_id TEXT,
    p_task_id TEXT DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_current_free INTEGER;
    v_current_paid INTEGER;
    v_new_free INTEGER;
    v_new_paid INTEGER;
    v_consumed_free INTEGER;
    v_consumed_paid INTEGER;
    v_existing_tx RECORD;
BEGIN
    -- 冪等性チェック: 既に同じrequest_id + task_idで処理済みか確認
    SELECT id, type INTO v_existing_tx
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
      AND request_id = p_request_id
      AND COALESCE(task_id, '') = COALESCE(p_task_id, '');

    IF v_existing_tx IS NOT NULL THEN
        -- 既に処理済み - 現在の残高を返す
        SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id;

        RETURN jsonb_build_object(
            'success', true,
            'free_credits', v_current_free,
            'paid_credits', v_current_paid,
            'already_processed', true
        );
    END IF;

    -- 現在の残高を取得（FOR UPDATE でロック）
    SELECT free_credits, paid_credits INTO v_current_free, v_current_paid
    FROM mcpist.credits
    WHERE user_id = p_user_id
    FOR UPDATE;

    IF v_current_free IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found'
        );
    END IF;

    -- 残高不足チェック
    IF (v_current_free + v_current_paid) < p_amount THEN
        RETURN jsonb_build_object(
            'success', false,
            'free_credits', v_current_free,
            'paid_credits', v_current_paid,
            'error', 'insufficient_credits'
        );
    END IF;

    -- 無料クレジットから優先的に消費
    IF v_current_free >= p_amount THEN
        v_consumed_free := p_amount;
        v_consumed_paid := 0;
        v_new_free := v_current_free - p_amount;
        v_new_paid := v_current_paid;
    ELSE
        v_consumed_free := v_current_free;
        v_consumed_paid := p_amount - v_current_free;
        v_new_free := 0;
        v_new_paid := v_current_paid - v_consumed_paid;
    END IF;

    -- クレジット更新
    UPDATE mcpist.credits
    SET free_credits = v_new_free, paid_credits = v_new_paid, updated_at = NOW()
    WHERE user_id = p_user_id;

    -- 履歴記録（無料クレジット消費分）
    IF v_consumed_free > 0 THEN
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, module, tool, request_id, task_id
        ) VALUES (
            p_user_id, 'consume', -v_consumed_free, 'free', p_module, p_tool, p_request_id, p_task_id
        );
    END IF;

    -- 履歴記録（有料クレジット消費分）
    IF v_consumed_paid > 0 THEN
        INSERT INTO mcpist.credit_transactions (
            user_id, type, amount, credit_type, module, tool, request_id, task_id
        ) VALUES (
            p_user_id, 'consume', -v_consumed_paid, 'paid', p_module, p_tool, p_request_id, p_task_id
        );
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free,
        'paid_credits', v_new_paid
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.consume_credit(
    p_user_id UUID,
    p_module TEXT,
    p_tool TEXT,
    p_amount INTEGER,
    p_request_id TEXT,
    p_task_id TEXT DEFAULT NULL
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.consume_credit(p_user_id, p_module, p_tool, p_amount, p_request_id, p_task_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.consume_credit(UUID, TEXT, TEXT, INTEGER, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.consume_credit(UUID, TEXT, TEXT, INTEGER, TEXT, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_module_token
-- モジュールが使用する外部サービスのトークンをVaultから取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_module_token(
    p_user_id UUID,
    p_module TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, vault, public
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
            'found', false,
            'error', 'token_not_found'
        );
    END IF;

    -- Vaultから復号されたシークレットを取得
    SELECT decrypted_secret::JSONB INTO v_credentials
    FROM vault.decrypted_secrets
    WHERE id = v_secret_id;

    IF v_credentials IS NULL THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'secret_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'found', true,
        'credentials', v_credentials
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_module_token(
    p_user_id UUID,
    p_module TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_module_token(p_user_id, p_module);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_module_token(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_module_token(UUID, TEXT) TO service_role;
