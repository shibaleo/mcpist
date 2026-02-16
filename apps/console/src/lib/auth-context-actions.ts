"use server"

import { rpc } from "@/lib/postgrest"
import { getUserId } from "@/lib/auth"

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
 * Fetch authenticated user's role and display_name via get_user_context RPC
 */
export async function fetchAuthUserContext(): Promise<AuthUserContext | null> {
  try {
    const userId = await getUserId()
    const rows = await rpc<UserContextRow[]>("get_user_context", { p_user_id: userId })
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
