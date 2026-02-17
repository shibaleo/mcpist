import { NextResponse } from "next/server"
import { workerFetch } from "@/lib/worker-client"
import { saveDefaultToolSettings } from "@/lib/mcp/tool-settings"

const GOOGLE_TOKEN_URL = "https://oauth2.googleapis.com/token"

export async function GET(request: Request) {
  const url = new URL(request.url)
  const code = url.searchParams.get("code")
  const error = url.searchParams.get("error")
  const stateParam = url.searchParams.get("state")

  // state から returnTo と module を取り出す
  let returnTo = "/tools"
  let moduleName: string = "google_calendar"  // デフォルト（後方互換性）
  if (stateParam) {
    try {
      const stateData = JSON.parse(Buffer.from(stateParam, "base64url").toString())
      if (stateData.returnTo) {
        returnTo = stateData.returnTo
      }
      if (stateData.module) {
        moduleName = stateData.module as string
      }
    } catch {
      // state のパースに失敗した場合はデフォルト値を使用
    }
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
    const credentials = await workerFetch<{ client_id: string; client_secret: string; redirect_uri: string; error?: string; message?: string }>(
      "GET", "/v1/oauth/apps/google/credentials"
    )

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "OAuth credentials not configured")
      return NextResponse.redirect(errorUrl)
    }

    // 認証コードをアクセストークンに交換
    const tokenResponse = await fetch(GOOGLE_TOKEN_URL, {
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
    // expires_at は Unix timestamp (秒) で保存 - dtl-itr-MOD-TVL.md 仕様準拠
    const tokenCredentials = {
      auth_type: "oauth2",
      access_token: tokenData.access_token,
      refresh_token: tokenData.refresh_token || null,
      token_type: tokenData.token_type || "Bearer",
      scope: tokenData.scope,
      expires_at: tokenData.expires_in
        ? Math.floor(Date.now() / 1000) + tokenData.expires_in
        : null,
    }

    await workerFetch("PUT", "/v1/credentials", {
      module: moduleName,
      credentials: tokenCredentials,
    })

    // デフォルトツール設定を保存
    await saveDefaultToolSettings(null, moduleName)

    // モジュール名を表示用に変換
    const moduleDisplayNames: Record<string, string> = {
      google_calendar: "Google Calendar",
      google_tasks: "Google Tasks",
      google_drive: "Google Drive",
      google_docs: "Google Docs",
      google_sheets: "Google Sheets",
    }
    const displayName = moduleDisplayNames[moduleName] || moduleName

    // 成功時はreturnToにリダイレクト
    const redirectUrl = new URL(returnTo, request.url)
    redirectUrl.searchParams.set("success", `${displayName} connected successfully`)
    return NextResponse.redirect(redirectUrl)
  } catch (err) {
    console.error("OAuth callback error:", err)
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth callback failed")
    return NextResponse.redirect(errorUrl)
  }
}
