/**
 * Well-Known Endpoints
 *
 * GET /.well-known/oauth-authorization-server
 *
 * OAuth 2.1 Authorization Server Metadata (RFC 8414)
 */

import { Hono } from 'hono'

const app = new Hono()

app.get('/oauth-authorization-server', (c) => {
  const baseUrl = process.env.OAUTH_SERVER_URL || 'http://oauth.localhost'

  const metadata = {
    issuer: baseUrl,
    authorization_endpoint: `${baseUrl}/authorize`,
    token_endpoint: `${baseUrl}/token`,
    jwks_uri: `${baseUrl}/jwks`,
    response_types_supported: ['code'],
    grant_types_supported: ['authorization_code', 'refresh_token'],
    code_challenge_methods_supported: ['S256'],
    token_endpoint_auth_methods_supported: ['none'],
    scopes_supported: ['openid', 'profile', 'email'],
  }

  return c.json(metadata, {
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'public, max-age=3600',
      'Access-Control-Allow-Origin': '*',
    },
  })
})

export const wellKnownRoute = app
