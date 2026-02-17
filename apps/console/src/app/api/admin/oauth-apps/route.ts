import { NextRequest, NextResponse } from "next/server"
import { workerFetch } from "@/lib/worker-client"

// GET: List OAuth apps (Worker handles admin check)
export async function GET(): Promise<NextResponse> {
  try {
    const data = await workerFetch("GET", "/v1/admin/oauth/apps")
    return NextResponse.json(data || [])
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    const status = err instanceof Error && err.message.includes("403") ? 403 : 500
    return NextResponse.json({ error: status === 403 ? "Forbidden" : "Internal server error" }, { status })
  }
}

// POST: Upsert OAuth app (Worker handles admin check)
export async function POST(request: NextRequest): Promise<NextResponse> {
  try {
    const body = await request.json()
    const { provider, client_id, client_secret, redirect_uri, enabled } = body

    if (!provider || !client_id) {
      return NextResponse.json({ error: "provider and client_id are required" }, { status: 400 })
    }

    const data = await workerFetch("PUT", "/v1/admin/oauth/apps", {
      provider,
      client_id,
      client_secret,
      redirect_uri,
      enabled: enabled ?? true,
    })

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    const status = err instanceof Error && err.message.includes("403") ? 403 : 500
    return NextResponse.json({ error: status === 403 ? "Forbidden" : "Internal server error" }, { status })
  }
}

// DELETE: Delete OAuth app (Worker handles admin check)
export async function DELETE(request: NextRequest): Promise<NextResponse> {
  try {
    const { searchParams } = new URL(request.url)
    const provider = searchParams.get("provider")

    if (!provider) {
      return NextResponse.json({ error: "provider is required" }, { status: 400 })
    }

    const data = await workerFetch("DELETE", `/v1/admin/oauth/apps/${encodeURIComponent(provider)}`)

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    const status = err instanceof Error && err.message.includes("403") ? 403 : 500
    return NextResponse.json({ error: status === 403 ? "Forbidden" : "Internal server error" }, { status })
  }
}
