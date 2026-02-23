-- =============================================================================
-- MCPist Baseline Schema
-- =============================================================================
-- All tables for GORM direct connection (no PostgREST, no Supabase Auth).
-- Auth: Clerk (clerk_id on users)
-- API keys: JWT (jwt_kid on api_keys)
-- =============================================================================

-- Schema
CREATE SCHEMA IF NOT EXISTS mcpist;

-- =============================================================================
-- Tables
-- =============================================================================

-- Plans (master data)
CREATE TABLE mcpist.plans (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    daily_limit    INTEGER NOT NULL,
    price_monthly  INTEGER NOT NULL DEFAULT 0,
    stripe_price_id TEXT,
    features       JSONB NOT NULL DEFAULT '{}'
);

-- Users
CREATE TABLE mcpist.users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clerk_id           TEXT UNIQUE,
    account_status     TEXT NOT NULL DEFAULT 'active',
    plan_id            TEXT NOT NULL DEFAULT 'free' REFERENCES mcpist.plans(id),
    display_name       TEXT,
    avatar_url         TEXT,
    email              TEXT,
    role               TEXT NOT NULL DEFAULT 'user',
    stripe_customer_id TEXT UNIQUE,
    settings           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_clerk_id ON mcpist.users(clerk_id) WHERE clerk_id IS NOT NULL;
CREATE INDEX idx_users_stripe_customer_id ON mcpist.users(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;
CREATE INDEX idx_users_account_status ON mcpist.users(account_status);

-- Modules (master data, synced from Go Server)
CREATE TABLE mcpist.modules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    status     TEXT NOT NULL DEFAULT 'active',
    tools      JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_modules_status ON mcpist.modules(status);

-- Module Settings (per-user)
CREATE TABLE mcpist.module_settings (
    user_id     UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id   UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, module_id)
);

-- Tool Settings (per-user per-module)
CREATE TABLE mcpist.tool_settings (
    user_id    UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id  UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    tool_id    TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, module_id, tool_id)
);

-- Prompts (user-created MCP prompts)
CREATE TABLE mcpist.prompts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id   UUID REFERENCES mcpist.modules(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    description TEXT,
    content     TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_prompts_user_id ON mcpist.prompts(user_id);

-- API Keys (JWT-based)
CREATE TABLE mcpist.api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    jwt_kid      TEXT,
    key_prefix   TEXT NOT NULL,
    name         TEXT NOT NULL,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_user_id ON mcpist.api_keys(user_id);

-- User Credentials (OAuth tokens per module, encrypted)
CREATE TABLE mcpist.user_credentials (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module                 TEXT NOT NULL,
    encrypted_credentials  TEXT NOT NULL,
    key_version            INTEGER NOT NULL DEFAULT 1,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, module)
);

CREATE INDEX idx_user_credentials_user_module ON mcpist.user_credentials(user_id, module);

-- OAuth Apps (admin-managed provider configs)
CREATE TABLE mcpist.oauth_apps (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider                TEXT NOT NULL UNIQUE,
    client_id               TEXT NOT NULL,
    encrypted_client_secret TEXT,
    redirect_uri            TEXT NOT NULL,
    enabled                 BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Usage Log (tool execution records)
CREATE TABLE mcpist.usage_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    meta_tool  TEXT NOT NULL,
    request_id TEXT,
    details    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_log_user_id ON mcpist.usage_log(user_id);
CREATE INDEX idx_usage_log_created_at ON mcpist.usage_log(created_at DESC);
CREATE INDEX idx_usage_log_request_id ON mcpist.usage_log(request_id) WHERE request_id IS NOT NULL;

-- Processed Webhook Events (Stripe idempotency)
CREATE TABLE mcpist.processed_webhook_events (
    event_id     TEXT PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =============================================================================
-- Seed Data
-- =============================================================================

-- Plans
INSERT INTO mcpist.plans (id, name, daily_limit, price_monthly, features) VALUES
    ('free',  'Free',  50,  0,    '{"modules": "all"}'),
    ('plus',  'Plus',  500, 980,  '{"modules": "all", "priority": true}'),
    ('team',  'Team',  2000, 4980, '{"modules": "all", "priority": true, "team": true}')
ON CONFLICT (id) DO NOTHING;
