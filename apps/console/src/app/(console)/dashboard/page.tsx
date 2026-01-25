"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Link2, Coins, Receipt, Loader2, Settings2 } from "lucide-react"
import { getUserCredits, getServiceConnections, type UserCredits, type ServiceConnection } from "@/lib/credits"

export default function DashboardPage() {
  const { user } = useAuth()
  const [credits, setCredits] = useState<UserCredits | null>(null)
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchData() {
      setLoading(true)
      try {
        const [creditsData, connectionsData] = await Promise.all([
          getUserCredits(),
          getServiceConnections(),
        ])
        setCredits(creditsData)
        setConnections(connectionsData)
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  const totalCredits = credits ? credits.free_credits + credits.paid_credits : 0

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">ダッシュボード</h1>
        <p className="text-muted-foreground mt-1">
          MCPistへようこそ、{user?.name}さん
        </p>
      </div>

      <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Connected Services */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Link2 className="h-4 w-4" />
              連携中のサービス
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : (
              <>
                <div className="text-3xl font-bold">{connections.length}</div>
                <p className="text-xs text-muted-foreground mt-1">サービス</p>
              </>
            )}
          </CardContent>
        </Card>

        {/* Enabled Tools (Mock) */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Settings2 className="h-4 w-4" />
              有効なツール
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : (
              <>
                <div className="text-3xl font-bold">-</div>
                <p className="text-xs text-muted-foreground mt-1">ツール設定で確認</p>
              </>
            )}
          </CardContent>
        </Card>

        {/* Credit Balance */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Coins className="h-4 w-4" />
              クレジット残高
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : (
              <>
                <div className="text-3xl font-bold">{totalCredits.toLocaleString()}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  無料: {credits?.free_credits.toLocaleString() ?? 0} / 有料: {credits?.paid_credits.toLocaleString() ?? 0}
                </p>
              </>
            )}
          </CardContent>
        </Card>

        {/* Usage This Month */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Receipt className="h-4 w-4" />
              今月の利用
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : (
              <>
                <div className="text-3xl font-bold">-</div>
                <p className="text-xs text-muted-foreground mt-1">詳細は課金ページで確認</p>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
