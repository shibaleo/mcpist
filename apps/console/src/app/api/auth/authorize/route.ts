/**
 * OAuth 2.1 Authorization Endpoint
 *
 * GET /api/auth/authorize
 *
 * This endpoint:
 * - Production: Redirects to Supabase OAuth Server
 * - Development: Uses custom implementation (validates params, checks login, redirects to consent)
 */

import { NextRequest, NextResponse } from 'next/server'
import { createServerClient } from '@supabase/ssr'
import { useSupabaseOAuthServer } from '@/lib/env'

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams

  // Required parameters
  const responseType = searchParams.get('response_type')
  const clientId = searchParams.get('client_id')
  const redirectUri = searchParams.get('redirect_uri')
  const codeChallenge = searchParams.get('code_challenge')
  const codeChallengeMethod = searchParams.get('code_challenge_method')
  const state = searchParams.get('state')

  // Optional parameters
  const scope = searchParams.get('scope') || 'openid profile'

  // Validate required parameters (same for both environments)
  if (responseType !== 'code') {
    return errorResponse(redirectUri, 'unsupported_response_type', 'Only response_type=code is supported', state)
  }

  if (!clientId) {
    return errorResponse(redirectUri, 'invalid_request', 'client_id is required', state)
  }

  if (!redirectUri) {
    return NextResponse.json({ error: 'invalid_request', error_description: 'redirect_uri is required' }, { status: 400 })
  }

  if (!codeChallenge) {
    return errorResponse(redirectUri, 'invalid_request', 'code_challenge is required (PKCE)', state)
  }

  if (codeChallengeMethod !== 'S256') {
    return errorResponse(redirectUri, 'invalid_request', 'code_challenge_method must be S256', state)
  }

  if (!state) {
    return errorResponse(redirectUri, 'invalid_request', 'state is required', state)
  }

  // Production: Redirect to Supabase OAuth Server
  if (useSupabaseOAuthServer) {
    const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
    if (!supabaseUrl) {
      return NextResponse.json({ error: 'server_error', error_description: 'SUPABASE_URL not configured' }, { status: 500 })
    }

    const supabaseAuthUrl = new URL(`${supabaseUrl}/auth/v1/authorize`)
    supabaseAuthUrl.searchParams.set('response_type', 'code')
    supabaseAuthUrl.searchParams.set('client_id', clientId)
    supabaseAuthUrl.searchParams.set('redirect_uri', redirectUri)
    supabaseAuthUrl.searchParams.set('code_challenge', codeChallenge)
    supabaseAuthUrl.searchParams.set('code_challenge_method', codeChallengeMethod)
    supabaseAuthUrl.searchParams.set('scope', scope)
    supabaseAuthUrl.searchParams.set('state', state)

    console.log('[authorize] Production: Redirecting to Supabase OAuth Server')
    return NextResponse.redirect(supabaseAuthUrl.toString())
  }

  // Development: Custom implementation
  console.log('[authorize] Development: Using custom OAuth implementation')

  // Check if user is logged in using request cookies
  const allCookies = request.cookies.getAll()
  console.log('[authorize] Cookies:', allCookies.map(c => c.name))

  let supabaseResponse = NextResponse.next({ request })
  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() {
          return request.cookies.getAll()
        },
        setAll(cookiesToSet) {
          cookiesToSet.forEach(({ name, value }) => request.cookies.set(name, value))
          supabaseResponse = NextResponse.next({ request })
          cookiesToSet.forEach(({ name, value, options }) =>
            supabaseResponse.cookies.set(name, value, options)
          )
        },
      },
    }
  )
  const { data: { user }, error } = await supabase.auth.getUser()
  console.log('[authorize] User:', user?.id, 'Error:', error?.message)

  // Store authorization request in session/query params for consent page
  const authRequest = {
    client_id: clientId,
    redirect_uri: redirectUri,
    code_challenge: codeChallenge,
    code_challenge_method: codeChallengeMethod,
    scope,
    state,
  }

  const encodedAuthRequest = Buffer.from(JSON.stringify(authRequest)).toString('base64url')
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'

  if (!user) {
    // Redirect to login page with return URL to consent page
    const consentUrl = baseUrl + '/oauth/consent?request=' + encodedAuthRequest
    const loginUrl = baseUrl + '/login?returnTo=' + encodeURIComponent(consentUrl)
    return NextResponse.redirect(loginUrl)
  }

  // User is logged in, redirect to consent page
  const consentUrl = baseUrl + '/oauth/consent?request=' + encodedAuthRequest
  return NextResponse.redirect(consentUrl)
}

function errorResponse(
  redirectUri: string | null,
  error: string,
  errorDescription: string,
  state: string | null
): NextResponse {
  if (!redirectUri) {
    return NextResponse.json({ error, error_description: errorDescription }, { status: 400 })
  }

  const url = new URL(redirectUri)
  url.searchParams.set('error', error)
  url.searchParams.set('error_description', errorDescription)
  if (state) {
    url.searchParams.set('state', state)
  }

  return NextResponse.redirect(url.toString())
}
