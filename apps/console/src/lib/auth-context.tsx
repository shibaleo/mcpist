"use client"

import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { createClient } from "@/lib/supabase/client"
import type { User as SupabaseUser, SupabaseClient } from "@supabase/supabase-js"

interface User {
  id: string
  name: string
  email: string
  avatar?: string
  isAdmin?: boolean
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

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [supabaseUser, setSupabaseUser] = useState<SupabaseUser | null>(null)
  const [supabase, setSupabase] = useState<SupabaseClient | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    const client = createClient()
    setSupabase(client)

    const getUser = async () => {
      const { data: { user: sbUser } } = await client.auth.getUser()
      setSupabaseUser(sbUser)

      if (sbUser) {
        // Check if user is admin from user_metadata or app_metadata
        const adminFlag = sbUser.user_metadata?.admin || sbUser.app_metadata?.admin || false
        setIsAdmin(adminFlag)

        setUser({
          id: sbUser.id,
          name: sbUser.user_metadata?.full_name || sbUser.email?.split("@")[0] || "User",
          email: sbUser.email || "",
          avatar: sbUser.user_metadata?.avatar_url,
          isAdmin: adminFlag,
        })
      } else {
        setUser(null)
        setIsAdmin(false)
      }
      setIsLoading(false)
    }

    getUser()

    const { data: { subscription } } = client.auth.onAuthStateChange((_event, session) => {
      const sbUser = session?.user ?? null
      setSupabaseUser(sbUser)

      if (sbUser) {
        const adminFlag = sbUser.user_metadata?.admin || sbUser.app_metadata?.admin || false
        setIsAdmin(adminFlag)

        setUser({
          id: sbUser.id,
          name: sbUser.user_metadata?.full_name || sbUser.email?.split("@")[0] || "User",
          email: sbUser.email || "",
          avatar: sbUser.user_metadata?.avatar_url,
          isAdmin: adminFlag,
        })
      } else {
        setUser(null)
        setIsAdmin(false)
      }
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
