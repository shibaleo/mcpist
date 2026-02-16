import { cache } from "react"
import { createClient } from "@/lib/supabase/server"

/**
 * Get the authenticated user's ID.
 * Cached per-request via React.cache so multiple Server Actions
 * in the same request share a single Supabase Auth verification.
 */
export const getUserId = cache(async (): Promise<string> => {
  const supabase = await createClient()
  const {
    data: { user },
  } = await supabase.auth.getUser()
  if (!user) throw new Error("Not authenticated")
  return user.id
})
