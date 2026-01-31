"use server"

import { createClient } from "@/lib/supabase/server"

interface RevokeApiKeyResult {
  success: boolean
}

/**
 * Revoke an API key
 * This is a Server Action that runs on Vercel's server
 */
export async function revokeApiKeyAction(
  keyId: string
): Promise<{ success: boolean; error?: string }> {
  try {
    const supabase = await createClient()

    // Verify authentication
    const {
      data: { user },
      error: authError,
    } = await supabase.auth.getUser()
    if (authError || !user) {
      return { success: false, error: "Not authenticated" }
    }

    // Call Supabase RPC to revoke the key
    const { data, error } = await supabase.rpc("revoke_my_api_key", {
      p_key_id: keyId,
    })
    const result = data as RevokeApiKeyResult | null

    if (error) {
      console.error("[RevokeApiKey] Supabase error:", error.message)
      return { success: false, error: error.message }
    }

    if (!result?.success) {
      return { success: false, error: "API key not found or already revoked" }
    }

    return { success: true }
  } catch (error) {
    console.error("[RevokeApiKey] Unexpected error:", error)
    return { success: false, error: "Internal server error" }
  }
}
