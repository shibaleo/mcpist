import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/postgrest"

const GITHUB_AUTHORIZE_URL = "https://github.com/login/oauth/authorize"

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
      { p_provider: "github" }
    )

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json({ error: "OAuth credentials not configured for GitHub" }, { status: 400 })
    }

    // state パラメータ（CSRF対策 + returnTo保存）
    const stateData = { returnTo }
    const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")

    // GitHub OAuth 認可URL構築
    const authParams = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: credentials.redirect_uri,
      scope: "repo read:user",
      state,
    })

    const authorizationUrl = `${GITHUB_AUTHORIZE_URL}?${authParams.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json({ error: "Failed to generate authorization URL" }, { status: 500 })
  }
}
