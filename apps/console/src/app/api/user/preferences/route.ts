import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"

export async function POST(request: Request) {
  try {
    const body = await request.json()

    const client = await createWorkerClient()
    const { data } = await client.PUT("/v1/user/settings", {
      body: { settings: body },
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
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/user/context")
    const ctx = data?.[0]
    return NextResponse.json(ctx?.settings || {})
  } catch (error) {
    console.error("Preferences API error:", error)
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    )
  }
}
