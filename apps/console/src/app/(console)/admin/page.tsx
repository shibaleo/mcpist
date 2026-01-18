"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, Activity, Server, CreditCard } from "lucide-react"

// モックデータ
const adminStats = {
  totalUsers: 42,
  activeConnections: 156,
  totalApiCalls: 12847,
  monthlyRevenue: 128000,
}

export default function AdminPage() {
  const { user, isAdmin, isLoading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isLoading && !isAdmin) {
      router.push("/dashboard")
    }
  }, [isLoading, isAdmin, router])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (!isAdmin) {
    return null
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY" }).format(price)
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">管理者パネル</h1>
        <p className="text-muted-foreground mt-1">
          システム全体の統計情報を確認できます
        </p>
      </div>

      <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Users */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Users className="h-4 w-4" />
              登録ユーザー数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{adminStats.totalUsers}</div>
            <p className="text-xs text-muted-foreground mt-1">ユーザー</p>
          </CardContent>
        </Card>

        {/* Active Connections */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Server className="h-4 w-4" />
              アクティブ接続
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{adminStats.activeConnections}</div>
            <p className="text-xs text-muted-foreground mt-1">接続</p>
          </CardContent>
        </Card>

        {/* API Calls */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Activity className="h-4 w-4" />
              API呼び出し
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{adminStats.totalApiCalls.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1">今月</p>
          </CardContent>
        </Card>

        {/* Monthly Revenue */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <CreditCard className="h-4 w-4" />
              月間売上
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{formatPrice(adminStats.monthlyRevenue)}</div>
            <p className="text-xs text-muted-foreground mt-1">今月</p>
          </CardContent>
        </Card>
      </div>

      {/* Placeholder for future admin features */}
      <Card>
        <CardHeader>
          <CardTitle>管理機能</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            将来的にはユーザー管理、システム設定、ログ閲覧などの機能が追加される予定です。
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
