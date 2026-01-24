import { createClient } from './supabase/client'

export interface UserCredits {
  free_credits: number
  paid_credits: number
  updated_at: string
}

export interface ServiceConnection {
  service: string
  connected_at: string
}

/**
 * Get the current user's credit balance
 */
export async function getUserCredits(): Promise<UserCredits | null> {
  const supabase = createClient()

  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return null
  }

  const { data, error } = await supabase
    .from('credits')
    .select('free_credits, paid_credits, updated_at')
    .eq('user_id', user.id)
    .single()

  if (error) {
    console.error('Failed to fetch credits:', error)
    return null
  }

  return data
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

  return data || []
}
