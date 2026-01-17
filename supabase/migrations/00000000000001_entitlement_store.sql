-- Entitlement Store Tables
-- Reference: spc-tbl.md

-- Ensure mcpist schema exists
CREATE SCHEMA IF NOT EXISTS mcpist;

-- Plans table (must be created first as it's referenced by subscriptions)
CREATE TABLE mcpist.plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    -- Rate limit settings
    rate_limit_rpm INTEGER NOT NULL DEFAULT 60,           -- Requests per minute
    rate_limit_burst INTEGER NOT NULL DEFAULT 10,         -- Burst allowance
    -- Quota settings
    quota_monthly INTEGER,                                -- NULL = unlimited
    -- Credit settings
    credit_enabled BOOLEAN NOT NULL DEFAULT false,
    -- Metadata
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default plans
INSERT INTO mcpist.plans (name, display_name, rate_limit_rpm, rate_limit_burst, quota_monthly, credit_enabled, is_active) VALUES
    ('free', 'Free', 10, 5, 1000, false, true),
    ('starter', 'Starter', 30, 10, 10000, false, false),
    ('pro', 'Pro', 60, 20, NULL, true, false),
    ('unlimited', 'Unlimited', 120, 50, NULL, true, false);

-- Users table (extends auth.users)
CREATE TABLE mcpist.users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    display_name TEXT,
    avatar_url TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Subscriptions table
CREATE TABLE mcpist.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES mcpist.plans(id),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'canceled', 'past_due', 'trialing')),
    -- PSP (Payment Service Provider) integration
    psp_customer_id TEXT,                                 -- Stripe customer ID
    psp_subscription_id TEXT,                             -- Stripe subscription ID
    -- Period
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)                                       -- One subscription per user
);

-- Modules table
CREATE TABLE mcpist.modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,                            -- e.g., 'notion', 'github'
    display_name TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- Requirements
    requires_oauth BOOLEAN NOT NULL DEFAULT false,
    oauth_provider TEXT,                                  -- e.g., 'google', 'github'
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert initial modules
INSERT INTO mcpist.modules (name, display_name, description, requires_oauth, oauth_provider) VALUES
    ('notion', 'Notion', 'Notion pages and databases', true, 'notion'),
    ('github', 'GitHub', 'GitHub repositories and issues', true, 'github'),
    ('google_calendar', 'Google Calendar', 'Google Calendar events', true, 'google'),
    ('microsoft_todo', 'Microsoft To Do', 'Microsoft To Do tasks', true, 'microsoft');

-- User module preferences
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

-- Usage tracking (monthly quota)
CREATE TABLE mcpist.usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    period_start DATE NOT NULL,                           -- First day of month
    request_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, period_start)
);

-- Credits balance
CREATE TABLE mcpist.credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    balance INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Credit transactions (audit log)
CREATE TABLE mcpist.credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    amount INTEGER NOT NULL,                              -- Positive = add, Negative = consume
    balance_after INTEGER NOT NULL,
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('purchase', 'consume', 'refund', 'bonus', 'expire')),
    description TEXT,
    reference_id TEXT,                                    -- e.g., tool call ID, Stripe payment ID
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tool costs (credit consumption per tool)
CREATE TABLE mcpist.tool_costs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    credit_cost INTEGER NOT NULL DEFAULT 1 CHECK (credit_cost >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(module_id, tool_name)
);

-- MCP tokens (long-lived tokens for MCP connections)
CREATE TABLE mcpist.mcp_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,                                   -- User-defined name
    token_hash TEXT NOT NULL UNIQUE,                      -- SHA-256 hash of token
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Processed webhook events (idempotency)
CREATE TABLE mcpist.processed_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL UNIQUE,                        -- PSP event ID
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_subscriptions_user_id ON mcpist.subscriptions(user_id);
CREATE INDEX idx_subscriptions_psp_customer_id ON mcpist.subscriptions(psp_customer_id);
CREATE INDEX idx_user_module_preferences_user_id ON mcpist.user_module_preferences(user_id);
CREATE INDEX idx_usage_user_id_period ON mcpist.usage(user_id, period_start);
CREATE INDEX idx_credits_user_id ON mcpist.credits(user_id);
CREATE INDEX idx_credit_transactions_user_id ON mcpist.credit_transactions(user_id);
CREATE INDEX idx_credit_transactions_created_at ON mcpist.credit_transactions(created_at);
CREATE INDEX idx_mcp_tokens_user_id ON mcpist.mcp_tokens(user_id);
CREATE INDEX idx_mcp_tokens_token_hash ON mcpist.mcp_tokens(token_hash);
CREATE INDEX idx_processed_webhook_events_event_id ON mcpist.processed_webhook_events(event_id);

-- Updated at trigger function
CREATE OR REPLACE FUNCTION mcpist.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.plans FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.users FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.subscriptions FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.modules FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.user_module_preferences FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.usage FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.credits FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.tool_costs FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
