/**
 * OAuth 2.1 Authorization Server Metadata (RFC 8414)
 *
 * MCPist 独自の Authorization Server メタデータを提供。
 * MCP Client は OAuth 2.1 + PKCE フローでアクセストークンを取得する。
 */
import { NextResponse } from 'next/server'

// MCPist のベース URL (issuer)
const BASE_URL = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'

export async function GET() {
  const metadata = {
    // issuer は MCPist のドメイン
    issuer: BASE_URL,
    // MCPist 独自の認可・トークンエンドポイント
    authorization_endpoint: `${BASE_URL}/api/auth/authorize`,
    token_endpoint: `${BASE_URL}/api/auth/token`,
    // MCPist 独自の JWKS エンドポイント
    jwks_uri: `${BASE_URL}/api/auth/jwks`,
    // サポートするフロー
    response_types_supported: ['code'],
    grant_types_supported: ['authorization_code', 'refresh_token'],
    code_challenge_methods_supported: ['S256'],
    token_endpoint_auth_methods_supported: ['none'],
    scopes_supported: ['openid', 'profile', 'email'],
  }

  return NextResponse.json(metadata, {
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'no-cache', // Development: disable cache
      'Access-Control-Allow-Origin': '*',
    },
  })
}
