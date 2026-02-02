import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createClient as createAdminClient } from "@supabase/supabase-js"

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

function getAdminClient() {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
  const secretKey = process.env.SUPABASE_SECRET_KEY
  if (!supabaseUrl || !secretKey) {
    throw new Error("Missing Supabase configuration")
  }
  return createAdminClient(supabaseUrl, secretKey, {
    auth: { autoRefreshToken: false, persistSession: false },
  })
}

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
  const module = url.searchParams.get("module") || "google_calendar"

  // スコープの取得（未知のモジュールはエラー）
  const scopes = MODULE_SCOPES[module]
  if (!scopes) {
    return NextResponse.json(
      { error: `Unknown module: ${module}` },
      { status: 400 }
    )
  }

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    const adminClient = getAdminClient()
    const { data: credentials, error: credError } = await adminClient.rpc("get_oauth_app_credentials", {
      p_provider: "google"
    })

    if (credError || !credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credError || credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Google" },
        { status: 400 }
      )
    }

    // state パラメータを生成（CSRF対策 + returnTo情報 + モジュール識別）
    const stateData = {
      nonce: crypto.randomUUID(),
      returnTo,
      module,
    }
    const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")

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
