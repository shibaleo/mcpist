-- RPC function to get current user's role
-- This function is in public schema so it can be called with anon key
-- Uses SECURITY DEFINER to access mcpist schema

CREATE OR REPLACE FUNCTION public.get_my_role()
RETURNS text
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT role FROM mcpist.users WHERE id = auth.uid();
$$;

-- Grant execute permission to authenticated users
GRANT EXECUTE ON FUNCTION public.get_my_role() TO authenticated;
