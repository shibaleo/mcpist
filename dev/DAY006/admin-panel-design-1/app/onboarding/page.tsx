"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { ServiceIcon } from "@/components/service-icon"
import { services } from "@/lib/data"
import { Check, ArrowRight, Sparkles, LinkIcon, PartyPopper } from "lucide-react"
import { cn } from "@/lib/utils"

const steps = [
  { id: 1, title: "ようこそ" },
  { id: 2, title: "サービス連携" },
  { id: 3, title: "準備完了" },
]

export default function OnboardingPage() {
  const router = useRouter()
  const [currentStep, setCurrentStep] = useState(1)
  const [selectedServices, setSelectedServices] = useState<string[]>([])

  const availableServices = services.filter((s) => s.status !== "no-permission").slice(0, 6)

  const handleServiceToggle = (serviceId: string) => {
    setSelectedServices((prev) =>
      prev.includes(serviceId) ? prev.filter((id) => id !== serviceId) : [...prev, serviceId],
    )
  }

  const handleNext = () => {
    if (currentStep < 3) {
      setCurrentStep(currentStep + 1)
    } else {
      router.push("/dashboard")
    }
  }

  const handleSkip = () => {
    router.push("/dashboard")
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {currentStep === 1 && (
          <div className="text-center space-y-6">
            <div className="flex justify-center">
              <div className="w-20 h-20 rounded-2xl bg-primary flex items-center justify-center">
                <Sparkles className="h-10 w-10 text-primary-foreground" />
              </div>
            </div>
            <div>
              <h1 className="text-3xl font-bold text-foreground">MCPistへようこそ</h1>
              <p className="text-muted-foreground mt-3 text-lg">
                AIアシスタントと外部サービスを連携し、
                <br />
                作業を効率化しましょう
              </p>
            </div>
            <div className="space-y-3 text-left bg-card p-6 rounded-xl border border-border">
              <div className="flex items-start gap-3">
                <Check className="h-5 w-5 text-success mt-0.5" />
                <div>
                  <p className="font-medium text-foreground">複数サービスを一元管理</p>
                  <p className="text-sm text-muted-foreground">GoogleカレンダーやSlackなど、様々なサービスを接続</p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <Check className="h-5 w-5 text-success mt-0.5" />
                <div>
                  <p className="font-medium text-foreground">安全なアクセス制御</p>
                  <p className="text-sm text-muted-foreground">必要な権限のみを付与し、セキュアに運用</p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <Check className="h-5 w-5 text-success mt-0.5" />
                <div>
                  <p className="font-medium text-foreground">チームでの権限管理</p>
                  <p className="text-sm text-muted-foreground">ロールベースで柔軟にアクセス権を設定</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {currentStep === 2 && (
          <div className="space-y-6">
            <div className="text-center">
              <div className="w-16 h-16 rounded-xl bg-secondary flex items-center justify-center mx-auto mb-4">
                <LinkIcon className="h-8 w-8 text-foreground" />
              </div>
              <h1 className="text-2xl font-bold text-foreground">最初のサービスを連携</h1>
              <p className="text-muted-foreground mt-2">よく使うサービスを選択してください（後から追加可能）</p>
            </div>
            <div className="grid grid-cols-2 gap-3">
              {availableServices.map((service) => (
                <Card
                  key={service.id}
                  className={cn(
                    "cursor-pointer transition-all hover:border-primary",
                    selectedServices.includes(service.id) && "border-primary bg-primary/5",
                  )}
                  onClick={() => handleServiceToggle(service.id)}
                >
                  <CardContent className="p-4 flex items-center gap-3">
                    <div
                      className={cn(
                        "w-10 h-10 rounded-lg flex items-center justify-center",
                        selectedServices.includes(service.id) ? "bg-primary" : "bg-secondary",
                      )}
                    >
                      <ServiceIcon
                        icon={service.icon}
                        className={cn(
                          "h-5 w-5",
                          selectedServices.includes(service.id) ? "text-primary-foreground" : "text-foreground",
                        )}
                      />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-foreground truncate">{service.name}</p>
                    </div>
                    {selectedServices.includes(service.id) && <Check className="h-5 w-5 text-primary shrink-0" />}
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        )}

        {currentStep === 3 && (
          <div className="text-center space-y-6">
            <div className="flex justify-center">
              <div className="w-20 h-20 rounded-2xl bg-success/20 flex items-center justify-center">
                <PartyPopper className="h-10 w-10 text-success" />
              </div>
            </div>
            <div>
              <h1 className="text-3xl font-bold text-foreground">準備完了！</h1>
              <p className="text-muted-foreground mt-3 text-lg">
                {selectedServices.length > 0
                  ? `${selectedServices.length}件のサービスを選択しました`
                  : "いつでもサービスを追加できます"}
              </p>
            </div>
            <div className="bg-card p-6 rounded-xl border border-border text-left">
              <p className="text-sm text-muted-foreground">
                ダッシュボードから連携状態の確認、新しいサービスの追加、権限の管理ができます。
              </p>
            </div>
          </div>
        )}

        <div className="mt-8 space-y-4">
          <Button className="w-full h-12" onClick={handleNext}>
            {currentStep === 3 ? (
              "ダッシュボードへ"
            ) : (
              <>
                次へ
                <ArrowRight className="h-4 w-4 ml-2" />
              </>
            )}
          </Button>
          {currentStep < 3 && (
            <button
              onClick={handleSkip}
              className="w-full text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              スキップ
            </button>
          )}
        </div>

        <div className="flex justify-center gap-2 mt-8">
          {steps.map((step) => (
            <div
              key={step.id}
              className={cn(
                "w-2 h-2 rounded-full transition-all",
                currentStep === step.id ? "bg-primary w-6" : "bg-muted",
              )}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
