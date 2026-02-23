"use server"

import { createWorkerClient } from "@/lib/worker"

export interface AuthUserContext {
  role: "user" | "admin"
  displayName: string | null
}

/**
 * Ensure user record exists in DB, then fetch profile.
 */
export async function fetchAuthUserContext(): Promise<AuthUserContext | null> {
  try {
    const client = await createWorkerClient()

    // Register (idempotent — creates user if not exists)
    await client.POST("/v1/me/register")

    const { data } = await client.GET("/v1/me/profile")
    const profile = data!
    if (!profile) return null

    return {
      role: profile.role === "admin" ? "admin" : "user",
      displayName: profile.display_name ?? null,
    }
  } catch {
    return null
  }
}
