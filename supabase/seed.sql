-- Seed data for local development
-- This file is run after migrations on `supabase db reset`
--
-- Note: API Key must be generated via Console UI because it requires vault access
--
-- Test users:
--   test@example.com  / testtest
--   admin@example.com / adminadmin
--
-- After login, go to MCP接続情報 page to generate API Key

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

-- =============================================================================
-- Test User
-- =============================================================================

INSERT INTO auth.users (
    id,
    instance_id,
    email,
    encrypted_password,
    email_confirmed_at,
    created_at,
    updated_at,
    aud,
    role,
    confirmation_token,
    recovery_token,
    email_change_token_new,
    email_change,
    raw_app_meta_data,
    raw_user_meta_data
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000000',
    'test@example.com',
    extensions.crypt('testtest', extensions.gen_salt('bf')),
    NOW(),
    NOW(),
    NOW(),
    'authenticated',
    'authenticated',
    '',
    '',
    '',
    '',
    '{"provider": "email", "providers": ["email"]}',
    '{}'
) ON CONFLICT (id) DO NOTHING;

-- Note: users and credits tables are populated by trigger on auth.users insert
-- If running on existing data, manually insert:
INSERT INTO mcpist.users (id, account_status)
VALUES ('11111111-1111-1111-1111-111111111111', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
VALUES ('11111111-1111-1111-1111-111111111111', 1000, 0)
ON CONFLICT (user_id) DO NOTHING;

-- =============================================================================
-- Admin User
-- =============================================================================

INSERT INTO auth.users (
    id,
    instance_id,
    email,
    encrypted_password,
    email_confirmed_at,
    created_at,
    updated_at,
    aud,
    role,
    confirmation_token,
    recovery_token,
    email_change_token_new,
    email_change,
    raw_app_meta_data,
    raw_user_meta_data
) VALUES (
    '22222222-2222-2222-2222-222222222222',
    '00000000-0000-0000-0000-000000000000',
    'admin@example.com',
    extensions.crypt('adminadmin', extensions.gen_salt('bf')),
    NOW(),
    NOW(),
    NOW(),
    'authenticated',
    'authenticated',
    '',
    '',
    '',
    '',
    '{"provider": "email", "providers": ["email"], "role": "admin"}',
    '{}'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO mcpist.users (id, account_status)
VALUES ('22222222-2222-2222-2222-222222222222', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
VALUES ('22222222-2222-2222-2222-222222222222', 1000, 0)
ON CONFLICT (user_id) DO NOTHING;
