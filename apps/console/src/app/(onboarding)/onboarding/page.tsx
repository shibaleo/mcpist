"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { ServiceIcon } from "@/components/service-icon"
import { services, getServiceIcon, getServiceDescription } from "@/lib/module-data"
import { getOAuthProviderForService, getOAuthAuthorizationUrl, OAuthAppError, OAUTH_CONFIGS } from "@/lib/oauth-apps"
import {
  Check,
  ArrowRight,
  Sparkles,
  Link2,
  PartyPopper,
  Loader2,
  Gift,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import { useAuth } from "@/lib/auth-context"
import { getMyConnections, type ServiceConnection } from "@/lib/token-vault"

const steps = [
  { id: 1, title: "サービス連携" },
  { id: 2, title: "特典を受け取る" },
  { id: 3, title: "準備完了" },
]

// OAuth対応サービス（OAUTH_CONFIGSから動的に取得）
const oauthServiceIds = new Set(Object.values(OAUTH_CONFIGS).map(c => c.serviceId))
const oauthServices = services.filter((s) => oauthServiceIds.has(s.id))

// APIキー認証サービス（OAuth対応以外のサービス）
const apiKeyServices = services.filter((s) => !oauthServiceIds.has(s.id))

export default function OnboardingPage() {
  const router = useRouter()
  const { user } = useAuth()
  const [currentStep, setCurrentStep] = useState(1)
  const [agreedToTerms, setAgreedToTerms] = useState(false)
  const [grantingCredits, setGrantingCredits] = useState(false)
  const [creditsGranted, setCreditsGranted] = useState(false)
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [connectingService, setConnectingService] = useState<string | null>(null)

  // 接続済みサービスを取得
  useEffect(() => {
    async function loadConnections() {
      if (user) {
        try {
          const data = await getMyConnections()
          setConnections(data)
        } catch (error) {
          console.error("Failed to load connections:", error)
        }
      }
    }
    loadConnections()
  }, [user])

  const connectedServiceIds = new Set(connections.map((c) => c.service))
  const hasAnyConnection = connectedServiceIds.size > 0

  // クレジット付与
  const handleGrantCredits = async () => {
    if (!user || !agreedToTerms) return

    setGrantingCredits(true)
    try {
      const response = await fetch("/api/credits/grant-signup-bonus", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      })

      const data = await response.json()

      if (data.success) {
        setCreditsGranted(true)
        toast.success("100クレジットを受け取りました！")
        setCurrentStep(3)
      } else if (data.error === "already_granted") {
        // 既に付与済みの場合はスキップ
        setCreditsGranted(true)
        setCurrentStep(3)
      } else {
        toast.error(data.message || "クレジットの付与に失敗しました")
      }
    } catch (error) {
      console.error("Failed to grant credits:", error)
      toast.error("クレジットの付与に失敗しました")
    } finally {
      setGrantingCredits(false)
    }
  }

  // OAuth連携
  const handleOAuthConnect = async (serviceId: string) => {
    const providerId = getOAuthProviderForService(serviceId)
    if (!providerId) {
      toast.error("OAuth設定が見つかりません")
      return
    }

    setConnectingService(serviceId)
    try {
      // returnToでオンボーディングに戻る
      const authUrl = await getOAuthAuthorizationUrl(providerId, "/onboarding")
      window.location.href = authUrl
    } catch (error) {
      if (error instanceof OAuthAppError) {
        toast.error(error.message)
      } else {
        toast.error("OAuth認可URLの取得に失敗しました")
      }
      setConnectingService(null)
    }
  }

  const handleNext = () => {
    if (currentStep === 1) {
      setCurrentStep(2)
    } else if (currentStep === 2) {
      handleGrantCredits()
    } else {
      router.push("/dashboard")
    }
  }

  const handleSkip = () => {
    if (currentStep === 1) {
      // サービス連携をスキップして特典受け取りへ
      setCurrentStep(2)
    } else if (currentStep === 2) {
      // 特典受け取りはスキップ不可（利用規約同意必須）
      return
    } else {
      router.push("/dashboard")
    }
  }

  // URLパラメータでステップを復元（OAuth後のリダイレクト用）
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const step = params.get("step")
    if (step) {
      const stepNum = parseInt(step)
      if (stepNum >= 1 && stepNum <= 3) {
        setCurrentStep(stepNum)
      }
    }
  }, [])

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {/* Step 1: サービス連携 */}
        {currentStep === 1 && (
          <div className="space-y-6">
            <div className="text-center">
              <div className="w-16 h-16 rounded-xl bg-primary flex items-center justify-center mx-auto mb-4">
                <Link2 className="h-8 w-8 text-primary-foreground" />
              </div>
              <h1 className="text-2xl font-bold text-foreground">MCPistへようこそ</h1>
              <p className="text-muted-foreground mt-2">
                まずはサービスを連携しましょう
              </p>
            </div>

            {/* OAuth対応サービス（推奨） */}
            <div className="space-y-3">
              <p className="text-sm font-medium text-muted-foreground">
                ワンクリックで連携（推奨）
              </p>
              {oauthServices.map((service) => {
                const isConnected = connectedServiceIds.has(service.id)
                const isConnecting = connectingService === service.id

                return (
                  <Card
                    key={service.id}
                    className={cn(
                      "transition-all",
                      isConnected && "border-green-500/50 bg-green-500/5"
                    )}
                  >
                    <CardContent className="p-4 flex items-center gap-4">
                      <div
                        className={cn(
                          "w-12 h-12 rounded-lg flex items-center justify-center",
                          isConnected ? "bg-green-500/20" : "bg-secondary"
                        )}
                      >
                        <ServiceIcon
                          icon={getServiceIcon(service.id)}
                          className={cn(
                            "h-6 w-6",
                            isConnected ? "text-green-600" : "text-foreground"
                          )}
                        />
                      </div>
                      <div className="flex-1">
                        <p className="font-medium text-foreground">{service.name}</p>
                        <p className="text-sm text-muted-foreground">
                          {getServiceDescription(service, "ja-JP")}
                        </p>
                      </div>
                      {isConnected ? (
                        <div className="flex items-center gap-2 text-green-600">
                          <Check className="h-5 w-5" />
                          <span className="text-sm font-medium">接続済み</span>
                        </div>
                      ) : (
                        <Button
                          onClick={() => handleOAuthConnect(service.id)}
                          disabled={isConnecting}
                        >
                          {isConnecting ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <>
                              <Link2 className="h-4 w-4 mr-2" />
                              連携
                            </>
                          )}
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>

            {/* APIキー認証サービス */}
            <div className="space-y-3">
              <p className="text-sm font-medium text-muted-foreground">
                APIキーで連携（後から設定可能）
              </p>
              <div className="grid grid-cols-3 gap-2">
                {apiKeyServices.map((service) => {
                  const isConnected = connectedServiceIds.has(service.id)

                  return (
                    <div
                      key={service.id}
                      className={cn(
                        "p-3 rounded-lg border text-center",
                        isConnected
                          ? "border-green-500/50 bg-green-500/5"
                          : "border-dashed border-muted-foreground/30"
                      )}
                    >
                      <ServiceIcon
                        icon={getServiceIcon(service.id)}
                        className={cn(
                          "h-6 w-6 mx-auto mb-2",
                          isConnected ? "text-green-600" : "text-muted-foreground"
                        )}
                      />
                      <p className={cn(
                        "text-xs font-medium",
                        isConnected ? "text-foreground" : "text-muted-foreground"
                      )}>
                        {service.name}
                      </p>
                      {isConnected && (
                        <Check className="h-3 w-3 text-green-600 mx-auto mt-1" />
                      )}
                    </div>
                  )
                })}
              </div>
              <p className="text-xs text-muted-foreground text-center">
                これらのサービスはダッシュボードの「サービス & ツール」から設定できます
              </p>
            </div>
          </div>
        )}

        {/* Step 2: 利用規約同意 + クレジット受け取り */}
        {currentStep === 2 && (
          <div className="text-center space-y-6">
            <div className="flex justify-center">
              <div className="w-20 h-20 rounded-2xl bg-primary/20 flex items-center justify-center">
                <Gift className="h-10 w-10 text-primary" />
              </div>
            </div>
            <div>
              <h1 className="text-3xl font-bold text-foreground">特典を受け取る</h1>
              <p className="text-muted-foreground mt-3 text-lg">
                利用規約に同意すると
                <br />
                <span className="text-primary font-semibold">100クレジット</span>をプレゼント
              </p>
            </div>
            <div className="space-y-3 text-left bg-card p-6 rounded-xl border border-border">
              <div className="flex items-start gap-3">
                <Check className="h-5 w-5 text-green-500 mt-0.5" />
                <div>
                  <p className="font-medium text-foreground">すぐに使える100クレジット</p>
                  <p className="text-sm text-muted-foreground">
                    サインアップ特典として無料でプレゼント
                  </p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <Check className="h-5 w-5 text-green-500 mt-0.5" />
                <div>
                  <p className="font-medium text-foreground">追加クレジットも購入可能</p>
                  <p className="text-sm text-muted-foreground">
                    必要に応じていつでもクレジットを追加できます
                  </p>
                </div>
              </div>
            </div>

            {/* 利用規約同意 */}
            <div className="flex items-start gap-3 p-4 bg-secondary/50 rounded-lg text-left">
              <Checkbox
                id="terms"
                checked={agreedToTerms}
                onCheckedChange={(checked) => setAgreedToTerms(checked === true)}
              />
              <label htmlFor="terms" className="text-sm text-muted-foreground cursor-pointer">
                <a href="/terms" target="_blank" className="text-primary hover:underline">
                  利用規約
                </a>
                および
                <a href="/privacy" target="_blank" className="text-primary hover:underline">
                  プライバシーポリシー
                </a>
                に同意し、100クレジットを受け取ります
              </label>
            </div>
          </div>
        )}

        {/* Step 3: 準備完了 */}
        {currentStep === 3 && (
          <div className="text-center space-y-6">
            <div className="flex justify-center">
              <div className="w-20 h-20 rounded-2xl bg-green-500/20 flex items-center justify-center">
                <PartyPopper className="h-10 w-10 text-green-600" />
              </div>
            </div>
            <div>
              <h1 className="text-3xl font-bold text-foreground">準備完了！</h1>
              <p className="text-muted-foreground mt-3 text-lg">
                {hasAnyConnection
                  ? `${connectedServiceIds.size}件のサービスを連携しました`
                  : "いつでもサービスを追加できます"}
              </p>
            </div>
            <div className="bg-card p-6 rounded-xl border border-border text-left space-y-3">
              <p className="font-medium text-foreground">次のステップ</p>
              <div className="space-y-2 text-sm text-muted-foreground">
                <p className="flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-primary/20 text-primary text-xs flex items-center justify-center">1</span>
                  MCPクライアント（Claude Desktop等）で接続設定
                </p>
                <p className="flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-primary/20 text-primary text-xs flex items-center justify-center">2</span>
                  AIアシスタントから外部サービスを操作
                </p>
                <p className="flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-primary/20 text-primary text-xs flex items-center justify-center">3</span>
                  ダッシュボードで使用状況を確認
                </p>
              </div>
            </div>
          </div>
        )}

        {/* ナビゲーションボタン */}
        <div className="mt-8 space-y-4">
          <Button
            className="w-full h-12"
            onClick={handleNext}
            disabled={
              (currentStep === 2 && (!agreedToTerms || grantingCredits)) ||
              connectingService !== null
            }
          >
            {grantingCredits ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                クレジットを付与中...
              </>
            ) : currentStep === 1 ? (
              hasAnyConnection ? (
                <>
                  次へ
                  <ArrowRight className="h-4 w-4 ml-2" />
                </>
              ) : (
                <>
                  スキップして次へ
                  <ArrowRight className="h-4 w-4 ml-2" />
                </>
              )
            ) : currentStep === 2 ? (
              <>
                同意して100クレジットを受け取る
                <ArrowRight className="h-4 w-4 ml-2" />
              </>
            ) : (
              "ダッシュボードへ"
            )}
          </Button>
          {currentStep === 1 && !hasAnyConnection && (
            <p className="text-xs text-muted-foreground text-center">
              サービス連携は後からでも設定できます
            </p>
          )}
        </div>

        {/* ステップインジケーター */}
        <div className="flex justify-center gap-2 mt-8">
          {steps.map((step) => (
            <div
              key={step.id}
              className={cn(
                "w-2 h-2 rounded-full transition-all",
                currentStep === step.id ? "bg-primary w-6" : "bg-muted"
              )}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
