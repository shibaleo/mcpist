import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import { generateState } from "@/lib/oauth/state"

const GOOGLE_AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth"

// モジュールごとのスコープ定義
const MODULE_SCOPES: Record<string, string[]> = {
  google_calendar: [
    "https://www.googleapis.com/auth/calendar",
    "https://www.googleapis.com/auth/calendar.events",
  ],
  google_tasks: [
    "https://www.googleapis.com/auth/tasks",
  ],
  google_drive: [
    "https://www.googleapis.com/auth/drive",
  ],
  google_docs: [
    "https://www.googleapis.com/auth/documents",
    "https://www.googleapis.com/auth/drive",  // Comments API requires Drive scope
  ],
  google_sheets: [
    "https://www.googleapis.com/auth/spreadsheets",
    "https://www.googleapis.com/auth/drive.readonly",  // For searching spreadsheets
  ],
  google_apps_script: [
    "https://www.googleapis.com/auth/script.projects",
    "https://www.googleapis.com/auth/script.deployments",
    "https://www.googleapis.com/auth/script.metrics",
    "https://www.googleapis.com/auth/script.processes",
    "https://www.googleapis.com/auth/script.scriptapp",  // For run_function (execute scripts)
    "https://www.googleapis.com/auth/drive.readonly",  // For listing projects
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
  const moduleName = url.searchParams.get("module") || "google_calendar"

  // スコープの取得（未知のモジュールはエラー）
  const scopes = MODULE_SCOPES[moduleName]
  if (!scopes) {
    return NextResponse.json(
      { error: `Unknown module: ${moduleName}` },
      { status: 400 }
    )
  }

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "google" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Google" },
        { status: 400 }
      )
    }

    // state パラメータ（HMAC-SHA256 署名付き）
    const state = generateState({ returnTo, module: moduleName })

    // 認可URLを構築
    const params = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: credentials.redirect_uri,
      response_type: "code",
      scope: scopes.join(" "),
      access_type: "offline",  // refresh_token を取得
      prompt: "consent",  // 毎回同意画面を表示（refresh_token取得のため）
      state,
    })

    const authorizationUrl = `${GOOGLE_AUTH_URL}?${params.toString()}`

    return NextResponse.json({ authorizationUrl })
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json(
      { error: "Failed to generate authorization URL" },
      { status: 500 }
    )
  }
}
