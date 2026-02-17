"use server"

import { rpc } from "@/lib/worker-client"

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
 * Get current user's settings via get_user_context
 */
export async function getUserSettings(): Promise<UserSettings> {
  try {
    const rows = await rpc<Array<{
      settings: Record<string, unknown> | null
      display_name: string | null
      language: string
    }>>("get_user_context")

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
    const result = await rpc<{ success: boolean }>("update_settings", {
      p_settings: settings,
    })
    return { success: result?.success ?? false }
  } catch (error) {
    console.error("Failed to update user settings:", error)
    return { success: false, error: String(error) }
  }
}
