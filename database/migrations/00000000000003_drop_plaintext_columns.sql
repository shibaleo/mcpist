-- Drop plaintext credential columns (all data is now encrypted)
ALTER TABLE mcpist.user_credentials DROP COLUMN IF EXISTS credentials;
ALTER TABLE mcpist.oauth_apps DROP COLUMN IF EXISTS client_secret;
