/**
 * Token Validator
 * Validates OAuth/API tokens via server-side API route to avoid CORS issues
 */

export interface TokenValidationResult {
  valid: boolean
  error?: string
  details?: Record<string, unknown>
}

/**
 * Validate token for a given service via API route
 */
export async function validateToken(
  service: string,
  token: string
): Promise<TokenValidationResult> {
  try {
    console.log('[token-validator] Calling API for service:', service)
    const response = await fetch('/api/validate-token', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ service, token }),
    })

    console.log('[token-validator] Response status:', response.status)

    if (!response.ok) {
      console.log('[token-validator] Response not ok')
      return {
        valid: false,
        error: 'トークン検証に失敗しました',
      }
    }

    const result = await response.json()
    console.log('[token-validator] Result:', JSON.stringify(result))
    return result
  } catch (error) {
    console.error('[token-validator] Error:', error)
    return {
      valid: false,
      error: 'ネットワークエラーが発生しました',
    }
  }
}
