import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createClient as createAdminClient } from "@supabase/supabase-js"
import { saveDefaultToolSettings } from "@/lib/tool-settings"
import crypto from "crypto"
import { cookies } from "next/headers"

// Trello OAuth 1.0a endpoints
const TRELLO_ACCESS_TOKEN_URL = "https://trello.com/1/OAuthGetAccessToken"

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

// OAuth 1.0a signature generation
function generateOAuthSignature(
  method: string,
  url: string,
  params: Record<string, string>,
  consumerSecret: string,
  tokenSecret: string = ""
): string {
  // Sort parameters alphabetically
  const sortedParams = Object.keys(params)
    .sort()
    .map((key) => `${encodeURIComponent(key)}=${encodeURIComponent(params[key])}`)
    .join("&")

  // Create signature base string
  const signatureBaseString = [
    method.toUpperCase(),
    encodeURIComponent(url),
    encodeURIComponent(sortedParams),
  ].join("&")

  // Create signing key
  const signingKey = `${encodeURIComponent(consumerSecret)}&${encodeURIComponent(tokenSecret)}`

  // Generate HMAC-SHA1 signature
  const signature = crypto
    .createHmac("sha1", signingKey)
    .update(signatureBaseString)
    .digest("base64")

  return signature
}

// Build OAuth Authorization header
function buildOAuthHeader(params: Record<string, string>): string {
  const headerParams = Object.keys(params)
    .sort()
    .map((key) => `${encodeURIComponent(key)}="${encodeURIComponent(params[key])}"`)
    .join(", ")
  return `OAuth ${headerParams}`
}

export async function GET(request: Request) {
  const url = new URL(request.url)
  const oauthToken = url.searchParams.get("oauth_token")
  const oauthVerifier = url.searchParams.get("oauth_verifier")

  // Get state from cookie
  const cookieStore = await cookies()
  const stateCookie = cookieStore.get("trello_oauth_state")

  let returnTo = "/tools"
  let oauthTokenSecret = ""

  if (stateCookie?.value) {
    try {
      const stateData = JSON.parse(Buffer.from(stateCookie.value, "base64url").toString())
      returnTo = stateData.returnTo || "/tools"
      oauthTokenSecret = stateData.oauthTokenSecret || ""
    } catch {
      // Parse error - use defaults
    }
  }

  // 認証チェック
  const supabase = await createClient()
  const {
    data: { user },
  } = await supabase.auth.getUser()

  if (!user) {
    return NextResponse.redirect(new URL("/login", request.url))
  }

  // Error handling
  if (!oauthToken || !oauthVerifier) {
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth authorization was denied or failed")
    const response = NextResponse.redirect(errorUrl)
    response.cookies.delete("trello_oauth_state")
    return response
  }

  if (!oauthTokenSecret) {
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth state expired. Please try again.")
    const response = NextResponse.redirect(errorUrl)
    response.cookies.delete("trello_oauth_state")
    return response
  }

  try {
    // OAuth App の認証情報を取得
    const adminClient = getAdminClient()
    const { data: credentials, error: credError } = await adminClient.rpc("get_oauth_app_credentials", {
      p_provider: "trello",
    })

    if (credError || !credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credError || credentials?.message)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "OAuth credentials not configured")
      const response = NextResponse.redirect(errorUrl)
      response.cookies.delete("trello_oauth_state")
      return response
    }

    const consumerKey = credentials.client_id
    const consumerSecret = credentials.client_secret

    // Step 3: Exchange request token for access token
    const timestamp = Math.floor(Date.now() / 1000).toString()
    const nonce = crypto.randomUUID().replace(/-/g, "")

    const oauthParams: Record<string, string> = {
      oauth_consumer_key: consumerKey,
      oauth_nonce: nonce,
      oauth_signature_method: "HMAC-SHA1",
      oauth_timestamp: timestamp,
      oauth_token: oauthToken,
      oauth_verifier: oauthVerifier,
      oauth_version: "1.0",
    }

    // Generate signature with token secret
    const signature = generateOAuthSignature(
      "POST",
      TRELLO_ACCESS_TOKEN_URL,
      oauthParams,
      consumerSecret,
      oauthTokenSecret
    )
    oauthParams.oauth_signature = signature

    // Request access token
    const accessTokenResponse = await fetch(TRELLO_ACCESS_TOKEN_URL, {
      method: "POST",
      headers: {
        Authorization: buildOAuthHeader(oauthParams),
        "Content-Type": "application/x-www-form-urlencoded",
      },
    })

    if (!accessTokenResponse.ok) {
      const errorText = await accessTokenResponse.text()
      console.error("Access token exchange failed:", errorText)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "Failed to exchange access token")
      const response = NextResponse.redirect(errorUrl)
      response.cookies.delete("trello_oauth_state")
      return response
    }

    const accessTokenText = await accessTokenResponse.text()
    const accessTokenParams = new URLSearchParams(accessTokenText)
    const accessToken = accessTokenParams.get("oauth_token")
    const accessTokenSecret = accessTokenParams.get("oauth_token_secret")

    if (!accessToken || !accessTokenSecret) {
      console.error("Invalid access token response:", accessTokenText)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "Invalid access token response")
      const response = NextResponse.redirect(errorUrl)
      response.cookies.delete("trello_oauth_state")
      return response
    }

    // Save token to vault
    // For Trello OAuth 1.0a:
    // - access_token: OAuth token
    // - username: API Key (consumer key) - needed for API calls
    // - metadata.token_secret: OAuth token secret - needed for signing requests
    const tokenCredentials = {
      auth_type: "oauth1",
      access_token: accessToken,
      username: consumerKey, // API Key stored in username for module.go compatibility
      refresh_token: null,
      token_type: "OAuth",
      expires_at: null, // Trello tokens don't expire (expiration: never)
      metadata: {
        token_secret: accessTokenSecret,
      },
    }

    const { error: saveError } = await supabase.rpc("upsert_my_credential", {
      p_module: "trello",
      p_credentials: tokenCredentials,
    })

    if (saveError) {
      console.error("Failed to save token:", saveError)
      const errorUrl = new URL(returnTo, request.url)
      errorUrl.searchParams.set("error", "Failed to save token")
      const response = NextResponse.redirect(errorUrl)
      response.cookies.delete("trello_oauth_state")
      return response
    }

    // デフォルトツール設定を保存
    await saveDefaultToolSettings(supabase, "trello")

    // 成功
    const redirectUrl = new URL(returnTo, request.url)
    redirectUrl.searchParams.set("success", "Trello connected successfully")
    const response = NextResponse.redirect(redirectUrl)
    response.cookies.delete("trello_oauth_state")
    return response
  } catch (err) {
    console.error("OAuth callback error:", err)
    const errorUrl = new URL(returnTo, request.url)
    errorUrl.searchParams.set("error", "OAuth callback failed")
    const response = NextResponse.redirect(errorUrl)
    response.cookies.delete("trello_oauth_state")
    return response
  }
}
