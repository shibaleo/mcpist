"use server"

import { rpc } from "@/lib/postgrest"
import { getUserId } from "@/lib/auth"

export interface ApiKey {
  id: string
  display_name: string
  key_prefix: string
  last_used_at: string | null
  expires_at: string | null
  revoked_at: string | null
}

export interface GenerateApiKeyResult {
  api_key: string
  key_prefix: string
}

export async function listApiKeys(): Promise<ApiKey[]> {
  const userId = await getUserId()
  return rpc<ApiKey[]>("list_api_keys", { p_user_id: userId })
}

export async function generateApiKey(
  name: string,
  expiresInDays: number | null = null
): Promise<GenerateApiKeyResult> {
  const userId = await getUserId()
  return rpc<GenerateApiKeyResult>("generate_api_key", {
    p_user_id: userId,
    p_display_name: name,
    p_expires_at: expiresInDays
      ? new Date(Date.now() + expiresInDays * 24 * 60 * 60 * 1000).toISOString()
      : undefined,
  })
}

export async function revokeApiKey(
  keyId: string
): Promise<{ success: boolean }> {
  const userId = await getUserId()
  return rpc<{ success: boolean }>("revoke_api_key", {
    p_user_id: userId,
    p_key_id: keyId,
  })
}
