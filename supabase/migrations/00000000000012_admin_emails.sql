-- =============================================================================
-- MCPist Admin Email Configuration
-- =============================================================================
-- This migration adds admin email management functionality.
-- Admin emails are stored in a config table and checked during user creation.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Admin Emails Table
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS mcpist.admin_emails (
    email TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE mcpist.admin_emails IS 'Emails that should be granted admin role on signup';

-- RLS Policy (service_role only)
ALTER TABLE mcpist.admin_emails ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role can manage admin_emails"
    ON mcpist.admin_emails
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- -----------------------------------------------------------------------------
-- Update handle_new_user trigger to check admin emails
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
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

    -- Create user record with pre_active status
    INSERT INTO mcpist.users (id, account_status)
    VALUES (NEW.id, 'pre_active'::mcpist.account_status);

    -- Create credits record with 0 credits (granted on onboarding completion)
    INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
    VALUES (NEW.id, 0, 0);

    RETURN NEW;
END;
$$;
