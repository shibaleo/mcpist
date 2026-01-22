"use client"

import { useAuth } from "@/lib/auth-context"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Link2, Wrench, Receipt } from "lucide-react"

// モックデータ
const dashboardStats = {
  connectedServices: 4,
  toolUsageCount: 128,
  nextBillingAmount: 1000,
}

export default function DashboardPage() {
  const { user } = useAuth()

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY" }).format(price)
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">ダッシュボード</h1>
        <p className="text-muted-foreground mt-1">
          MCPistへようこそ、{user?.name}さん
        </p>
      </div>

      <div className="grid md:grid-cols-3 gap-6">
        {/* Connected Services */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Link2 className="h-4 w-4" />
              連携中のサービス
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{dashboardStats.connectedServices}</div>
            <p className="text-xs text-muted-foreground mt-1">サービス</p>
          </CardContent>
        </Card>

        {/* Tool Usage */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Wrench className="h-4 w-4" />
              ツール使用回数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{dashboardStats.toolUsageCount.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1">今月</p>
          </CardContent>
        </Card>

        {/* Next Billing */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Receipt className="h-4 w-4" />
              来月の請求額
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{formatPrice(dashboardStats.nextBillingAmount)}</div>
            <p className="text-xs text-muted-foreground mt-1">2月請求予定</p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
