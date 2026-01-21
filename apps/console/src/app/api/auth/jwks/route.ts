/**
 * JWKS (JSON Web Key Set) Endpoint (Proxy)
 *
 * GET /api/auth/jwks
 *
 * This endpoint proxies to the appropriate OAuth server:
 * - Production: Supabase JWKS
 * - Development: OAuth Mock Server
 */

import { NextResponse } from 'next/server'
import { getOAuthServerInternalUrl } from '@/lib/env'

export async function GET() {
  const oauthServerUrl = getOAuthServerInternalUrl()
  const jwksUrl = `${oauthServerUrl}/jwks`

  console.log(`[jwks] Proxying to OAuth server: ${jwksUrl}`)

  try {
    const response = await fetch(jwksUrl)
    const data = await response.json()

    return NextResponse.json(data, {
      headers: {
        'Content-Type': 'application/json',
        'Cache-Control': 'public, max-age=3600',
        'Access-Control-Allow-Origin': '*',
      },
    })
  } catch (error) {
    console.error('[jwks] Error proxying to OAuth server:', error)
    return NextResponse.json(
      { error: 'Failed to fetch JWKS' },
      { status: 500 }
    )
  }
}
