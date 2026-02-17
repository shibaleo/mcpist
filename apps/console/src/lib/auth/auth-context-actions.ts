"use server"

import { workerFetch } from "@/lib/worker-client"

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

export interface AuthUserContext {
  role: "user" | "admin"
  displayName: string | null
}

/**
 * Fetch authenticated user's role and display_name via GET /v1/user/context
 */
export async function fetchAuthUserContext(): Promise<AuthUserContext | null> {
  try {
    const rows = await workerFetch<UserContextRow[]>("GET", "/v1/user/context")
    const ctx = Array.isArray(rows) ? rows[0] : rows
    if (!ctx) return null

    return {
      role: ctx.role === "admin" ? "admin" : "user",
      displayName: ctx.display_name,
    }
  } catch {
    return null
  }
}
