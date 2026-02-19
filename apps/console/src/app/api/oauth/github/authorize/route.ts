import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"

const GITHUB_AUTHORIZE_URL = "https://github.com/login/oauth/authorize"

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
      params: { path: { provider: "github" } },
    })

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
