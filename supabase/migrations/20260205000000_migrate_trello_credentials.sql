-- =============================================================================
-- Trello credentials migration: username → api_key
-- =============================================================================
-- 背景: credentials JSON 構造を v2.2 仕様に準拠させるため
-- 変更: Trello の api_key を username フィールドから api_key フィールドに移動
-- =============================================================================

-- Trello の credentials を username → api_key に変換
UPDATE mcpist.user_credentials
SET
  credentials = (
    -- 新しい JSON を構築
    jsonb_build_object(
      'auth_type', COALESCE(credentials::jsonb->>'auth_type', 'api_key'),
      'api_key', credentials::jsonb->>'username',
      'access_token', credentials::jsonb->>'access_token'
    )
    -- metadata があれば追加
    || CASE
      WHEN credentials::jsonb ? 'metadata'
      THEN jsonb_build_object('metadata', credentials::jsonb->'metadata')
      ELSE '{}'::jsonb
    END
  )::text,
  updated_at = NOW()
WHERE module = 'trello'
  AND credentials::jsonb ? 'username'
  AND NOT (credentials::jsonb ? 'api_key');

-- 確認用: マイグレーション後の Trello credentials を表示
-- SELECT user_id, module, credentials::jsonb FROM mcpist.user_credentials WHERE module = 'trello';
