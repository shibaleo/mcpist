"use client"

import { useState, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { getUserContext, type UserCredits } from "@/lib/credits"
import { Coins, Gift, Loader2, CheckCircle, Sparkles, Info } from "lucide-react"
import { toast } from "sonner"

// モジュールレベルキャッシュ
let cachedCredits: UserCredits | null = null
let cachedAccountStatus: string | null = null

export const dynamic = "force-dynamic"

export default function BillingPage() {
  const searchParams = useSearchParams()
  const hasCached = cachedCredits !== null
  const [credits, setCredits] = useState<UserCredits | null>(cachedCredits)
  const [accountStatus, setAccountStatus] = useState<string | null>(cachedAccountStatus)
  const [loading, setLoading] = useState(!hasCached)
  const [purchasing, setPurchasing] = useState(false)
  const [claiming, setClaiming] = useState(false)

  // Handle success/cancel from Stripe Checkout
  useEffect(() => {
    const success = searchParams.get("success")
    const canceled = searchParams.get("canceled")

    if (success === "true") {
      toast.success("クレジットを取得しました！", {
        description: "100クレジットがアカウントに追加されました。",
      })
      // Clear URL params
      window.history.replaceState({}, "", "/credits")
    } else if (canceled === "true") {
      toast.error("購入がキャンセルされました", {
        description: "クレジットは追加されませんでした。",
      })
      window.history.replaceState({}, "", "/credits")
    }
  }, [searchParams])

  // Fetch user context (account status and credits)
  const fetchUserContext = async () => {
    setLoading(true)
    try {
      const context = await getUserContext()
      const newAccountStatus = context?.account_status ?? null
      const newCredits = context ? {
        free_credits: context.free_credits,
        paid_credits: context.paid_credits,
        updated_at: new Date().toISOString(),
      } : null
      cachedAccountStatus = newAccountStatus
      cachedCredits = newCredits
      setAccountStatus(newAccountStatus)
      setCredits(newCredits)
    } catch (error) {
      console.error("Failed to fetch user context:", error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUserContext()
  }, [searchParams]) // Refetch when returning from Stripe

  const totalCredits = credits ? credits.free_credits + credits.paid_credits : 0

  const handleClaimSignupBonus = async () => {
    setClaiming(true)
    try {
      const response = await fetch("/api/credits/grant-signup-bonus", {
        method: "POST",
      })
      const data = await response.json()

      if (data.success) {
        toast.success("クレジットを受け取りました！", {
          description: "100クレジットがアカウントに追加されました。",
        })
        // Refetch user context
        await fetchUserContext()
      } else if (data.error === "already_granted") {
        toast.info("既にクレジットを受け取っています")
        setAccountStatus("active")
      } else {
        throw new Error(data.message || "Failed to claim bonus")
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

  const handleGetFreeCredits = async () => {
    setPurchasing(true)
    try {
      const response = await fetch("/api/stripe/checkout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      })

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || "Failed to create checkout session")
      }

      // Redirect to Stripe Checkout
      if (data.url) {
        window.location.href = data.url
      }
    } catch (error) {
      console.error("Checkout error:", error)
      toast.error("エラーが発生しました", {
        description: "しばらくしてからもう一度お試しください。",
      })
      setPurchasing(false)
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">クレジット</h1>
        <p className="text-muted-foreground mt-1">クレジット残高の確認と購入</p>
      </div>

      {/* クレジット残高 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Coins className="h-5 w-5 text-primary" />
            クレジット残高
          </CardTitle>
          <CardDescription>
            MCPツールの実行に使用されます
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="text-muted-foreground">読み込み中...</span>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex items-baseline gap-2 flex-wrap">
                <span className="text-4xl font-bold">{totalCredits.toLocaleString()}</span>
                <span className="text-muted-foreground">クレジット</span>
              </div>
              <div className="flex flex-wrap gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="bg-info/20 text-info shrink-0">
                    無料
                  </Badge>
                  <span>{credits?.free_credits.toLocaleString() ?? 0}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="bg-success/20 text-success shrink-0">
                    有料
                  </Badge>
                  <span>{credits?.paid_credits.toLocaleString() ?? 0}</span>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 初回クレジット取得（pre_active のみ） */}
      {accountStatus === "pre_active" && (
        <Card className="animate-pulse-border border-primary shadow-lg shadow-primary/20">
          <CardHeader>
            <CardTitle className="text-lg flex items-start gap-2">
              <Sparkles className="h-5 w-5 text-primary shrink-0 mt-0.5" />
              ようこそ！初回クレジットを受け取る
            </CardTitle>
            <CardDescription>
              MCPistを始めるための100クレジットをプレゼント
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="bg-primary/10 rounded-lg p-4">
                <div className="flex flex-wrap items-center gap-4">
                  <div>
                    <p className="font-medium">スタートボーナス</p>
                    <p className="text-sm text-muted-foreground">
                      今すぐ受け取れます
                    </p>
                  </div>
                  <div className="ml-auto text-right">
                    <p className="text-2xl font-bold text-primary">100</p>
                    <p className="text-sm text-muted-foreground">クレジット</p>
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
                    <Gift className="h-4 w-4 mr-2" />
                    100クレジットを受け取る
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* テスト用クレジット取得（active のみ） */}
      {accountStatus === "active" && (
        <Card className="border-dashed border-2 border-primary/30">
          <CardHeader>
            <CardTitle className="text-lg flex items-start gap-2">
              <Gift className="h-5 w-5 text-primary shrink-0 mt-0.5" />
              テスト用クレジットを取得
            </CardTitle>
            <CardDescription>
              テスト期間中は何度でも取得できます
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="bg-muted/50 rounded-lg p-4">
                <div className="flex flex-wrap items-center gap-4">
                  <div className="flex items-baseline gap-2">
                    <span className="text-4xl font-bold">100</span>
                    <span className="text-muted-foreground">クレジット</span>
                  </div>
                  <div className="ml-auto text-right">
                    <p className="text-2xl font-bold text-primary">無料</p>
                    <p className="text-sm text-muted-foreground">テストクレジット</p>
                  </div>
                </div>
              </div>
              <Button
                className="w-full"
                size="lg"
                onClick={handleGetFreeCredits}
                disabled={purchasing}
              >
                {purchasing ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    処理中...
                  </>
                ) : (
                  <>
                    <Gift className="h-4 w-4 mr-2" />
                    テストクレジットを取得
                  </>
                )}
              </Button>
              <p className="text-xs text-muted-foreground text-center">
                テスト期間中のため支払いは発生しません
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* クレジットの使い方 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Info className="h-5 w-5 text-primary" />
            クレジットについて
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2 text-sm text-muted-foreground">
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 mt-0.5 text-success shrink-0" />
              <span>MCPツールの実行ごとに1クレジットを消費します</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 mt-0.5 text-success shrink-0" />
              <span>無料クレジットが優先的に消費されます</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 mt-0.5 text-success shrink-0" />
              <span>有料クレジットに有効期限はありません</span>
            </li>
          </ul>
        </CardContent>
      </Card>
    </div>
  )
}
