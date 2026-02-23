/**
 * OAuth Client Registration (RFC 7591 Dynamic Client Registration)
 *
 * MCP クライアントと同様に、Clerk の registration_endpoint に POST して
 * client_id を動的に取得する。取得した client_id は sessionStorage にキャッシュ。
 */

const STORAGE_KEY = "mcpist_oauth_client"

interface StoredClient {
  client_id: string
  client_secret?: string
}

/**
 * Get cached client_id from sessionStorage, or null if not cached.
 */
export function getOAuthClientId(): string {
  if (typeof window === "undefined") return ""
  const stored = sessionStorage.getItem(STORAGE_KEY)
  if (stored) {
    try {
      const parsed: StoredClient = JSON.parse(stored)
      return parsed.client_id
    } catch {
      // ignore
    }
  }
  return ""
}

/**
 * Register a new OAuth client via Dynamic Client Registration (RFC 7591).
 * Fetches the registration_endpoint from authorization server metadata,
 * then POSTs a client registration request.
 * Caches the result in sessionStorage.
 */
export async function getOrRegisterOAuthClient(): Promise<string> {
  // Return cached client_id if available
  const cached = getOAuthClientId()
  if (cached) return cached

  // Fetch authorization server metadata to get registration_endpoint
  const metadataRes = await fetch("/.well-known/oauth-authorization-server")
  const metadata = await metadataRes.json()

  const registrationEndpoint = metadata.registration_endpoint
  if (!registrationEndpoint) {
    throw new Error("Authorization server does not support dynamic client registration")
  }

  // Register client (RFC 7591)
  const res = await fetch(registrationEndpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_name: "MCPist",
      redirect_uris: [`${window.location.origin}/oauth/callback`],
      grant_types: ["authorization_code", "refresh_token"],
      response_types: ["code"],
      scope: "openid profile email",
      token_endpoint_auth_method: "none",
    }),
  })

  if (!res.ok) {
    const err = await res.text()
    throw new Error(`Client registration failed: ${err}`)
  }

  const client = await res.json()
  const stored: StoredClient = {
    client_id: client.client_id,
    client_secret: client.client_secret,
  }

  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(stored))
  return stored.client_id
}

/**
 * Clear stored OAuth client info.
 */
export function clearOAuthClient(): void {
  if (typeof window === "undefined") return
  sessionStorage.removeItem(STORAGE_KEY)
}
