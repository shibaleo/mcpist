-- =============================================================================
-- MCPist Database Schema and Tables
-- =============================================================================
-- This migration creates:
-- 1. mcpist schema
-- 2. All entitlement store tables (plans, users, subscriptions, etc.)
-- 3. Token vault tables (oauth_tokens, oauth_token_history)
-- 4. OAuth authorization codes table
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
-- Utility Functions
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -----------------------------------------------------------------------------
-- Plans Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    rate_limit_rpm INTEGER NOT NULL DEFAULT 60,
    rate_limit_burst INTEGER NOT NULL DEFAULT 10,
    quota_monthly INTEGER,
    credit_enabled BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO mcpist.plans (name, display_name, rate_limit_rpm, rate_limit_burst, quota_monthly, credit_enabled, is_active) VALUES
    ('free', 'Free', 10, 5, 1000, false, true),
    ('starter', 'Starter', 30, 10, 10000, false, false),
    ('pro', 'Pro', 60, 20, NULL, true, false),
    ('unlimited', 'Unlimited', 120, 50, NULL, true, false);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.plans
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- Users Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    display_name TEXT,
    avatar_url TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.users
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- Subscriptions Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES mcpist.plans(id),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'canceled', 'past_due', 'trialing')),
    psp_customer_id TEXT,
    psp_subscription_id TEXT,
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_subscriptions_user_id ON mcpist.subscriptions(user_id);
CREATE INDEX idx_subscriptions_psp_customer_id ON mcpist.subscriptions(psp_customer_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.subscriptions
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- Modules Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    requires_oauth BOOLEAN NOT NULL DEFAULT false,
    oauth_provider TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO mcpist.modules (name, display_name, description, requires_oauth, oauth_provider) VALUES
    ('notion', 'Notion', 'Notion pages and databases', true, 'notion'),
    ('github', 'GitHub', 'GitHub repositories and issues', true, 'github'),
    ('google_calendar', 'Google Calendar', 'Google Calendar events', true, 'google'),
    ('microsoft_todo', 'Microsoft To Do', 'Microsoft To Do tasks', true, 'microsoft');

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.modules
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- User Module Preferences Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.user_module_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module_id)
);

CREATE INDEX idx_user_module_preferences_user_id ON mcpist.user_module_preferences(user_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.user_module_preferences
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- Usage Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    period_start DATE NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, period_start)
);

CREATE INDEX idx_usage_user_id_period ON mcpist.usage(user_id, period_start);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.usage
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- Credits Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    balance INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_credits_user_id ON mcpist.credits(user_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.credits
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- Credit Transactions Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    amount INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('purchase', 'consume', 'refund', 'bonus', 'expire')),
    description TEXT,
    reference_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credit_transactions_user_id ON mcpist.credit_transactions(user_id);
CREATE INDEX idx_credit_transactions_created_at ON mcpist.credit_transactions(created_at);

-- -----------------------------------------------------------------------------
-- Tool Costs Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.tool_costs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    credit_cost INTEGER NOT NULL DEFAULT 1 CHECK (credit_cost >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(module_id, tool_name)
);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.tool_costs
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- MCP Tokens Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.mcp_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mcp_tokens_user_id ON mcpist.mcp_tokens(user_id);
CREATE INDEX idx_mcp_tokens_token_hash ON mcpist.mcp_tokens(token_hash);

-- -----------------------------------------------------------------------------
-- OAuth Tokens Table (Token Vault)
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    access_token_secret_id UUID,
    refresh_token_secret_id UUID,
    token_type TEXT DEFAULT 'Bearer',
    scope TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, service)
);

CREATE INDEX idx_oauth_tokens_user_id ON mcpist.oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_service ON mcpist.oauth_tokens(service);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();

-- -----------------------------------------------------------------------------
-- OAuth Token History Table (Audit Trail)
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.oauth_token_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    access_token_secret_id UUID,
    refresh_token_secret_id UUID,
    token_type TEXT DEFAULT 'Bearer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ,
    expired_reason TEXT,
    created_by_ip TEXT,
    expired_by_ip TEXT
);

CREATE INDEX idx_oauth_token_history_user_service ON mcpist.oauth_token_history(user_id, service);
CREATE INDEX idx_oauth_token_history_created_at ON mcpist.oauth_token_history(created_at DESC);

-- -----------------------------------------------------------------------------
-- OAuth Authorization Codes Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.oauth_authorization_codes (
    code TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    scope TEXT,
    state TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_codes_expires_at ON mcpist.oauth_authorization_codes(expires_at);
CREATE INDEX idx_oauth_codes_user_id ON mcpist.oauth_authorization_codes(user_id);

-- -----------------------------------------------------------------------------
-- Processed Webhook Events Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.processed_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_processed_webhook_events_event_id ON mcpist.processed_webhook_events(event_id);
