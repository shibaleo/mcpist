import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import { generateState } from "@/lib/oauth/state"

const TODOIST_AUTH_URL = "https://todoist.com/oauth/authorize"

// Todoist OAuth scopes
// data:read - Read user data
// data:read_write - Read and write user data
// data:delete - Delete user data
// task:add - Add tasks (Quick Add only)
const TODOIST_SCOPES = ["data:read_write", "data:delete"]

export async function GET(request: Request) {
  // 認証チェック（ユーザーセッション確認）
  const { userId } = await auth()
  if (!userId) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  // パラメータを取得
  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "todoist" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Todoist" },
        { status: 400 }
      )
    }

    // state パラメータ（HMAC-SHA256 署名付き）
    const state = generateState({ returnTo })

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
