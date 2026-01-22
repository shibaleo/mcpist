"use client"

import { AuthProvider } from "@/lib/auth-context"
import { ConsoleLayout } from "@/components/console-layout"

export default function ConsoleRootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <AuthProvider>
      <ConsoleLayout>{children}</ConsoleLayout>
    </AuthProvider>
  )
}
