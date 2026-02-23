import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"

export async function POST(request: Request) {
  try {
    const body = await request.json()

    const client = await createWorkerClient()
    const { data } = await client.PUT("/v1/me/settings", {
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
    const { data } = await client.GET("/v1/me/profile")
    return NextResponse.json(data?.settings || {})
  } catch (error) {
    console.error("Preferences API error:", error)
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    )
  }
}
