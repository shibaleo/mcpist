import { NextResponse } from "next/server"
import { workerFetch } from "@/lib/worker-client"

export async function POST(request: Request) {
  try {
    const body = await request.json()

    const data = await workerFetch("PUT", "/v1/user/settings", {
      settings: body,
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
    const data = await workerFetch<{ settings: Record<string, unknown> }[]>("GET", "/v1/user/context")
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
