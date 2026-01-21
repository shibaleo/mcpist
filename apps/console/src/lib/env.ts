/**
 * Environment utilities
 */

export const isDevelopment = process.env.ENVIRONMENT === 'development'
export const isProduction = process.env.ENVIRONMENT === 'production'

/**
 * Check if Supabase OAuth Server should be used
 * - Production: Use Supabase OAuth Server (Cloud feature)
 * - Development: Use OAuth Mock Server (oauth.localhost)
 */
export const useSupabaseOAuthServer = isProduction

/**
 * Get the OAuth Server URL (for browser redirects)
 * - Production: Supabase OAuth Server
 * - Development: OAuth Mock Server (oauth.localhost)
 */
export function getOAuthServerUrl(): string {
  if (useSupabaseOAuthServer) {
    const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
    return `${supabaseUrl}/auth/v1/oauth`
  }
  // Browser-accessible URL
  return process.env.OAUTH_SERVER_URL || 'http://oauth.localhost'
}

/**
 * Get the internal OAuth Server URL (for server-to-server communication)
 * - Production: Supabase OAuth Server
 * - Development: OAuth Mock Server container (internal DNS)
 */
export function getOAuthServerInternalUrl(): string {
  if (useSupabaseOAuthServer) {
    const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
    return `${supabaseUrl}/auth/v1/oauth`
  }
  // In Docker: use container name for server-to-server calls
  return process.env.OAUTH_SERVER_INTERNAL_URL || process.env.OAUTH_SERVER_URL || 'http://oauth.localhost'
}

/**
 * Get the OAuth authorization endpoint URL (for client-side redirect)
 */
export function getOAuthAuthorizeUrl(params: {
  clientId: string
  redirectUri: string
  codeChallenge: string
  codeChallengeMethod: string
  scope: string
  state: string
}): string {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL

  if (useSupabaseOAuthServer && supabaseUrl) {
    // Supabase OAuth Server endpoint
    const url = new URL(`${supabaseUrl}/auth/v1/oauth/authorize`)
    url.searchParams.set('response_type', 'code')
    url.searchParams.set('client_id', params.clientId)
    url.searchParams.set('redirect_uri', params.redirectUri)
    url.searchParams.set('code_challenge', params.codeChallenge)
    url.searchParams.set('code_challenge_method', params.codeChallengeMethod)
    url.searchParams.set('scope', params.scope)
    url.searchParams.set('state', params.state)
    return url.toString()
  }

  // Development: proxy through Console to OAuth Mock Server
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
  const url = new URL(`${baseUrl}/api/auth/authorize`)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('client_id', params.clientId)
  url.searchParams.set('redirect_uri', params.redirectUri)
  url.searchParams.set('code_challenge', params.codeChallenge)
  url.searchParams.set('code_challenge_method', params.codeChallengeMethod)
  url.searchParams.set('scope', params.scope)
  url.searchParams.set('state', params.state)
  return url.toString()
}

/**
 * Get the OAuth token endpoint URL
 */
export function getOAuthTokenUrl(): string {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL

  if (useSupabaseOAuthServer && supabaseUrl) {
    return `${supabaseUrl}/auth/v1/oauth/token`
  }

  // Development: proxy through Console
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
  return `${baseUrl}/api/auth/token`
}

/**
 * Get the OAuth JWKS endpoint URL
 */
export function getOAuthJwksUrl(): string {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL

  if (useSupabaseOAuthServer && supabaseUrl) {
    return `${supabaseUrl}/auth/v1/.well-known/jwks.json`
  }

  // Development: proxy through Console
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
  return `${baseUrl}/api/auth/jwks`
}
