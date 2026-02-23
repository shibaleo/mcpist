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

/** Format a Date to YYYY-MM-DD for the usage API */
function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${day}`
}

/**
 * Get the current user's plan info
 */
export async function getUserPlan(): Promise<UserPlan | null> {
  try {
    const client = await createWorkerClient()
    const today = formatDate(new Date())
    const [profileRes, usageRes] = await Promise.all([
      client.GET("/v1/me/profile"),
      client.GET("/v1/me/usage", {
        params: { query: { start: today, end: today } },
      }),
    ])
    const profile = profileRes.data!
    const usage = usageRes.data
    if (!profile) return null

    return {
      plan_id: profile.plan_id,
      daily_used: usage?.total_used ?? 0,
      daily_limit: profile.daily_limit,
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
    const { data } = await client.GET("/v1/me/credentials")
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
    const today = formatDate(new Date())
    const [profileRes, usageRes] = await Promise.all([
      client.GET("/v1/me/profile"),
      client.GET("/v1/me/usage", {
        params: { query: { start: today, end: today } },
      }),
    ])
    const profile = profileRes.data!
    const usage = usageRes.data
    if (!profile) return null

    return {
      account_status: profile.account_status,
      plan_id: profile.plan_id,
      daily_used: usage?.total_used ?? 0,
      daily_limit: profile.daily_limit,
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
    const { data } = await client.GET("/v1/me/usage", {
      params: {
        query: {
          start: formatDate(startDate),
          end: formatDate(endDate),
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
