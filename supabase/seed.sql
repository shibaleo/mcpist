-- Seed data for local development
-- This file is run after migrations on `supabase db reset`
--
-- Note: API Key must be generated via Console UI because it requires vault access
-- Test user: test@example.com / password123
-- After login, go to MCP接続情報 page to generate API Key

-- Create test user in auth.users
INSERT INTO auth.users (
    id,
    instance_id,
    email,
    encrypted_password,
    email_confirmed_at,
    created_at,
    updated_at,
    aud,
    role
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000000',
    'test@example.com',
    crypt('password123', gen_salt('bf')),
    NOW(),
    NOW(),
    NOW(),
    'authenticated',
    'authenticated'
) ON CONFLICT (id) DO NOTHING;

-- Create mcpist user record
INSERT INTO mcpist.users (id, display_name, status)
VALUES ('11111111-1111-1111-1111-111111111111', 'Test User', 'active')
ON CONFLICT (id) DO NOTHING;

-- Enable notion module for test user
INSERT INTO mcpist.user_module_preferences (user_id, module_id, is_enabled)
SELECT
    '11111111-1111-1111-1111-111111111111',
    id,
    true
FROM mcpist.modules
WHERE name = 'notion'
ON CONFLICT (user_id, module_id) DO NOTHING;
