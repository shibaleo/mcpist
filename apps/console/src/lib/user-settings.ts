"use server"

import { createClient } from "@/lib/supabase/server"

export type Language = "en-US" | "ja-JP"

export interface UserSettings {
  language: Language
}

const DEFAULT_SETTINGS: UserSettings = {
  language: "en-US",
}

/**
 * Get current user's settings
 */
export async function getUserSettings(): Promise<UserSettings> {
  const supabase = await createClient()

  const { data, error } = await supabase.rpc("get_my_preferences")

  if (error || !data) {
    console.error("Failed to get user settings:", error)
    return DEFAULT_SETTINGS
  }

  const prefs = data as Record<string, unknown>
  return {
    language: (prefs?.language as Language) || DEFAULT_SETTINGS.language,
  }
}

/**
 * Update current user's settings
 */
export async function updateUserSettings(
  settings: Partial<UserSettings>
): Promise<{ success: boolean; error?: string }> {
  const supabase = await createClient()

  const { data, error } = await supabase.rpc("update_my_preferences", {
    p_preferences: settings,
  })

  if (error) {
    console.error("Failed to update user settings:", error)
    return { success: false, error: error.message }
  }

  const result = data as { success: boolean } | null
  return { success: result?.success ?? false }
}
