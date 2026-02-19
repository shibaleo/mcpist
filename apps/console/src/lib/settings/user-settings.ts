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
 * Get current user's settings via GET /v1/me/profile
 */
export async function getUserSettings(): Promise<UserSettings> {
  try {
    const client = await createWorkerClient()
    const { data } = await client.GET("/v1/me/profile")
    const profile = data!
    if (!profile) return DEFAULT_SETTINGS

    const settings = profile.settings as Record<string, unknown> | null
    return {
      language: (settings?.language as Language) || DEFAULT_SETTINGS.language,
      display_name: profile.display_name || DEFAULT_SETTINGS.display_name,
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
    const { data } = await client.PUT("/v1/me/settings", {
      body: { settings },
    })
    return { success: data?.success ?? false }
  } catch (error) {
    console.error("Failed to update user settings:", error)
    return { success: false, error: String(error) }
  }
}
