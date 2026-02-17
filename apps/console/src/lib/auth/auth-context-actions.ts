"use server"

import { createWorkerClient } from "@/lib/worker"

export interface AuthUserContext {
  role: "user" | "admin"
  displayName: string | null
}

/**
 * Fetch authenticated user's role and display_name via GET /v1/user/context
 */
export async function fetchAuthUserContext(): Promise<AuthUserContext | null> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/user/context")
    const rows = data!
    const ctx = rows[0]
    if (!ctx) return null

    return {
      role: ctx.role === "admin" ? "admin" : "user",
      displayName: ctx.display_name ?? null,
    }
  } catch {
    return null
  }
}
