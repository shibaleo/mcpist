import { createClient } from './supabase/client'

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

export class ApiKeyError extends Error {
  constructor(message: string, public code?: string) {
    super(message)
    this.name = 'ApiKeyError'
  }
}

export async function listApiKeys(): Promise<ApiKey[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_my_api_keys')

  if (error) {
    throw new ApiKeyError(error.message, error.code)
  }

  return data || []
}

export async function generateApiKey(
  name: string,
  expiresInDays: number | null = null
): Promise<GenerateApiKeyResult> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('generate_my_api_key', {
    p_display_name: name,
    p_expires_at: expiresInDays ? new Date(Date.now() + expiresInDays * 24 * 60 * 60 * 1000).toISOString() : undefined,
  })

  if (error) {
    throw new ApiKeyError(error.message, error.code)
  }

  if (!data) {
    throw new ApiKeyError('Failed to generate API key')
  }

  return data as unknown as GenerateApiKeyResult
}

// Note: revokeApiKey has been moved to a Server Action
// See: apps/console/src/app/(console)/my/api-keys/actions.ts
