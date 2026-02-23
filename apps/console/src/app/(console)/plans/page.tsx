"use client"

import { useState, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { getUserContext, type UserContext } from "@/lib/billing/plan"
import { Loader2, Sparkles, Info, CheckCircle, Zap, Crown, ExternalLink } from "lucide-react"
import { toast } from "sonner"

// モジュールレベルキャッシュ
let cachedContext: UserContext | null = null

export const dynamic = "force-dynamic"

const planDisplayNames: Record<string, string> = {
  free: "Free",
  plus: "Plus",
}

export default function PlanPage() {
  const searchParams = useSearchParams()
  const hasCached = cachedContext !== null
  const [context, setContext] = useState<UserContext | null>(cachedContext)
  const [loading, setLoading] = useState(!hasCached)
  const [claiming, setClaiming] = useState(false)
  const [upgrading, setUpgrading] = useState(false)
  const [managingSubscription, setManagingSubscription] = useState(false)

  // Fetch user context
  const fetchContext = async () => {
    setLoading(true)
    try {
      const data = await getUserContext()
      cachedContext = data
      setContext(data)
      return data
    } catch (error) {
      console.error("Failed to fetch context:", error)
      return null
    } finally {
      setLoading(false)
    }
  }

  // Webhook処理完了を待ってからデータを再取得（リトライ付き）
  const waitForPlanUpdate = async (expectedPlan: string, maxRetries = 5) => {
    for (let i = 0; i < maxRetries; i++) {
      await new Promise((r) => setTimeout(r, 2000))
      const data = await fetchContext()
      if (data?.plan_id === expectedPlan) return true
    }
    return false
  }

  // Handle success/cancel from Stripe Checkout
  useEffect(() => {
    const success = searchParams.get("success")
    const canceled = searchParams.get("canceled")

    if (success === "true") {
      window.history.replaceState({}, "", "/plans")
      cachedContext = null
      toast.promise(waitForPlanUpdate("plus"), {
        loading: "プランを更新中...",
        success: "Plusプランが適用されました！",
        error: "プランの反映に時間がかかっています。ページを再読み込みしてください。",
      })
    } else if (canceled === "true") {
      toast.error("アップグレードがキャンセルされました")
      window.history.replaceState({}, "", "/plans")
    }
  }, [searchParams])

  useEffect(() => {
    fetchContext()
  }, [])

  const handleClaimSignupBonus = async () => {
    setClaiming(true)
    try {
      const response = await fetch("/api/credits/grant-signup-bonus", {
        method: "POST",
      })
      const data = await response.json()

      if (data.success) {
        toast.success("アカウントが有効になりました！", {
          description: "MCPistをお楽しみください。",
        })
        await fetchContext()
      } else if (data.error === "already_granted") {
        toast.info("既にアカウントは有効です")
        await fetchContext()
      } else {
        throw new Error(data.message || "Failed to activate")
      }
    } catch (error) {
      console.error("Claim error:", error)
      toast.error("エラーが発生しました", {
        description: "しばらくしてからもう一度お試しください。",
      })
    } finally {
      setClaiming(false)
    }
  }

  const handleUpgrade = async () => {
    setUpgrading(true)
    try {
      const response = await fetch("/api/stripe/checkout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      })

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || "Failed to create checkout session")
      }

      if (data.url) {
        window.location.href = data.url
      }
    } catch (error) {
      console.error("Checkout error:", error)
      toast.error("エラーが発生しました", {
        description: "しばらくしてからもう一度お試しください。",
      })
      setUpgrading(false)
    }
  }

  const handleManageSubscription = async () => {
    setManagingSubscription(true)
    try {
      const response = await fetch("/api/stripe/portal", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      })

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || "Failed to create portal session")
      }

      if (data.url) {
        window.location.href = data.url
      }
    } catch (error) {
      console.error("Portal error:", error)
      toast.error("エラーが発生しました", {
        description: "しばらくしてからもう一度お試しください。",
      })
      setManagingSubscription(false)
    }
  }

  const usagePercent = context ? Math.min(100, (context.daily_used / context.daily_limit) * 100) : 0
  const isNearLimit = context ? context.daily_used >= context.daily_limit * 0.8 : false

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">プラン</h1>
        <p className="text-muted-foreground mt-1">プランと使用量の確認</p>
      </div>

      {/* 現在のプラン */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Zap className="h-5 w-5 text-primary" />
            現在のプラン
          </CardTitle>
          <CardDescription>
            MCPツールの1日あたりの実行回数上限
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="text-muted-foreground">読み込み中...</span>
            </div>
          ) : context ? (
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <Badge variant={context.plan_id === "plus" ? "default" : "secondary"} className="text-base px-3 py-1">
                  {context.plan_id === "plus" && <Crown className="h-4 w-4 mr-1" />}
                  {planDisplayNames[context.plan_id] ?? context.plan_id}
                </Badge>
                <span className="text-muted-foreground text-sm">
                  {context.daily_limit.toLocaleString()} 回/日
                </span>
              </div>

              {/* Daily usage progress */}
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">本日の使用量</span>
                  <span className={isNearLimit ? "text-warning font-medium" : ""}>
                    {context.daily_used.toLocaleString()} / {context.daily_limit.toLocaleString()}
                  </span>
                </div>
                <Progress
                  value={usagePercent}
                  className="h-2"
                />
                <p className="text-xs text-muted-foreground">
                  毎日 UTC 0:00 (JST 9:00) にリセットされます
                </p>
              </div>
            </div>
          ) : (
            <p className="text-muted-foreground">データを取得できませんでした</p>
          )}
        </CardContent>
      </Card>

      {/* 初回アクティベーション（pre_active のみ） */}
      {context?.account_status === "pre_active" && (
        <Card className="animate-pulse-border border-primary shadow-lg shadow-primary/20">
          <CardHeader>
            <CardTitle className="text-lg flex items-start gap-2">
              <Sparkles className="h-5 w-5 text-primary shrink-0 mt-0.5" />
              ようこそ！アカウントを有効にする
            </CardTitle>
            <CardDescription>
              MCPistを始めるためにアカウントを有効化します
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="bg-primary/10 rounded-lg p-4">
                <div className="flex flex-wrap items-center gap-4">
                  <div>
                    <p className="font-medium">Freeプラン</p>
                    <p className="text-sm text-muted-foreground">
                      1日100回のツール実行
                    </p>
                  </div>
                  <div className="ml-auto text-right">
                    <p className="text-2xl font-bold text-primary">無料</p>
                    <p className="text-sm text-muted-foreground">で始められます</p>
                  </div>
                </div>
              </div>
              <Button
                className="w-full"
                size="lg"
                onClick={handleClaimSignupBonus}
                disabled={claiming}
              >
                {claiming ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    処理中...
                  </>
                ) : (
                  <>
                    <Sparkles className="h-4 w-4 mr-2" />
                    アカウントを有効にする
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* アップグレード CTA（free プランの active ユーザーのみ） */}
      {context?.account_status === "active" && context?.plan_id === "free" && (
        <Card className="border-dashed border-2 border-primary/30">
          <CardHeader>
            <CardTitle className="text-lg flex items-start gap-2">
              <Crown className="h-5 w-5 text-primary shrink-0 mt-0.5" />
              Plusプランにアップグレード
            </CardTitle>
            <CardDescription>
              より多くのツール実行回数で生産性を向上
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="bg-muted/50 rounded-lg p-4">
                <div className="flex flex-wrap items-center gap-4">
                  <div>
                    <p className="font-medium">Plusプラン</p>
                    <p className="text-sm text-muted-foreground">
                      1日500回のツール実行
                    </p>
                  </div>
                  <div className="ml-auto text-right">
                    <p className="text-2xl font-bold text-primary">¥980</p>
                    <p className="text-sm text-muted-foreground">/月</p>
                  </div>
                </div>
              </div>
              <Button
                className="w-full"
                size="lg"
                onClick={handleUpgrade}
                disabled={upgrading}
              >
                {upgrading ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    処理中...
                  </>
                ) : (
                  <>
                    <Crown className="h-4 w-4 mr-2" />
                    Plusにアップグレード
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* サブスクリプション管理（plus プランのみ） */}
      {context?.account_status === "active" && context?.plan_id === "plus" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Crown className="h-5 w-5 text-primary" />
              サブスクリプション管理
            </CardTitle>
            <CardDescription>
              お支払い方法の変更、プランの解約はこちらから
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              variant="outline"
              onClick={handleManageSubscription}
              disabled={managingSubscription}
            >
              {managingSubscription ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  処理中...
                </>
              ) : (
                <>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  サブスクリプションを管理
                </>
              )}
            </Button>
          </CardContent>
        </Card>
      )}

      {/* プランについて */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Info className="h-5 w-5 text-primary" />
            プランについて
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2 text-sm text-muted-foreground">
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 mt-0.5 text-success shrink-0" />
              <span>MCPツールの実行ごとに1回としてカウントされます</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 mt-0.5 text-success shrink-0" />
              <span>使用量は毎日 UTC 0:00（JST 9:00）にリセットされます</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 mt-0.5 text-success shrink-0" />
              <span>Plusプランは月額サブスクリプションです</span>
            </li>
          </ul>
        </CardContent>
      </Card>
    </div>
  )
}
