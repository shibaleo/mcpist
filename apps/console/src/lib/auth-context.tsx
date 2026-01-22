"use client"

import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { createClient } from "@/lib/supabase/client"
import type { User as SupabaseUser, SupabaseClient } from "@supabase/supabase-js"

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
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

async function fetchUserRole(client: SupabaseClient): Promise<"user" | "admin"> {
  const { data } = await client.rpc("get_my_role")
  return (data as "user" | "admin") || "user"
}

async function buildUser(client: SupabaseClient, sbUser: SupabaseUser): Promise<User> {
  const role = await fetchUserRole(client)
  return {
    id: sbUser.id,
    name: sbUser.user_metadata?.full_name || sbUser.email?.split("@")[0] || "User",
    email: sbUser.email || "",
    avatar: sbUser.user_metadata?.avatar_url,
    role,
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [supabaseUser, setSupabaseUser] = useState<SupabaseUser | null>(null)
  const [supabase, setSupabase] = useState<SupabaseClient | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    const client = createClient()
    setSupabase(client)

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

    const { data: { subscription } } = client.auth.onAuthStateChange(async (_event, session) => {
      const sbUser = session?.user ?? null
      setSupabaseUser(sbUser)

      if (sbUser) {
        // setTimeout prevents deadlock - do not remove!
        setTimeout(async () => {
          try {
            const appUser = await buildUser(client, sbUser)
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
    client.auth.getSession().then(({ data: { session } }) => {
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
  }, [])

  const signOut = async () => {
    if (supabase) {
      await supabase.auth.signOut()
    }
  }

  return (
    <AuthContext.Provider value={{ user, supabaseUser, supabase, isLoading, isAdmin, signOut }}>
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
