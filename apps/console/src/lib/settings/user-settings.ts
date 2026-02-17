"use server"

import { createWorkerClient } from "@/lib/worker"

export type Language = "en-US" | "ja-JP"

export interface UserSettings {
  language: Language
  display_name: string
}

const DEFAULT_SETTINGS: UserSettings = {
  language: "en-US",
  display_name: "",
}

/**
 * Get current user's settings via GET /v1/user/context
 */
export async function getUserSettings(): Promise<UserSettings> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/user/context")
    const ctx = data![0]
    if (!ctx) return DEFAULT_SETTINGS

    return {
      language: (ctx.language as Language) || DEFAULT_SETTINGS.language,
      display_name: ctx.display_name || DEFAULT_SETTINGS.display_name,
    }
  } catch {
    return DEFAULT_SETTINGS
  }
}

/**
 * Update current user's settings
 */
export async function updateUserSettings(
  settings: Partial<UserSettings>
): Promise<{ success: boolean; error?: string }> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.PUT("/v1/user/settings", {
      body: { settings },
    })
    return { success: data?.success ?? false }
  } catch (error) {
    console.error("Failed to update user settings:", error)
    return { success: false, error: String(error) }
  }
}
