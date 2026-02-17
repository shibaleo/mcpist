/**
 * OAuth Client ID Management
 *
 * OAuth Client ID is managed via environment variable NEXT_PUBLIC_OAUTH_CLIENT_ID.
 * The client must be pre-registered in Supabase OAuth Apps dashboard.
 *
 * For local development, use 'mcpist-console' (registered in OAuth Mock Server).
 * For production, use the UUID from Supabase OAuth Apps dashboard.
 */

/**
 * Default client ID for development
 */
const DEFAULT_CLIENT_ID = 'mcpist-console'

/**
 * Get OAuth client ID from environment variable
 *
 * Priority:
 * 1. NEXT_PUBLIC_OAUTH_CLIENT_ID environment variable
 * 2. Default fallback ('mcpist-console' for development)
 */
export function getOAuthClientId(): string {
  return process.env.NEXT_PUBLIC_OAUTH_CLIENT_ID || DEFAULT_CLIENT_ID
}

/**
 * Get or register OAuth client (async version for compatibility)
 *
 * This is now just a wrapper around getOAuthClientId() for backward compatibility.
 * Client registration is done manually via Supabase dashboard.
 */
export async function getOrRegisterOAuthClient(): Promise<string> {
  return getOAuthClientId()
}

/**
 * Clear stored OAuth client info (no-op for compatibility)
 */
export function clearOAuthClient(): void {
  // No-op: client ID is now from environment variable
}
