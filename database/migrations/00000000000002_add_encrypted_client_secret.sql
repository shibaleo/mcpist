-- Add encrypted_client_secret column to oauth_apps table
ALTER TABLE mcpist.oauth_apps ADD COLUMN IF NOT EXISTS encrypted_client_secret TEXT;
