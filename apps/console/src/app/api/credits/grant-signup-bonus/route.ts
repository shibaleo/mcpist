import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/postgrest"

/**
 * POST /api/credits/grant-signup-bonus
 * Activate new user account (set status to active, assign free plan)
 * Uses idempotency key to prevent duplicate activations
 */
export async function POST() {
  try {
    const supabase = await createClient()
    const {
      data: { user },
    } = await supabase.auth.getUser()

    if (!user) {
      return NextResponse.json(
        { success: false, error: "unauthorized", message: "ログインが必要です" },
        { status: 401 }
      )
    }

    // Complete onboarding (activate account + assign free plan)
    const eventId = `onboarding:${user.id}`
    const result = await rpc<{
      success: boolean
      already_completed?: boolean
      plan_id?: string
      error?: string
      message?: string
    }>("complete_user_onboarding", {
      p_user_id: user.id,
      p_event_id: eventId,
    })

    if (result?.success) {
      if (result.already_completed) {
        console.log(
          `[api/credits/grant-signup-bonus] Onboarding already completed for ${user.id}`
        )
        return NextResponse.json({
          success: false,
          error: "already_granted",
          message: "既にアカウントは有効です",
        })
      }
      console.log(
        `[api/credits/grant-signup-bonus] Onboarding completed for user ${user.id}, plan: ${result.plan_id}`
      )
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
