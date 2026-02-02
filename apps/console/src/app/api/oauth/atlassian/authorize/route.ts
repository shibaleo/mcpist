import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createClient as createAdminClient } from "@supabase/supabase-js"

const ATLASSIAN_AUTH_URL = "https://auth.atlassian.com/authorize"

// モジュールごとのスコープ定義
// Jira と Confluence で共通のスコープを使用
const MODULE_SCOPES: Record<string, string[]> = {
  jira: [
    "read:jira-work",
    "write:jira-work",
    "read:jira-user",
    "manage:jira-project",
    "offline_access",
  ],
  confluence: [
    "read:confluence-content.all",
    "write:confluence-content",
    "read:confluence-space.summary",
    "offline_access",
  ],
  // Jira + Confluence の両方を使う場合
  atlassian: [
    "read:jira-work",
    "write:jira-work",
    "read:jira-user",
    "manage:jira-project",
    "read:confluence-content.all",
    "write:confluence-content",
    "read:confluence-space.summary",
    "offline_access",
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
  const module = url.searchParams.get("module") || "atlassian"

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
      p_provider: "atlassian"
    })

    if (credError || !credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credError || credentials?.message)
      return NextResponse.json(
        { error: "OAuth credentials not configured for Atlassian" },
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
