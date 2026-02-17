"use server"

import { createWorkerClient } from "@/lib/worker"
import type { components } from "@/lib/worker"

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

export type UsageStats = components["schemas"]["UsageData"]

/**
 * Get the current user's plan info
 */
export async function getUserPlan(): Promise<UserPlan | null> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/user/context")
    const context = data![0]
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
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/credentials")
    const rows = data!

    return rows.map((item) => ({
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
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/user/context")
    const context = data![0]
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
 * Get the current user's usage statistics for a period
 */
export async function getMyUsage(startDate: Date, endDate: Date): Promise<UsageStats | null> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/user/usage", {
      params: {
        query: {
          start: startDate.toISOString(),
          end: endDate.toISOString(),
        },
      },
    })
    return data!
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
