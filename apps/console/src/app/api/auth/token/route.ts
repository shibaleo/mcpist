/**
 * OAuth 2.1 Token Endpoint
 *
 * POST /api/auth/token
 *
 * Supports:
 * - authorization_code grant (with PKCE verification)
 * - refresh_token grant (TODO)
 *
 * Production: Proxies to Supabase OAuth Server
 * Development: Uses custom implementation
 */

import { NextRequest, NextResponse } from 'next/server'
import { consumeAuthorizationCode } from '../lib/codes'
import { verifyPKCE } from '../lib/pkce'
import { signJWT } from '../lib/jwt'
import { createClient } from '@supabase/supabase-js'
import { useSupabaseOAuthServer, getOAuthTokenUrl } from '@/lib/env'

export async function POST(request: NextRequest) {
  // Production: Proxy to Supabase OAuth Server
  if (useSupabaseOAuthServer) {
    return proxyToSupabaseOAuthServer(request)
  }

  // Development: Custom implementation
  return handleCustomTokenEndpoint(request)
}

/**
 * Proxy token request to Supabase OAuth Server
 */
async function proxyToSupabaseOAuthServer(request: NextRequest): Promise<NextResponse> {
  const tokenUrl = getOAuthTokenUrl()

  console.log('[token] Production: Proxying to Supabase OAuth Server:', tokenUrl)

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
    console.error('[token] Error proxying to Supabase OAuth Server:', error)
    return errorResponse('server_error', 'Failed to communicate with OAuth server')
  }
}

/**
 * Custom token endpoint implementation (for development)
 */
async function handleCustomTokenEndpoint(request: NextRequest): Promise<NextResponse> {
  console.log('[token] Development: Using custom token endpoint')
  // Token endpoint accepts application/x-www-form-urlencoded
  const contentType = request.headers.get('content-type')

  let params: URLSearchParams
  if (contentType?.includes('application/x-www-form-urlencoded')) {
    const body = await request.text()
    params = new URLSearchParams(body)
  } else if (contentType?.includes('application/json')) {
    const body = await request.json()
    params = new URLSearchParams(body)
  } else {
    return errorResponse('invalid_request', 'Content-Type must be application/x-www-form-urlencoded or application/json')
  }

  const grantType = params.get('grant_type')

  if (grantType === 'authorization_code') {
    return handleAuthorizationCodeGrant(params)
  } else if (grantType === 'refresh_token') {
    return handleRefreshTokenGrant(params)
  } else {
    return errorResponse('unsupported_grant_type', 'Only authorization_code and refresh_token are supported')
  }
}

async function handleAuthorizationCodeGrant(params: URLSearchParams): Promise<NextResponse> {
  const code = params.get('code')
  const redirectUri = params.get('redirect_uri')
  const clientId = params.get('client_id')
  const codeVerifier = params.get('code_verifier')

  // Validate required parameters
  if (!code) {
    return errorResponse('invalid_request', 'code is required')
  }
  if (!redirectUri) {
    return errorResponse('invalid_request', 'redirect_uri is required')
  }
  if (!clientId) {
    return errorResponse('invalid_request', 'client_id is required')
  }
  if (!codeVerifier) {
    return errorResponse('invalid_request', 'code_verifier is required (PKCE)')
  }

  // Retrieve authorization code from database
  const codeData = await consumeAuthorizationCode(code)
  if (!codeData) {
    return errorResponse('invalid_grant', 'Authorization code is invalid, expired, or already used')
  }

  // Verify client_id and redirect_uri match
  if (codeData.client_id !== clientId) {
    return errorResponse('invalid_grant', 'client_id mismatch')
  }
  if (codeData.redirect_uri !== redirectUri) {
    return errorResponse('invalid_grant', 'redirect_uri mismatch')
  }

  // Verify PKCE
  const pkceValid = await verifyPKCE(codeVerifier, codeData.code_challenge, codeData.code_challenge_method)
  if (!pkceValid) {
    return errorResponse('invalid_grant', 'PKCE verification failed')
  }

  // Get user email from Supabase
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL || process.env.SUPABASE_URL
  const serviceKey = process.env.SUPABASE_SERVICE_ROLE_KEY
  if (!supabaseUrl || !serviceKey) {
    return errorResponse('server_error', 'Server configuration error')
  }

  const supabase = createClient(supabaseUrl, serviceKey)
  const { data: userData } = await supabase.auth.admin.getUserById(codeData.user_id)
  const email = userData?.user?.email

  // Generate JWT - audience is the Worker URL (API Gateway)
  const mcpServerUrl = process.env.MCP_SERVER_URL || 'http://localhost:8787'
  const accessToken = await signJWT({
    sub: codeData.user_id,
    aud: mcpServerUrl,
    scope: codeData.scope || 'openid profile',
    email,
  })

  // TODO: Generate refresh token and store in database

  return NextResponse.json({
    access_token: accessToken,
    token_type: 'Bearer',
    expires_in: 3600, // 1 hour
    scope: codeData.scope || 'openid profile',
    // refresh_token: refreshToken, // TODO
  }, {
    headers: {
      'Cache-Control': 'no-store',
      'Pragma': 'no-cache',
    },
  })
}

async function handleRefreshTokenGrant(params: URLSearchParams): Promise<NextResponse> {
  const refreshToken = params.get('refresh_token')

  if (!refreshToken) {
    return errorResponse('invalid_request', 'refresh_token is required')
  }

  // TODO: Implement refresh token validation and new access token generation
  return errorResponse('unsupported_grant_type', 'refresh_token grant not yet implemented')
}

function errorResponse(error: string, errorDescription: string): NextResponse {
  return NextResponse.json(
    {
      error,
      error_description: errorDescription,
    },
    {
      status: 400,
      headers: {
        'Cache-Control': 'no-store',
        'Pragma': 'no-cache',
      },
    }
  )
}
