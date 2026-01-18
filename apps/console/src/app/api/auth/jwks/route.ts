/**
 * JWKS (JSON Web Key Set) Endpoint
 *
 * GET /api/auth/jwks
 *
 * Returns the public keys used to verify JWTs issued by MCPist Auth Server.
 * MCP Server uses this to verify access tokens.
 */

import { NextResponse } from 'next/server'
import { getJWKS } from '../lib/jwt'

export async function GET() {
  try {
    const jwks = await getJWKS()

    return NextResponse.json(jwks, {
      headers: {
        'Content-Type': 'application/json',
        'Cache-Control': 'public, max-age=3600', // Cache for 1 hour
        'Access-Control-Allow-Origin': '*',
      },
    })
  } catch (error) {
    console.error('Failed to generate JWKS:', error)
    return NextResponse.json(
      { error: 'Failed to generate JWKS' },
      { status: 500 }
    )
  }
}
