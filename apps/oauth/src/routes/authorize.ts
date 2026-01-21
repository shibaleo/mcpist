/**
 * OAuth 2.1 Authorization Endpoint
 *
 * GET /authorize
 *
 * Validates authorization request parameters and redirects to consent page.
 * Mimics Supabase OAuth Server behavior:
 * 1. Validate parameters
 * 2. Generate authorization_id
 * 3. Redirect to {CONSOLE_URL}/oauth/consent?authorization_id=xxx
 */

import { Hono } from 'hono'
import { storeAuthorizationRequest } from '../lib/codes.js'
import { generateAuthorizationId } from '../lib/pkce.js'

const app = new Hono()

app.get('/', async (c) => {
  const query = c.req.query()

  // Required parameters
  const responseType = query.response_type
  const clientId = query.client_id
  const redirectUri = query.redirect_uri
  const codeChallenge = query.code_challenge
  const codeChallengeMethod = query.code_challenge_method
  const state = query.state

  // Optional parameters
  const scope = query.scope || 'openid profile email'

  // Validate required parameters
  if (responseType !== 'code') {
    return errorResponse(c, redirectUri, 'unsupported_response_type', 'Only response_type=code is supported', state)
  }

  if (!clientId) {
    return errorResponse(c, redirectUri, 'invalid_request', 'client_id is required', state)
  }

  if (!redirectUri) {
    return c.json({ error: 'invalid_request', error_description: 'redirect_uri is required' }, 400)
  }

  if (!codeChallenge) {
    return errorResponse(c, redirectUri, 'invalid_request', 'code_challenge is required (PKCE)', state)
  }

  if (codeChallengeMethod !== 'S256') {
    return errorResponse(c, redirectUri, 'invalid_request', 'code_challenge_method must be S256', state)
  }

  // state is recommended but not strictly required
  // However, we'll require it for better security
  if (!state) {
    return errorResponse(c, redirectUri, 'invalid_request', 'state is required', state)
  }

  // Generate authorization_id and store request
  const authorizationId = generateAuthorizationId()

  try {
    await storeAuthorizationRequest({
      id: authorizationId,
      client_id: clientId,
      redirect_uri: redirectUri,
      code_challenge: codeChallenge,
      code_challenge_method: codeChallengeMethod,
      scope,
      state,
    })
  } catch (error) {
    console.error('[authorize] Failed to store authorization request:', error)
    return errorResponse(c, redirectUri, 'server_error', 'Failed to process authorization request', state)
  }

  // Redirect to consent page with authorization_id
  // This mimics Supabase OAuth Server behavior
  const consoleUrl = process.env.CONSOLE_URL || 'http://console.localhost'
  const consentUrl = new URL('/oauth/consent', consoleUrl)
  consentUrl.searchParams.set('authorization_id', authorizationId)

  console.log(`[authorize] Redirecting to consent: ${consentUrl.toString()}`)
  return c.redirect(consentUrl.toString())
})

function errorResponse(
  c: ReturnType<typeof Hono.prototype.get>,
  redirectUri: string | undefined,
  error: string,
  errorDescription: string,
  state: string | undefined
) {
  if (!redirectUri) {
    return c.json({ error, error_description: errorDescription }, 400)
  }

  const url = new URL(redirectUri)
  url.searchParams.set('error', error)
  url.searchParams.set('error_description', errorDescription)
  if (state) {
    url.searchParams.set('state', state)
  }

  return c.redirect(url.toString())
}

export const authorizeRoute = app
