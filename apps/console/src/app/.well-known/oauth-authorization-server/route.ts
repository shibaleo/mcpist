/**
 * OAuth 2.1 Authorization Server Metadata (RFC 8414)
 *
 * Supabase OAuth Server のエンドポイントを返す。
 * MCP Client は OAuth 2.1 + PKCE フローでアクセストークンを取得する。
 */
import { NextResponse } from 'next/server'

// Supabase URL
const SUPABASE_URL = process.env.NEXT_PUBLIC_SUPABASE_URL

export async function GET() {
  const metadata = {
    // issuer は Supabase のドメイン
    issuer: SUPABASE_URL,
    // Supabase OAuth Server エンドポイント
    authorization_endpoint: `${SUPABASE_URL}/auth/v1/oauth/authorize`,
    token_endpoint: `${SUPABASE_URL}/auth/v1/oauth/token`,
    // Supabase JWKS エンドポイント
    jwks_uri: `${SUPABASE_URL}/auth/v1/.well-known/jwks.json`,
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
      'Cache-Control': 'no-cache',
      'Access-Control-Allow-Origin': '*',
    },
  })
}
