"use server"

import { revokeApiKey } from "@/lib/mcp/api-keys"

/**
 * Revoke an API key — thin wrapper for backward compatibility
 */
export async function revokeApiKeyAction(
  keyId: string
): Promise<{ success: boolean; error?: string }> {
  try {
    const result = await revokeApiKey(keyId)
    if (!result.success) {
      return { success: false, error: "API key not found or already revoked" }
    }
    return { success: true }
  } catch (error) {
    console.error("[RevokeApiKey] Error:", error)
    return { success: false, error: "Internal server error" }
  }
}
