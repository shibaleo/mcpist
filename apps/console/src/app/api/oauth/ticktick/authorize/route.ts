import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import { generateState } from "@/lib/oauth/state"

const TICKTICK_AUTHORIZE_URL = "https://ticktick.com/oauth/authorize"

const TICKTICK_SCOPES = [
  "tasks:read",
  "tasks:write",
]

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
      params: { path: { provider: "ticktick" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json({ error: "OAuth credentials not configured for TickTick" }, { status: 400 })
    }

    // state パラメータ（HMAC-SHA256 署名付き）
    const state = generateState({ returnTo })

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
