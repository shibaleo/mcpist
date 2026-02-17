import { NextResponse } from "next/server"
import { rpc } from "@/lib/worker-client"

/**
 * POST /api/credits/grant-signup-bonus
 * Activate new user account (set status to active, assign free plan)
 * Uses idempotency key to prevent duplicate activations
 */
export async function POST() {
  try {
    // p_event_id は Worker 側で注入される p_user_id を使って生成されるため、
    // ここでは固定プレフィックスのみ渡す（RPC 側で p_user_id と結合）
    const result = await rpc<{
      success: boolean
      already_completed?: boolean
      plan_id?: string
      error?: string
      message?: string
    }>("complete_user_onboarding", {
      p_event_id: "onboarding",
    })

    if (result?.success) {
      if (result.already_completed) {
        return NextResponse.json({
          success: false,
          error: "already_granted",
          message: "既にアカウントは有効です",
        })
      }
      return NextResponse.json({
        success: true,
        plan_id: result.plan_id,
      })
    } else {
      console.error("[api/credits/grant-signup-bonus] Unexpected response:", result)
      return NextResponse.json(
        { success: false, error: result?.error || "unexpected", message: result?.message || "予期せぬエラー" },
        { status: 500 }
      )
    }
  } catch (err) {
    console.error("[api/credits/grant-signup-bonus] Exception:", err)
    return NextResponse.json(
      { success: false, error: "internal_error", message: "内部エラーが発生しました" },
      { status: 500 }
    )
  }
}
