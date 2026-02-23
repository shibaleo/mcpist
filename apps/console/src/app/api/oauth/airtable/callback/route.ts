import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"
import { verifyState } from "@/lib/oauth/state"

const AIRTABLE_TOKEN_URL = "https://airtable.com/oauth2/v1/token"

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

  // Cookie から code_verifier を取得（PKCE）
  const cookieHeader = request.headers.get("cookie") || ""
  const cookies = Object.fromEntries(
    cookieHeader.split(";").map(c => {
      const [key, ...rest] = c.trim().split("=")
      return [key, rest.join("=")]
    })
  )
  const codeVerifier = cookies["airtable_code_verifier"]

  if (!codeVerifier) {
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "PKCE code_verifier not found. Please try connecting again.")
    return NextResponse.redirect(errorUrl)
  }

  try {
    // OAuth App の認証情報を取得
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "airtable" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "OAuth credentials not configured")
      return NextResponse.redirect(errorUrl)
    }

    // 認証コードをアクセストークンに交換
    // Airtable requires Basic auth header with client_id:client_secret
    // https://airtable.com/developers/web/api/oauth-reference#token
    const basicAuth = Buffer.from(`${credentials.client_id}:${credentials.client_secret}`).toString("base64")

    const tokenParams = new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: credentials.redirect_uri,
      code_verifier: codeVerifier,
    })

    const tokenResponse = await fetch(AIRTABLE_TOKEN_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        "Authorization": `Basic ${basicAuth}`,
      },
      body: tokenParams.toString(),
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
      console.error("Airtable OAuth error:", tokenData.error, tokenData.error_description)
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
    // Airtable tokens expire in ~2 months, refresh tokens also provided
    const expiresAt = tokenData.expires_in
      ? Math.floor(Date.now() / 1000) + tokenData.expires_in
      : null

    const tokenCredentials = {
      auth_type: "oauth2",
      access_token: tokenData.access_token,
      refresh_token: tokenData.refresh_token || null,
      token_type: tokenData.token_type || "Bearer",
      scope: tokenData.scope || null,
      expires_at: expiresAt,
    }

    await client.PUT("/v1/me/credentials/{module}", {
      params: { path: { module: "airtable" } },
      body: { credentials: tokenCredentials },
    })

    // code_verifier Cookie を削除して成功リダイレクト
    const redirectUrl = new URL(returnTo, request.url)
    redirectUrl.searchParams.set("success", "Airtable connected successfully")
    const response = NextResponse.redirect(redirectUrl)
    response.cookies.delete("airtable_code_verifier")
    return response
  } catch (err) {
    console.error("OAuth callback error:", err)
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth callback failed")
    return NextResponse.redirect(errorUrl)
  }
}
