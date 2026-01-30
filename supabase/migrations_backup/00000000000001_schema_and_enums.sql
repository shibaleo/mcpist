-- =============================================================================
-- MCPist Database Schema and Enums
-- =============================================================================
-- This migration creates:
-- 1. mcpist schema
-- 2. Enum types
-- 3. Utility functions
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Schema Setup
-- -----------------------------------------------------------------------------

CREATE SCHEMA IF NOT EXISTS mcpist;

GRANT USAGE ON SCHEMA mcpist TO postgres, anon, authenticated, service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcpist
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO postgres, service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA mcpist
GRANT SELECT ON TABLES TO anon, authenticated;

-- -----------------------------------------------------------------------------
-- Enum Types
-- -----------------------------------------------------------------------------

CREATE TYPE mcpist.account_status AS ENUM (
    'active',      -- アクティブ
    'suspended',   -- 一時停止
    'disabled'     -- 無効化
);

CREATE TYPE mcpist.module_status AS ENUM (
    'active',       -- 利用可能
    'coming_soon',  -- 近日公開
    'maintenance',  -- メンテナンス中
    'beta',         -- ベータ版
    'deprecated',   -- 非推奨
    'disabled'      -- 無効
);

CREATE TYPE mcpist.credit_transaction_type AS ENUM (
    'consume',       -- クレジット消費
    'purchase',      -- クレジット購入
    'monthly_reset'  -- 月次リセット
);

-- -----------------------------------------------------------------------------
-- Utility Functions
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Enable pgcrypto extension for gen_random_bytes (API key generation)
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA extensions;
