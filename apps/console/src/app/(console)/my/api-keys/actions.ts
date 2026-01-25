"use server"

import { createClient } from "@/lib/supabase/server"

interface RevokeApiKeyResult {
  revoked: boolean
  key_hash: string | null
}

// Server-side Worker URL (uses the same proxy as client)
const WORKER_URL = process.env.MCP_SERVER_URL || "http://mcp.localhost"
// Internal secret for Console → Worker communication
const INTERNAL_SECRET = process.env.INTERNAL_SECRET || ""

/**
 * Revoke an API key and invalidate the Worker cache
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
    const { data, error } = await supabase.rpc("revoke_api_key", {
      p_key_id: keyId,
    })
    const result = data as RevokeApiKeyResult | null

    if (error) {
      console.error("[RevokeApiKey] Supabase error:", error.message)
      return { success: false, error: error.message }
    }

    if (!result?.revoked) {
      return { success: false, error: "API key not found or already revoked" }
    }

    // Invalidate Worker cache if we have the key hash
    if (result.key_hash) {
      try {
        await invalidateWorkerCache(result.key_hash)
        console.log(
          `[RevokeApiKey] Cache invalidated for hash: ${result.key_hash.substring(0, 8)}...`
        )
      } catch (cacheError) {
        // Log but don't fail - the key is already revoked in DB
        console.error("[RevokeApiKey] Failed to invalidate cache:", cacheError)
      }
    }

    return { success: true }
  } catch (error) {
    console.error("[RevokeApiKey] Unexpected error:", error)
    return { success: false, error: "Internal server error" }
  }
}

/**
 * Invalidate the API key cache in the Worker
 */
async function invalidateWorkerCache(keyHash: string): Promise<void> {
  const url = `${WORKER_URL}/internal/invalidate-api-key`

  console.log(`[InvalidateCache] URL: ${url}`)
  console.log(`[InvalidateCache] INTERNAL_SECRET set: ${INTERNAL_SECRET ? 'yes' : 'no'}`)
  console.log(`[InvalidateCache] INTERNAL_SECRET length: ${INTERNAL_SECRET.length}`)

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Internal-Secret": INTERNAL_SECRET,
    },
    body: JSON.stringify({ key_hash: keyHash }),
  })

  console.log(`[InvalidateCache] Response status: ${response.status}`)

  if (!response.ok) {
    const errorText = await response.text()
    console.error(`[InvalidateCache] Error response: ${errorText}`)
    throw new Error(`Worker returned ${response.status}: ${errorText}`)
  }

  const responseText = await response.text()
  console.log(`[InvalidateCache] Success response: ${responseText}`)
}
