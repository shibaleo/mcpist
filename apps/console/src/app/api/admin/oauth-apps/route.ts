import { NextRequest, NextResponse } from "next/server"
import { createClient } from "@supabase/supabase-js"
import { createClient as createServerClient } from "@/lib/supabase/server"

// Service role client for Vault operations
function createAdminClient() {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
  const secretKey = process.env.SUPABASE_SECRET_KEY

  if (!supabaseUrl || !secretKey) {
    throw new Error("Missing Supabase configuration (SUPABASE_SECRET_KEY)")
  }

  return createClient(supabaseUrl, secretKey, {
    auth: {
      autoRefreshToken: false,
      persistSession: false,
    },
  })
}

// Verify user is admin
async function verifyAdmin(): Promise<{ isAdmin: boolean; userId?: string }> {
  const supabase = await createServerClient()
  const { data: { user } } = await supabase.auth.getUser()

  if (!user) {
    return { isAdmin: false }
  }

  const { data: role } = await supabase.rpc("get_user_role")
  return { isAdmin: role === "admin", userId: user.id }
}

// GET: List OAuth apps
export async function GET(): Promise<NextResponse> {
  const { isAdmin } = await verifyAdmin()
  if (!isAdmin) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 403 })
  }

  try {
    const adminClient = createAdminClient()
    const { data, error } = await adminClient.rpc("list_oauth_apps")

    if (error) {
      console.error("[admin/oauth-apps] list error:", error)
      return NextResponse.json({ error: error.message }, { status: 500 })
    }

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

    const adminClient = createAdminClient()
    const { data, error } = await adminClient.rpc("upsert_oauth_app", {
      p_provider: provider,
      p_client_id: client_id,
      p_client_secret: client_secret,
      p_redirect_uri: redirect_uri,
      p_enabled: enabled ?? true,
    })

    if (error) {
      console.error("[admin/oauth-apps] upsert error:", error)
      return NextResponse.json({ error: error.message }, { status: 500 })
    }

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

    const adminClient = createAdminClient()
    const { data, error } = await adminClient.rpc("delete_oauth_app", {
      p_provider: provider,
    })

    if (error) {
      console.error("[admin/oauth-apps] delete error:", error)
      return NextResponse.json({ error: error.message }, { status: 500 })
    }

    return NextResponse.json(data)
  } catch (err) {
    console.error("[admin/oauth-apps] error:", err)
    return NextResponse.json({ error: "Internal server error" }, { status: 500 })
  }
}
