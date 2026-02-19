/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * https://modelcontextprotocol.io/specification/draft/basic/authorization
 *
 * MCP Server が返す 401 レスポンスの WWW-Authenticate ヘッダーで
 * このメタデータの URL が指定される。
 */
import { NextResponse } from 'next/server'

const BASE_URL = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
const WORKER_URL = process.env.NEXT_PUBLIC_WORKER_URL || process.env.NEXT_PUBLIC_MCP_SERVER_URL!

export async function GET() {
  const metadata = {
    resource: `${WORKER_URL}/v1/mcp`,
    authorization_servers: [`${WORKER_URL}/.well-known/oauth-authorization-server`],
    scopes_supported: ['openid', 'profile', 'email'],
    bearer_methods_supported: ['header'],
  }

  return NextResponse.json(metadata, {
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'public, max-age=3600',
      'Access-Control-Allow-Origin': '*',
    },
  })
}
