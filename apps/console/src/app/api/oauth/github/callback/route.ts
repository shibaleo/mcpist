import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"
import { verifyState } from "@/lib/oauth/state"

const GITHUB_TOKEN_URL = "https://github.com/login/oauth/access_token"

export async function GET(request: Request) {
  const url = new URL(request.url)
  const code = url.searchParams.get("code")
  const error = url.searchParams.get("error")
  const stateParam = url.searchParams.get("state")

  // state の署名検証 + デコード
  let returnTo = "/tools"
  try {
    const stateData = verifyState(stateParam || "")
    if (typeof stateData.returnTo === "string") returnTo = stateData.returnTo
  } catch {
    const errorUrl = new URL("/tools", request.url)
    errorUrl.searchParams.set("error", "Invalid or expired OAuth state")
    return NextResponse.redirect(errorUrl)
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
    // OAuth App の認証情報を取得
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "github" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "OAuth credentials not configured")
      return NextResponse.redirect(errorUrl)
    }

    // 認証コードをアクセストークンに交換
    const tokenResponse = await fetch(GITHUB_TOKEN_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: JSON.stringify({
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

    if (tokenData.error) {
      console.error("GitHub OAuth error:", tokenData.error, tokenData.error_description)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", tokenData.error_description || tokenData.error)
      return NextResponse.redirect(errorUrl)
    }

    if (!tokenData.access_token) {
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "No access token received")
      return NextResponse.redirect(errorUrl)
    }

    // トークン情報を保存
    // Note: GitHub OAuth tokens do NOT expire (unless revoked)
    // They also don't provide refresh tokens
    const tokenCredentials = {
      auth_type: "oauth2",
      access_token: tokenData.access_token,
      refresh_token: null,  // GitHub doesn't provide refresh tokens
      token_type: tokenData.token_type || "Bearer",
      expires_at: null,  // GitHub tokens don't expire
      scope: tokenData.scope,
    }

    await client.PUT("/v1/me/credentials/{module}", {
      params: { path: { module: "github" } },
      body: { credentials: tokenCredentials },
    })

    // 成功時はreturnToにリダイレクト
    const redirectUrl = new URL(returnTo, request.url)
    redirectUrl.searchParams.set("success", "GitHub connected successfully")
    return NextResponse.redirect(redirectUrl)
  } catch (err) {
    console.error("OAuth callback error:", err)
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth callback failed")
    return NextResponse.redirect(errorUrl)
  }
}
