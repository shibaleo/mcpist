-- =============================================================================
-- MCPist RPC Functions for MCP Server
-- =============================================================================
-- This migration creates RPC functions used by MCP Server (service_role):
-- 1. lookup_user_by_key_hash - APIキーハッシュからuser_idを取得
-- 2. get_user_context - ユーザー情報取得
-- 3. consume_user_credits - クレジット消費
-- 4. get_user_credential - ユーザー認証情報取得
-- 5. upsert_user_credential - ユーザー認証情報保存/更新
-- 6. sync_modules - モジュール同期
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
-- ツール実行に必要なユーザー情報を一括取得（最適化版）
-- enabled_modules は enabled_tools のキーから導出
-- -----------------------------------------------------------------------------

CREATE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_account_status TEXT;
    v_free_credits INTEGER;
    v_paid_credits INTEGER;
    v_language TEXT;
    v_module_data JSONB;
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_module_descriptions JSONB;
BEGIN
    -- 1. Get user status and language
    SELECT u.account_status::TEXT, COALESCE(u.settings->>'language', 'en-US')
    INTO v_account_status, v_language
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- 2. Get credit balance
    SELECT c.free_credits, c.paid_credits
    INTO v_free_credits, v_paid_credits
    FROM mcpist.credits c
    WHERE c.user_id = p_user_id;

    IF v_free_credits IS NULL THEN
        v_free_credits := 0;
        v_paid_credits := 0;
    END IF;

    -- 3. Get enabled tools grouped by module with descriptions
    SELECT
        COALESCE(jsonb_object_agg(
            module_name,
            jsonb_build_object(
                'tools', tools,
                'description', description
            )
        ), '{}'::JSONB)
    INTO v_module_data
    FROM (
        SELECT
            m.name AS module_name,
            array_agg(ts.tool_id) AS tools,
            ms.description
        FROM mcpist.tool_settings ts
        JOIN mcpist.modules m ON m.id = ts.module_id
        LEFT JOIN mcpist.module_settings ms
            ON ms.user_id = ts.user_id AND ms.module_id = ts.module_id
        WHERE ts.user_id = p_user_id
          AND ts.enabled = true
          AND m.status IN ('active', 'beta')
        GROUP BY m.name, ms.description
    ) AS subq;

    -- 4. Extract enabled_modules
    SELECT array_agg(key)
    INTO v_enabled_modules
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 5. Extract enabled_tools: {module: [tool_ids]}
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'tools'),
        '{}'::JSONB
    )
    INTO v_enabled_tools
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 6. Extract module_descriptions
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'description'),
        '{}'::JSONB
    )
    INTO v_module_descriptions
    FROM jsonb_object_keys(v_module_data) AS key
    WHERE v_module_data->key->>'description' IS NOT NULL
      AND v_module_data->key->>'description' != '';

    IF v_enabled_modules IS NULL THEN
        v_enabled_modules := ARRAY[]::TEXT[];
    END IF;

    RETURN QUERY SELECT
        v_account_status,
        v_free_credits,
        v_paid_credits,
        v_enabled_modules,
        v_enabled_tools,
        v_language,
        v_module_descriptions;
END;
$$;

-- public schema wrapper
CREATE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    free_credits INTEGER,
    paid_credits INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;

-- -----------------------------------------------------------------------------
-- consume_user_credits
-- クレジットを消費し、履歴を記録する（冪等性対応）
-- 無料クレジットを優先的に消費
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.consume_user_credits(
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
    -- 冪等性チェック
    SELECT id, type INTO v_existing_tx
    FROM mcpist.credit_transactions
    WHERE user_id = p_user_id
      AND request_id = p_request_id
      AND COALESCE(task_id, '') = COALESCE(p_task_id, '');

    IF v_existing_tx IS NOT NULL THEN
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
CREATE OR REPLACE FUNCTION public.consume_user_credits(
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
    SELECT mcpist.consume_user_credits(p_user_id, p_module, p_tool, p_amount, p_request_id, p_task_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.consume_user_credits(UUID, TEXT, TEXT, INTEGER, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.consume_user_credits(UUID, TEXT, TEXT, INTEGER, TEXT, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_user_credential
-- ユーザーのモジュール用認証情報を取得（MCP Server 向け）
-- 旧名: get_module_token
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_user_credential(
    p_user_id UUID,
    p_module TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_credentials TEXT;
    v_credentials_jsonb JSONB;
BEGIN
    -- user_credentials から認証情報を取得
    SELECT uc.credentials INTO v_credentials
    FROM mcpist.user_credentials uc
    WHERE uc.user_id = p_user_id AND uc.module = p_module;

    IF v_credentials IS NULL THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'token_not_found'
        );
    END IF;

    -- Parse JSON
    BEGIN
        v_credentials_jsonb := v_credentials::JSONB;
    EXCEPTION WHEN OTHERS THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'invalid_credentials_format'
        );
    END;

    RETURN jsonb_build_object(
        'found', true,
        'user_id', p_user_id,
        'service', p_module,
        'auth_type', v_credentials_jsonb->>'auth_type',
        'credentials', v_credentials_jsonb,
        'metadata', v_credentials_jsonb->'metadata'
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.get_user_credential(
    p_user_id UUID,
    p_module TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_user_credential(p_user_id, p_module);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_credential(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_credential(UUID, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- upsert_user_credential
-- ユーザーの認証情報を保存/更新（MCP Server 向け）
-- 旧名: update_module_token / upsert_service_token (統合)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_user_credential(
    p_user_id UUID,
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    -- UPSERT: INSERT or UPDATE
    INSERT INTO mcpist.user_credentials (user_id, module, credentials)
    VALUES (p_user_id, p_module, p_credentials::TEXT)
    ON CONFLICT (user_id, module)
    DO UPDATE SET
        credentials = p_credentials::TEXT,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.upsert_user_credential(
    p_user_id UUID,
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_user_credential(p_user_id, p_module, p_credentials);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_user_credential(UUID, TEXT, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_user_credential(UUID, TEXT, JSONB) TO service_role;

-- -----------------------------------------------------------------------------
-- sync_modules
-- Server startup時にGoサーバーから呼び出され、モジュールをDBに同期
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.sync_modules(p_modules TEXT[])
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_inserted INTEGER := 0;
    v_module TEXT;
BEGIN
    FOREACH v_module IN ARRAY p_modules LOOP
        INSERT INTO mcpist.modules (name, status)
        VALUES (v_module, 'active')
        ON CONFLICT (name) DO NOTHING;

        IF FOUND THEN
            v_inserted := v_inserted + 1;
        END IF;
    END LOOP;

    RETURN jsonb_build_object(
        'success', true,
        'inserted', v_inserted,
        'total', array_length(p_modules, 1)
    );
END;
$$;

COMMENT ON FUNCTION public.sync_modules(TEXT[]) IS 'Sync registered modules from Go server to database. Called on server startup.';
