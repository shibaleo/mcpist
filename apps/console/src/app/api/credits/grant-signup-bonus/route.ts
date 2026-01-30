import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createAdminClient } from "@/lib/supabase/admin"

/**
 * POST /api/credits/grant-signup-bonus
 * Grant 100 free credits to new users during onboarding
 * Uses idempotency key to prevent duplicate grants
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

    const adminClient = createAdminClient()

    // Complete onboarding (grant credits + set status to active)
    const eventId = `onboarding:${user.id}`
    const { data, error } = await adminClient.rpc("complete_onboarding", {
      p_user_id: user.id,
      p_event_id: eventId,
    })

    if (error) {
      console.error("[api/credits/grant-signup-bonus] RPC error:", error)
      return NextResponse.json(
        { success: false, error: "rpc_error", message: "クレジットの付与に失敗しました" },
        { status: 500 }
      )
    }

    if (data?.success) {
      if (data.already_completed) {
        console.log(
          `[api/credits/grant-signup-bonus] Onboarding already completed for ${user.id}`
        )
        return NextResponse.json({
          success: false,
          error: "already_granted",
          message: "既にクレジットを受け取っています",
        })
      }
      console.log(
        `[api/credits/grant-signup-bonus] Onboarding completed for user ${user.id}, granted ${data.credits_granted} credits`
      )
      return NextResponse.json({
        success: true,
        credits_granted: data.credits_granted,
        status: data.status,
      })
    } else {
      console.error("[api/credits/grant-signup-bonus] Unexpected response:", data)
      return NextResponse.json(
        { success: false, error: data?.error || "unexpected", message: data?.message || "予期せぬエラー" },
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
