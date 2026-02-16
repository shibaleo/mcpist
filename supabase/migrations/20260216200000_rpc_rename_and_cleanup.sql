-- =============================================================================
-- RPC Rename & Cleanup: auth.uid() → p_user_id, _my_/_user_ prefix removal
-- =============================================================================
-- Phase 1 of Console RPC migration.
-- - All _my_ functions: add p_user_id param, replace auth.uid(), rename
-- - Server _user_ functions: rename (already have p_user_id)
-- - Deprecated functions: DROP
-- - New: get_module_config
-- - Extended: get_user_context (add role, settings, connected_count)
-- - All GRANTs: service_role only
-- =============================================================================

-- =============================================================================
-- 1. API Keys: generate_my_api_key → generate_api_key
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.generate_api_key(
    p_user_id UUID,
    p_display_name TEXT,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
DECLARE
    v_key TEXT;
    v_key_hash TEXT;
    v_key_prefix TEXT;
    v_key_id UUID;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    v_key := 'mpt_' || encode(gen_random_bytes(16), 'hex');
    v_key_prefix := substring(v_key from 1 for 8) || '...' || substring(v_key from length(v_key) - 3 for 4);
    v_key_hash := encode(sha256(v_key::bytea), 'hex');

    INSERT INTO mcpist.api_keys (user_id, name, key_hash, key_prefix, expires_at)
    VALUES (p_user_id, p_display_name, v_key_hash, v_key_prefix, p_expires_at)
    RETURNING id INTO v_key_id;

    RETURN jsonb_build_object(
        'api_key', v_key,
        'key_prefix', v_key_prefix
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.generate_api_key(
    p_user_id UUID,
    p_display_name TEXT,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.generate_api_key(p_user_id, p_display_name, p_expires_at);
$$;

GRANT EXECUTE ON FUNCTION mcpist.generate_api_key(UUID, TEXT, TIMESTAMPTZ) TO service_role;
GRANT EXECUTE ON FUNCTION public.generate_api_key(UUID, TEXT, TIMESTAMPTZ) TO service_role;

DROP FUNCTION IF EXISTS public.generate_my_api_key(TEXT, TIMESTAMPTZ);
DROP FUNCTION IF EXISTS mcpist.generate_my_api_key(TEXT, TIMESTAMPTZ);

-- =============================================================================
-- 2. API Keys: list_my_api_keys → list_api_keys
-- =============================================================================

DROP FUNCTION IF EXISTS public.list_api_keys(UUID);
DROP FUNCTION IF EXISTS mcpist.list_api_keys(UUID);

CREATE FUNCTION mcpist.list_api_keys(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    key_prefix TEXT,
    display_name TEXT,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        k.id,
        k.key_prefix,
        k.name AS display_name,
        k.expires_at,
        k.last_used_at,
        k.revoked_at
    FROM mcpist.api_keys k
    WHERE k.user_id = p_user_id
      AND k.revoked_at IS NULL
    ORDER BY k.created_at DESC;
END;
$$;

CREATE FUNCTION public.list_api_keys(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    key_prefix TEXT,
    display_name TEXT,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_api_keys(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_api_keys(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.list_api_keys(UUID) TO service_role;

DROP FUNCTION IF EXISTS public.list_my_api_keys();
DROP FUNCTION IF EXISTS mcpist.list_my_api_keys();

-- =============================================================================
-- 3. API Keys: revoke_my_api_key → revoke_api_key
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.revoke_api_key(p_user_id UUID, p_key_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_affected INTEGER;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    UPDATE mcpist.api_keys
    SET revoked_at = NOW()
    WHERE id = p_key_id
      AND user_id = p_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('success', v_affected > 0);
END;
$$;

CREATE OR REPLACE FUNCTION public.revoke_api_key(p_user_id UUID, p_key_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.revoke_api_key(p_user_id, p_key_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.revoke_api_key(UUID, UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.revoke_api_key(UUID, UUID) TO service_role;

DROP FUNCTION IF EXISTS public.revoke_my_api_key(UUID);
DROP FUNCTION IF EXISTS mcpist.revoke_my_api_key(UUID);

-- =============================================================================
-- 4. Credentials: list_my_credentials → list_credentials
-- =============================================================================

DROP FUNCTION IF EXISTS public.list_credentials(UUID);
DROP FUNCTION IF EXISTS mcpist.list_credentials(UUID);

CREATE FUNCTION mcpist.list_credentials(p_user_id UUID)
RETURNS TABLE (
    module TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        uc.module,
        uc.created_at,
        uc.updated_at
    FROM mcpist.user_credentials uc
    WHERE uc.user_id = p_user_id
    ORDER BY uc.module;
END;
$$;

CREATE FUNCTION public.list_credentials(p_user_id UUID)
RETURNS TABLE (
    module TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_credentials(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_credentials(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.list_credentials(UUID) TO service_role;

DROP FUNCTION IF EXISTS public.list_my_credentials();
DROP FUNCTION IF EXISTS mcpist.list_my_credentials();

-- =============================================================================
-- 5. Credentials: upsert_my_credential + upsert_user_credential → upsert_credential
-- =============================================================================
-- Unified: both Console and Server use the same function.

CREATE OR REPLACE FUNCTION mcpist.upsert_credential(
    p_user_id UUID,
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    INSERT INTO mcpist.user_credentials (user_id, module, credentials)
    VALUES (p_user_id, p_module, p_credentials::TEXT)
    ON CONFLICT (user_id, module)
    DO UPDATE SET
        credentials = p_credentials::TEXT,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'success', true,
        'module', p_module
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_credential(
    p_user_id UUID,
    p_module TEXT,
    p_credentials JSONB
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_credential(p_user_id, p_module, p_credentials);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_credential(UUID, TEXT, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_credential(UUID, TEXT, JSONB) TO service_role;

-- Drop old functions
DROP FUNCTION IF EXISTS public.upsert_my_credential(TEXT, JSONB);
DROP FUNCTION IF EXISTS mcpist.upsert_my_credential(TEXT, JSONB);
DROP FUNCTION IF EXISTS public.upsert_user_credential(UUID, TEXT, JSONB);
DROP FUNCTION IF EXISTS mcpist.upsert_user_credential(UUID, TEXT, JSONB);

-- =============================================================================
-- 6. Credentials: delete_my_credential → delete_credential
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.delete_credential(p_user_id UUID, p_module TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_affected INTEGER;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    DELETE FROM mcpist.user_credentials
    WHERE user_id = p_user_id AND module = p_module;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('success', v_affected > 0);
END;
$$;

CREATE OR REPLACE FUNCTION public.delete_credential(p_user_id UUID, p_module TEXT)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_credential(p_user_id, p_module);
$$;

GRANT EXECUTE ON FUNCTION mcpist.delete_credential(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.delete_credential(UUID, TEXT) TO service_role;

DROP FUNCTION IF EXISTS public.delete_my_credential(TEXT);
DROP FUNCTION IF EXISTS mcpist.delete_my_credential(TEXT);

-- =============================================================================
-- 7. Credentials: get_user_credential → get_credential
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.get_credential(
    p_user_id UUID,
    p_module TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_credentials TEXT;
    v_credentials_jsonb JSONB;
BEGIN
    SELECT uc.credentials INTO v_credentials
    FROM mcpist.user_credentials uc
    WHERE uc.user_id = p_user_id AND uc.module = p_module;

    IF v_credentials IS NULL THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'token_not_found'
        );
    END IF;

    BEGIN
        v_credentials_jsonb := v_credentials::JSONB;
    EXCEPTION WHEN OTHERS THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'invalid_credentials_format'
        );
    END;

    RETURN jsonb_build_object(
        'found', true,
        'user_id', p_user_id,
        'service', p_module,
        'auth_type', v_credentials_jsonb->>'auth_type',
        'credentials', v_credentials_jsonb,
        'metadata', v_credentials_jsonb->'metadata'
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.get_credential(
    p_user_id UUID,
    p_module TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_credential(p_user_id, p_module);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_credential(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_credential(UUID, TEXT) TO service_role;

DROP FUNCTION IF EXISTS public.get_user_credential(UUID, TEXT);
DROP FUNCTION IF EXISTS mcpist.get_user_credential(UUID, TEXT);

-- =============================================================================
-- 8. Prompts: list_my_prompts → list_prompts
-- =============================================================================

DROP FUNCTION IF EXISTS public.list_prompts(UUID, TEXT);
DROP FUNCTION IF EXISTS mcpist.list_prompts(UUID, TEXT);

CREATE FUNCTION mcpist.list_prompts(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    RETURN QUERY
    SELECT
        p.id,
        m.name AS module_name,
        p.name,
        p.description,
        p.content,
        p.enabled,
        p.created_at,
        p.updated_at
    FROM mcpist.prompts p
    LEFT JOIN mcpist.modules m ON m.id = p.module_id
    WHERE p.user_id = p_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY p.updated_at DESC;
END;
$$;

CREATE FUNCTION public.list_prompts(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    module_name TEXT,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_prompts(p_user_id, p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_prompts(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.list_prompts(UUID, TEXT) TO service_role;

DROP FUNCTION IF EXISTS public.list_my_prompts(TEXT);
DROP FUNCTION IF EXISTS mcpist.list_my_prompts(TEXT);

-- =============================================================================
-- 9. Prompts: get_my_prompt → get_prompt
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.get_prompt(p_user_id UUID, p_prompt_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_prompt RECORD;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    SELECT
        p.id,
        m.name AS module_name,
        p.name,
        p.description,
        p.content,
        p.enabled,
        p.created_at,
        p.updated_at
    INTO v_prompt
    FROM mcpist.prompts p
    LEFT JOIN mcpist.modules m ON m.id = p.module_id
    WHERE p.id = p_prompt_id AND p.user_id = p_user_id;

    IF v_prompt IS NULL THEN
        RETURN jsonb_build_object(
            'found', false,
            'error', 'prompt_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'found', true,
        'id', v_prompt.id,
        'module_name', v_prompt.module_name,
        'name', v_prompt.name,
        'description', v_prompt.description,
        'content', v_prompt.content,
        'enabled', v_prompt.enabled,
        'created_at', v_prompt.created_at,
        'updated_at', v_prompt.updated_at
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.get_prompt(p_user_id UUID, p_prompt_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_prompt(p_user_id, p_prompt_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_prompt(UUID, UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_prompt(UUID, UUID) TO service_role;

DROP FUNCTION IF EXISTS public.get_my_prompt(UUID);
DROP FUNCTION IF EXISTS mcpist.get_my_prompt(UUID);

-- =============================================================================
-- 10. Prompts: upsert_my_prompt → upsert_prompt
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.upsert_prompt(
    p_user_id UUID,
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL,
    p_enabled BOOLEAN DEFAULT TRUE,
    p_description TEXT DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
    v_result_id UUID;
    v_action TEXT;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    IF p_module_name IS NOT NULL THEN
        SELECT id INTO v_module_id
        FROM mcpist.modules
        WHERE name = p_module_name;

        IF v_module_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'module_not_found'
            );
        END IF;
    END IF;

    IF p_prompt_id IS NOT NULL THEN
        UPDATE mcpist.prompts
        SET
            name = p_name,
            description = p_description,
            content = p_content,
            module_id = v_module_id,
            enabled = p_enabled,
            updated_at = NOW()
        WHERE id = p_prompt_id AND user_id = p_user_id
        RETURNING id INTO v_result_id;

        IF v_result_id IS NULL THEN
            RETURN jsonb_build_object(
                'success', false,
                'error', 'prompt_not_found'
            );
        END IF;
        v_action := 'updated';
    ELSE
        INSERT INTO mcpist.prompts (user_id, module_id, name, description, content, enabled)
        VALUES (p_user_id, v_module_id, p_name, p_description, p_content, p_enabled)
        RETURNING id INTO v_result_id;
        v_action := 'created';
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'id', v_result_id,
        'action', v_action
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_prompt(
    p_user_id UUID,
    p_name TEXT,
    p_content TEXT,
    p_module_name TEXT DEFAULT NULL,
    p_prompt_id UUID DEFAULT NULL,
    p_enabled BOOLEAN DEFAULT TRUE,
    p_description TEXT DEFAULT NULL
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_prompt(p_user_id, p_name, p_content, p_module_name, p_prompt_id, p_enabled, p_description);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_prompt(UUID, TEXT, TEXT, TEXT, UUID, BOOLEAN, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_prompt(UUID, TEXT, TEXT, TEXT, UUID, BOOLEAN, TEXT) TO service_role;

DROP FUNCTION IF EXISTS public.upsert_my_prompt(TEXT, TEXT, TEXT, UUID, BOOLEAN, TEXT);
DROP FUNCTION IF EXISTS mcpist.upsert_my_prompt(TEXT, TEXT, TEXT, UUID, BOOLEAN, TEXT);
-- Also drop older signature
DROP FUNCTION IF EXISTS public.upsert_my_prompt(TEXT, TEXT, TEXT, UUID);
DROP FUNCTION IF EXISTS mcpist.upsert_my_prompt(TEXT, TEXT, TEXT, UUID);

-- =============================================================================
-- 11. Prompts: delete_my_prompt → delete_prompt
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.delete_prompt(p_user_id UUID, p_prompt_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_deleted_id UUID;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    DELETE FROM mcpist.prompts
    WHERE id = p_prompt_id AND user_id = p_user_id
    RETURNING id INTO v_deleted_id;

    IF v_deleted_id IS NULL THEN
        RETURN jsonb_build_object(
            'success', false,
            'error', 'prompt_not_found'
        );
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'deleted_id', v_deleted_id
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.delete_prompt(p_user_id UUID, p_prompt_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.delete_prompt(p_user_id, p_prompt_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.delete_prompt(UUID, UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.delete_prompt(UUID, UUID) TO service_role;

DROP FUNCTION IF EXISTS public.delete_my_prompt(UUID);
DROP FUNCTION IF EXISTS mcpist.delete_my_prompt(UUID);

-- =============================================================================
-- 12. Prompts: get_user_prompts → get_prompts
-- =============================================================================

DROP FUNCTION IF EXISTS public.get_prompts(UUID, TEXT, BOOLEAN);
DROP FUNCTION IF EXISTS mcpist.get_prompts(UUID, TEXT, BOOLEAN);

CREATE FUNCTION mcpist.get_prompts(
    p_user_id UUID,
    p_prompt_name TEXT DEFAULT NULL,
    p_enabled_only BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.id,
        p.name,
        p.description,
        p.content,
        p.enabled
    FROM mcpist.prompts p
    WHERE p.user_id = p_user_id
      AND (p_prompt_name IS NULL OR p.name = p_prompt_name)
      AND (NOT p_enabled_only OR p.enabled = TRUE)
    ORDER BY p.updated_at DESC;
END;
$$;

CREATE FUNCTION public.get_prompts(
    p_user_id UUID,
    p_prompt_name TEXT DEFAULT NULL,
    p_enabled_only BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    description TEXT,
    content TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_prompts(p_user_id, p_prompt_name, p_enabled_only);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_prompts(UUID, TEXT, BOOLEAN) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_prompts(UUID, TEXT, BOOLEAN) TO service_role;

DROP FUNCTION IF EXISTS public.get_user_prompts(UUID, TEXT, BOOLEAN);
DROP FUNCTION IF EXISTS mcpist.get_user_prompts(UUID, TEXT, BOOLEAN);

-- =============================================================================
-- 13. Tool Settings: upsert_my_tool_settings → upsert_tool_settings
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.upsert_tool_settings(
    p_user_id UUID,
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    IF p_enabled_tools IS NOT NULL AND array_length(p_enabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT p_user_id, v_module_id, unnest(p_enabled_tools), true
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = true;
    END IF;

    IF p_disabled_tools IS NOT NULL AND array_length(p_disabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT p_user_id, v_module_id, unnest(p_disabled_tools), false
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = false;
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'enabled_count', COALESCE(array_length(p_enabled_tools, 1), 0),
        'disabled_count', COALESCE(array_length(p_disabled_tools, 1), 0)
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_tool_settings(
    p_user_id UUID,
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_tool_settings(p_user_id, p_module_name, p_enabled_tools, p_disabled_tools);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_tool_settings(UUID, TEXT, TEXT[], TEXT[]) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_tool_settings(UUID, TEXT, TEXT[], TEXT[]) TO service_role;

DROP FUNCTION IF EXISTS public.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]);
DROP FUNCTION IF EXISTS mcpist.upsert_my_tool_settings(TEXT, TEXT[], TEXT[]);

-- =============================================================================
-- 14. Module Descriptions: upsert_my_module_description → upsert_module_description
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.upsert_module_description(
    p_user_id UUID,
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    INSERT INTO mcpist.module_settings (user_id, module_id, enabled, description)
    VALUES (p_user_id, v_module_id, true, p_description)
    ON CONFLICT (user_id, module_id)
    DO UPDATE SET description = p_description;

    RETURN jsonb_build_object('success', true, 'module', p_module_name);
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_module_description(
    p_user_id UUID,
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_module_description(p_user_id, p_module_name, p_description);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_module_description(UUID, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.upsert_module_description(UUID, TEXT, TEXT) TO service_role;

DROP FUNCTION IF EXISTS public.upsert_my_module_description(TEXT, TEXT);
DROP FUNCTION IF EXISTS mcpist.upsert_my_module_description(TEXT, TEXT);

-- =============================================================================
-- 15. Settings: update_my_settings → update_settings
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.update_settings(p_user_id UUID, p_settings JSONB)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_current JSONB;
    v_updated JSONB;
    v_display_name TEXT;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    SELECT COALESCE(settings, '{}'::JSONB) INTO v_current
    FROM mcpist.users
    WHERE id = p_user_id;

    -- Extract display_name if present, handle it separately
    IF p_settings ? 'display_name' THEN
        v_display_name := p_settings->>'display_name';
        p_settings := p_settings - 'display_name';
    END IF;

    v_updated := v_current || p_settings;

    UPDATE mcpist.users
    SET settings = v_updated,
        display_name = COALESCE(v_display_name, display_name),
        updated_at = NOW()
    WHERE id = p_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'settings', v_updated || jsonb_build_object('display_name', COALESCE(v_display_name, (SELECT display_name FROM mcpist.users WHERE id = p_user_id)))
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.update_settings(p_user_id UUID, p_settings JSONB)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.update_settings(p_user_id, p_settings);
$$;

GRANT EXECUTE ON FUNCTION mcpist.update_settings(UUID, JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.update_settings(UUID, JSONB) TO service_role;

DROP FUNCTION IF EXISTS public.update_my_settings(JSONB);
DROP FUNCTION IF EXISTS mcpist.update_my_settings(JSONB);

-- =============================================================================
-- 16. Usage: get_my_usage → get_usage
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.get_usage(
    p_user_id UUID,
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_total_used INTEGER;
    v_module_usage JSONB;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    SELECT COUNT(*)::INTEGER
    INTO v_total_used
    FROM mcpist.usage_log
    WHERE user_id = p_user_id
      AND created_at >= p_start_date
      AND created_at < p_end_date;

    SELECT COALESCE(
        jsonb_object_agg(module_name, usage),
        '{}'::JSONB
    )
    INTO v_module_usage
    FROM (
        SELECT
            d->>'module' AS module_name,
            COUNT(*)::INTEGER AS usage
        FROM mcpist.usage_log ul,
             jsonb_array_elements(ul.details) AS d
        WHERE ul.user_id = p_user_id
          AND ul.created_at >= p_start_date
          AND ul.created_at < p_end_date
        GROUP BY d->>'module'
        ORDER BY usage DESC
    ) sub
    WHERE module_name IS NOT NULL;

    RETURN jsonb_build_object(
        'total_used', v_total_used,
        'by_module', v_module_usage,
        'period', jsonb_build_object(
            'start', p_start_date,
            'end', p_end_date
        )
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.get_usage(
    p_user_id UUID,
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_usage(p_user_id, p_start_date, p_end_date);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_usage(UUID, TIMESTAMPTZ, TIMESTAMPTZ) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_usage(UUID, TIMESTAMPTZ, TIMESTAMPTZ) TO service_role;

DROP FUNCTION IF EXISTS public.get_my_usage(TIMESTAMPTZ, TIMESTAMPTZ);
DROP FUNCTION IF EXISTS mcpist.get_my_usage(TIMESTAMPTZ, TIMESTAMPTZ);

-- =============================================================================
-- 17. OAuth: list_my_oauth_consents → list_oauth_consents
-- =============================================================================

DROP FUNCTION IF EXISTS public.list_oauth_consents(UUID);
DROP FUNCTION IF EXISTS mcpist.list_oauth_consents(UUID);

CREATE FUNCTION mcpist.list_oauth_consents(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    client_id UUID,
    client_name TEXT,
    scopes TEXT,
    granted_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.client_id,
        cl.client_name,
        c.scopes,
        c.granted_at
    FROM auth.oauth_consents c
    LEFT JOIN auth.oauth_clients cl ON c.client_id = cl.id
    WHERE c.user_id = p_user_id
      AND c.revoked_at IS NULL
    ORDER BY c.granted_at DESC;
END;
$$;

CREATE FUNCTION public.list_oauth_consents(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    client_id UUID,
    client_name TEXT,
    scopes TEXT,
    granted_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.list_oauth_consents(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.list_oauth_consents(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.list_oauth_consents(UUID) TO service_role;

DROP FUNCTION IF EXISTS public.list_my_oauth_consents();
DROP FUNCTION IF EXISTS mcpist.list_my_oauth_consents();

-- =============================================================================
-- 18. OAuth: revoke_my_oauth_consent → revoke_oauth_consent
-- =============================================================================

CREATE OR REPLACE FUNCTION mcpist.revoke_oauth_consent(p_user_id UUID, p_consent_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_affected INTEGER;
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    UPDATE auth.oauth_consents
    SET revoked_at = NOW()
    WHERE id = p_consent_id
      AND user_id = p_user_id
      AND revoked_at IS NULL;

    GET DIAGNOSTICS v_affected = ROW_COUNT;
    RETURN jsonb_build_object('revoked', v_affected > 0);
END;
$$;

CREATE OR REPLACE FUNCTION public.revoke_oauth_consent(p_user_id UUID, p_consent_id UUID)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.revoke_oauth_consent(p_user_id, p_consent_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.revoke_oauth_consent(UUID, UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.revoke_oauth_consent(UUID, UUID) TO service_role;

DROP FUNCTION IF EXISTS public.revoke_my_oauth_consent(UUID);
DROP FUNCTION IF EXISTS mcpist.revoke_my_oauth_consent(UUID);

-- =============================================================================
-- 19. Deprecated: get_my_role, get_my_settings, get_my_tool_settings, get_my_module_descriptions
-- =============================================================================
-- These are replaced by get_user_context (role + settings) and get_module_config.

DROP FUNCTION IF EXISTS public.get_my_role();
DROP FUNCTION IF EXISTS mcpist.get_my_role();

DROP FUNCTION IF EXISTS public.get_my_settings();
DROP FUNCTION IF EXISTS mcpist.get_my_settings();

DROP FUNCTION IF EXISTS public.get_my_tool_settings(TEXT);
DROP FUNCTION IF EXISTS mcpist.get_my_tool_settings(TEXT);

DROP FUNCTION IF EXISTS public.get_my_module_descriptions();
DROP FUNCTION IF EXISTS mcpist.get_my_module_descriptions();

-- =============================================================================
-- 20. Extended: get_user_context (add role, settings, connected_count)
-- =============================================================================

DROP FUNCTION IF EXISTS public.get_user_context(UUID);
DROP FUNCTION IF EXISTS mcpist.get_user_context(UUID);

CREATE FUNCTION mcpist.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    plan_id TEXT,
    daily_used INTEGER,
    daily_limit INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB,
    role TEXT,
    settings JSONB,
    display_name TEXT,
    connected_count INTEGER
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_account_status TEXT;
    v_plan_id TEXT;
    v_daily_used INTEGER;
    v_daily_limit INTEGER;
    v_language TEXT;
    v_role TEXT;
    v_settings JSONB;
    v_display_name TEXT;
    v_connected_count INTEGER;
    v_module_data JSONB;
    v_enabled_modules TEXT[];
    v_enabled_tools JSONB;
    v_module_descriptions JSONB;
BEGIN
    -- 1. Get user status, plan, language, settings, display_name
    SELECT
        u.account_status::TEXT,
        u.plan_id,
        COALESCE(u.settings->>'language', 'en-US'),
        COALESCE(u.settings, '{}'::JSONB),
        u.display_name
    INTO v_account_status, v_plan_id, v_language, v_settings, v_display_name
    FROM mcpist.users u
    WHERE u.id = p_user_id;

    IF v_account_status IS NULL THEN
        RETURN;  -- User not found
    END IF;

    -- 2. Get role from auth.users
    SELECT COALESCE(raw_app_meta_data->>'role', 'user')
    INTO v_role
    FROM auth.users
    WHERE id = p_user_id;

    -- 3. Get plan daily limit
    SELECT p.daily_limit INTO v_daily_limit
    FROM mcpist.plans p
    WHERE p.id = v_plan_id;

    IF v_daily_limit IS NULL THEN
        v_daily_limit := 100;
    END IF;

    -- 4. Count today's usage (UTC day boundary)
    SELECT COUNT(*)::INTEGER INTO v_daily_used
    FROM mcpist.usage_log
    WHERE user_id = p_user_id
      AND created_at >= (CURRENT_DATE AT TIME ZONE 'UTC');

    -- 5. Count connected services
    SELECT COUNT(*)::INTEGER INTO v_connected_count
    FROM mcpist.user_credentials
    WHERE user_id = p_user_id;

    -- 6. Get enabled tools grouped by module with descriptions
    SELECT
        COALESCE(jsonb_object_agg(
            module_name,
            jsonb_build_object(
                'tools', tools,
                'description', description
            )
        ), '{}'::JSONB)
    INTO v_module_data
    FROM (
        SELECT
            m.name AS module_name,
            array_agg(ts.tool_id) AS tools,
            ms.description
        FROM mcpist.tool_settings ts
        JOIN mcpist.modules m ON m.id = ts.module_id
        LEFT JOIN mcpist.module_settings ms
            ON ms.user_id = ts.user_id AND ms.module_id = ts.module_id
        WHERE ts.user_id = p_user_id
          AND ts.enabled = true
          AND m.status IN ('active', 'beta')
        GROUP BY m.name, ms.description
    ) AS subq;

    -- 7. Extract enabled_modules
    SELECT array_agg(key)
    INTO v_enabled_modules
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 8. Extract enabled_tools: {module: [tool_ids]}
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'tools'),
        '{}'::JSONB
    )
    INTO v_enabled_tools
    FROM jsonb_object_keys(v_module_data) AS key;

    -- 9. Extract module_descriptions
    SELECT COALESCE(
        jsonb_object_agg(key, v_module_data->key->'description'),
        '{}'::JSONB
    )
    INTO v_module_descriptions
    FROM jsonb_object_keys(v_module_data) AS key
    WHERE v_module_data->key->>'description' IS NOT NULL
      AND v_module_data->key->>'description' != '';

    IF v_enabled_modules IS NULL THEN
        v_enabled_modules := ARRAY[]::TEXT[];
    END IF;

    RETURN QUERY SELECT
        v_account_status,
        v_plan_id,
        v_daily_used,
        v_daily_limit,
        v_enabled_modules,
        v_enabled_tools,
        v_language,
        v_module_descriptions,
        v_role,
        v_settings,
        v_display_name,
        v_connected_count;
END;
$$;

CREATE FUNCTION public.get_user_context(p_user_id UUID)
RETURNS TABLE (
    account_status TEXT,
    plan_id TEXT,
    daily_used INTEGER,
    daily_limit INTEGER,
    enabled_modules TEXT[],
    enabled_tools JSONB,
    language TEXT,
    module_descriptions JSONB,
    role TEXT,
    settings JSONB,
    display_name TEXT,
    connected_count INTEGER
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_user_context(p_user_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_user_context(UUID) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_user_context(UUID) TO service_role;

-- =============================================================================
-- 21. New: get_module_config (replaces get_my_tool_settings + get_my_module_descriptions)
-- =============================================================================

CREATE FUNCTION mcpist.get_module_config(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    description TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'p_user_id is required';
    END IF;

    RETURN QUERY
    SELECT
        m.name AS module_name,
        ms.description,
        ts.tool_id,
        ts.enabled
    FROM mcpist.tool_settings ts
    JOIN mcpist.modules m ON m.id = ts.module_id
    LEFT JOIN mcpist.module_settings ms
        ON ms.user_id = ts.user_id AND ms.module_id = ts.module_id
    WHERE ts.user_id = p_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY m.name, ts.tool_id;
END;
$$;

CREATE FUNCTION public.get_module_config(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    description TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_module_config(p_user_id, p_module_name);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_module_config(UUID, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.get_module_config(UUID, TEXT) TO service_role;

-- =============================================================================
-- 22. Revoke authenticated grants for remaining old functions
-- =============================================================================
-- list_all_oauth_consents still uses auth.uid() for admin check — keep for now,
-- but restrict to service_role (admin check will be done in Console backend).

REVOKE EXECUTE ON FUNCTION public.list_all_oauth_consents() FROM authenticated;
GRANT EXECUTE ON FUNCTION public.list_all_oauth_consents() TO service_role;
