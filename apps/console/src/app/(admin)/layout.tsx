"use client"

import { useRouter } from "next/navigation"
import { useEffect } from "react"
import { AuthProvider, useAuth } from "@/lib/auth/auth-context"
import { ConsoleLayout } from "@/components/console-layout"
import { Loader2 } from "lucide-react"

function AdminGuard({ children }: { children: React.ReactNode }) {
  const { isLoading, isAdmin, user } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && user && !isAdmin) {
      router.replace("/dashboard")
    }
  }, [isLoading, isAdmin, user, router])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!isAdmin) {
    return null
  }

  return <>{children}</>
}

export default function AdminRootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <AuthProvider>
      <ConsoleLayout>
        <AdminGuard>{children}</AdminGuard>
      </ConsoleLayout>
    </AuthProvider>
  )
}
