"use server"

import { workerFetch } from "@/lib/worker-client"

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
    const rows = await workerFetch<Array<{
      settings: Record<string, unknown> | null
      display_name: string | null
      language: string
    }>>("GET", "/v1/user/context")

    const ctx = Array.isArray(rows) ? rows[0] : rows
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
    const result = await workerFetch<{ success: boolean }>("PUT", "/v1/user/settings", {
      settings,
    })
    return { success: result?.success ?? false }
  } catch (error) {
    console.error("Failed to update user settings:", error)
    return { success: false, error: String(error) }
  }
}
