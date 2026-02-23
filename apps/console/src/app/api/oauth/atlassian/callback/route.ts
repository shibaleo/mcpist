import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"
import { verifyState } from "@/lib/oauth/state"

const ATLASSIAN_TOKEN_URL = "https://auth.atlassian.com/oauth/token"
const ATLASSIAN_RESOURCES_URL = "https://api.atlassian.com/oauth/token/accessible-resources"

// Atlassian Cloud サイト情報を取得
async function getAccessibleResources(accessToken: string): Promise<{ id: string; url: string; name: string }[]> {
  const response = await fetch(ATLASSIAN_RESOURCES_URL, {
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "Accept": "application/json",
    },
  })

  if (!response.ok) {
    throw new Error("Failed to get accessible resources")
  }

  return response.json()
}

export async function GET(request: Request) {
  const url = new URL(request.url)
  const code = url.searchParams.get("code")
  const error = url.searchParams.get("error")
  const stateParam = url.searchParams.get("state")

  // state の署名検証 + デコード
  let returnTo = "/tools"
  let moduleName: string = ""
  try {
    const stateData = verifyState(stateParam || "")
    if (typeof stateData.returnTo === "string") returnTo = stateData.returnTo
    if (typeof stateData.module === "string") moduleName = stateData.module
  } catch {
    const errorUrl = new URL("/tools", request.url)
    errorUrl.searchParams.set("error", "Invalid or expired OAuth state")
    return NextResponse.redirect(errorUrl)
  }

  // module チェック
  if (!moduleName) {
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "Missing module in state")
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
    // OAuth App の認証情報を取得（service role 権限で）
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "atlassian" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "OAuth credentials not configured")
      return NextResponse.redirect(errorUrl)
    }

    // 認証コードをアクセストークンに交換
    const tokenResponse = await fetch(ATLASSIAN_TOKEN_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        grant_type: "authorization_code",
        client_id: credentials.client_id,
        client_secret: credentials.client_secret,
        code,
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

    // アクセス可能なリソース（Cloud サイト）を取得
    // これにより cloudId を取得し、API エンドポイントの構築に使用
    let resources: { id: string; url: string; name: string }[] = []
    try {
      resources = await getAccessibleResources(tokenData.access_token)
    } catch (err) {
      console.error("Failed to get accessible resources:", err)
      // リソース取得失敗はエラーにしない（後で手動設定可能）
    }

    // 最初のリソースのドメインを保存（複数サイトがある場合は最初のものを使用）
    const firstResource = resources[0]
    const cloudId = firstResource?.id || ""
    const siteName = firstResource?.name || ""
    // URLからドメインを抽出 (例: https://your-domain.atlassian.net -> your-domain.atlassian.net)
    const domain = firstResource?.url ? new URL(firstResource.url).hostname : ""

    // トークン情報を保存
    // expires_at は Unix timestamp (秒) で保存
    const tokenCredentials = {
      auth_type: "oauth2",
      access_token: tokenData.access_token,
      refresh_token: tokenData.refresh_token || null,
      token_type: tokenData.token_type || "Bearer",
      scope: tokenData.scope,
      expires_at: tokenData.expires_in
        ? Math.floor(Date.now() / 1000) + tokenData.expires_in
        : null,
      metadata: {
        cloud_id: cloudId,
        domain: domain,
        site_name: siteName,
      },
    }

    // モジュールにクレデンシャルを保存
    await client.PUT("/v1/me/credentials/{module}", {
      params: { path: { module: moduleName } },
      body: { credentials: tokenCredentials },
    })

    // モジュール名を表示用に変換
    const moduleDisplayNames: Record<string, string> = {
      jira: "Jira",
      confluence: "Confluence",
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
