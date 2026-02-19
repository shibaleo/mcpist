/**
 * OAuth 2.1 Authorization Server Metadata (RFC 8414)
 *
 * Returns Clerk JWKS URL for token verification.
 * The actual OAuth metadata is served by Worker at /.well-known/oauth-authorization-server.
 */
import { NextResponse } from 'next/server'

const CLERK_JWKS_URL = process.env.CLERK_JWKS_URL || process.env.NEXT_PUBLIC_CLERK_JWKS_URL

export async function GET() {
  const metadata = {
    jwks_uri: CLERK_JWKS_URL,
    response_types_supported: ['code'],
    grant_types_supported: ['authorization_code'],
    scopes_supported: ['openid', 'profile', 'email'],
  }

  return NextResponse.json(metadata, {
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'public, max-age=3600',
      'Access-Control-Allow-Origin': '*',
    },
  })
}
