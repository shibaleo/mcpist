"use client"

import type React from "react"
import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Sidebar } from "./sidebar"
import { MobileHeader } from "./mobile-header"
import { useMediaQuery } from "@/hooks/use-media-query"
import { useAuth } from "@/lib/auth/auth-context"

interface ConsoleLayoutProps {
  children: React.ReactNode
}

export function ConsoleLayout({ children }: ConsoleLayoutProps) {
  const isTablet = useMediaQuery("(min-width: 768px) and (max-width: 1023px)")
  const [collapsed, setCollapsed] = useState(isTablet)
  const { user, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [isLoading, user, router])

  if (isLoading) {
    return (
      <div className="flex h-dvh items-center justify-center bg-background">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (!user) {
    return null
  }

  return (
    <div className="flex h-dvh bg-background relative">
      {/* Dot Grid Background - covers entire screen */}
      <div className="absolute inset-0 dot-grid pointer-events-none" />

      {/* Desktop/Tablet Sidebar */}
      <div className="hidden md:block relative z-10">
        <Sidebar collapsed={collapsed} onCollapsedChange={setCollapsed} />
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden relative z-10">
        {/* Page Content */}
        <main className="flex-1 overflow-auto">{children}</main>
      </div>

      {/* Mobile Header - outside z-10 stacking context */}
      <MobileHeader />
    </div>
  )
}
