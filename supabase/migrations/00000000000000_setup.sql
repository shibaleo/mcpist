-- MCPist Schema Initialization
-- This migration creates the mcpist schema

-- Create mcpist schema
CREATE SCHEMA IF NOT EXISTS mcpist;

-- Grant usage on mcpist schema
GRANT USAGE ON SCHEMA mcpist TO postgres, anon, authenticated, service_role;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA mcpist
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO postgres, service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcpist
GRANT SELECT ON TABLES TO anon, authenticated;
