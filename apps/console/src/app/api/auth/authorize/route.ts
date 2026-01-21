/**
 * OAuth 2.1 Authorization Endpoint (Proxy)
 *
 * GET /api/auth/authorize
 *
 * This endpoint proxies to the appropriate OAuth server:
 * - Production: Supabase OAuth Server
 * - Development: OAuth Mock Server
 */

import { NextRequest, NextResponse } from 'next/server'
import { getOAuthServerUrl, useSupabaseOAuthServer } from '@/lib/env'

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams
  const oauthServerUrl = getOAuthServerUrl()

  // Build the OAuth server authorize URL
  const authorizeUrl = new URL(`${oauthServerUrl}/authorize`)

  // Forward all query parameters
  searchParams.forEach((value, key) => {
    authorizeUrl.searchParams.set(key, value)
  })

  console.log(`[authorize] Redirecting to OAuth server: ${authorizeUrl.toString()}`)

  // Redirect to the OAuth server
  return NextResponse.redirect(authorizeUrl.toString())
}
