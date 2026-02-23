/**
 * OAuth 2.1 Authorization Server Metadata (RFC 8414)
 *
 * Clerk の /.well-known/oauth-authorization-server をプロキシ。
 * MCP クライアントが Console 経由でメタデータを取得するケース用。
 * メインの配信は Worker 側。
 */
import { NextResponse } from 'next/server'

const CLERK_JWKS_URL = process.env.CLERK_JWKS_URL || process.env.NEXT_PUBLIC_CLERK_JWKS_URL || ''
const CLERK_ISSUER = CLERK_JWKS_URL.replace(/\/\.well-known\/jwks\.json$/, '')

export async function GET() {
  const res = await fetch(`${CLERK_ISSUER}/.well-known/oauth-authorization-server`)
  const metadata = await res.json()

  return NextResponse.json(metadata, {
    headers: {
      'Cache-Control': 'public, max-age=3600',
      'Access-Control-Allow-Origin': '*',
    },
  })
}
