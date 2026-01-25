-- =============================================================================
-- Rename get_my_role to get_user_role
-- =============================================================================

-- Drop the old function
DROP FUNCTION IF EXISTS public.get_my_role();

-- Create the new function with the new name
CREATE OR REPLACE FUNCTION public.get_user_role()
RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id UUID;
    v_role TEXT;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RETURN NULL;
    END IF;

    -- auth.usersからraw_app_meta_data.roleを取得
    SELECT COALESCE(
        raw_app_meta_data->>'role',
        'user'
    ) INTO v_role
    FROM auth.users
    WHERE id = v_user_id;

    RETURN v_role;
END;
$$;

GRANT EXECUTE ON FUNCTION public.get_user_role() TO authenticated;
