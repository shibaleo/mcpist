/**
 * OAuth Token Management Utilities
 *
 * Handles access token storage, expiration tracking, and automatic refresh.
 */

import { getOAuthClientId } from '@/lib/oauth/client'

export interface TokenData {
  accessToken: string
  refreshToken: string | null
  expiresAt: number  // Unix timestamp in milliseconds
  scope: string
}

/**
 * Get stored token data from sessionStorage
 */
export function getStoredTokens(): TokenData | null {
  if (typeof window === 'undefined') return null

  const accessToken = sessionStorage.getItem('mcp_access_token')
  if (!accessToken) return null

  return {
    accessToken,
    refreshToken: sessionStorage.getItem('mcp_refresh_token'),
    expiresAt: parseInt(sessionStorage.getItem('mcp_token_expires_at') || '0'),
    scope: sessionStorage.getItem('mcp_token_scope') || '',
  }
}

/**
 * Store token data in sessionStorage
 */
export function storeTokens(data: TokenData): void {
  if (typeof window === 'undefined') return

  sessionStorage.setItem('mcp_access_token', data.accessToken)
  sessionStorage.setItem('mcp_refresh_token', data.refreshToken || '')
  sessionStorage.setItem('mcp_token_expires_at', String(data.expiresAt))
  sessionStorage.setItem('mcp_token_scope', data.scope)
}

/**
 * Clear all stored token data
 */
export function clearTokens(): void {
  if (typeof window === 'undefined') return

  sessionStorage.removeItem('mcp_access_token')
  sessionStorage.removeItem('mcp_refresh_token')
  sessionStorage.removeItem('mcp_token_expires_at')
  sessionStorage.removeItem('mcp_token_scope')
  sessionStorage.removeItem('oauth_processed_code')
}

/**
 * Check if the access token is expired or about to expire
 * Returns true if token expires within the buffer time (default 5 minutes)
 */
export function isTokenExpired(tokens: TokenData | null, bufferMs: number = 5 * 60 * 1000): boolean {
  if (!tokens) return true
  return Date.now() + bufferMs >= tokens.expiresAt
}

/**
 * Get time until token expiration in milliseconds
 * Returns negative value if already expired
 */
export function getTimeUntilExpiration(tokens: TokenData | null): number {
  if (!tokens) return -1
  return tokens.expiresAt - Date.now()
}

/**
 * Format time until expiration as human-readable string
 */
export function formatTimeUntilExpiration(tokens: TokenData | null): string {
  const timeMs = getTimeUntilExpiration(tokens)

  if (timeMs < 0) return '期限切れ'

  const seconds = Math.floor(timeMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (hours > 0) {
    return `${hours}時間${minutes % 60}分`
  }
  if (minutes > 0) {
    return `${minutes}分${seconds % 60}秒`
  }
  return `${seconds}秒`
}

/**
 * Refresh the access token using the refresh token
 * Returns new token data on success, null on failure
 */
export async function refreshAccessToken(tokens: TokenData): Promise<TokenData | null> {
  if (!tokens.refreshToken) {
    console.error('[OAuth] No refresh token available')
    return null
  }

  try {
    // Get OAuth metadata for token endpoint
    const metadataRes = await fetch('/.well-known/oauth-authorization-server')
    if (!metadataRes.ok) {
      throw new Error('Failed to fetch OAuth metadata')
    }
    const metadata = await metadataRes.json()

    // Request new tokens using refresh token
    const tokenRes = await fetch(metadata.token_endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: new URLSearchParams({
        grant_type: 'refresh_token',
        refresh_token: tokens.refreshToken,
        client_id: getOAuthClientId(),
      }),
    })

    if (!tokenRes.ok) {
      const errorData = await tokenRes.json().catch(() => ({}))
      console.error('[OAuth] Token refresh failed:', errorData)
      return null
    }

    const tokenData = await tokenRes.json()

    // Calculate new expiration time
    const expiresAt = Date.now() + (tokenData.expires_in || 3600) * 1000

    const newTokens: TokenData = {
      accessToken: tokenData.access_token,
      refreshToken: tokenData.refresh_token || null,
      expiresAt,
      scope: tokenData.scope || tokens.scope,
    }

    // Store new tokens
    storeTokens(newTokens)

    console.log('[OAuth] Token refreshed successfully')
    return newTokens
  } catch (error) {
    console.error('[OAuth] Token refresh error:', error)
    return null
  }
}

/**
 * Get a valid access token, refreshing if necessary
 * Returns the access token string or null if unable to get a valid token
 */
export async function getValidAccessToken(): Promise<string | null> {
  let tokens = getStoredTokens()

  if (!tokens) {
    return null
  }

  // Check if token is expired or about to expire (within 5 minutes)
  if (isTokenExpired(tokens)) {
    console.log('[OAuth] Token expired or expiring soon, attempting refresh...')

    // Try to refresh
    tokens = await refreshAccessToken(tokens)
    if (!tokens) {
      // Refresh failed, clear tokens
      clearTokens()
      return null
    }
  }

  return tokens.accessToken
}
