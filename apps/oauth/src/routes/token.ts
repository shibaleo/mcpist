/**
 * OAuth 2.1 Token Endpoint
 *
 * POST /token
 *
 * Supports:
 * - authorization_code grant (with PKCE verification)
 * - refresh_token grant (with token rotation)
 */

import { Hono } from 'hono'
import { consumeAuthorizationCode, storeRefreshToken, consumeRefreshToken } from '../lib/codes.js'
import { verifyPKCE } from '../lib/pkce.js'
import { signJWT } from '../lib/jwt.js'
import { createClient } from '@supabase/supabase-js'

// Generate secure random token
function generateRefreshToken(): string {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  return 'mrt_' + Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('')
}

const app = new Hono()

app.post('/', async (c) => {
  // Token endpoint accepts application/x-www-form-urlencoded
  const contentType = c.req.header('content-type')

  let params: URLSearchParams
  if (contentType?.includes('application/x-www-form-urlencoded')) {
    const body = await c.req.text()
    params = new URLSearchParams(body)
  } else if (contentType?.includes('application/json')) {
    const body = await c.req.json()
    params = new URLSearchParams(body)
  } else {
    return errorResponse(c, 'invalid_request', 'Content-Type must be application/x-www-form-urlencoded or application/json')
  }

  const grantType = params.get('grant_type')

  if (grantType === 'authorization_code') {
    return handleAuthorizationCodeGrant(c, params)
  } else if (grantType === 'refresh_token') {
    return handleRefreshTokenGrant(c, params)
  } else {
    return errorResponse(c, 'unsupported_grant_type', 'Only authorization_code and refresh_token are supported')
  }
})

async function handleAuthorizationCodeGrant(c: any, params: URLSearchParams) {
  const code = params.get('code')
  const redirectUri = params.get('redirect_uri')
  const clientId = params.get('client_id')
  const codeVerifier = params.get('code_verifier')

  // Validate required parameters
  if (!code) {
    return errorResponse(c, 'invalid_request', 'code is required')
  }
  if (!redirectUri) {
    return errorResponse(c, 'invalid_request', 'redirect_uri is required')
  }
  if (!clientId) {
    return errorResponse(c, 'invalid_request', 'client_id is required')
  }
  if (!codeVerifier) {
    return errorResponse(c, 'invalid_request', 'code_verifier is required (PKCE)')
  }

  // Retrieve authorization code from database
  const codeData = await consumeAuthorizationCode(code)
  if (!codeData) {
    return errorResponse(c, 'invalid_grant', 'Authorization code is invalid, expired, or already used')
  }

  // Verify client_id and redirect_uri match
  if (codeData.client_id !== clientId) {
    return errorResponse(c, 'invalid_grant', 'client_id mismatch')
  }
  if (codeData.redirect_uri !== redirectUri) {
    return errorResponse(c, 'invalid_grant', 'redirect_uri mismatch')
  }

  // Verify PKCE
  const pkceValid = await verifyPKCE(codeVerifier, codeData.code_challenge, codeData.code_challenge_method)
  if (!pkceValid) {
    return errorResponse(c, 'invalid_grant', 'PKCE verification failed')
  }

  // Get user email from Supabase
  const supabaseUrl = process.env.SUPABASE_URL
  const serviceKey = process.env.SUPABASE_SERVICE_ROLE_KEY
  if (!supabaseUrl || !serviceKey) {
    return errorResponse(c, 'server_error', 'Server configuration error')
  }

  const supabase = createClient(supabaseUrl, serviceKey)
  const { data: userData } = await supabase.auth.admin.getUserById(codeData.user_id)
  const email = userData?.user?.email

  // Generate JWT - audience is the Worker URL (API Gateway)
  const mcpServerUrl = process.env.MCP_SERVER_URL || 'http://mcp.localhost'
  const accessToken = await signJWT({
    sub: codeData.user_id,
    aud: mcpServerUrl,
    scope: codeData.scope || 'openid profile email',
    email,
    client_id: codeData.client_id,
  })

  // Generate and store refresh token
  const refreshToken = generateRefreshToken()
  await storeRefreshToken({
    token: refreshToken,
    user_id: codeData.user_id,
    client_id: codeData.client_id,
    scope: codeData.scope,
  })

  return c.json({
    access_token: accessToken,
    token_type: 'Bearer',
    expires_in: 3600, // 1 hour
    refresh_token: refreshToken,
    scope: codeData.scope || 'openid profile email',
  }, {
    headers: {
      'Cache-Control': 'no-store',
      'Pragma': 'no-cache',
    },
  })
}

async function handleRefreshTokenGrant(c: any, params: URLSearchParams) {
  const refreshToken = params.get('refresh_token')
  const clientId = params.get('client_id')

  if (!refreshToken) {
    return errorResponse(c, 'invalid_request', 'refresh_token is required')
  }
  if (!clientId) {
    return errorResponse(c, 'invalid_request', 'client_id is required')
  }

  // Validate and consume the refresh token (token rotation)
  const tokenData = await consumeRefreshToken(refreshToken)
  if (!tokenData) {
    return errorResponse(c, 'invalid_grant', 'Refresh token is invalid, expired, or already used')
  }

  // Verify client_id matches
  if (tokenData.client_id !== clientId) {
    return errorResponse(c, 'invalid_grant', 'client_id mismatch')
  }

  // Get user email from Supabase
  const supabaseUrl = process.env.SUPABASE_URL
  const serviceKey = process.env.SUPABASE_SERVICE_ROLE_KEY
  if (!supabaseUrl || !serviceKey) {
    return errorResponse(c, 'server_error', 'Server configuration error')
  }

  const supabase = createClient(supabaseUrl, serviceKey)
  const { data: userData } = await supabase.auth.admin.getUserById(tokenData.user_id)
  const email = userData?.user?.email

  // Generate new access token
  const mcpServerUrl = process.env.MCP_SERVER_URL || 'http://mcp.localhost'
  const accessToken = await signJWT({
    sub: tokenData.user_id,
    aud: mcpServerUrl,
    scope: tokenData.scope || 'openid profile email',
    email,
    client_id: tokenData.client_id,
  })

  // Generate new refresh token (token rotation)
  const newRefreshToken = generateRefreshToken()
  await storeRefreshToken({
    token: newRefreshToken,
    user_id: tokenData.user_id,
    client_id: tokenData.client_id,
    scope: tokenData.scope,
  })

  return c.json({
    access_token: accessToken,
    token_type: 'Bearer',
    expires_in: 3600, // 1 hour
    refresh_token: newRefreshToken,
    scope: tokenData.scope || 'openid profile email',
  }, {
    headers: {
      'Cache-Control': 'no-store',
      'Pragma': 'no-cache',
    },
  })
}

function errorResponse(c: any, error: string, errorDescription: string) {
  return c.json(
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

export const tokenRoute = app
