-- =============================================================================
-- API Keys Table and Functions
-- =============================================================================
-- This migration creates:
-- 1. mcpist.api_keys table for storing API keys
-- 2. RPC functions in public schema for API key management
-- =============================================================================

-- Enable pgcrypto extension for gen_random_bytes
-- Supabase では拡張機能は extensions スキーマに作成される
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA extensions;

-- -----------------------------------------------------------------------------
-- API Keys Table
-- -----------------------------------------------------------------------------

CREATE TABLE mcpist.api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,                    -- キーの名前（識別用）
  key_hash TEXT NOT NULL UNIQUE,         -- SHA-256ハッシュ
  key_prefix TEXT NOT NULL,              -- mpt_xxxx（表示用）
  service TEXT NOT NULL DEFAULT 'mcpist', -- サービス名
  scopes TEXT[] DEFAULT '{}',            -- 権限スコープ
  last_used_at TIMESTAMPTZ,              -- 最終使用日時
  expires_at TIMESTAMPTZ,                -- 有効期限（NULL = 無期限）
  created_at TIMESTAMPTZ DEFAULT NOW(),
  revoked_at TIMESTAMPTZ                 -- 削除日時（論理削除）
);

-- インデックス
CREATE INDEX idx_api_keys_user_id ON mcpist.api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON mcpist.api_keys(key_hash) WHERE revoked_at IS NULL;

-- -----------------------------------------------------------------------------
-- RLS Policies
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.api_keys ENABLE ROW LEVEL SECURITY;

-- ユーザーは自分のAPIキーのみ参照可能
CREATE POLICY "Users can view own api_keys"
    ON mcpist.api_keys FOR SELECT
    TO authenticated
    USING (auth.uid() = user_id);

-- ユーザーは自分のAPIキーのみ作成可能
CREATE POLICY "Users can insert own api_keys"
    ON mcpist.api_keys FOR INSERT
    TO authenticated
    WITH CHECK (auth.uid() = user_id);

-- ユーザーは自分のAPIキーのみ更新可能（revoke用）
CREATE POLICY "Users can update own api_keys"
    ON mcpist.api_keys FOR UPDATE
    TO authenticated
    USING (auth.uid() = user_id);

-- service_roleは全て参照可能（Worker からの検証用）
CREATE POLICY "Service role can view all api_keys"
    ON mcpist.api_keys FOR SELECT
    TO service_role
    USING (true);

-- service_roleは更新可能（last_used_at更新用）
CREATE POLICY "Service role can update all api_keys"
    ON mcpist.api_keys FOR UPDATE
    TO service_role
    USING (true);

-- -----------------------------------------------------------------------------
-- RPC Functions (public schema)
-- -----------------------------------------------------------------------------

-- APIキー生成
-- 返り値: { id, name, key, key_prefix, expires_at }
-- keyは生成時のみ返され、以降は参照不可
CREATE OR REPLACE FUNCTION public.generate_api_key(
  p_name TEXT,
  p_expires_in_days INTEGER DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
DECLARE
  v_user_id UUID;
  v_key TEXT;
  v_key_hash TEXT;
  v_key_prefix TEXT;
  v_expires_at TIMESTAMPTZ;
  v_key_id UUID;
BEGIN
  -- 認証確認
  v_user_id := auth.uid();
  IF v_user_id IS NULL THEN
    RAISE EXCEPTION 'Not authenticated';
  END IF;

  -- キー生成（mpt_ + 32文字のランダム16進数）
  v_key := 'mpt_' || encode(gen_random_bytes(16), 'hex');
  v_key_prefix := substring(v_key from 1 for 8) || '...' || substring(v_key from length(v_key) - 3 for 4);
  v_key_hash := encode(sha256(v_key::bytea), 'hex');

  -- 有効期限設定
  IF p_expires_in_days IS NOT NULL THEN
    v_expires_at := NOW() + (p_expires_in_days || ' days')::INTERVAL;
  END IF;

  -- 挿入
  INSERT INTO mcpist.api_keys (user_id, name, key_hash, key_prefix, expires_at)
  VALUES (v_user_id, p_name, v_key_hash, v_key_prefix, v_expires_at)
  RETURNING id INTO v_key_id;

  RETURN jsonb_build_object(
    'id', v_key_id,
    'name', p_name,
    'key', v_key,
    'key_prefix', v_key_prefix,
    'expires_at', v_expires_at
  );
END;
$$;

-- APIキー一覧取得
CREATE OR REPLACE FUNCTION public.list_api_keys()
RETURNS TABLE (
  id UUID,
  name TEXT,
  key_prefix TEXT,
  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ,
  is_expired BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
BEGIN
  RETURN QUERY
  SELECT
    k.id,
    k.name,
    k.key_prefix,
    k.last_used_at,
    k.expires_at,
    k.created_at,
    (k.expires_at IS NOT NULL AND k.expires_at < NOW()) AS is_expired
  FROM mcpist.api_keys k
  WHERE k.user_id = auth.uid()
    AND k.revoked_at IS NULL
  ORDER BY k.created_at DESC;
END;
$$;

-- APIキー削除（論理削除）
-- 返り値: { revoked: boolean, key_hash: string | null }
CREATE OR REPLACE FUNCTION public.revoke_api_key(p_key_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
DECLARE
  v_user_id UUID;
  v_key_hash TEXT;
  v_affected INTEGER;
BEGIN
  v_user_id := auth.uid();
  IF v_user_id IS NULL THEN
    RAISE EXCEPTION 'Not authenticated';
  END IF;

  -- 先にkey_hashを取得
  SELECT key_hash INTO v_key_hash
  FROM mcpist.api_keys
  WHERE id = p_key_id
    AND user_id = v_user_id
    AND revoked_at IS NULL;

  IF v_key_hash IS NULL THEN
    RETURN jsonb_build_object('revoked', false, 'key_hash', NULL);
  END IF;

  -- 論理削除
  UPDATE mcpist.api_keys
  SET revoked_at = NOW()
  WHERE id = p_key_id
    AND user_id = v_user_id
    AND revoked_at IS NULL;

  GET DIAGNOSTICS v_affected = ROW_COUNT;
  RETURN jsonb_build_object('revoked', v_affected > 0, 'key_hash', v_key_hash);
END;
$$;

-- APIキー検証（Worker から呼び出し）
CREATE OR REPLACE FUNCTION public.validate_api_key(p_key TEXT)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, extensions
AS $$
DECLARE
  v_key_hash TEXT;
  v_key_record RECORD;
BEGIN
  -- キーのハッシュ化
  v_key_hash := encode(sha256(p_key::bytea), 'hex');

  -- キー検索
  SELECT
    k.id,
    k.user_id,
    k.name,
    k.scopes,
    k.expires_at,
    k.revoked_at
  INTO v_key_record
  FROM mcpist.api_keys k
  WHERE k.key_hash = v_key_hash;

  -- キーが存在しない
  IF v_key_record IS NULL THEN
    RETURN jsonb_build_object('valid', false, 'error', 'invalid_key');
  END IF;

  -- 削除済み
  IF v_key_record.revoked_at IS NOT NULL THEN
    RETURN jsonb_build_object('valid', false, 'error', 'revoked');
  END IF;

  -- 有効期限切れ
  IF v_key_record.expires_at IS NOT NULL AND v_key_record.expires_at < NOW() THEN
    RETURN jsonb_build_object('valid', false, 'error', 'expired');
  END IF;

  -- 最終使用日時を更新
  UPDATE mcpist.api_keys
  SET last_used_at = NOW()
  WHERE id = v_key_record.id;

  RETURN jsonb_build_object(
    'valid', true,
    'user_id', v_key_record.user_id,
    'key_name', v_key_record.name,
    'scopes', v_key_record.scopes
  );
END;
$$;

-- Grant execute permissions
GRANT EXECUTE ON FUNCTION public.generate_api_key(TEXT, INTEGER) TO authenticated;
GRANT EXECUTE ON FUNCTION public.list_api_keys() TO authenticated;
GRANT EXECUTE ON FUNCTION public.revoke_api_key(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION public.validate_api_key(TEXT) TO anon, authenticated, service_role;
