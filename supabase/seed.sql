-- Seed data for local development
-- This file is run after migrations on `supabase db reset`
--
-- Note: API Key must be generated via Console UI because it requires vault access
--
-- Admin users are configured in mcpist.admin_emails table.
-- When these users sign up via OAuth, they automatically get admin role.
--
-- After login, go to MCP接続情報 page to generate API Key

-- =============================================================================
-- Admin Emails (users who should get admin role on signup)
-- =============================================================================

INSERT INTO mcpist.admin_emails (email) VALUES
    ('shiba.dog.leo.private@gmail.com')
ON CONFLICT (email) DO NOTHING;

-- =============================================================================
-- Modules Master Data
-- =============================================================================

INSERT INTO mcpist.modules (name, status) VALUES
    ('notion', 'active'),
    ('github', 'active'),
    ('jira', 'active'),
    ('confluence', 'active'),
    ('supabase', 'beta'),
    ('google_calendar', 'active'),
    ('microsoft_todo', 'active'),
    ('rag', 'active')
ON CONFLICT (name) DO UPDATE SET status = EXCLUDED.status;
