-- =============================================================================
-- list_modules_with_tools: モジュール+ツール一覧をDBから取得
-- Console が tools.json の代わりに使用
-- =============================================================================

-- 1. modules テーブルに descriptions カラム追加（モジュール説明の多言語対応）
ALTER TABLE mcpist.modules
    ADD COLUMN IF NOT EXISTS descriptions JSONB DEFAULT '{}'::JSONB;

COMMENT ON COLUMN mcpist.modules.descriptions IS 'Module descriptions: {"en-US": "...", "ja-JP": "..."}';

-- 2. sync_modules を更新して descriptions も UPSERT するように

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
    v_descriptions JSONB;
BEGIN
    FOR v_module IN SELECT jsonb_array_elements(p_modules) LOOP
        v_name := v_module->>'name';
        v_status := COALESCE(v_module->>'status', 'active');
        v_tools := COALESCE(v_module->'tools', '[]'::JSONB);
        v_descriptions := COALESCE(v_module->'descriptions', '{}'::JSONB);

        INSERT INTO mcpist.modules (name, status, tools, descriptions)
        VALUES (v_name, v_status::mcpist.module_status, v_tools, v_descriptions)
        ON CONFLICT (name) DO UPDATE SET
            status = EXCLUDED.status,
            tools = EXCLUDED.tools,
            descriptions = EXCLUDED.descriptions;

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

-- 3. list_modules_with_tools RPC

CREATE OR REPLACE FUNCTION mcpist.list_modules_with_tools()
RETURNS TABLE (
    id TEXT,
    name TEXT,
    status TEXT,
    descriptions JSONB,
    tools JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS id,
        m.name,
        m.status::TEXT,
        COALESCE(m.descriptions, '{}'::JSONB),
        COALESCE(m.tools, '[]'::JSONB)
    FROM mcpist.modules m
    WHERE m.status IN ('active', 'beta')
    ORDER BY m.name;
END;
$$;

-- Public wrapper for PostgREST
CREATE OR REPLACE FUNCTION public.list_modules_with_tools()
RETURNS TABLE (
    id TEXT,
    name TEXT,
    status TEXT,
    descriptions JSONB,
    tools JSONB
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
    SELECT * FROM mcpist.list_modules_with_tools();
$$;

-- Console uses anon for public pages, authenticated for logged-in pages
GRANT EXECUTE ON FUNCTION public.list_modules_with_tools() TO authenticated;
GRANT EXECUTE ON FUNCTION public.list_modules_with_tools() TO anon;
