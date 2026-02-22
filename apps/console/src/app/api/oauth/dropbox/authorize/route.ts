import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import { generateState } from "@/lib/oauth/state"

const DROPBOX_AUTHORIZE_URL = "https://www.dropbox.com/oauth2/authorize"

export async function GET(request: Request) {
  // 認証チェック
  const { userId } = await auth()
  if (!userId) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "dropbox" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json({ error: "OAuth credentials not configured for Dropbox" }, { status: 400 })
    }

    // state パラメータ（HMAC-SHA256 署名付き）
    const state = generateState({ returnTo })

    // Dropbox OAuth 認可URL構築
    // https://www.dropbox.com/developers/documentation/http/documentation#oauth2-authorize
    const authParams = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: credentials.redirect_uri,
      response_type: "code",
      state,
      token_access_type: "offline",
    })

    const authorizationUrl = `${DROPBOX_AUTHORIZE_URL}?${authParams.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json({ error: "Failed to generate authorization URL" }, { status: 500 })
  }
}
