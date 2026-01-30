import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"

export async function POST(request: Request) {
  try {
    const supabase = await createClient()
    const { data: { user } } = await supabase.auth.getUser()

    if (!user) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const body = await request.json()

    // preferences を更新
    const { data, error } = await supabase.rpc("update_my_preferences", {
      p_preferences: body,
    })

    if (error) {
      console.error("Failed to update preferences:", error)
      return NextResponse.json(
        { error: "Failed to update preferences" },
        { status: 500 }
      )
    }

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

    const { data, error } = await supabase.rpc("get_my_preferences")

    if (error) {
      console.error("Failed to get preferences:", error)
      return NextResponse.json(
        { error: "Failed to get preferences" },
        { status: 500 }
      )
    }

    return NextResponse.json(data || {})
  } catch (error) {
    console.error("Preferences API error:", error)
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    )
  }
}
