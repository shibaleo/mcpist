/**
 * Authorization Code storage utilities
 * Uses Supabase RPC for persistent storage
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

export interface AuthorizationRequest {
  id: string
  client_id: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: string
  scope: string
  state: string | null
  created_at: string
  expires_at: string
}

function getSupabaseAdmin() {
  const url = process.env.SUPABASE_URL
  const serviceKey = process.env.SUPABASE_SERVICE_ROLE_KEY

  if (!url || !serviceKey) {
    throw new Error('Supabase configuration missing')
  }

  return createClient(url, serviceKey)
}

/**
 * Store authorization request (pending consent)
 * Returns authorization_id for the consent flow
 */
export async function storeAuthorizationRequest(data: {
  id: string
  client_id: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: string
  scope: string
  state: string | null
}): Promise<void> {
  const supabase = getSupabaseAdmin()

  // Request expires in 10 minutes (time to complete consent)
  const expiresAt = new Date(Date.now() + 10 * 60 * 1000).toISOString()

  const { error } = await supabase.rpc('store_oauth_authorization_request', {
    p_id: data.id,
    p_client_id: data.client_id,
    p_redirect_uri: data.redirect_uri,
    p_code_challenge: data.code_challenge,
    p_code_challenge_method: data.code_challenge_method,
    p_scope: data.scope,
    p_state: data.state,
    p_expires_at: expiresAt,
  })

  if (error) {
    throw new Error('Failed to store authorization request: ' + error.message)
  }
}

/**
 * Get authorization request details by ID
 */
export async function getAuthorizationRequest(id: string): Promise<AuthorizationRequest | null> {
  const supabase = getSupabaseAdmin()

  const { data, error } = await supabase.rpc('get_oauth_authorization_request', {
    p_id: id,
  })

  if (error) {
    console.error('Failed to get authorization request:', error)
    return null
  }

  if (!data || data.length === 0) {
    return null
  }

  return data[0] as AuthorizationRequest
}

/**
 * Approve authorization request and generate authorization code
 * Returns the code and redirect_uri
 */
export async function approveAuthorizationRequest(
  authorizationId: string,
  userId: string
): Promise<{ code: string; redirect_uri: string; state: string | null } | null> {
  const supabase = getSupabaseAdmin()

  const { data, error } = await supabase.rpc('approve_oauth_authorization', {
    p_authorization_id: authorizationId,
    p_user_id: userId,
  })

  if (error) {
    console.error('Failed to approve authorization:', error)
    return null
  }

  if (!data || data.length === 0) {
    return null
  }

  return data[0] as { code: string; redirect_uri: string; state: string | null }
}

/**
 * Deny authorization request
 * Returns redirect_uri and state for error redirect
 */
export async function denyAuthorizationRequest(
  authorizationId: string
): Promise<{ redirect_uri: string; state: string | null } | null> {
  const supabase = getSupabaseAdmin()

  const { data, error } = await supabase.rpc('deny_oauth_authorization', {
    p_authorization_id: authorizationId,
  })

  if (error) {
    console.error('Failed to deny authorization:', error)
    return null
  }

  if (!data || data.length === 0) {
    return null
  }

  return data[0] as { redirect_uri: string; state: string | null }
}

/**
 * Store authorization code in database via RPC
 */
export async function storeAuthorizationCode(data: Omit<AuthorizationCodeData, 'expires_at'>): Promise<void> {
  const supabase = getSupabaseAdmin()

  // Code expires in 10 minutes (matching Supabase)
  const expiresAt = new Date(Date.now() + 10 * 60 * 1000).toISOString()

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

// === Refresh Token Management ===

export interface RefreshTokenData {
  token: string
  user_id: string
  client_id: string
  scope: string | null
  expires_at: string
}

/**
 * Store refresh token in database
 */
export async function storeRefreshToken(data: {
  token: string
  user_id: string
  client_id: string
  scope: string | null
}): Promise<void> {
  const supabase = getSupabaseAdmin()

  // Refresh token expires in 30 days
  const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()

  const { error } = await supabase.rpc('store_oauth_refresh_token', {
    p_token: data.token,
    p_user_id: data.user_id,
    p_client_id: data.client_id,
    p_scope: data.scope,
    p_expires_at: expiresAt,
  })

  if (error) {
    throw new Error('Failed to store refresh token: ' + error.message)
  }
}

/**
 * Validate and rotate refresh token
 * Returns the token data if valid, null otherwise
 * The old token is invalidated and a new token should be generated
 */
export async function consumeRefreshToken(token: string): Promise<RefreshTokenData | null> {
  const supabase = getSupabaseAdmin()

  const { data, error } = await supabase.rpc('consume_oauth_refresh_token', {
    p_token: token,
  })

  if (error) {
    console.error('Failed to consume refresh token:', error)
    return null
  }

  if (!data || data.length === 0) {
    return null
  }

  return data[0] as RefreshTokenData
}
