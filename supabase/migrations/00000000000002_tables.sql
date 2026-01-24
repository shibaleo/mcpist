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
-- 9. service_tokens
-- 10. processed_webhook_events
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Users Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    account_status mcpist.account_status NOT NULL DEFAULT 'active',
    preferences JSONB DEFAULT '{}',
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
    free_credits INTEGER NOT NULL DEFAULT 1000 CHECK (free_credits >= 0 AND free_credits <= 1000),
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id)
);

CREATE INDEX idx_module_settings_user_id ON mcpist.module_settings(user_id);

-- -----------------------------------------------------------------------------
-- Tool Settings Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.tool_settings (
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE RESTRICT,
    tool_name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id, tool_name)
);

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
-- Service Tokens Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.service_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    credentials_secret_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, service)
);

CREATE INDEX idx_service_tokens_user_id ON mcpist.service_tokens(user_id);

CREATE TRIGGER set_service_tokens_updated_at
    BEFORE UPDATE ON mcpist.service_tokens
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();

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
