/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * https://modelcontextprotocol.io/specification/draft/basic/authorization
 *
 * MCP Server が返す 401 レスポンスの WWW-Authenticate ヘッダーで
 * このメタデータの URL が指定される。
 */
import { NextResponse } from 'next/server'

const BASE_URL = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
const MCP_SERVER_URL = process.env.NEXT_PUBLIC_MCP_SERVER_URL || 'http://localhost:8089'

export async function GET() {
  const metadata = {
    // MCP Server (Go) is the protected resource
    resource: `${MCP_SERVER_URL}/mcp`,
    // MCPist 独自の Authorization Server を指定
    authorization_servers: [BASE_URL],
    scopes_supported: ['openid', 'profile', 'email'],
    bearer_methods_supported: ['header'],
  }

  return NextResponse.json(metadata, {
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'no-cache', // Development: disable cache
      'Access-Control-Allow-Origin': '*',
    },
  })
}
