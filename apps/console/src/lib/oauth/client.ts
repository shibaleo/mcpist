/**
 * OAuth Client ID Management
 *
 * OAuth Client ID for MCP connection flow.
 */

const DEFAULT_CLIENT_ID = 'mcpist-console'

/**
 * Get OAuth client ID from environment variable
 */
export function getOAuthClientId(): string {
  return process.env.NEXT_PUBLIC_OAUTH_CLIENT_ID || DEFAULT_CLIENT_ID
}

/**
 * Get or register OAuth client (async version for compatibility)
 */
export async function getOrRegisterOAuthClient(): Promise<string> {
  return getOAuthClientId()
}

/**
 * Clear stored OAuth client info (no-op for compatibility)
 */
export function clearOAuthClient(): void {
  // No-op
}
