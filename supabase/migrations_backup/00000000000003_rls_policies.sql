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
ALTER TABLE mcpist.service_tokens ENABLE ROW LEVEL SECURITY;
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
-- Service Tokens Policies
-- -----------------------------------------------------------------------------

CREATE POLICY service_tokens_select ON mcpist.service_tokens
    FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY service_tokens_insert ON mcpist.service_tokens
    FOR INSERT WITH CHECK (auth.uid() = user_id);

CREATE POLICY service_tokens_update ON mcpist.service_tokens
    FOR UPDATE USING (auth.uid() = user_id);

CREATE POLICY service_tokens_delete ON mcpist.service_tokens
    FOR DELETE USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- Processed Webhook Events Policies
-- -----------------------------------------------------------------------------

-- processed_webhook_events INSERT is done via RPC with service_role
-- No direct user access needed
