import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/postgrest"

const TODOIST_AUTH_URL = "https://todoist.com/oauth/authorize"

// Todoist OAuth scopes
// data:read - Read user data
// data:read_write - Read and write user data
// data:delete - Delete user data
// task:add - Add tasks (Quick Add only)
const TODOIST_SCOPES = ["data:read_write", "data:delete"]

export async function GET(request: Request) {
  // 認証チェック（ユーザーセッション確認）
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()

  if (!user) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  // パラメータを取得
  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    const credentials = await rpc<{ client_id: string; client_secret: string; redirect_uri: string; scopes?: string; error?: string; message?: string }>(
      "get_oauth_app_credentials",
      { p_provider: "todoist" }
    )

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Todoist" },
        { status: 400 }
      )
    }

    // state パラメータを生成（CSRF対策 + returnTo情報）
    const stateData = {
      nonce: crypto.randomUUID(),
      returnTo,
    }
    const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")

    // 認可URLを構築
    const params = new URLSearchParams({
      client_id: credentials.client_id,
      scope: TODOIST_SCOPES.join(","),
      state,
    })

    const authorizationUrl = `${TODOIST_AUTH_URL}?${params.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json(
      { error: "Failed to generate authorization URL" },
      { status: 500 }
    )
  }
}
