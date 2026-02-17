import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { rpc } from "@/lib/worker-client"

const NOTION_AUTH_URL = "https://api.notion.com/v1/oauth/authorize"

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
      { p_provider: "notion" }
    )

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Notion" },
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
    // Notion OAuth の仕様に従う
    const params = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: credentials.redirect_uri,
      response_type: "code",
      owner: "user",
      state,
    })

    const authorizationUrl = `${NOTION_AUTH_URL}?${params.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json(
      { error: "Failed to generate authorization URL" },
      { status: 500 }
    )
  }
}
