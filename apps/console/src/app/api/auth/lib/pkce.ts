/**
 * PKCE (Proof Key for Code Exchange) utilities
 * RFC 7636: https://datatracker.ietf.org/doc/html/rfc7636
 */

/**
 * Verify PKCE code_verifier against code_challenge
 * Only S256 method is supported (as per OAuth 2.1)
 */
export async function verifyPKCE(
  codeVerifier: string,
  codeChallenge: string,
  method: string = 'S256'
): Promise<boolean> {
  if (method !== 'S256') {
    // OAuth 2.1 requires S256, plain is not allowed
    return false
  }

  // S256: BASE64URL(SHA256(code_verifier)) == code_challenge
  const encoder = new TextEncoder()
  const data = encoder.encode(codeVerifier)
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)

  // Convert to base64url
  const hashArray = new Uint8Array(hashBuffer)
  const base64 = btoa(String.fromCharCode(...hashArray))
  const base64url = base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')

  return base64url === codeChallenge
}

/**
 * Generate a random authorization code
 */
export function generateAuthorizationCode(): string {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  return Array.from(array, (b) => b.toString(16).padStart(2, '0')).join('')
}
