-- Token Vault Tables
-- Reference: spc-tbl.md

-- OAuth tokens table
-- Stores reference to encrypted tokens in vault.secrets
CREATE TABLE mcpist.oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,                                -- e.g., 'notion', 'github', 'google'
    -- Token reference (stored in vault.secrets)
    access_token_secret_id UUID,                          -- Reference to vault.secrets
    refresh_token_secret_id UUID,                         -- Reference to vault.secrets
    -- Token metadata (not sensitive)
    token_type TEXT DEFAULT 'Bearer',
    scope TEXT,
    expires_at TIMESTAMPTZ,
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, service)
);

-- Indexes
CREATE INDEX idx_oauth_tokens_user_id ON mcpist.oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_service ON mcpist.oauth_tokens(service);

-- Updated at trigger
CREATE TRIGGER set_updated_at BEFORE UPDATE ON mcpist.oauth_tokens FOR EACH ROW EXECUTE FUNCTION mcpist.set_updated_at();
