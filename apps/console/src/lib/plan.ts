"use server"

import { rpc } from "@/lib/postgrest"
import { getUserId } from "@/lib/auth"

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

interface UserContextRow {
  account_status: string
  plan_id: string
  daily_used: number
  daily_limit: number
  role: string
  settings: Record<string, unknown> | null
  display_name: string | null
  connected_count: number
}

/**
 * Get the current user's plan info
 */
export async function getUserPlan(): Promise<UserPlan | null> {
  try {
    const userId = await getUserId()
    const rows = await rpc<UserContextRow[]>("get_user_context", { p_user_id: userId })
    const context = Array.isArray(rows) ? rows[0] : rows
    if (!context) return null

    return {
      plan_id: context.plan_id,
      daily_used: context.daily_used,
      daily_limit: context.daily_limit,
    }
  } catch {
    return null
  }
}

/**
 * Get the list of connected services for the current user
 */
export async function getServiceConnections(): Promise<ServiceConnection[]> {
  try {
    const userId = await getUserId()
    const data = await rpc<Array<{ module: string; created_at: string; updated_at: string }>>(
      "list_credentials",
      { p_user_id: userId }
    )

    return (data || []).map((item) => ({
      id: item.module,
      service: item.module,
      connected_at: item.created_at,
      updated_at: item.updated_at,
    }))
  } catch {
    return []
  }
}

/**
 * Get the current user's context including account status and plan
 */
export async function getUserContext(): Promise<UserContext | null> {
  try {
    const userId = await getUserId()
    const rows = await rpc<UserContextRow[]>("get_user_context", { p_user_id: userId })
    const context = Array.isArray(rows) ? rows[0] : rows
    if (!context) return null

    return {
      account_status: context.account_status,
      plan_id: context.plan_id,
      daily_used: context.daily_used,
      daily_limit: context.daily_limit,
    }
  } catch {
    return null
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
 */
export async function getMyUsage(startDate: Date, endDate: Date): Promise<UsageStats | null> {
  try {
    const userId = await getUserId()
    return rpc<UsageStats>("get_usage", {
      p_user_id: userId,
      p_start_date: startDate.toISOString(),
      p_end_date: endDate.toISOString(),
    })
  } catch {
    return null
  }
}

/**
 * Get usage for the current month (1st to now)
 */
export async function getMyMonthlyUsage(): Promise<UsageStats | null> {
  const now = new Date()
  const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1)
  return getMyUsage(startOfMonth, now)
}
