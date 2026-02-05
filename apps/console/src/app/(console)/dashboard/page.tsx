"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Link2, Coins, Receipt, Loader2, Settings2, ChevronRight } from "lucide-react"
import { getUserContext, getServiceConnections, getMyMonthlyUsage, type UserCredits, type ServiceConnection, type UsageStats } from "@/lib/credits"
import { getMyToolSettings, type ToolSetting } from "@/lib/tool-settings"
import Link from "next/link"
import { cn } from "@/lib/utils"

// オンボーディングステップの判定
type OnboardingStep = "connections" | "billing" | "complete"

function getOnboardingStep(
  connections: ServiceConnection[],
  accountStatus: string | null
): OnboardingStep {
  // 1. サービス未連携
  if (connections.length === 0) {
    return "connections"
  }
  // 2. 初回クレジット未取得（pre_active）
  if (accountStatus === "pre_active") {
    return "billing"
  }
  // 全て完了
  return "complete"
}

// ハイライトカードのラッパー
function HighlightCard({
  children,
  href,
  highlight,
  className,
}: {
  children: React.ReactNode
  href: string
  highlight?: boolean
  className?: string
}) {
  return (
    <Link href={href} className="block group">
      <Card
        className={cn(
          "relative transition-all duration-300 cursor-pointer h-full",
          "hover:border-primary/50 hover:shadow-md",
          highlight && "animate-pulse-border border-primary shadow-lg shadow-primary/20",
          className
        )}
      >
        {children}
        <div className="absolute bottom-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity">
          <ChevronRight className="h-4 w-4 text-muted-foreground" />
        </div>
      </Card>
    </Link>
  )
}

export default function DashboardPage() {
  const { user } = useAuth()
  const [credits, setCredits] = useState<UserCredits | null>(null)
  const [accountStatus, setAccountStatus] = useState<string | null>(null)
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [toolSettings, setToolSettings] = useState<ToolSetting[]>([])
  const [monthlyUsage, setMonthlyUsage] = useState<UsageStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchData() {
      setLoading(true)
      try {
        const [contextData, connectionsData, toolSettingsData, usageData] = await Promise.all([
          getUserContext(),
          getServiceConnections(),
          getMyToolSettings(),
          getMyMonthlyUsage(),
        ])
        setAccountStatus(contextData?.account_status ?? null)
        setCredits(contextData ? {
          free_credits: contextData.free_credits,
          paid_credits: contextData.paid_credits,
          updated_at: new Date().toISOString(),
        } : null)
        setConnections(connectionsData)
        setToolSettings(toolSettingsData)
        setMonthlyUsage(usageData)
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  const totalCredits = credits ? credits.free_credits + credits.paid_credits : 0
  const enabledToolCount = toolSettings.filter((t) => t.enabled).length
  const totalToolCount = toolSettings.length
  const onboardingStep = getOnboardingStep(connections, accountStatus)

  // 残高アラート（active ユーザーで残高50以下）
  const LOW_CREDIT_THRESHOLD = 50
  const showLowCreditAlert = accountStatus === "active" && totalCredits <= LOW_CREDIT_THRESHOLD

  // オンボーディングメッセージ
  const onboardingMessages: Record<OnboardingStep, string> = {
    connections: "まずはサービスを連携しましょう",
    billing: "初回クレジットを受け取りましょう",
    complete: "",
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">ダッシュボード</h1>
        <p className="text-muted-foreground mt-1">
          MCPistへようこそ、{user?.name}さん
        </p>
      </div>

      {/* オンボーディングガイド */}
      {!loading && onboardingStep !== "complete" && (
        <div className="bg-primary/10 border border-primary/30 rounded-lg p-4 flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
            <span className="text-primary font-bold text-sm">!</span>
          </div>
          <div>
            <p className="font-medium text-foreground">セットアップを続けましょう</p>
            <p className="text-sm text-muted-foreground">{onboardingMessages[onboardingStep]}</p>
          </div>
        </div>
      )}

      {/* 残高アラート（オンボーディング完了後、残高が少ない場合） */}
      {!loading && onboardingStep === "complete" && showLowCreditAlert && (
        <div className="bg-warning/10 border border-warning/30 rounded-lg p-4 flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-warning/20 flex items-center justify-center">
            <Coins className="h-4 w-4 text-warning" />
          </div>
          <div>
            <p className="font-medium text-foreground">クレジット残高が少なくなっています</p>
            <p className="text-sm text-muted-foreground">クレジットを追加して、引き続きMCPistをご利用ください</p>
          </div>
        </div>
      )}

      <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Connected Services */}
        <HighlightCard
          href="/tools"
          highlight={onboardingStep === "connections"}
        >
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
        </HighlightCard>

        {/* Enabled Tools */}
        <HighlightCard href="/tools">
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
                <div className="text-3xl font-bold">{enabledToolCount}</div>
                <p className="text-xs text-muted-foreground mt-1">{totalToolCount} ツール中</p>
              </>
            )}
          </CardContent>
        </HighlightCard>

        {/* Credit Balance */}
        <HighlightCard
          href="/billing"
          highlight={onboardingStep === "billing" || showLowCreditAlert}
        >
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
        </HighlightCard>

        {/* Usage This Month */}
        <HighlightCard href="/billing">
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
                <div className="text-3xl font-bold">
                  {monthlyUsage?.total_consumed?.toLocaleString() ?? 0}
                </div>
                <p className="text-xs text-muted-foreground mt-1">クレジット消費</p>
              </>
            )}
          </CardContent>
        </HighlightCard>
      </div>
    </div>
  )
}
