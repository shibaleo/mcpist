import type React from "react"
import { AuthProvider } from "@/lib/auth-context"
import { AdminLayout } from "@/components/admin-layout"

export default function AdminRootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <AuthProvider>
      <AdminLayout>{children}</AdminLayout>
    </AuthProvider>
  )
}
