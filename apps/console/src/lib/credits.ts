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

export interface UserContext {
  account_status: string
  free_credits: number
  paid_credits: number
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

  const { data, error } = await supabase.rpc('list_my_credentials')

  if (error) {
    console.error('Failed to fetch service connections:', error)
    return []
  }

  // Map RPC response to ServiceConnection interface
  return (data || []).map((item: { module: string; created_at: string; updated_at: string }) => ({
    id: item.module,
    service: item.module,
    connected_at: item.created_at,
    updated_at: item.updated_at
  }))
}

/**
 * Get the current user's context including account status and credits
 * Uses get_user_context RPC since mcpist schema is not exposed
 */
export async function getUserContext(): Promise<UserContext | null> {
  const supabase = createClient()

  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return null
  }

  const { data, error } = await supabase.rpc('get_user_context', {
    p_user_id: user.id
  })

  if (error) {
    console.error('Failed to fetch user context:', error)
    return null
  }

  // get_user_context returns an array, take first result
  const context = Array.isArray(data) ? data[0] : data
  if (!context) {
    return null
  }

  return {
    account_status: context.account_status,
    free_credits: context.free_credits,
    paid_credits: context.paid_credits,
  }
}

/**
 * Usage statistics for a period
 */
export interface UsageStats {
  total_consumed: number
  by_module: Record<string, number>
  period: {
    start: string
    end: string
  }
}

/**
 * Get the current user's usage statistics for a period
 * @param startDate - Start of the period (inclusive)
 * @param endDate - End of the period (exclusive)
 */
export async function getMyUsage(startDate: Date, endDate: Date): Promise<UsageStats | null> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('get_my_usage', {
    p_start_date: startDate.toISOString(),
    p_end_date: endDate.toISOString(),
  })

  if (error) {
    console.error('Failed to fetch usage:', error)
    return null
  }

  return data as unknown as UsageStats
}

/**
 * Get usage for the current month (1st to now)
 */
export async function getMyMonthlyUsage(): Promise<UsageStats | null> {
  const now = new Date()
  const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1)
  // End is "now" for current month
  return getMyUsage(startOfMonth, now)
}
