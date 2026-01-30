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
