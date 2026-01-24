import { createClient } from './supabase/client'

export interface UserCredits {
  free_credits: number
  paid_credits: number
  updated_at: string
}

export interface ServiceConnection {
  id: string
  service: string
  connected_at: string
  updated_at: string
}

/**
 * Get the current user's credit balance
 * Uses get_user_context RPC since mcpist schema is not exposed
 */
export async function getUserCredits(): Promise<UserCredits | null> {
  const supabase = createClient()

  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return null
  }

  const { data, error } = await supabase.rpc('get_user_context', {
    p_user_id: user.id
  })

  if (error) {
    console.error('Failed to fetch credits:', error)
    return null
  }

  // get_user_context returns an array, take first result
  const context = Array.isArray(data) ? data[0] : data
  if (!context) {
    return null
  }

  return {
    free_credits: context.free_credits,
    paid_credits: context.paid_credits,
    updated_at: new Date().toISOString() // RPC doesn't return updated_at
  }
}

/**
 * Get the list of connected services for the current user
 */
export async function getServiceConnections(): Promise<ServiceConnection[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_service_connections')

  if (error) {
    console.error('Failed to fetch service connections:', error)
    return []
  }

  // Map RPC response to ServiceConnection interface
  return (data || []).map((item: { id: string; service: string; created_at: string; updated_at: string }) => ({
    id: item.id,
    service: item.service,
    connected_at: item.created_at,
    updated_at: item.updated_at
  }))
}
