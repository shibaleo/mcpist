-- Row Level Security Policies
-- Reference: spc-sec.md

-- Enable RLS on all tables
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
ALTER TABLE mcpist.processed_webhook_events ENABLE ROW LEVEL SECURITY;

-- Plans: Public read, no user write
CREATE POLICY "Plans are viewable by everyone"
    ON mcpist.plans FOR SELECT
    USING (is_active = true);

-- Modules: Public read, no user write
CREATE POLICY "Modules are viewable by everyone"
    ON mcpist.modules FOR SELECT
    USING (is_active = true);

-- Tool costs: Public read, no user write
CREATE POLICY "Tool costs are viewable by everyone"
    ON mcpist.tool_costs FOR SELECT
    USING (true);

-- Users: Own data only
CREATE POLICY "Users can view own data"
    ON mcpist.users FOR SELECT
    USING (auth.uid() = id);

CREATE POLICY "Users can update own data"
    ON mcpist.users FOR UPDATE
    USING (auth.uid() = id);

-- Subscriptions: Own data only
CREATE POLICY "Users can view own subscription"
    ON mcpist.subscriptions FOR SELECT
    USING (auth.uid() = user_id);

-- User module preferences: Own data only
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

-- Usage: Own data only (read)
CREATE POLICY "Users can view own usage"
    ON mcpist.usage FOR SELECT
    USING (auth.uid() = user_id);

-- Credits: Own data only (read)
CREATE POLICY "Users can view own credits"
    ON mcpist.credits FOR SELECT
    USING (auth.uid() = user_id);

-- Credit transactions: Own data only (read)
CREATE POLICY "Users can view own credit transactions"
    ON mcpist.credit_transactions FOR SELECT
    USING (auth.uid() = user_id);

-- MCP tokens: Own data only
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

-- OAuth tokens: Own data only
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

-- Processed webhook events: Service role only (no user access)
-- No policies = no access for anon/authenticated
