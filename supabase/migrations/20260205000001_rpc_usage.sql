-- =============================================================================
-- MCPist RPC Functions for Usage Statistics
-- =============================================================================
-- This migration creates RPC functions for usage statistics:
-- 1. get_my_usage - 指定期間のクレジット消費量を取得
-- =============================================================================

-- -----------------------------------------------------------------------------
-- get_my_usage
-- 指定期間のクレジット消費量を取得
-- p_start_date: 開始日時 (inclusive)
-- p_end_date: 終了日時 (exclusive)
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.get_my_usage(
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_user_id UUID;
    v_total_consumed INTEGER;
    v_module_usage JSONB;
BEGIN
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- 合計消費量を取得（type = 'consume' のみ）
    SELECT COALESCE(SUM(ABS(amount)), 0)::INTEGER
    INTO v_total_consumed
    FROM mcpist.credit_transactions
    WHERE user_id = v_user_id
      AND type = 'consume'
      AND created_at >= p_start_date
      AND created_at < p_end_date;

    -- モジュール別消費量を取得
    SELECT COALESCE(
        jsonb_object_agg(module, usage),
        '{}'::JSONB
    )
    INTO v_module_usage
    FROM (
        SELECT
            module,
            SUM(ABS(amount))::INTEGER AS usage
        FROM mcpist.credit_transactions
        WHERE user_id = v_user_id
          AND type = 'consume'
          AND module IS NOT NULL
          AND created_at >= p_start_date
          AND created_at < p_end_date
        GROUP BY module
        ORDER BY usage DESC
    ) sub;

    RETURN jsonb_build_object(
        'total_consumed', v_total_consumed,
        'by_module', v_module_usage,
        'period', jsonb_build_object(
            'start', p_start_date,
            'end', p_end_date
        )
    );
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_usage(
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.get_my_usage(p_start_date, p_end_date);
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_usage(TIMESTAMPTZ, TIMESTAMPTZ) TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_usage(TIMESTAMPTZ, TIMESTAMPTZ) TO authenticated;
