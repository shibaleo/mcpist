-- =============================================================================
-- MCPist Row Level Security Policies
-- =============================================================================
-- This migration enables RLS and creates policies for all tables.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Enable RLS on All Tables
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.user_module_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.credits ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.credit_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.tool_costs ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.mcp_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.oauth_authorization_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.processed_webhook_events ENABLE ROW LEVEL SECURITY;

-- -----------------------------------------------------------------------------
-- Public Read Policies (Plans, Modules, Tool Costs)
-- -----------------------------------------------------------------------------

CREATE POLICY "Plans are viewable by everyone"
    ON mcpist.plans FOR SELECT
    USING (is_active = true);

CREATE POLICY "Modules are viewable by everyone"
    ON mcpist.modules FOR SELECT
    USING (is_active = true);

CREATE POLICY "Tool costs are viewable by everyone"
    ON mcpist.tool_costs FOR SELECT
    USING (true);

-- -----------------------------------------------------------------------------
-- Users Policies (Own Data Only)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own data"
    ON mcpist.users FOR SELECT
    USING (auth.uid() = id);

CREATE POLICY "Users can update own data"
    ON mcpist.users FOR UPDATE
    USING (auth.uid() = id);

-- -----------------------------------------------------------------------------
-- Subscriptions Policies (Own Data Only)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own subscription"
    ON mcpist.subscriptions FOR SELECT
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- User Module Preferences Policies (Own Data Only)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own preferences"
    ON mcpist.user_module_preferences FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert own preferences"
    ON mcpist.user_module_preferences FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update own preferences"
    ON mcpist.user_module_preferences FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete own preferences"
    ON mcpist.user_module_preferences FOR DELETE
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- Usage Policies (Own Data Only, Read)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own usage"
    ON mcpist.usage FOR SELECT
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- Credits Policies (Own Data Only, Read)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own credits"
    ON mcpist.credits FOR SELECT
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- Credit Transactions Policies (Own Data Only, Read)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own credit transactions"
    ON mcpist.credit_transactions FOR SELECT
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- MCP Tokens Policies (Own Data Only)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own MCP tokens"
    ON mcpist.mcp_tokens FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert own MCP tokens"
    ON mcpist.mcp_tokens FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update own MCP tokens"
    ON mcpist.mcp_tokens FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete own MCP tokens"
    ON mcpist.mcp_tokens FOR DELETE
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- OAuth Tokens Policies (Own Data Only)
-- -----------------------------------------------------------------------------

CREATE POLICY "Users can view own OAuth tokens"
    ON mcpist.oauth_tokens FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert own OAuth tokens"
    ON mcpist.oauth_tokens FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update own OAuth tokens"
    ON mcpist.oauth_tokens FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete own OAuth tokens"
    ON mcpist.oauth_tokens FOR DELETE
    USING (auth.uid() = user_id);

-- -----------------------------------------------------------------------------
-- OAuth Authorization Codes (Service Role Only)
-- No policies for anon/authenticated - service_role bypasses RLS
-- -----------------------------------------------------------------------------

-- -----------------------------------------------------------------------------
-- Processed Webhook Events (Service Role Only)
-- No policies = no access for anon/authenticated
-- -----------------------------------------------------------------------------
