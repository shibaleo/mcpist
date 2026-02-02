import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createClient as createAdminClient } from "@supabase/supabase-js"
import { saveDefaultToolSettings } from "@/lib/tool-settings"

const TODOIST_TOKEN_URL = "https://todoist.com/oauth/access_token"

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
  const url = new URL(request.url)
  const code = url.searchParams.get("code")
  const error = url.searchParams.get("error")
  const stateParam = url.searchParams.get("state")

  // state から returnTo を取り出す
  let returnTo = "/connections"
  if (stateParam) {
    try {
      const stateData = JSON.parse(Buffer.from(stateParam, "base64url").toString())
      if (stateData.returnTo) {
        returnTo = stateData.returnTo
      }
    } catch {
      // state のパースに失敗した場合はデフォルト値を使用
    }
  }

  // 認証チェック
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()

  if (!user) {
    return NextResponse.redirect(new URL("/login", request.url))
  }

  // エラーチェック
  if (error) {
    const errorDescription = url.searchParams.get("error_description") || error
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", errorDescription)
    return NextResponse.redirect(errorUrl)
  }

  if (!code) {
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "No authorization code received")
    return NextResponse.redirect(errorUrl)
  }

  try {
    // OAuth App の認証情報を取得（service role 権限で）
    const adminClient = getAdminClient()
    const { data: credentials, error: credError } = await adminClient.rpc("get_oauth_app_credentials", {
      p_provider: "todoist"
    })

    if (credError || !credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credError || credentials?.message)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "OAuth credentials not configured")
      return NextResponse.redirect(errorUrl)
    }

    // 認証コードをアクセストークンに交換
    // Todoist uses form-encoded POST (not JSON)
    const tokenResponse = await fetch(TODOIST_TOKEN_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        client_id: credentials.client_id,
        client_secret: credentials.client_secret,
        code,
      }),
    })

    if (!tokenResponse.ok) {
      const errorText = await tokenResponse.text()
      console.error("Token exchange failed:", errorText)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "Failed to exchange token")
      return NextResponse.redirect(errorUrl)
    }

    const tokenData = await tokenResponse.json()

    if (!tokenData.access_token) {
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "No access token received")
      return NextResponse.redirect(errorUrl)
    }

    // トークン情報を保存
    // Note: Todoist does NOT provide refresh_token - tokens are long-lived
    const tokenCredentials = {
      auth_type: "oauth2",
      access_token: tokenData.access_token,
      refresh_token: null,  // Todoist doesn't provide refresh tokens
      token_type: tokenData.token_type || "Bearer",
      expires_at: null,  // Todoist tokens don't expire (until revoked)
    }

    const { error: saveError } = await supabase.rpc("upsert_my_credential", {
      p_module: "todoist",
      p_credentials: tokenCredentials,
    })

    if (saveError) {
      console.error("Failed to save token:", saveError)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "Failed to save token")
      return NextResponse.redirect(errorUrl)
    }

    // デフォルトツール設定を保存
    await saveDefaultToolSettings(supabase, "todoist")

    // 成功時はreturnToにリダイレクト
    const redirectUrl = new URL(returnTo, request.url)
    redirectUrl.searchParams.set("success", "Todoist connected successfully")
    return NextResponse.redirect(redirectUrl)
  } catch (err) {
    console.error("OAuth callback error:", err)
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth callback failed")
    return NextResponse.redirect(errorUrl)
  }
}
