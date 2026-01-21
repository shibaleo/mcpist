/**
 * OAuth Mock Server
 *
 * Development environment mock for Supabase OAuth Server.
 * Implements the same API contract as Supabase OAuth Server.
 *
 * Endpoints:
 * - GET  /authorize              - Authorization request
 * - GET  /authorization/:id      - Get authorization details
 * - POST /authorization/:id/approve - Approve authorization
 * - POST /authorization/:id/deny    - Deny authorization
 * - POST /token                  - Token exchange
 * - GET  /jwks                   - Public keys (JWKS)
 * - GET  /.well-known/oauth-authorization-server - OAuth metadata
 */

import { Hono } from 'hono'
import { cors } from 'hono/cors'
import { logger } from 'hono/logger'
import { authorizeRoute } from './routes/authorize.js'
import { authorizationRoutes } from './routes/authorization.js'
import { tokenRoute } from './routes/token.js'
import { jwksRoute } from './routes/jwks.js'
import { wellKnownRoute } from './routes/well-known.js'

const app = new Hono()

// Middleware
app.use('*', logger())
app.use('*', cors({
  origin: ['http://console.localhost', 'http://localhost:3000'],
  credentials: true,
}))

// Health check
app.get('/health', (c) => {
  return c.json({ status: 'ok', service: 'oauth-mock-server' })
})

// OAuth endpoints (matching Supabase OAuth Server paths)
// Note: Supabase uses /auth/v1/oauth/* but we use root paths for simplicity
// Console proxies to this server so paths are transparent to clients

app.route('/authorize', authorizeRoute)
app.route('/authorization', authorizationRoutes)
app.route('/token', tokenRoute)
app.route('/jwks', jwksRoute)
app.route('/.well-known', wellKnownRoute)

// Start server
import { serve } from '@hono/node-server'

const port = parseInt(process.env.PORT || '4000', 10)

console.log(`OAuth Mock Server starting on port ${port}`)

serve({
  fetch: app.fetch,
  port,
})
