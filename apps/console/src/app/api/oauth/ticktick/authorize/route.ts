import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/postgrest"

const TICKTICK_AUTHORIZE_URL = "https://ticktick.com/oauth/authorize"

const TICKTICK_SCOPES = [
  "tasks:read",
  "tasks:write",
]

export async function GET(request: Request) {
  // 認証チェック
  const supabase = await createClient()
  const {
    data: { user },
  } = await supabase.auth.getUser()

  if (!user) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得
    const credentials = await rpc<{ client_id: string; client_secret: string; redirect_uri: string; scopes?: string; error?: string; message?: string }>(
      "get_oauth_app_credentials",
      { p_provider: "ticktick" }
    )

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json({ error: "OAuth credentials not configured for TickTick" }, { status: 400 })
    }

    // state パラメータ（CSRF対策 + returnTo保存）
    const stateData = { returnTo }
    const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")

    // TickTick OAuth 認可URL構築
    // https://developer.ticktick.com/api#/openapi
    const authParams = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: credentials.redirect_uri,
      response_type: "code",
      scope: TICKTICK_SCOPES.join(" "),
      state,
    })

    const authorizationUrl = `${TICKTICK_AUTHORIZE_URL}?${authParams.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json({ error: "Failed to generate authorization URL" }, { status: 500 })
  }
}
