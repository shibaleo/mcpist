-- RPC: sync_modules
-- Server startup時にGoサーバーから呼び出され、モジュールをDBに同期する

CREATE OR REPLACE FUNCTION public.sync_modules(p_modules TEXT[])
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_inserted INTEGER := 0;
    v_module TEXT;
BEGIN
    FOREACH v_module IN ARRAY p_modules LOOP
        INSERT INTO mcpist.modules (name, status)
        VALUES (v_module, 'active')
        ON CONFLICT (name) DO NOTHING;

        IF FOUND THEN
            v_inserted := v_inserted + 1;
        END IF;
    END LOOP;

    RETURN jsonb_build_object(
        'success', true,
        'inserted', v_inserted,
        'total', array_length(p_modules, 1)
    );
END;
$$;

COMMENT ON FUNCTION public.sync_modules(TEXT[]) IS 'Sync registered modules from Go server to database. Called on server startup.';
