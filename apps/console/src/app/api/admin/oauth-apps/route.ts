import { NextRequest, NextResponse } from "next/server"
import { createWorkerClient, WorkerAPIError } from "@/lib/worker"

// GET: List OAuth apps (Worker handles admin check)
export async function GET(): Promise<NextResponse> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/admin/oauth/apps")
    const apps = (data || []).map((app: Record<string, unknown>) => ({
      ...app,
      has_credentials: !!app.client_id,
    }))
    return NextResponse.json(apps)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    const status = err instanceof WorkerAPIError && err.status === 403 ? 403 : 500
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

    const client = await createWorkerClient()
    const { data } = await client.PUT("/v1/admin/oauth/apps/{provider}", {
      params: { path: { provider } },
      body: {
        client_id,
        client_secret,
        redirect_uri,
        enabled: enabled ?? true,
      },
    })

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    const status = err instanceof WorkerAPIError && err.status === 403 ? 403 : 500
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

    const client = await createWorkerClient()
    const { data } = await client.DELETE("/v1/admin/oauth/apps/{provider}", {
      params: { path: { provider } },
    })

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    const status = err instanceof WorkerAPIError && err.status === 403 ? 403 : 500
    return NextResponse.json({ error: status === 403 ? "Forbidden" : "Internal server error" }, { status })
  }
}
