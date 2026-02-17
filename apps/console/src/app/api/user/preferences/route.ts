import { NextResponse } from "next/server"
import { rpc } from "@/lib/worker-client"

export async function POST(request: Request) {
  try {
    const body = await request.json()

    const data = await rpc("update_settings", {
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
    const data = await rpc<{ settings: Record<string, unknown> }[]>("get_user_context")
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
