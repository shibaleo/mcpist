import { NextRequest, NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/worker-client"

interface UserContextRow {
  role: string
}

// Verify user is admin
async function verifyAdmin(): Promise<{ isAdmin: boolean; userId?: string }> {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()

  if (!user) {
    return { isAdmin: false }
  }

  const rows = await rpc<UserContextRow[]>("get_user_context", { p_user_id: user.id })
  const ctx = Array.isArray(rows) ? rows[0] : rows
  const role = (ctx as UserContextRow | undefined)?.role
  return { isAdmin: role === "admin", userId: user.id }
}

// GET: List OAuth apps
export async function GET(): Promise<NextResponse> {
  const { isAdmin } = await verifyAdmin()
  if (!isAdmin) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 403 })
  }

  try {
    const data = await rpc("list_oauth_apps")
    return NextResponse.json(data || [])
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    return NextResponse.json({ error: "Internal server error" }, { status: 500 })
  }
}

// POST: Upsert OAuth app
export async function POST(request: NextRequest): Promise<NextResponse> {
  const { isAdmin } = await verifyAdmin()
  if (!isAdmin) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 403 })
  }

  try {
    const body = await request.json()
    const { provider, client_id, client_secret, redirect_uri, enabled } = body

    if (!provider || !client_id) {
      return NextResponse.json({ error: "provider and client_id are required" }, { status: 400 })
    }

    const data = await rpc("upsert_oauth_app", {
      p_provider: provider,
      p_client_id: client_id,
      p_client_secret: client_secret,
      p_redirect_uri: redirect_uri,
      p_enabled: enabled ?? true,
    })

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    return NextResponse.json({ error: "Internal server error" }, { status: 500 })
  }
}

// DELETE: Delete OAuth app
export async function DELETE(request: NextRequest): Promise<NextResponse> {
  const { isAdmin } = await verifyAdmin()
  if (!isAdmin) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 403 })
  }

  try {
    const { searchParams } = new URL(request.url)
    const provider = searchParams.get("provider")

    if (!provider) {
      return NextResponse.json({ error: "provider is required" }, { status: 400 })
    }

    const data = await rpc("delete_oauth_app", {
      p_provider: provider,
    })

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    return NextResponse.json({ error: "Internal server error" }, { status: 500 })
  }
}
