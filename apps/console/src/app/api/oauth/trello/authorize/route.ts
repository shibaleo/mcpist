import { NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createClient as createAdminClient } from "@supabase/supabase-js"
import crypto from "crypto"

// Trello OAuth 1.0a endpoints
const TRELLO_REQUEST_TOKEN_URL = "https://trello.com/1/OAuthGetRequestToken"
const TRELLO_AUTHORIZE_URL = "https://trello.com/1/OAuthAuthorizeToken"

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
  // 認証チェック
  console.log("[trello/authorize] Starting...")
  const supabase = await createClient()
  const {
    data: { user },
    error: authError,
  } = await supabase.auth.getUser()

  console.log("[trello/authorize] Auth result:", { user: user?.id, error: authError?.message })

  if (!user) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得
    const adminClient = getAdminClient()
    const { data: credentials, error: credError } = await adminClient.rpc("get_oauth_app_credentials", {
      p_provider: "trello",
    })

    if (credError || !credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credError || credentials?.message)
      return NextResponse.json({ error: "OAuth credentials not configured for Trello" }, { status: 400 })
    }

    const consumerKey = credentials.client_id
    const consumerSecret = credentials.client_secret

    // Callback URL
    const callbackUrl = `${url.origin}/api/oauth/trello/callback`

    // Step 1: Get Request Token
    const timestamp = Math.floor(Date.now() / 1000).toString()
    const nonce = crypto.randomUUID().replace(/-/g, "")

    const oauthParams: Record<string, string> = {
      oauth_callback: callbackUrl,
      oauth_consumer_key: consumerKey,
      oauth_nonce: nonce,
      oauth_signature_method: "HMAC-SHA1",
      oauth_timestamp: timestamp,
      oauth_version: "1.0",
    }

    // Generate signature
    const signature = generateOAuthSignature(
      "POST",
      TRELLO_REQUEST_TOKEN_URL,
      oauthParams,
      consumerSecret
    )
    oauthParams.oauth_signature = signature

    // Request token
    const requestTokenResponse = await fetch(TRELLO_REQUEST_TOKEN_URL, {
      method: "POST",
      headers: {
        Authorization: buildOAuthHeader(oauthParams),
        "Content-Type": "application/x-www-form-urlencoded",
      },
    })

    if (!requestTokenResponse.ok) {
      const errorText = await requestTokenResponse.text()
      console.error("Request token failed:", errorText)
      return NextResponse.json({ error: "Failed to get request token" }, { status: 500 })
    }

    const requestTokenText = await requestTokenResponse.text()
    const requestTokenParams = new URLSearchParams(requestTokenText)
    const oauthToken = requestTokenParams.get("oauth_token")
    const oauthTokenSecret = requestTokenParams.get("oauth_token_secret")

    if (!oauthToken || !oauthTokenSecret) {
      console.error("Invalid request token response:", requestTokenText)
      return NextResponse.json({ error: "Invalid request token response" }, { status: 500 })
    }

    // Store oauth_token_secret temporarily (needed for access token exchange)
    // We'll store it in a cookie since it's needed in the callback
    const stateData = {
      oauthTokenSecret,
      returnTo,
    }
    const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")

    // Step 2: Build authorization URL
    const authParams = new URLSearchParams({
      oauth_token: oauthToken,
      name: "mcpist",
      scope: "read,write",
      expiration: "never",
    })

    const authorizationUrl = `${TRELLO_AUTHORIZE_URL}?${authParams.toString()}`

    // Create response with state cookie
    const response = NextResponse.json({ authorizationUrl })
    response.cookies.set("trello_oauth_state", state, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 10, // 10 minutes
      path: "/",
    })

    return response
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json({ error: "Failed to generate authorization URL" }, { status: 500 })
  }
}
