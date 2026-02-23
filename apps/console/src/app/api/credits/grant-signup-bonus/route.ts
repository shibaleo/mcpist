import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"

/**
 * POST /api/credits/grant-signup-bonus
 * Activate new user account (set status to active, assign free plan)
 */
export async function POST() {
  try {
    const client = await createWorkerClient()
    const { data: result } = await client.POST("/v1/me/onboarding", {
      body: { event_id: "onboarding" },
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
