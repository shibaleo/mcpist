"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { ServiceIcon } from "@/components/service-icon"
import { Sparkles, ArrowRight, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"

// TODO: プロダクトツアーを実装
// - ステップ1: MCPistとは何か、何ができるかの説明
// - ステップ2: サービス連携 → ツール設定 → AIが使えるようになる流れをアニメーションで視覚的に説明
// - ステップ3: ダッシュボードへの誘導

// オンボーディングで選択可能なサービス
const selectableServices = [
  {
    id: "google_calendar",
    name: "Google Calendar",
    icon: "google-calendar",
    description: "予定の確認・作成",
  },
  {
    id: "github",
    name: "GitHub",
    icon: "github",
    description: "リポジトリ・Issue管理",
  },
  {
    id: "notion",
    name: "Notion",
    icon: "notion",
    description: "ドキュメント・DB操作",
  },
  {
    id: "jira",
    name: "Jira",
    icon: "jira",
    description: "チケット管理",
  },
  {
    id: "microsoft_todo",
    name: "Microsoft To Do",
    icon: "microsoft-todo",
    description: "タスク管理",
  },
  {
    id: "supabase",
    name: "Supabase",
    icon: "supabase",
    description: "データベース操作",
  },
]

export default function OnboardingPage() {
  const router = useRouter()
  const [selectedServices, setSelectedServices] = useState<string[]>([])
  const [saving, setSaving] = useState(false)

  const toggleService = (serviceId: string) => {
    setSelectedServices((prev) =>
      prev.includes(serviceId)
        ? prev.filter((id) => id !== serviceId)
        : [...prev, serviceId]
    )
  }

  const handleContinue = async () => {
    setSaving(true)
    try {
      // preferences に preferred_services を保存
      const response = await fetch("/api/user/preferences", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ preferred_services: selectedServices }),
      })

      if (!response.ok) {
        throw new Error("Failed to save preferences")
      }

      router.push("/dashboard")
    } catch (error) {
      console.error("Failed to save preferences:", error)
      toast.error("設定の保存に失敗しました")
      // エラーでもダッシュボードに遷移
      router.push("/dashboard")
    } finally {
      setSaving(false)
    }
  }

  const handleSkip = () => {
    router.push("/dashboard")
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-lg text-center space-y-8">
        {/* アイコン */}
        <div className="flex justify-center">
          <div className="w-20 h-20 rounded-2xl bg-primary/20 flex items-center justify-center">
            <Sparkles className="h-10 w-10 text-primary" />
          </div>
        </div>

        {/* タイトル */}
        <div>
          <h1 className="text-3xl font-bold text-foreground">MCPistへようこそ</h1>
          <p className="text-muted-foreground mt-3 text-lg">
            どのサービスを使いますか？
          </p>
          <p className="text-muted-foreground text-sm mt-1">
            選択したサービスが優先的に表示されます
          </p>
        </div>

        {/* サービス選択グリッド */}
        <div className="grid grid-cols-2 gap-3">
          {selectableServices.map((service) => {
            const isSelected = selectedServices.includes(service.id)
            return (
              <div
                key={service.id}
                onClick={() => toggleService(service.id)}
                className={cn(
                  "relative p-4 rounded-xl border-2 cursor-pointer transition-all",
                  "hover:border-primary/50 hover:bg-primary/5",
                  isSelected
                    ? "border-primary bg-primary/10"
                    : "border-border bg-card"
                )}
              >
                {/* チェックマーク */}
                <div className="absolute top-2 right-2">
                  <Checkbox
                    checked={isSelected}
                    onCheckedChange={() => toggleService(service.id)}
                    className="pointer-events-none"
                  />
                </div>

                <div className="flex flex-col items-center gap-2 pt-2">
                  <div
                    className={cn(
                      "w-12 h-12 rounded-xl flex items-center justify-center",
                      isSelected ? "bg-primary/20" : "bg-secondary"
                    )}
                  >
                    <ServiceIcon
                      icon={service.icon}
                      className={cn(
                        "h-6 w-6",
                        isSelected ? "text-primary" : "text-muted-foreground"
                      )}
                    />
                  </div>
                  <div className="text-center">
                    <p
                      className={cn(
                        "font-medium text-sm",
                        isSelected ? "text-foreground" : "text-muted-foreground"
                      )}
                    >
                      {service.name}
                    </p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {service.description}
                    </p>
                  </div>
                </div>
              </div>
            )
          })}
        </div>

        {/* ボタン */}
        <div className="space-y-3">
          <Button
            className="w-full h-12"
            onClick={handleContinue}
            disabled={saving}
          >
            {saving ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                保存中...
              </>
            ) : selectedServices.length > 0 ? (
              <>
                {selectedServices.length}個のサービスを選択して続ける
                <ArrowRight className="h-4 w-4 ml-2" />
              </>
            ) : (
              <>
                ダッシュボードへ
                <ArrowRight className="h-4 w-4 ml-2" />
              </>
            )}
          </Button>

          {selectedServices.length > 0 && (
            <Button
              variant="ghost"
              className="w-full"
              onClick={handleSkip}
              disabled={saving}
            >
              スキップ
            </Button>
          )}
        </div>

        <p className="text-xs text-muted-foreground">
          サービス連携やツール設定はダッシュボードから行えます
        </p>
      </div>
    </div>
  )
}
