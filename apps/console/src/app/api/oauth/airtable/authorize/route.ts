import { NextResponse } from "next/server"
import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import crypto from "crypto"
import { generateState } from "@/lib/oauth/state"

const AIRTABLE_AUTHORIZE_URL = "https://airtable.com/oauth2/v1/authorize"

const AIRTABLE_SCOPES = [
  "data.records:read",
  "data.records:write",
  "schema.bases:read",
  "schema.bases:write",
]

// PKCE: Generate code_verifier (43-128 characters, URL-safe)
function generateCodeVerifier(): string {
  return crypto.randomBytes(64).toString("base64url")
}

// PKCE: Generate code_challenge from code_verifier (S256)
function generateCodeChallenge(verifier: string): string {
  return crypto.createHash("sha256").update(verifier).digest("base64url")
}

export async function GET(request: Request) {
  // 認証チェック
  const { userId } = await auth()
  if (!userId) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const url = new URL(request.url)
  const returnTo = url.searchParams.get("returnTo") || "/tools"

  try {
    // OAuth App の認証情報を取得
    const client = await createWorkerClient()
    const { data: credentials } = await client.GET("/v1/oauth/apps/{provider}/credentials", {
      params: { path: { provider: "airtable" } },
    })

    if (!credentials || credentials.error) {
      console.error("Failed to get OAuth credentials:", credentials?.message)
      return NextResponse.json({ error: "OAuth credentials not configured for Airtable" }, { status: 400 })
    }

    // PKCE: code_verifier と code_challenge を生成
    const codeVerifier = generateCodeVerifier()
    const codeChallenge = generateCodeChallenge(codeVerifier)

    // state パラメータ（HMAC-SHA256 署名付き）
    const state = generateState({ returnTo })

    // Airtable OAuth 認可URL構築
    // https://airtable.com/developers/web/api/oauth-reference#authorize
    const authParams = new URLSearchParams({
      client_id: credentials.client_id,
      redirect_uri: credentials.redirect_uri,
      response_type: "code",
      scope: AIRTABLE_SCOPES.join(" "),
      state,
      code_challenge: codeChallenge,
      code_challenge_method: "S256",
    })

    const authorizationUrl = `${AIRTABLE_AUTHORIZE_URL}?${authParams.toString()}`

    // code_verifier を Cookie に保存（callback で使用）
    const response = NextResponse.json({ authorizationUrl })
    response.cookies.set("airtable_code_verifier", codeVerifier, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 600, // 10 minutes
      path: "/api/oauth/airtable",
    })

    return response
  } catch (err) {
    console.error("Failed to generate authorization URL:", err)
    return NextResponse.json({ error: "Failed to generate authorization URL" }, { status: 500 })
  }
}
