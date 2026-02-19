"use client"

import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { useUser, useClerk } from "@clerk/nextjs"
import { fetchAuthUserContext } from "./auth-context-actions"

interface User {
  id: string
  name: string
  email: string
  avatar?: string
  role: "user" | "admin"
}

interface AuthContextType {
  user: User | null
  isLoading: boolean
  isAdmin: boolean
  signOut: () => Promise<void>
  updateName: (name: string) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const { user: clerkUser, isLoaded: clerkLoaded } = useUser()
  const clerk = useClerk()
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    if (!clerkLoaded) return

    if (!clerkUser) {
      setUser(null)
      setIsAdmin(false)
      setIsLoading(false)
      return
    }

    const appUser: User = {
      id: clerkUser.id,
      name: clerkUser.fullName || clerkUser.firstName || clerkUser.emailAddresses[0]?.emailAddress?.split("@")[0] || "User",
      email: clerkUser.emailAddresses[0]?.emailAddress || "",
      avatar: clerkUser.imageUrl,
      role: "user",
    }
    setUser(appUser)

    // Fetch role from backend before clearing isLoading
    // so admin guards don't redirect prematurely
    fetchAuthUserContext().then((ctx) => {
      if (ctx) {
        setIsAdmin(ctx.role === "admin")
        setUser((prev) => prev ? { ...prev, role: ctx.role } : prev)
      }
    }).catch(() => {
      // ignore
    }).finally(() => {
      setIsLoading(false)
    })
  }, [clerkUser, clerkLoaded])

  const signOut = async () => {
    await clerk.signOut()
  }

  const updateName = (name: string) => {
    setUser((prev) => prev ? { ...prev, name } : prev)
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, isAdmin, signOut, updateName }}>
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
