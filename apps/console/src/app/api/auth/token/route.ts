/**
 * OAuth 2.1 Token Endpoint (Proxy)
 *
 * POST /api/auth/token
 *
 * This endpoint proxies to the appropriate OAuth server:
 * - Production: Supabase OAuth Server
 * - Development: OAuth Mock Server
 */

import { NextRequest, NextResponse } from 'next/server'
import { getOAuthServerInternalUrl } from '@/lib/env'

export async function POST(request: NextRequest) {
  const oauthServerUrl = getOAuthServerInternalUrl()
  const tokenUrl = `${oauthServerUrl}/token`

  console.log(`[token] Proxying to OAuth server: ${tokenUrl}`)

  try {
    const body = await request.text()

    const response = await fetch(tokenUrl, {
      method: 'POST',
      headers: {
        'Content-Type': request.headers.get('content-type') || 'application/x-www-form-urlencoded',
      },
      body,
    })

    const data = await response.json()

    return NextResponse.json(data, {
      status: response.status,
      headers: {
        'Cache-Control': 'no-store',
        'Pragma': 'no-cache',
      },
    })
  } catch (error) {
    console.error('[token] Error proxying to OAuth server:', error)
    return NextResponse.json(
      { error: 'server_error', error_description: 'Failed to communicate with OAuth server' },
      { status: 500 }
    )
  }
}
