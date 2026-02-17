"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth/auth-context"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Cable, Zap, Receipt, Loader2, Settings2, ChevronRight, Calendar } from "lucide-react"
import { getUserContext, getServiceConnections, getMyUsage, type UserContext, type ServiceConnection, type UsageStats } from "@/lib/billing/plan"
import { getMyToolSettings, type ToolSetting } from "@/lib/mcp/tool-settings"
import Link from "next/link"
import { cn } from "@/lib/utils"

// ダッシュボードデータのモジュールレベルキャッシュ
let cachedContext: UserContext | null = null
let cachedConnections: ServiceConnection[] | null = null
let cachedToolSettings: ToolSetting[] | null = null
let cachedUsage: UsageStats | null = null

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
  // 2. 初回アクティベーション未完了（pre_active）
  if (accountStatus === "pre_active") {
    return "billing"
  }
  // 全て完了
  return "complete"
}

// 日付をYYYY-MM-DD形式に変換（ローカルタイムゾーン）
function formatDateForInput(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 文字列から日付を安全にパース
function parseDateFromInput(value: string, fallback: Date): Date {
  if (!value) return fallback
  const parsed = new Date(value + 'T00:00:00')
  return isNaN(parsed.getTime()) ? fallback : parsed
}

// 今月の開始日を取得
function getStartOfMonth(): Date {
  const now = new Date()
  return new Date(now.getFullYear(), now.getMonth(), 1)
}

// 今月の終了日（翌月1日）を取得
function getEndOfMonth(): Date {
  const now = new Date()
  return new Date(now.getFullYear(), now.getMonth() + 1, 1)
}

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
  const hasCachedData = cachedConnections !== null
  const [context, setContext] = useState<UserContext | null>(cachedContext)
  const [connections, setConnections] = useState<ServiceConnection[]>(cachedConnections ?? [])
  const [toolSettings, setToolSettings] = useState<ToolSetting[]>(cachedToolSettings ?? [])
  const [usage, setUsage] = useState<UsageStats | null>(cachedUsage)
  const [usageLoading, setUsageLoading] = useState(!cachedUsage)
  const [loading, setLoading] = useState(!hasCachedData)

  // 期間選択 (デフォルト: 今月)
  const [usageStartDate, setUsageStartDate] = useState<Date>(getStartOfMonth())
  const [usageEndDate, setUsageEndDate] = useState<Date>(getEndOfMonth())

  // 初期データ取得
  useEffect(() => {
    async function fetchData() {
      setLoading(true)
      try {
        const [contextData, connectionsData, toolSettingsData] = await Promise.all([
          getUserContext(),
          getServiceConnections(),
          getMyToolSettings(),
        ])
        cachedContext = contextData
        cachedConnections = connectionsData
        cachedToolSettings = toolSettingsData
        setContext(contextData)
        setConnections(connectionsData)
        setToolSettings(toolSettingsData)
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  // 利用量データ取得 (期間変更時)
  useEffect(() => {
    async function fetchUsage() {
      setUsageLoading(true)
      try {
        const usageData = await getMyUsage(usageStartDate, usageEndDate)
        cachedUsage = usageData
        setUsage(usageData)
      } catch (error) {
        console.error('Failed to fetch usage data:', error)
      } finally {
        setUsageLoading(false)
      }
    }

    fetchUsage()
  }, [usageStartDate, usageEndDate])

  const enabledToolCount = toolSettings.filter((t) => t.enabled).length
  const totalToolCount = toolSettings.length
  const onboardingStep = getOnboardingStep(connections, context?.account_status ?? null)

  // 使用量アラート（active ユーザーで80%以上使用）
  const isNearLimit = context?.account_status === "active" && context.daily_used >= context.daily_limit * 0.8

  // オンボーディングメッセージ
  const onboardingMessages: Record<OnboardingStep, string> = {
    connections: "まずはサービスを連携しましょう",
    billing: "アカウントを有効にしましょう",
    complete: "",
  }

  return (
    <div className="p-6 md:p-8 space-y-6">
      <div className="pl-8 md:pl-0">
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

      {/* 使用量アラート（オンボーディング完了後、使用量が多い場合） */}
      {!loading && onboardingStep === "complete" && isNearLimit && (
        <div className="bg-warning/10 border border-warning/30 rounded-lg p-4 flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-warning/20 flex items-center justify-center">
            <Zap className="h-4 w-4 text-warning" />
          </div>
          <div>
            <p className="font-medium text-foreground">本日の使用量が上限に近づいています</p>
            <p className="text-sm text-muted-foreground">プランのアップグレードをご検討ください</p>
          </div>
        </div>
      )}

      <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Connected Services */}
        <HighlightCard
          href="/services"
          highlight={!loading && onboardingStep === "connections"}
        >
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Cable className="h-4 w-4" />
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

        {/* Daily Usage / Plan */}
        <HighlightCard
          href="/plans"
          highlight={!loading && (onboardingStep === "billing" || isNearLimit)}
        >
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Zap className="h-4 w-4" />
              本日の使用量
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : context ? (
              <>
                <div className="text-3xl font-bold">{context.daily_used.toLocaleString()}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  / {context.daily_limit.toLocaleString()} 回
                </p>
              </>
            ) : (
              <>
                <div className="text-3xl font-bold">-</div>
                <p className="text-xs text-muted-foreground mt-1">データなし</p>
              </>
            )}
          </CardContent>
        </HighlightCard>

        {/* Usage with Date Range */}
        <Card className="relative">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Receipt className="h-4 w-4" />
              期間利用量
            </CardTitle>
          </CardHeader>
          <CardContent>
            {usageLoading ? (
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            ) : (
              <>
                <div className="text-3xl font-bold">
                  {usage?.total_used?.toLocaleString() ?? 0}
                </div>
                <p className="text-xs text-muted-foreground mt-1">回実行</p>
              </>
            )}
            {/* 期間選択 */}
            <div className="mt-4 pt-3 border-t border-border">
              <div className="flex items-center gap-1 text-xs text-muted-foreground mb-2">
                <Calendar className="h-3 w-3" />
                期間
              </div>
              <div className="flex gap-2">
                <input
                  type="date"
                  value={formatDateForInput(usageStartDate)}
                  onChange={(e) => setUsageStartDate(parseDateFromInput(e.target.value, usageStartDate))}
                  className="flex-1 px-2 py-1 text-xs rounded border border-input bg-background"
                />
                <span className="text-muted-foreground self-center">〜</span>
                <input
                  type="date"
                  value={formatDateForInput(usageEndDate)}
                  onChange={(e) => setUsageEndDate(parseDateFromInput(e.target.value, usageEndDate))}
                  className="flex-1 px-2 py-1 text-xs rounded border border-input bg-background"
                />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
