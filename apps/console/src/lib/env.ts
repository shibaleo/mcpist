/**
 * Environment utilities
 */

export const isDevelopment = process.env.ENVIRONMENT === 'development'
export const isProduction = process.env.ENVIRONMENT === 'production'

/**
 * Check if Supabase OAuth Server should be used
 * - Production: Use Supabase OAuth Server (Cloud feature)
 * - Development: Use custom implementation (local Supabase doesn't support OAuth Server)
 */
export const useSupabaseOAuthServer = isProduction

/**
 * Get the OAuth authorization endpoint URL
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
    const url = new URL(`${supabaseUrl}/auth/v1/authorize`)
    url.searchParams.set('response_type', 'code')
    url.searchParams.set('client_id', params.clientId)
    url.searchParams.set('redirect_uri', params.redirectUri)
    url.searchParams.set('code_challenge', params.codeChallenge)
    url.searchParams.set('code_challenge_method', params.codeChallengeMethod)
    url.searchParams.set('scope', params.scope)
    url.searchParams.set('state', params.state)
    return url.toString()
  }

  // Custom implementation endpoint
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
    return `${supabaseUrl}/auth/v1/token`
  }

  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
  return `${baseUrl}/api/auth/token`
}
