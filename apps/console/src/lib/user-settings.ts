"use server"

import { createClient } from "@/lib/supabase/server"

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
 * Get current user's settings
 */
export async function getUserSettings(): Promise<UserSettings> {
  const supabase = await createClient()

  const { data, error } = await supabase.rpc("get_my_settings")

  if (error || !data) {
    console.error("Failed to get user settings:", error)
    return DEFAULT_SETTINGS
  }

  const prefs = data as Record<string, unknown>
  return {
    language: (prefs?.language as Language) || DEFAULT_SETTINGS.language,
    display_name: (prefs?.display_name as string) || DEFAULT_SETTINGS.display_name,
  }
}

/**
 * Update current user's settings
 */
export async function updateUserSettings(
  settings: Partial<UserSettings>
): Promise<{ success: boolean; error?: string }> {
  const supabase = await createClient()

  const { data, error } = await supabase.rpc("update_my_settings", {
    p_settings: settings,
  })

  if (error) {
    console.error("Failed to update user settings:", error)
    return { success: false, error: error.message }
  }

  const result = data as { success: boolean } | null
  return { success: result?.success ?? false }
}
