import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import { generateState } from "@/lib/oauth/state"

const ATLASSIAN_AUTH_URL = "https://auth.atlassian.com/authorize"

// モジュールごとのスコープ定義
const MODULE_SCOPES: Record<string, string[]> = {
  jira: [
    "read:jira-work",
    "write:jira-work",
    "read:jira-user",
    "manage:jira-project",
    "offline_access",
  ],
  confluence: [
    // Granular scopes (required for Confluence Cloud API v2)
    "read:space:confluence",
    "read:page:confluence",
    "write:page:confluence",
    "read:content-details:confluence",
    "write:comment:confluence",
    "read:comment:confluence",
    "read:label:confluence",
    "write:label:confluence",
    "search:confluence",
    "offline_access",
  ],
}

export async function GET(request: Request) {
  // 認証チェック（ユーザーセッション確認）
  const { userId } = await auth()
  if (!userId) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  // パラメータを取得
  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"
  const moduleName = url.searchParams.get("module")
  if (!moduleName || !(moduleName in MODULE_SCOPES)) {
    return NextResponse.json(
      { error: `Unknown or missing module: ${moduleName}` },
      { status: 400 }
    )
  }

  const scopes = MODULE_SCOPES[moduleName]

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "atlassian" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Atlassian" },
        { status: 400 }
      )
    }

    // state パラメータ（HMAC-SHA256 署名付き）
    const state = generateState({ returnTo, module: moduleName })

    // 認可URLを構築
    // Atlassian OAuth 2.0 (3LO) の仕様に従う
    const params = new URLSearchParams({
      audience: "api.atlassian.com",
      client_id: credentials.client_id,
      scope: scopes.join(" "),
      redirect_uri: credentials.redirect_uri,
      state,
      response_type: "code",
      prompt: "consent",  // 毎回同意画面を表示（refresh_token取得のため）
    })

    const authorizationUrl = `${ATLASSIAN_AUTH_URL}?${params.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json(
      { error: "Failed to generate authorization URL" },
      { status: 500 }
    )
  }
}
