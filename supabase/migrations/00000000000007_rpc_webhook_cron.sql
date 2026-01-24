-- =============================================================================
-- MCPist RPC Functions for Webhook and Cron
-- =============================================================================
-- This migration creates RPC functions:
-- 1. add_paid_credits - 有料クレジット加算（Webhook用）
-- 2. reset_free_credits - 月次無料クレジットリセット（Cron用）
-- =============================================================================

-- -----------------------------------------------------------------------------
-- add_paid_credits
-- 有料クレジットを加算（PSP Webhook処理用、冪等性対応）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.add_paid_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_existing_event RECORD;
    v_current_paid INTEGER;
    v_new_paid INTEGER;
BEGIN
    -- 冪等性チェック: 既に処理済みか確認
    SELECT event_id INTO v_existing_event
    FROM mcpist.processed_webhook_events
    WHERE event_id = p_event_id;

    IF v_existing_event IS NOT NULL THEN
        -- 既に処理済み - 現在の残高を返す
        SELECT paid_credits INTO v_current_paid
        FROM mcpist.credits
        WHERE user_id = p_user_id;

        RETURN jsonb_build_object(
            'success', true,
            'paid_credits', COALESCE(v_current_paid, 0),
            'already_processed', true
        );
    END IF;

    -- 処理済みイベントを記録
    INSERT INTO mcpist.processed_webhook_events (event_id, user_id)
    VALUES (p_event_id, p_user_id);

    -- クレジットを加算
    UPDATE mcpist.credits
    SET paid_credits = paid_credits + p_amount, updated_at = NOW()
    WHERE user_id = p_user_id
    RETURNING paid_credits INTO v_new_paid;

    IF v_new_paid IS NULL THEN
        -- creditsレコードがない場合は作成
        INSERT INTO mcpist.credits (user_id, free_credits, paid_credits)
        VALUES (p_user_id, 0, p_amount)
        RETURNING paid_credits INTO v_new_paid;
    END IF;

    -- 履歴記録
    INSERT INTO mcpist.credit_transactions (
        user_id, type, amount, credit_type
    ) VALUES (
        p_user_id, 'purchase', p_amount, 'paid'
    );

    RETURN jsonb_build_object(
        'success', true,
        'paid_credits', v_new_paid,
        'already_processed', false
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.add_paid_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_event_id TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.add_paid_credits(p_user_id, p_amount, p_event_id);
$$;

GRANT EXECUTE ON FUNCTION mcpist.add_paid_credits(UUID, INTEGER, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION public.add_paid_credits(UUID, INTEGER, TEXT) TO service_role;

-- -----------------------------------------------------------------------------
-- reset_free_credits
-- 月次無料クレジットリセット（Cron用）
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION mcpist.reset_free_credits()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_updated_count INTEGER;
    v_reset_amount INTEGER := 1000;
BEGIN
    -- activeユーザーの無料クレジットをリセット
    WITH updated AS (
        UPDATE mcpist.credits c
        SET free_credits = v_reset_amount, updated_at = NOW()
        FROM mcpist.users u
        WHERE c.user_id = u.id
          AND u.account_status = 'active'
          AND c.free_credits < v_reset_amount
        RETURNING c.user_id, v_reset_amount - c.free_credits AS reset_diff
    )
    SELECT COUNT(*) INTO v_updated_count FROM updated;

    -- 履歴記録（リセットされたユーザーのみ）
    INSERT INTO mcpist.credit_transactions (user_id, type, amount, credit_type)
    SELECT
        c.user_id,
        'monthly_reset'::mcpist.credit_transaction_type,
        v_reset_amount - c.free_credits,
        'free'
    FROM mcpist.credits c
    JOIN mcpist.users u ON c.user_id = u.id
    WHERE u.account_status = 'active'
      AND c.free_credits < v_reset_amount;

    RETURN jsonb_build_object(
        'updated_count', v_updated_count
    );
END;
$$;

-- public schema wrapper
CREATE OR REPLACE FUNCTION public.reset_free_credits()
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.reset_free_credits();
$$;

GRANT EXECUTE ON FUNCTION mcpist.reset_free_credits() TO service_role;
GRANT EXECUTE ON FUNCTION public.reset_free_credits() TO service_role;
