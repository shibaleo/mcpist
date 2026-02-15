-- =============================================================================
-- MCPist: Add tools JSONB to modules + Replace sync_modules RPC
-- =============================================================================
-- modules テーブルにツール定義 (JSONB) を追加し、
-- Go サーバー起動時にモジュール+ツール情報を動的に同期する。
--
-- 変更点:
--   1. modules テーブルに tools JSONB カラム追加
--   2. 旧 sync_modules(TEXT[]) を DROP
--   3. 新 sync_modules(JSONB) を作成 — モジュール名+ステータス+ツール一覧を UPSERT
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. modules テーブルに tools カラム追加
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.modules
    ADD COLUMN IF NOT EXISTS tools JSONB DEFAULT '[]'::JSONB;

COMMENT ON COLUMN mcpist.modules.tools IS 'Tool definitions from Go registry: [{id, name, descriptions, annotations}]';

-- -----------------------------------------------------------------------------
-- 2. 旧 sync_modules を DROP
-- -----------------------------------------------------------------------------

DROP FUNCTION IF EXISTS public.sync_modules(TEXT[]);

-- -----------------------------------------------------------------------------
-- 3. 新 sync_modules RPC
-- Go サーバー起動時に呼び出し、モジュール+ツール情報を UPSERT
-- 入力: JSONB 配列 [{name, status, tools: [{id, name, descriptions, annotations}]}]
-- 出力: {success, upserted, total}
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.sync_modules(p_modules JSONB)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_upserted INTEGER := 0;
    v_module JSONB;
    v_name TEXT;
    v_status TEXT;
    v_tools JSONB;
BEGIN
    FOR v_module IN SELECT jsonb_array_elements(p_modules) LOOP
        v_name := v_module->>'name';
        v_status := COALESCE(v_module->>'status', 'active');
        v_tools := COALESCE(v_module->'tools', '[]'::JSONB);

        INSERT INTO mcpist.modules (name, status, tools)
        VALUES (v_name, v_status::mcpist.module_status, v_tools)
        ON CONFLICT (name) DO UPDATE SET
            status = EXCLUDED.status,
            tools = EXCLUDED.tools;

        v_upserted := v_upserted + 1;
    END LOOP;

    RETURN jsonb_build_object(
        'success', true,
        'upserted', v_upserted,
        'total', jsonb_array_length(p_modules)
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.sync_modules(p_modules JSONB)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.sync_modules(p_modules);
$$;

GRANT EXECUTE ON FUNCTION mcpist.sync_modules(JSONB) TO service_role;
GRANT EXECUTE ON FUNCTION public.sync_modules(JSONB) TO service_role;

COMMENT ON FUNCTION mcpist.sync_modules(JSONB) IS 'Sync registered modules and tools from Go server to database. Called on server startup.';
COMMENT ON FUNCTION public.sync_modules(JSONB) IS 'Sync registered modules and tools from Go server to database. Called on server startup.';
