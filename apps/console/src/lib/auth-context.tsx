"use client"

import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { createClient } from "@/lib/supabase/client"
import type { User as SupabaseUser, SupabaseClient } from "@supabase/supabase-js"

// Development auth bypass - set NEXT_PUBLIC_DEV_AUTH_BYPASS=true in .env.local
const DEV_AUTH_BYPASS = process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "true"

interface User {
  id: string
  name: string
  email: string
  avatar?: string
  role: "user" | "admin"
}

interface AuthContextType {
  user: User | null
  supabaseUser: SupabaseUser | null
  supabase: SupabaseClient | null
  isLoading: boolean
  isAdmin: boolean
  signOut: () => Promise<void>
  updateName: (name: string) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

// Dummy user for development bypass
const DEV_BYPASS_USER: User = {
  id: "dev-bypass-user-id",
  name: "Dev User",
  email: "dev@localhost",
  role: "admin",
}

async function fetchUserRole(client: SupabaseClient): Promise<"user" | "admin"> {
  const { data } = await client.rpc("get_my_role")
  // RPC returns { role: "user" | "admin" }
  const role = (data as { role: string } | null)?.role
  return role === "admin" ? "admin" : "user"
}

async function buildUser(client: SupabaseClient, sbUser: SupabaseUser): Promise<User> {
  const [role, settings] = await Promise.all([
    fetchUserRole(client),
    client.rpc("get_my_settings").then(({ data }) => data as Record<string, unknown> | null),
  ])
  const displayName = (settings?.display_name as string) || sbUser.user_metadata?.full_name || sbUser.email?.split("@")[0] || "User"
  return {
    id: sbUser.id,
    name: displayName,
    email: sbUser.email || "",
    avatar: sbUser.user_metadata?.avatar_url,
    role,
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => DEV_AUTH_BYPASS ? DEV_BYPASS_USER : null)
  const [supabaseUser, setSupabaseUser] = useState<SupabaseUser | null>(null)
  const [supabase] = useState<SupabaseClient | null>(() => {
    if (typeof window === "undefined") return null
    return createClient()
  })
  const [isLoading, setIsLoading] = useState(!DEV_AUTH_BYPASS)
  const [isAdmin, setIsAdmin] = useState(DEV_AUTH_BYPASS)

  useEffect(() => {
    if (DEV_AUTH_BYPASS) {
      console.warn("[Auth] DEV_AUTH_BYPASS enabled - using dummy user")
      return
    }

    if (!supabase) return

    // IMPORTANT: Auth initialization order matters to avoid deadlocks
    // See: https://github.com/supabase/supabase-js/issues/1594
    //
    // 1. Set up onAuthStateChange BEFORE calling getSession()
    //    - This ensures the listener is ready when session events fire
    //
    // 2. Use .then() instead of await for getSession()
    //    - Prevents blocking the event loop during initialization
    //
    // 3. Wrap async operations (like buildUser) in setTimeout(..., 0)
    //    - Defers execution to next tick, avoiding deadlocks with Supabase internals
    //    - The Web Locks API can cause hangs if async calls happen synchronously
    //      within onAuthStateChange callback

    const { data: { subscription } } = supabase.auth.onAuthStateChange(async (_event, session) => {
      const sbUser = session?.user ?? null
      setSupabaseUser(sbUser)

      if (sbUser) {
        // setTimeout prevents deadlock - do not remove!
        setTimeout(async () => {
          try {
            const appUser = await buildUser(supabase, sbUser)
            setUser(appUser)
            setIsAdmin(appUser.role === "admin")
          } catch {
            // Role fetch failed, default to user
            setUser({
              id: sbUser.id,
              name: sbUser.user_metadata?.full_name || sbUser.email?.split("@")[0] || "User",
              email: sbUser.email || "",
              avatar: sbUser.user_metadata?.avatar_url,
              role: "user",
            })
            setIsAdmin(false)
          }
          setIsLoading(false)
        }, 0)
      } else {
        setUser(null)
        setIsAdmin(false)
        setIsLoading(false)
      }
    })

    // Use .then() not await - prevents blocking during init
    supabase.auth.getSession().then(({ data: { session } }) => {
      // onAuthStateChange will handle the state update
      if (!session) {
        setIsLoading(false)
      }
    }).catch(() => {
      setIsLoading(false)
    })

    return () => {
      subscription.unsubscribe()
    }
  }, [supabase])

  const signOut = async () => {
    if (supabase) {
      await supabase.auth.signOut()
    }
  }

  const updateName = (name: string) => {
    setUser((prev) => prev ? { ...prev, name } : prev)
  }

  return (
    <AuthContext.Provider value={{ user, supabaseUser, supabase, isLoading, isAdmin, signOut, updateName }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
