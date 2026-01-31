-- =============================================================================
-- MCPist Database Schema and Enums
-- =============================================================================
-- This migration creates:
-- 1. mcpist schema
-- 2. Enum types
-- 3. Utility functions
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Schema Setup
-- -----------------------------------------------------------------------------

CREATE SCHEMA IF NOT EXISTS mcpist;

GRANT USAGE ON SCHEMA mcpist TO postgres, anon, authenticated, service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcpist
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO postgres, service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcpist
GRANT SELECT ON TABLES TO anon, authenticated;

-- -----------------------------------------------------------------------------
-- Enum Types
-- -----------------------------------------------------------------------------

CREATE TYPE mcpist.account_status AS ENUM (
    'pre_active',  -- オンボーディング待ち（サインアップ直後）
    'active',      -- アクティブ
    'suspended',   -- 一時停止
    'disabled'     -- 無効化
);

CREATE TYPE mcpist.module_status AS ENUM (
    'active',       -- 利用可能
    'coming_soon',  -- 近日公開
    'maintenance',  -- メンテナンス中
    'beta',         -- ベータ版
    'deprecated',   -- 非推奨
    'disabled'      -- 無効
);

CREATE TYPE mcpist.credit_transaction_type AS ENUM (
    'consume',       -- クレジット消費
    'purchase',      -- クレジット購入
    'monthly_reset', -- 月次リセット
    'bonus'          -- ボーナス付与（サインアップボーナス等）
);

-- -----------------------------------------------------------------------------
-- Utility Functions
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Enable pgcrypto extension for gen_random_bytes (API key generation)
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA extensions;
-- =============================================================================
-- MCPist Core Tables
-- =============================================================================
-- This migration creates all core tables:
-- 1. users
-- 2. credits
-- 3. credit_transactions
-- 4. modules
-- 5. module_settings
-- 6. tool_settings
-- 7. prompts
-- 8. api_keys
-- 9. processed_webhook_events
--
-- Note: service_tokens table removed (replaced by user_credentials with pgsodium TCE)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Users Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    account_status mcpist.account_status NOT NULL DEFAULT 'pre_active',
    display_name TEXT,
    avatar_url TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_account_status ON mcpist.users(account_status);

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON mcpist.users
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();

-- -----------------------------------------------------------------------------
-- Credits Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.credits (
    user_id UUID PRIMARY KEY REFERENCES mcpist.users(id) ON DELETE CASCADE,
    free_credits INTEGER NOT NULL DEFAULT 0 CHECK (free_credits >= 0 AND free_credits <= 1000),
    paid_credits INTEGER NOT NULL DEFAULT 0 CHECK (paid_credits >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_credits_updated_at
    BEFORE UPDATE ON mcpist.credits
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();

-- -----------------------------------------------------------------------------
-- Credit Transactions Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    type mcpist.credit_transaction_type NOT NULL,
    amount INTEGER NOT NULL,
    credit_type TEXT CHECK (credit_type IN ('free', 'paid')),
    module TEXT,
    tool TEXT,
    request_id TEXT,
    task_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credit_transactions_user_id ON mcpist.credit_transactions(user_id);
CREATE INDEX idx_credit_transactions_created_at ON mcpist.credit_transactions(created_at DESC);
CREATE INDEX idx_credit_transactions_type ON mcpist.credit_transactions(type);
CREATE INDEX idx_credit_transactions_request_id ON mcpist.credit_transactions(request_id) WHERE request_id IS NOT NULL;

-- 冪等性のためのUNIQUE制約（consume時のみ使用）
CREATE UNIQUE INDEX idx_credit_transactions_idempotency
    ON mcpist.credit_transactions(user_id, request_id, COALESCE(task_id, ''))
    WHERE request_id IS NOT NULL;

-- -----------------------------------------------------------------------------
-- Modules Table (Master Data)
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    status mcpist.module_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_modules_status ON mcpist.modules(status);

-- -----------------------------------------------------------------------------
-- Module Settings Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.module_settings (
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id)
);

COMMENT ON COLUMN mcpist.module_settings.description IS 'User-defined additional description for the module';

CREATE INDEX idx_module_settings_user_id ON mcpist.module_settings(user_id);

-- -----------------------------------------------------------------------------
-- Tool Settings Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.tool_settings (
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE RESTRICT,
    tool_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id, tool_id)
);

COMMENT ON COLUMN mcpist.tool_settings.tool_id IS 'Tool ID in format: {module}:{tool_name} (e.g., notion:search)';

CREATE INDEX idx_tool_settings_user_module ON mcpist.tool_settings(user_id, module_id);

-- -----------------------------------------------------------------------------
-- Prompts Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID REFERENCES mcpist.modules(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, module_id, name)
);

CREATE INDEX idx_prompts_user_id ON mcpist.prompts(user_id);
CREATE INDEX idx_prompts_user_module ON mcpist.prompts(user_id, module_id);

CREATE TRIGGER set_prompts_updated_at
    BEFORE UPDATE ON mcpist.prompts
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();

-- -----------------------------------------------------------------------------
-- API Keys Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    name TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user_id ON mcpist.api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON mcpist.api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_expires_at ON mcpist.api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- -----------------------------------------------------------------------------
-- Processed Webhook Events Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.processed_webhook_events (
    event_id TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_processed_webhook_events_user_id ON mcpist.processed_webhook_events(user_id);
CREATE INDEX idx_processed_webhook_events_processed_at ON mcpist.processed_webhook_events(processed_at DESC);
-- =============================================================================
-- MCPist User Credentials Table (pgsodium TCE)
-- =============================================================================
-- ユーザーのモジュール認証情報を pgsodium Transparent Column Encryption で暗号化保存
--
-- 旧: service_tokens + vault.secrets
-- 新: user_credentials (pgsodium TCE)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- User Credentials Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.user_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module TEXT NOT NULL,
    credentials TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module)
);

CREATE INDEX idx_user_credentials_user_id ON mcpist.user_credentials(user_id);

CREATE TRIGGER set_user_credentials_updated_at
    BEFORE UPDATE ON mcpist.user_credentials
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();

COMMENT ON TABLE mcpist.user_credentials IS 'User credentials for external services (OAuth tokens, API keys, etc.) - encrypted with pgsodium TCE';
COMMENT ON COLUMN mcpist.user_credentials.module IS 'Module name (e.g., google, microsoft, jira, notion)';
COMMENT ON COLUMN mcpist.user_credentials.credentials IS 'JSON-encoded credentials (encrypted with pgsodium TCE)';

-- -----------------------------------------------------------------------------
-- pgsodium Transparent Column Encryption Setup
-- -----------------------------------------------------------------------------
-- Note: pgsodium TCE requires the following setup:
-- 1. Create an encryption key (done once per database)
-- 2. Apply SECURITY LABEL to the column
--
-- This must be done after the table is created, and the key_id must be known.
-- The actual encryption setup will be done in a separate step after migration.
-- -----------------------------------------------------------------------------

-- Placeholder comment for TCE setup (to be configured manually or via seed script):
-- SELECT pgsodium.create_key(name := 'user_credentials_key', key_type := 'aead-det');
-- SECURITY LABEL FOR pgsodium ON COLUMN mcpist.user_credentials.credentials
--     IS 'ENCRYPT WITH KEY ID <key_id> ASSOCIATED (id, user_id)';
-- =============================================================================
-- MCPist Row Level Security Policies
-- =============================================================================
-- This migration enables RLS and creates policies for all tables
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Enable RLS
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.credits ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.credit_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.module_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.tool_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.user_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.processed_webhook_events ENABLE ROW LEVEL SECURITY;

-- -----------------------------------------------------------------------------
-- Users Policies
-- -----------------------------------------------------------------------------

CREATE POLICY users_select ON mcpist.users
    FOR SELECT USING (auth.uid() = id);

CREATE POLICY users_update ON mcpist.users
    FOR UPDATE USING (auth.uid() = id);

-- -----------------------------------------------------------------------------
-- Credits Policies
-- -----------------------------------------------------------------------------

CREATE POLICY credits_select ON mcpist.credits
    FOR SELECT USING (auth.uid() = user_id);

-- credits UPDATE is done via RPC with service_role

-- -----------------------------------------------------------------------------
-- Credit Transactions Policies
-- -----------------------------------------------------------------------------

CREATE POLICY credit_transactions_select ON mcpist.credit_transactions
    FOR SELECT USING (auth.uid() = user_id);

-- credit_transactions INSERT is done via RPC with service_role

-- -----------------------------------------------------------------------------
-- Modules Policies (public read)
-- -----------------------------------------------------------------------------

CREATE POLICY modules_select ON mcpist.modules
    FOR SELECT USING (true);

-- -----------------------------------------------------------------------------
-- Module Settings Policies
-- -----------------------------------------------------------------------------

CREATE POLICY module_settings_select ON mcpist.module_settings
    FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY module_settings_insert ON mcpist.module_settings
    FOR INSERT WITH CHECK (auth.uid() = user_id);

CREATE POLICY module_settings_update ON mcpist.module_settings
    FOR UPDATE USING (auth.uid() = user_id);

CREATE POLICY module_settings_delete ON mcpist.module_settings
    FOR DELETE USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- Tool Settings Policies
-- -----------------------------------------------------------------------------

CREATE POLICY tool_settings_select ON mcpist.tool_settings
    FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY tool_settings_insert ON mcpist.tool_settings
    FOR INSERT WITH CHECK (auth.uid() = user_id);

CREATE POLICY tool_settings_update ON mcpist.tool_settings
    FOR UPDATE USING (auth.uid() = user_id);

CREATE POLICY tool_settings_delete ON mcpist.tool_settings
    FOR DELETE USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- Prompts Policies
-- -----------------------------------------------------------------------------

CREATE POLICY prompts_select ON mcpist.prompts
    FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY prompts_insert ON mcpist.prompts
    FOR INSERT WITH CHECK (auth.uid() = user_id);

CREATE POLICY prompts_update ON mcpist.prompts
    FOR UPDATE USING (auth.uid() = user_id);

CREATE POLICY prompts_delete ON mcpist.prompts
    FOR DELETE USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- API Keys Policies
-- -----------------------------------------------------------------------------

CREATE POLICY api_keys_select ON mcpist.api_keys
    FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY api_keys_insert ON mcpist.api_keys
    FOR INSERT WITH CHECK (auth.uid() = user_id);

CREATE POLICY api_keys_update ON mcpist.api_keys
    FOR UPDATE USING (auth.uid() = user_id);

CREATE POLICY api_keys_delete ON mcpist.api_keys
    FOR DELETE USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- User Credentials Policies
-- -----------------------------------------------------------------------------
-- SELECT: authenticated users can read their own credentials
-- INSERT/UPDATE/DELETE: done via RPC with service_role (SECURITY DEFINER)

CREATE POLICY user_credentials_select ON mcpist.user_credentials
    FOR SELECT USING (auth.uid() = user_id);

-- service_role can do everything (for RPC functions)
CREATE POLICY user_credentials_service_role ON mcpist.user_credentials
    FOR ALL TO service_role USING (true) WITH CHECK (true);

-- -----------------------------------------------------------------------------
-- Processed Webhook Events Policies
-- -----------------------------------------------------------------------------

-- processed_webhook_events INSERT is done via RPC with service_role
-- No direct user access needed
-- =============================================================================
-- MCPist User Management Triggers
-- =============================================================================
-- This migration creates triggers for user creation
-- New users are created with 'pre_active' status and 0 credits.
-- Onboarding completion grants credits and sets status to 'active'.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Handle New User (triggered when auth.users row is created)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    -- Create user record with pre_active status
    INSERT INTO mcpist.users (id, account_status)
    VALUES (NEW.id, 'pre_active'::mcpist.account_status);

    -- Create credits record with 0 credits (granted on onboarding completion)
    INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
    VALUES (NEW.id, 0, 0);

    RETURN NEW;
END;
$$;

-- Trigger on auth.users
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.handle_new_user();
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
-- =============================================================================
-- MCPist RPC Functions for Console Frontend
-- =============================================================================
-- This migration creates RPC functions used by Console (authenticated):
-- 1. generate_my_api_key - APIキー生成
-- 2. list_my_api_keys - APIキー一覧取得
-- 3. revoke_my_api_key - APIキー削除（論理削除）
-- 4. list_my_credentials - サービス接続一覧
-- 5. upsert_my_credential - サービストークン登録/更新
-- 6. delete_my_credential - サービストークン削除
-- 7. get_my_role - ユーザーロール取得
-- 8. get_my_preferences - ユーザー設定取得
-- 9. update_my_preferences - ユーザー設定更新
-- 10. get_my_tool_settings - ツール設定取得
-- 11. upsert_my_tool_settings - ツール設定更新
-- 12. get_my_module_descriptions - モジュール説明取得
-- 13. upsert_my_module_description - モジュール説明更新
-- 14. list_modules - モジュール一覧取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- generate_my_api_key
-- APIキーを生成（キーは生成時のみ返される）
-- 旧名: generate_api_key
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.generate_my_api_key(
    p_display_name TEXT,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
DECLARE
    v_user_id UUID;
    v_key TEXT;
    v_key_hash TEXT;
    v_key_prefix TEXT;
    v_key_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- キー生成（mpt_ + 32文字のランダム16進数）
    v_key := 'mpt_' || encode(gen_random_bytes(16), 'hex');
    v_key_prefix := substring(v_key from 1 for 8) || '...' || substring(v_key from length(v_key) - 3 for 4);
    v_key_hash := encode(sha256(v_key::bytea), 'hex');

    -- 挿入
    INSERT INTO mcpist.api_keys (user_id, name, key_hash, key_prefix, expires_at)
    VALUES (v_user_id, p_display_name, v_key_hash, v_key_prefix, p_expires_at)
    RETURNING id INTO v_key_id;

    RETURN jsonb_build_object(
        'api_key', v_key,
        'key_prefix', v_key_prefix
    );
END;
$$;

GRANT EXECUTE ON FUNCTION public.generate_my_api_key(TEXT, TIMESTAMPTZ) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_my_api_keys
-- ユーザーのAPIキー一覧を取得
-- 旧名: list_api_keys
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_my_api_keys()
RETURNS TABLE (
    id UUID,
    key_prefix TEXT,
    display_name TEXT,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        k.id,
        k.key_prefix,
        k.name AS display_name,
        k.expires_at,
        k.last_used_at,
        k.revoked_at
    FROM mcpist.api_keys k
    WHERE k.user_id = auth.uid()
      AND k.revoked_at IS NULL
    ORDER BY k.created_at DESC;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_my_api_keys() TO authenticated;

-- -----------------------------------------------------------------------------
-- revoke_my_api_key
-- APIキーを論理削除
-- 旧名: revoke_api_key
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.revoke_my_api_key(p_key_id UUID)
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

    UPDATE mcpist.api_keys
    SET revoked_at = NOW()
    WHERE id = p_key_id
      AND user_id = v_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('success', v_affected > 0);
END;
$$;

GRANT EXECUTE ON FUNCTION public.revoke_my_api_key(UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_my_credentials
-- サービス接続一覧を取得
-- 旧名: list_service_connections
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_my_credentials()
RETURNS TABLE (
    module TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        uc.module,
        uc.created_at,
        uc.updated_at
    FROM mcpist.user_credentials uc
    WHERE uc.user_id = auth.uid()
    ORDER BY uc.module;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_my_credentials() TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_credential
-- サービストークンを登録/更新
-- 旧名: upsert_service_token
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.upsert_my_credential(
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- UPSERT
    INSERT INTO mcpist.user_credentials (user_id, module, credentials)
    VALUES (v_user_id, p_module, p_credentials::TEXT)
    ON CONFLICT (user_id, module)
    DO UPDATE SET
        credentials = p_credentials::TEXT,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true,
        'module', p_module
    );
END;
$$;

GRANT EXECUTE ON FUNCTION public.upsert_my_credential(TEXT, JSONB) TO authenticated;

-- -----------------------------------------------------------------------------
-- delete_my_credential
-- サービストークンを削除
-- 旧名: delete_service_token
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.delete_my_credential(p_module TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_affected INTEGER;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    DELETE FROM mcpist.user_credentials
    WHERE user_id = v_user_id AND module = p_module;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('success', v_affected > 0);
END;
$$;

GRANT EXECUTE ON FUNCTION public.delete_my_credential(TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_role
-- 現在ログインしているユーザーのロールを取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.get_my_role()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id UUID;
    v_role TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RETURN jsonb_build_object('role', NULL);
    END IF;

    SELECT COALESCE(
        raw_app_meta_data->>'role',
        'user'
    ) INTO v_role
    FROM auth.users
    WHERE id = v_user_id;

    RETURN jsonb_build_object('role', v_role);
END;
$$;

GRANT EXECUTE ON FUNCTION public.get_my_role() TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_settings
-- Get current user's settings
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_settings()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_settings JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT settings INTO v_settings
    FROM mcpist.users
    WHERE id = v_user_id;

    RETURN COALESCE(v_settings, '{}'::JSONB);
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_settings()
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_settings();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_settings() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_settings() TO authenticated;

-- -----------------------------------------------------------------------------
-- update_my_settings
-- Update current user's settings (merge with existing)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.update_my_settings(p_settings JSONB)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_current JSONB;
    v_updated JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT COALESCE(settings, '{}'::JSONB) INTO v_current
    FROM mcpist.users
    WHERE id = v_user_id;

    v_updated := v_current || p_settings;

    UPDATE mcpist.users
    SET settings = v_updated
    WHERE id = v_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'settings', v_updated
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.update_my_settings(p_settings JSONB)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.update_my_settings(p_settings);
$$;

GRANT EXECUTE ON FUNCTION mcpist.update_my_settings(JSONB) TO authenticated;
GRANT EXECUTE ON FUNCTION public.update_my_settings(JSONB) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_tool_settings
-- ユーザーのツール設定を取得
-- -----------------------------------------------------------------------------

CREATE FUNCTION mcpist.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ts.tool_id,
        ts.enabled
    FROM mcpist.tool_settings ts
    JOIN mcpist.modules m ON m.id = ts.module_id
    WHERE ts.user_id = auth.uid()
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY m.name, ts.tool_id;
END;
$$;

CREATE FUNCTION public.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_tool_settings(p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_tool_settings(TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_tool_settings(TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_tool_settings
-- ツール設定を一括更新（モジュール単位）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_tool_settings(
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    -- 有効ツールをUPSERT
    IF p_enabled_tools IS NOT NULL AND array_length(p_enabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT v_user_id, v_module_id, unnest(p_enabled_tools), true
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = true;
    END IF;

    -- 無効ツールをUPSERT
    IF p_disabled_tools IS NOT NULL AND array_length(p_disabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT v_user_id, v_module_id, unnest(p_disabled_tools), false
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = false;
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'enabled_count', COALESCE(array_length(p_enabled_tools, 1), 0),
        'disabled_count', COALESCE(array_length(p_disabled_tools, 1), 0)
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_tool_settings(
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_tool_settings(p_module_name, p_enabled_tools, p_disabled_tools);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_module_descriptions
-- ユーザーのモジュール説明を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ms.description
    FROM mcpist.module_settings ms
    JOIN mcpist.modules m ON m.id = ms.module_id
    WHERE ms.user_id = auth.uid()
      AND ms.description IS NOT NULL
      AND ms.description != ''
    ORDER BY m.name;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_module_descriptions();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_module_descriptions() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_module_descriptions() TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_module_description
-- モジュール説明を更新
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    INSERT INTO mcpist.module_settings (user_id, module_id, enabled, description)
    VALUES (v_user_id, v_module_id, true, p_description)
    ON CONFLICT (user_id, module_id)
    DO UPDATE SET description = p_description;

    RETURN jsonb_build_object('success', true, 'module', p_module_name);
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_module_description(p_module_name, p_description);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_module_description(TEXT, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_module_description(TEXT, TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_modules
-- 利用可能なモジュール一覧を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_modules()
RETURNS TABLE (
    id UUID,
    name TEXT,
    status TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.id,
        m.name,
        m.status::TEXT,
        m.created_at
    FROM mcpist.modules m
    WHERE m.status IN ('active', 'beta')
    ORDER BY m.name;
END;
$$;

GRANT EXECUTE ON FUNCTION public.list_modules() TO authenticated;
GRANT EXECUTE ON FUNCTION public.list_modules() TO anon;
-- =============================================================================
-- MCPist: OAuth Apps Table and RPCs
-- =============================================================================
-- OAuth プロバイダー（Google, Microsoft等）のクライアント認証情報を管理
-- クライアントID/Secretは Vault に暗号化保存（運営シークレット）
--
-- Note: ユーザートークンは user_credentials テーブルに保存（pgsodium TCE）
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

    SELECT decrypted_secret::JSONB INTO v_credentials
    FROM vault.decrypted_secrets
    WHERE id = v_secret_id;

    IF v_credentials IS NULL THEN
        RETURN jsonb_build_object(
            'error', 'secret_not_found',
            'message', 'Credentials not found in vault for provider: ' || p_provider
        );
    END IF;

    RETURN jsonb_build_object(
        'provider', p_provider,
        'client_id', v_credentials->>'client_id',
        'client_secret', v_credentials->>'client_secret',
        'redirect_uri', v_redirect_uri
    );
END;
$$;

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
    v_orphan_secret_id UUID;
    v_new_secret_id UUID;
    v_credentials TEXT;
    v_secret_name TEXT;
BEGIN
    v_secret_name := 'oauth_app_' || p_provider;

    -- oauth_apps テーブルから既存レコードを検索
    SELECT oa.secret_id INTO v_existing_secret_id
    FROM mcpist.oauth_apps oa
    WHERE oa.provider = p_provider;

    v_credentials := jsonb_build_object(
        'client_id', p_client_id,
        'client_secret', p_client_secret
    )::TEXT;

    IF v_existing_secret_id IS NOT NULL THEN
        -- 既存シークレットを更新（vault.update_secret を使用）
        PERFORM vault.update_secret(
            v_existing_secret_id,
            v_credentials,
            NULL,  -- scope (not changed)
            v_secret_name,
            'OAuth client credentials for ' || p_provider
        );

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
        -- 孤立したシークレット（oauth_appsに紐づいていないが同名のもの）を検索
        SELECT id INTO v_orphan_secret_id
        FROM vault.secrets
        WHERE name = v_secret_name;

        IF v_orphan_secret_id IS NOT NULL THEN
            -- 孤立したシークレットを更新して再利用
            PERFORM vault.update_secret(
                v_orphan_secret_id,
                v_credentials,
                NULL,
                v_secret_name,
                'OAuth client credentials for ' || p_provider
            );
            v_new_secret_id := v_orphan_secret_id;
        ELSE
            -- 新規シークレットを作成（vault.create_secret を使用）
            v_new_secret_id := vault.create_secret(
                v_credentials,
                v_secret_name,
                'OAuth client credentials for ' || p_provider
            );
        END IF;

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
-- Admin Console から呼び出し: OAuthアプリ設定一覧を取得
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

    DELETE FROM mcpist.oauth_apps WHERE provider = p_provider;

    -- vault.delete_secret を使用してシークレットを削除
    PERFORM vault.delete_secret(v_secret_id);

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

GRANT EXECUTE ON FUNCTION mcpist.delete_oauth_app(TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.delete_oauth_app(TEXT) TO service_role;
-- =============================================================================
-- Migration: Stripe Integration
-- =============================================================================
-- Stripe連携機能:
-- 1. stripe_customer_id カラム追加
-- 2. add_user_credits - 統合クレジット追加RPC
-- 3. complete_user_onboarding - オンボーディング完了RPC
-- 4. link_stripe_customer - StripeカスタマーID紐付け
-- 5. get_user_by_stripe_customer - StripeカスタマーIDからユーザー取得
-- 6. get_stripe_customer_id - ユーザーのStripeカスタマーID取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- stripe_customer_id カラム追加
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.users
ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id
ON mcpist.users(stripe_customer_id)
WHERE stripe_customer_id IS NOT NULL;

COMMENT ON COLUMN mcpist.users.stripe_customer_id IS 'Stripe Customer ID (cus_xxx) for payment integration';

-- -----------------------------------------------------------------------------
-- add_user_credits RPC
-- クレジット追加（free/paid統合、冪等性保証）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.add_user_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,  -- 'free' or 'paid'
    p_event_id TEXT      -- idempotency key
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_new_free_credits INTEGER;
    v_new_paid_credits INTEGER;
BEGIN
    -- Validate credit_type
    IF p_credit_type NOT IN ('free', 'paid') THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_credit_type',
            'message', 'credit_type must be "free" or "paid"'
        );
    END IF;

    -- Check if event already processed (idempotency)
    IF EXISTS (
        SELECT 1 FROM mcpist.processed_webhook_events
        WHERE event_id = p_event_id
    ) THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'event_already_processed',
            'message', 'This event has already been processed'
        );
    END IF;

    -- Validate amount
    IF p_amount <= 0 THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_amount',
            'message', 'Amount must be positive'
        );
    END IF;

    -- Add credits based on type
    IF p_credit_type = 'free' THEN
        UPDATE mcpist.credits
        SET
            free_credits = free_credits + p_amount,
            updated_at = NOW()
        WHERE user_id = p_user_id
        RETURNING free_credits, paid_credits INTO v_new_free_credits, v_new_paid_credits;
    ELSE
        UPDATE mcpist.credits
        SET
            paid_credits = paid_credits + p_amount,
            updated_at = NOW()
        WHERE user_id = p_user_id
        RETURNING free_credits, paid_credits INTO v_new_free_credits, v_new_paid_credits;
    END IF;

    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found',
            'message', 'User not found'
        );
    END IF;

    -- Record transaction
    INSERT INTO mcpist.credit_transactions (
        user_id,
        type,
        amount,
        credit_type,
        request_id
    ) VALUES (
        p_user_id,
        (CASE WHEN p_credit_type = 'free' THEN 'bonus' ELSE 'purchase' END)::mcpist.credit_transaction_type,
        p_amount,
        p_credit_type,
        p_event_id
    );

    -- Mark event as processed
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id, processed_at)
    VALUES (p_event_id, p_user_id, NOW());

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', v_new_free_credits,
        'paid_credits', v_new_paid_credits
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.add_user_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.add_user_credits(p_user_id, p_amount, p_credit_type, p_event_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.add_user_credits(UUID, INTEGER, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.add_user_credits(UUID, INTEGER, TEXT, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- complete_user_onboarding RPC
-- オンボーディング完了（pre_active → active + 初期クレジット付与）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.complete_user_onboarding(
    p_user_id UUID,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_current_status mcpist.account_status;
    v_result JSONB;
BEGIN
    -- Check current status
    SELECT account_status INTO v_current_status
    FROM mcpist.users
    WHERE id = p_user_id;

    -- If already active, return success (idempotent)
    IF v_current_status = 'active' THEN
        RETURN jsonb_build_object(
            'success', true,
            'already_completed', true,
            'message', 'Onboarding already completed'
        );
    END IF;

    -- If not pre_active, something is wrong
    IF v_current_status IS NULL OR v_current_status != 'pre_active' THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'invalid_status',
            'message', 'User is not in pre_active status'
        );
    END IF;

    -- Grant signup bonus credits using add_user_credits (handles idempotency)
    SELECT mcpist.add_user_credits(p_user_id, 100, 'free', p_event_id) INTO v_result;

    IF NOT (v_result->>'success')::boolean THEN
        IF v_result->>'error' = 'event_already_processed' THEN
            NULL;  -- Continue to update status
        ELSE
            RETURN v_result;
        END IF;
    END IF;

    -- Update status to active
    UPDATE mcpist.users
    SET account_status = 'active'::mcpist.account_status,
        updated_at = NOW()
    WHERE id = p_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'free_credits', 100
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.complete_user_onboarding(
    p_user_id UUID,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.complete_user_onboarding(p_user_id, p_event_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.complete_user_onboarding(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.complete_user_onboarding(UUID, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- link_stripe_customer RPC
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.link_stripe_customer(
    p_user_id UUID,
    p_stripe_customer_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    UPDATE mcpist.users
    SET stripe_customer_id = p_stripe_customer_id
    WHERE id = p_user_id;

    IF NOT FOUND THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'user_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'stripe_customer_id', p_stripe_customer_id
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.link_stripe_customer(
    p_user_id UUID,
    p_stripe_customer_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.link_stripe_customer(p_user_id, p_stripe_customer_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.link_stripe_customer(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.link_stripe_customer(UUID, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_user_by_stripe_customer RPC
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_user_by_stripe_customer(
    p_stripe_customer_id TEXT
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
BEGIN
    SELECT id INTO v_user_id
    FROM mcpist.users
    WHERE stripe_customer_id = p_stripe_customer_id;

    RETURN v_user_id;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_user_by_stripe_customer(p_stripe_customer_id TEXT)
RETURNS UUID
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_user_by_stripe_customer(p_stripe_customer_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_by_stripe_customer(TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_by_stripe_customer(TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- get_stripe_customer_id RPC
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_stripe_customer_id(p_user_id UUID)
RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_stripe_customer_id TEXT;
BEGIN
    SELECT stripe_customer_id INTO v_stripe_customer_id
    FROM mcpist.users
    WHERE id = p_user_id;

    RETURN v_stripe_customer_id;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_stripe_customer_id(p_user_id UUID)
RETURNS TEXT
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_stripe_customer_id(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_stripe_customer_id(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_stripe_customer_id(UUID) TO service_role;
-- =============================================================================
-- MCPist RPC Functions for Prompts
-- =============================================================================
-- Console (User) 向けプロンプト管理RPC:
-- 1. list_my_prompts - プロンプト一覧取得
-- 2. get_my_prompt - プロンプト詳細取得
-- 3. upsert_my_prompt - プロンプト作成/更新
-- 4. delete_my_prompt - プロンプト削除
-- =============================================================================

-- -----------------------------------------------------------------------------
-- list_my_prompts
-- 認証ユーザーのプロンプト一覧を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.list_my_prompts(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    content TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    RETURN QUERY
    SELECT
        p.id,
        m.name AS module_name,
        p.name,
        p.content,
        p.created_at,
        p.updated_at
    FROM mcpist.prompts p
    LEFT JOIN mcpist.modules m ON m.id = p.module_id
    WHERE p.user_id = v_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY p.updated_at DESC;
END;
$$;

CREATE OR REPLACE FUNCTION public.list_my_prompts(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    content TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_my_prompts(p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_my_prompts(TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.list_my_prompts(TEXT) TO authenticated;

-- -----------------------------------------------------------------------------
-- get_my_prompt
-- 認証ユーザーのプロンプト詳細を取得
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_prompt RECORD;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT
        p.id,
        m.name AS module_name,
        p.name,
        p.content,
        p.created_at,
        p.updated_at
    INTO v_prompt
    FROM mcpist.prompts p
    LEFT JOIN mcpist.modules m ON m.id = p.module_id
    WHERE p.id = p_prompt_id AND p.user_id = v_user_id;

    IF v_prompt IS NULL THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'prompt_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'found', true,
        'id', v_prompt.id,
        'module_name', v_prompt.module_name,
        'name', v_prompt.name,
        'content', v_prompt.content,
        'created_at', v_prompt.created_at,
        'updated_at', v_prompt.updated_at
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_prompt(p_prompt_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_prompt(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_prompt(UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- upsert_my_prompt
-- 認証ユーザーのプロンプトを作成/更新
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.upsert_my_prompt(
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_module_id UUID;
    v_result_id UUID;
    v_action TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- モジュール名からIDを取得（指定された場合）
    IF p_module_name IS NOT NULL THEN
        SELECT id INTO v_module_id
        FROM mcpist.modules
        WHERE name = p_module_name AND status IN ('active', 'beta');

        IF v_module_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'module_not_found'
            );
        END IF;
    END IF;

    -- 更新の場合
    IF p_prompt_id IS NOT NULL THEN
        UPDATE mcpist.prompts
        SET
            name = p_name,
            content = p_content,
            module_id = v_module_id
        WHERE id = p_prompt_id AND user_id = v_user_id
        RETURNING id INTO v_result_id;

        IF v_result_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'prompt_not_found'
            );
        END IF;

        v_action := 'updated';
    ELSE
        -- 新規作成
        INSERT INTO mcpist.prompts (user_id, module_id, name, content)
        VALUES (v_user_id, v_module_id, p_name, p_content)
        ON CONFLICT (user_id, module_id, name) DO UPDATE
        SET content = EXCLUDED.content
        RETURNING id INTO v_result_id;

        v_action := 'created';
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'id', v_result_id,
        'action', v_action
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_prompt(
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_prompt(p_name, p_content, p_module_name, p_prompt_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_prompt(TEXT, TEXT, TEXT, UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_prompt(TEXT, TEXT, TEXT, UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- delete_my_prompt
-- 認証ユーザーのプロンプトを削除
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.delete_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_deleted_id UUID;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    DELETE FROM mcpist.prompts
    WHERE id = p_prompt_id AND user_id = v_user_id
    RETURNING id INTO v_deleted_id;

    IF v_deleted_id IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'prompt_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'deleted_id', v_deleted_id
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.delete_my_prompt(p_prompt_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_my_prompt(p_prompt_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.delete_my_prompt(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION public.delete_my_prompt(UUID) TO authenticated;
-- =============================================================================
-- MCPist RPC Functions for OAuth Consents
-- =============================================================================
-- This migration creates RPC functions for OAuth consent management:
-- 1. list_my_oauth_consents - ユーザーのOAuthコンセント一覧を取得
-- 2. revoke_my_oauth_consent - OAuthコンセントを取り消し
-- 3. list_all_oauth_consents - 全ユーザーのOAuthコンセント一覧を取得（管理者用）
--
-- Note: These RPCs access auth.oauth_consents (Supabase OAuth Server internal table)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- list_my_oauth_consents
-- ユーザーのOAuthコンセント一覧を取得（auth.oauth_consentsから）
-- 旧名: list_oauth_consents
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.list_my_oauth_consents()
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

GRANT EXECUTE ON FUNCTION public.list_my_oauth_consents() TO authenticated;

-- -----------------------------------------------------------------------------
-- revoke_my_oauth_consent
-- OAuthコンセントを取り消し（論理削除）
-- 旧名: revoke_oauth_consent
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.revoke_my_oauth_consent(p_consent_id UUID)
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

GRANT EXECUTE ON FUNCTION public.revoke_my_oauth_consent(UUID) TO authenticated;

-- -----------------------------------------------------------------------------
-- list_all_oauth_consents (Admin only)
-- 全ユーザーのOAuthコンセント一覧を取得（管理者用）
-- Note: 管理者用なのでプレフィックスなし
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
-- =============================================================================
-- MCPist Admin Email Configuration
-- =============================================================================
-- This migration adds admin email management functionality.
-- Admin emails are stored in a config table and checked during user creation.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Admin Emails Table
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS mcpist.admin_emails (
    email TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE mcpist.admin_emails IS 'Emails that should be granted admin role on signup';

-- RLS Policy (service_role only)
ALTER TABLE mcpist.admin_emails ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role can manage admin_emails"
    ON mcpist.admin_emails
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- -----------------------------------------------------------------------------
-- Update handle_new_user trigger to check admin emails
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_is_admin BOOLEAN;
BEGIN
    -- Check if email is in admin_emails table
    SELECT EXISTS(
        SELECT 1 FROM mcpist.admin_emails WHERE email = NEW.email
    ) INTO v_is_admin;

    -- If admin email, update raw_app_meta_data to include admin role
    IF v_is_admin THEN
        UPDATE auth.users
        SET raw_app_meta_data = COALESCE(raw_app_meta_data, '{}'::jsonb) || '{"role": "admin"}'::jsonb
        WHERE id = NEW.id;
    END IF;

    -- Create user record with pre_active status
    INSERT INTO mcpist.users (id, account_status)
    VALUES (NEW.id, 'pre_active'::mcpist.account_status);

    -- Create credits record with 0 credits (granted on onboarding completion)
    INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
    VALUES (NEW.id, 0, 0);

    RETURN NEW;
END;
$$;
