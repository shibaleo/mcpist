/**
 * JWKS Endpoint
 *
 * GET /jwks
 *
 * Returns JSON Web Key Set for JWT verification.
 * MCP Server (Worker) uses this to verify access tokens.
 */

import { Hono } from 'hono'
import { getJWKS } from '../lib/jwt.js'

const app = new Hono()

app.get('/', async (c) => {
  const jwks = await getJWKS()

  return c.json(jwks, {
    headers: {
      'Cache-Control': 'public, max-age=3600', // Cache for 1 hour
      'Content-Type': 'application/json',
    },
  })
})

export const jwksRoute = app
