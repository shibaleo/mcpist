/**
 * Authorization Management Endpoints
 *
 * GET  /authorization/:id         - Get authorization details
 * POST /authorization/:id/approve - Approve authorization (generate code)
 * POST /authorization/:id/deny    - Deny authorization
 *
 * These endpoints mimic Supabase's auth.oauth.* methods:
 * - getAuthorizationDetails(authorization_id)
 * - approveAuthorization(authorization_id)
 * - denyAuthorization(authorization_id)
 */

import { Hono } from 'hono'
import { getAuthorizationRequest, approveAuthorizationRequest, denyAuthorizationRequest } from '../lib/codes.js'

const app = new Hono()

/**
 * GET /authorization/:id
 * Get authorization request details
 * Called by consent page to display client info and scopes
 */
app.get('/:id', async (c) => {
  const authorizationId = c.req.param('id')

  const request = await getAuthorizationRequest(authorizationId)

  if (!request) {
    return c.json({
      error: 'invalid_request',
      error_description: 'Authorization request not found or expired',
    }, 404)
  }

  // Return details needed for consent screen
  return c.json({
    id: request.id,
    client_id: request.client_id,
    redirect_uri: request.redirect_uri,
    scope: request.scope,
    state: request.state,
    created_at: request.created_at,
    expires_at: request.expires_at,
  })
})

/**
 * POST /authorization/:id/approve
 * Approve authorization and generate code
 * Called by consent page when user clicks "Allow"
 */
app.post('/:id/approve', async (c) => {
  const authorizationId = c.req.param('id')

  // Get user_id from request body
  const body = await c.req.json().catch(() => ({}))
  const userId = body.user_id

  if (!userId) {
    return c.json({
      error: 'invalid_request',
      error_description: 'user_id is required',
    }, 400)
  }

  const result = await approveAuthorizationRequest(authorizationId, userId)

  if (!result) {
    return c.json({
      error: 'invalid_request',
      error_description: 'Authorization request not found, expired, or already processed',
    }, 404)
  }

  // Return code and redirect info
  // Client (consent page) should redirect to redirect_uri with code and state
  return c.json({
    code: result.code,
    redirect_uri: result.redirect_uri,
    state: result.state,
  })
})

/**
 * POST /authorization/:id/deny
 * Deny authorization
 * Called by consent page when user clicks "Deny"
 */
app.post('/:id/deny', async (c) => {
  const authorizationId = c.req.param('id')

  const result = await denyAuthorizationRequest(authorizationId)

  if (!result) {
    return c.json({
      error: 'invalid_request',
      error_description: 'Authorization request not found, expired, or already processed',
    }, 404)
  }

  // Return redirect info with error
  // Client (consent page) should redirect to redirect_uri with error
  return c.json({
    redirect_uri: result.redirect_uri,
    state: result.state,
    error: 'access_denied',
    error_description: 'User denied the authorization request',
  })
})

export const authorizationRoutes = app
