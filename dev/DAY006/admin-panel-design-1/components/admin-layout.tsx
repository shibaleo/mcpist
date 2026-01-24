"use client"

import type React from "react"

import { useState, useEffect } from "react"
import { Sidebar } from "./sidebar"
import { MobileHeader } from "./mobile-header"
import { useMediaQuery } from "@/hooks/use-media-query"

interface AdminLayoutProps {
  children: React.ReactNode
}

export function AdminLayout({ children }: AdminLayoutProps) {
  const [collapsed, setCollapsed] = useState(false)
  const isTablet = useMediaQuery("(min-width: 768px) and (max-width: 1023px)")
  const isMobile = useMediaQuery("(max-width: 767px)")

  useEffect(() => {
    if (isTablet) {
      setCollapsed(true)
    } else if (!isMobile) {
      setCollapsed(false)
    }
  }, [isTablet, isMobile])

  return (
    <div className="flex h-screen bg-background">
      {/* Desktop/Tablet Sidebar */}
      <div className="hidden md:block">
        <Sidebar collapsed={collapsed} onCollapsedChange={setCollapsed} />
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Mobile Header */}
        <MobileHeader />

        {/* Page Content */}
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  )
}
