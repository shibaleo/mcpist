/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * https://modelcontextprotocol.io/specification/draft/basic/authorization
 *
 * MCP クライアントが認可サーバーを発見するために使用。
 * authorization_servers は Clerk の issuer URL を直接指す。
 */
import { NextResponse } from 'next/server'

const WORKER_URL = process.env.NEXT_PUBLIC_WORKER_URL || process.env.NEXT_PUBLIC_MCP_SERVER_URL!
const CLERK_JWKS_URL = process.env.CLERK_JWKS_URL || process.env.NEXT_PUBLIC_CLERK_JWKS_URL || ''
const CLERK_ISSUER = CLERK_JWKS_URL.replace(/\/\.well-known\/jwks\.json$/, '')

export async function GET() {
  const metadata = {
    resource: `${WORKER_URL}/v1/mcp`,
    authorization_servers: [CLERK_ISSUER],
    scopes_supported: ['openid', 'profile', 'email'],
    bearer_methods_supported: ['header'],
  }

  return NextResponse.json(metadata, {
    headers: {
      'Cache-Control': 'public, max-age=3600',
      'Access-Control-Allow-Origin': '*',
    },
  })
}
