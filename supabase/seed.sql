-- Seed data for local development
-- This file is run after migrations on `supabase db reset`
--
-- Note: API Key must be generated via Console UI because it requires vault access
--
-- Test users:
--   test@example.com  / testtest   (role: user)
--   admin@example.com / adminadmin (role: admin)
--
-- After login, go to MCP接続情報 page to generate API Key

-- Enable pgcrypto extension in extensions schema (for gen_salt/crypt functions)
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA extensions;

-- =============================================================================
-- Test User (role: user)
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

INSERT INTO mcpist.users (id, display_name, status, role)
VALUES ('11111111-1111-1111-1111-111111111111', 'Test User', 'active', 'user')
ON CONFLICT (id) DO UPDATE SET role = 'user', display_name = 'Test User';

-- =============================================================================
-- Admin User (role: admin)
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
    '{"provider": "email", "providers": ["email"]}',
    '{}'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO mcpist.users (id, display_name, status, role)
VALUES ('22222222-2222-2222-2222-222222222222', 'Admin User', 'active', 'admin')
ON CONFLICT (id) DO UPDATE SET role = 'admin', display_name = 'Admin User';

-- =============================================================================
-- Enable notion module for both users
-- =============================================================================

INSERT INTO mcpist.user_module_preferences (user_id, module_id, is_enabled)
SELECT
    '11111111-1111-1111-1111-111111111111',
    id,
    true
FROM mcpist.modules
WHERE name = 'notion'
ON CONFLICT (user_id, module_id) DO NOTHING;

INSERT INTO mcpist.user_module_preferences (user_id, module_id, is_enabled)
SELECT
    '22222222-2222-2222-2222-222222222222',
    id,
    true
FROM mcpist.modules
WHERE name = 'notion'
ON CONFLICT (user_id, module_id) DO NOTHING;
