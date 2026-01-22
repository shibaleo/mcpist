/**
 * Authorization Code storage utilities
 * Uses Supabase RPC for persistent storage (mcpist schema is not exposed via PostgREST)
 */

import { createClient } from '@supabase/supabase-js'

export interface AuthorizationCodeData {
  code: string
  user_id: string
  client_id: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: string
  scope: string | null
  state: string | null
  expires_at: string
}

function getSupabaseAdmin() {
  const url = process.env.NEXT_PUBLIC_SUPABASE_URL || process.env.SUPABASE_URL
  const serviceKey = process.env.SUPABASE_SERVICE_ROLE_KEY

  if (!url || !serviceKey) {
    throw new Error('Supabase configuration missing')
  }

  return createClient(url, serviceKey)
}

/**
 * Store authorization code in database via RPC
 */
export async function storeAuthorizationCode(data: Omit<AuthorizationCodeData, 'expires_at'>): Promise<void> {
  const supabase = getSupabaseAdmin()

  // Code expires in 5 minutes
  const expiresAt = new Date(Date.now() + 5 * 60 * 1000).toISOString()

  const { error } = await supabase.rpc('store_oauth_code', {
    p_code: data.code,
    p_user_id: data.user_id,
    p_client_id: data.client_id,
    p_redirect_uri: data.redirect_uri,
    p_code_challenge: data.code_challenge,
    p_code_challenge_method: data.code_challenge_method,
    p_scope: data.scope,
    p_state: data.state,
    p_expires_at: expiresAt,
  })

  if (error) {
    throw new Error('Failed to store authorization code: ' + error.message)
  }
}

/**
 * Retrieve and consume authorization code via RPC
 * Returns null if code is invalid, expired, or already used
 */
export async function consumeAuthorizationCode(code: string): Promise<AuthorizationCodeData | null> {
  const supabase = getSupabaseAdmin()

  const { data, error } = await supabase.rpc('consume_oauth_code', {
    p_code: code,
  })

  if (error) {
    console.error('Failed to consume authorization code:', error)
    return null
  }

  // RPC returns an array, get first row
  if (!data || data.length === 0) {
    return null
  }

  return data[0] as AuthorizationCodeData
}
