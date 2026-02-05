-- 1. Backfill display_name from auth.users for existing users
UPDATE mcpist.users u
SET display_name = a.raw_user_meta_data->>'full_name',
    updated_at = NOW()
FROM auth.users a
WHERE u.id = a.id
  AND u.display_name IS NULL
  AND a.raw_user_meta_data->>'full_name' IS NOT NULL;

-- 2. Update handle_new_user trigger to set display_name on signup
CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'mcpist', 'public'
AS $function$
DECLARE
    v_is_admin BOOLEAN;
BEGIN
    -- Check if email is in admin_emails table
    SELECT EXISTS(
        SELECT 1 FROM mcpist.admin_emails WHERE email = NEW.email
    ) INTO v_is_admin;

    -- If admin email, update raw_app_meta_data to include admin role
    IF v_is_admin THEN
        UPDATE auth.users
        SET raw_app_meta_data = COALESCE(raw_app_meta_data, '{}'::jsonb) || '{"role": "admin"}'::jsonb
        WHERE id = NEW.id;
    END IF;

    -- Create user record with display_name from OAuth provider
    INSERT INTO mcpist.users (id, account_status, display_name)
    VALUES (
        NEW.id,
        'pre_active'::mcpist.account_status,
        NEW.raw_user_meta_data->>'full_name'
    );

    RETURN NEW;
END;
$function$;

-- 3. Extend get_my_settings to include display_name
CREATE OR REPLACE FUNCTION mcpist.get_my_settings()
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'mcpist', 'public'
AS $function$
DECLARE
    v_user_id UUID;
    v_settings JSONB;
    v_display_name TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT settings, display_name
    INTO v_settings, v_display_name
    FROM mcpist.users
    WHERE id = v_user_id;

    RETURN COALESCE(v_settings, '{}'::JSONB) || jsonb_build_object('display_name', v_display_name);
END;
$function$;

-- 4. Extend update_my_settings to handle display_name
CREATE OR REPLACE FUNCTION mcpist.update_my_settings(p_settings jsonb)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'mcpist', 'public'
AS $function$
DECLARE
    v_user_id UUID;
    v_current JSONB;
    v_updated JSONB;
    v_display_name TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    SELECT COALESCE(settings, '{}'::JSONB) INTO v_current
    FROM mcpist.users
    WHERE id = v_user_id;

    -- Extract display_name if present, handle it separately
    IF p_settings ? 'display_name' THEN
        v_display_name := p_settings->>'display_name';
        -- Remove display_name from settings JSONB (it's a dedicated column)
        p_settings := p_settings - 'display_name';
    END IF;

    v_updated := v_current || p_settings;

    UPDATE mcpist.users
    SET settings = v_updated,
        display_name = COALESCE(v_display_name, display_name),
        updated_at = NOW()
    WHERE id = v_user_id;

    RETURN jsonb_build_object(
        'success', true,
        'settings', v_updated || jsonb_build_object('display_name', COALESCE(v_display_name, (SELECT display_name FROM mcpist.users WHERE id = v_user_id)))
    );
END;
$function$;
