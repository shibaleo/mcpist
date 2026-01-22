import { createClient } from './supabase/client'

export interface ApiKey {
  id: string
  name: string
  key_prefix: string
  last_used_at: string | null
  expires_at: string | null
  created_at: string
  is_expired: boolean
}

export interface GenerateApiKeyResult {
  id: string
  name: string
  key: string
  key_prefix: string
  expires_at: string | null
}

export class ApiKeyError extends Error {
  constructor(message: string, public code?: string) {
    super(message)
    this.name = 'ApiKeyError'
  }
}

export async function listApiKeys(): Promise<ApiKey[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_api_keys')

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

  const { data, error } = await supabase.rpc('generate_api_key', {
    p_name: name,
    p_expires_in_days: expiresInDays,
  })

  if (error) {
    throw new ApiKeyError(error.message, error.code)
  }

  return data
}

// Note: revokeApiKey has been moved to a Server Action
// See: apps/console/src/app/(console)/my/api-keys/actions.ts
