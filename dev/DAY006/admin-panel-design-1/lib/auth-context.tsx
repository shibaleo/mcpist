"use client"

import { createContext, useContext, useState, type ReactNode } from "react"

type UserRole = "admin" | "user"

interface User {
  id: string
  name: string
  email: string
  role: UserRole
  avatar?: string
}

interface AuthContextType {
  user: User | null
  setUser: (user: User | null) => void
  isAdmin: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>({
    id: "1",
    name: "山田 太郎",
    email: "yamada@example.com",
    role: "admin",
    avatar: undefined,
  })

  const isAdmin = user?.role === "admin"

  return <AuthContext.Provider value={{ user, setUser, isAdmin }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
