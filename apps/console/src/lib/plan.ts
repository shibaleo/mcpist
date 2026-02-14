import { createClient } from './supabase/client'

export interface UserPlan {
  plan_id: string
  daily_used: number
  daily_limit: number
}

export interface ServiceConnection {
  id: string
  service: string
  connected_at: string
  updated_at: string
}

export interface UserContext {
  account_status: string
  plan_id: string
  daily_used: number
  daily_limit: number
}

/**
 * Get the current user's plan info
 * Uses get_user_context RPC since mcpist schema is not exposed
 */
export async function getUserPlan(): Promise<UserPlan | null> {
  const supabase = createClient()

  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return null
  }

  const { data, error } = await supabase.rpc('get_user_context', {
    p_user_id: user.id
  })

  if (error) {
    console.error('Failed to fetch plan:', error)
    return null
  }

  // get_user_context returns an array, take first result
  const context = Array.isArray(data) ? data[0] : data
  if (!context) {
    return null
  }

  return {
    plan_id: context.plan_id,
    daily_used: context.daily_used,
    daily_limit: context.daily_limit,
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
 * Get the current user's context including account status and plan
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
    plan_id: context.plan_id,
    daily_used: context.daily_used,
    daily_limit: context.daily_limit,
  }
}

/**
 * Usage statistics for a period
 */
export interface UsageStats {
  total_used: number
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
