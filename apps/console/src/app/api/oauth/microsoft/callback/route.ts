import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"

const MICROSOFT_TOKEN_URL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"

export async function GET(request: Request) {
  const url = new URL(request.url)
  const code = url.searchParams.get("code")
  const error = url.searchParams.get("error")
  const state = url.searchParams.get("state")

  // 認証チェック
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()

  if (!user) {
    return NextResponse.redirect(new URL("/login", request.url))
  }

  // エラーチェック
  if (error) {
    const errorDescription = url.searchParams.get("error_description") || error
    return NextResponse.redirect(
      new URL(`/connections?error=${encodeURIComponent(errorDescription)}`, request.url)
    )
  }

  if (!code) {
    return NextResponse.redirect(
      new URL("/connections?error=No authorization code received", request.url)
    )
  }

  try {
    // OAuth App の認証情報を取得
    const { data: credentials, error: credError } = await supabase.rpc("get_oauth_app_credentials", {
      p_provider: "microsoft"
    })

    if (credError || !credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credError || credentials?.message)
      return NextResponse.redirect(
        new URL("/connections?error=OAuth credentials not configured", request.url)
      )
    }

    // 認証コードをアクセストークンに交換
    const tokenResponse = await fetch(MICROSOFT_TOKEN_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        client_id: credentials.client_id,
        client_secret: credentials.client_secret,
        code,
        grant_type: "authorization_code",
        redirect_uri: credentials.redirect_uri,
        scope: "offline_access Tasks.ReadWrite",
      }),
    })

    if (!tokenResponse.ok) {
      const errorText = await tokenResponse.text()
      console.error("Token exchange failed:", errorText)
      return NextResponse.redirect(
        new URL(`/connections?error=${encodeURIComponent("Failed to exchange token")}`, request.url)
      )
    }

    const tokenData = await tokenResponse.json()

    if (!tokenData.access_token) {
      return NextResponse.redirect(
        new URL("/connections?error=No access token received", request.url)
      )
    }

    // トークン情報を保存
    const tokenCredentials = {
      access_token: tokenData.access_token,
      refresh_token: tokenData.refresh_token || null,
      token_type: tokenData.token_type || "Bearer",
      scope: tokenData.scope,
      expires_at: tokenData.expires_in
        ? new Date(Date.now() + tokenData.expires_in * 1000).toISOString()
        : null,
    }

    const { error: saveError } = await supabase.rpc("upsert_service_token", {
      p_service: "microsoft_todo",
      p_credentials: tokenCredentials,
    })

    if (saveError) {
      console.error("Failed to save token:", saveError)
      return NextResponse.redirect(
        new URL(`/connections?error=${encodeURIComponent("Failed to save token")}`, request.url)
      )
    }

    // 成功時はconnectionsページにリダイレクト
    return NextResponse.redirect(
      new URL("/connections?success=Microsoft To Do connected successfully", request.url)
    )
  } catch (err) {
    console.error("OAuth callback error:", err)
    return NextResponse.redirect(
      new URL(`/connections?error=${encodeURIComponent("OAuth callback failed")}`, request.url)
    )
  }
}
