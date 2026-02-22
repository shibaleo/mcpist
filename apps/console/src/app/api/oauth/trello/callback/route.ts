import { NextResponse } from "next/server"
import { createWorkerClient } from "@/lib/worker"
import { verifyState } from "@/lib/oauth/state"
import crypto from "crypto"
import { cookies } from "next/headers"

// Trello OAuth 1.0a endpoints
const TRELLO_ACCESS_TOKEN_URL = "https://trello.com/1/OAuthGetAccessToken"

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

  // Get state from cookie and verify signature
  const cookieStore = await cookies()
  const stateCookie = cookieStore.get("trello_oauth_state")

  let returnTo = "/tools"
  let oauthTokenSecret = ""

  try {
    const stateData = verifyState(stateCookie?.value || "")
    if (typeof stateData.returnTo === "string") returnTo = stateData.returnTo
    if (typeof stateData.oauthTokenSecret === "string") oauthTokenSecret = stateData.oauthTokenSecret
  } catch {
    const errorUrl = new URL("/tools", request.url)
    errorUrl.searchParams.set("error", "Invalid or expired OAuth state")
    const response = NextResponse.redirect(errorUrl)
    response.cookies.delete("trello_oauth_state")
    return response
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
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "trello" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
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

    // Save token to vault (OAuth 1.0a standard fields)
    const tokenCredentials = {
      auth_type: "oauth1",
      consumer_key: consumerKey,
      consumer_secret: consumerSecret,
      access_token: accessToken,
      access_token_secret: accessTokenSecret,
    }

    await client.PUT("/v1/me/credentials/{module}", {
      params: { path: { module: "trello" } },
      body: { credentials: tokenCredentials },
    })

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
