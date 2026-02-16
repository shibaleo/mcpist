import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/postgrest"

export async function POST(request: Request) {
  try {
    const supabase = await createClient()
    const { data: { user } } = await supabase.auth.getUser()

    if (!user) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const body = await request.json()

    // preferences を更新
    const data = await rpc("update_settings", {
      p_user_id: user.id,
      p_settings: body,
    })

    return NextResponse.json({ success: true, data })
  } catch (error) {
    console.error("Preferences API error:", error)
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    )
  }
}

export async function GET() {
  try {
    const supabase = await createClient()
    const { data: { user } } = await supabase.auth.getUser()

    if (!user) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const data = await rpc<{ settings: Record<string, unknown> }[]>("get_user_context", {
      p_user_id: user.id,
    })
    const ctx = Array.isArray(data) ? data[0] : data
    return NextResponse.json(ctx?.settings || {})
  } catch (error) {
    console.error("Preferences API error:", error)
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    )
  }
}
