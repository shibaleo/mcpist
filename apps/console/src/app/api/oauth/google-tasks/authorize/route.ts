import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createClient as createAdminClient } from "@supabase/supabase-js"

const GOOGLE_AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth"
const GOOGLE_TASKS_SCOPES = [
  "https://www.googleapis.com/auth/tasks",
]

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

  // returnTo パラメータを取得
  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    // Google Tasks は google プロバイダーの OAuth App を共有
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

    // state パラメータを生成（CSRF対策 + returnTo情報）
    const stateData = {
      nonce: crypto.randomUUID(),
      returnTo,
      module: "google_tasks", // callbackでモジュールを識別
    }
    const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")

    // redirect_uri を google-tasks 用に変更
    const redirectUri = credentials.redirect_uri.replace("/google/callback", "/google-tasks/callback")

    // 認可URLを構築
    const params = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: redirectUri,
      response_type: "code",
      scope: GOOGLE_TASKS_SCOPES.join(" "),
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
