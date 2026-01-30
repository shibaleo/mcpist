"use client"

import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Sparkles, ArrowRight } from "lucide-react"

// TODO: プロダクトツアーを実装
// - ステップ1: MCPistとは何か、何ができるかの説明
// - ステップ2: サービス連携 → ツール設定 → AIが使えるようになる流れをアニメーションで視覚的に説明
// - ステップ3: ダッシュボードへの誘導

export default function OnboardingPage() {
  const router = useRouter()

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
            AIアシスタントから外部サービスを操作できるMCPサーバーです
          </p>
        </div>

        {/* プレースホルダー */}
        <div className="bg-card p-8 rounded-xl border border-dashed border-muted-foreground/30 text-muted-foreground">
          <p className="text-sm">
            ここにプロダクトツアーが入ります
          </p>
          <p className="text-xs mt-2">
            Coming Soon...
          </p>
        </div>

        {/* スキップボタン */}
        <Button
          className="w-full h-12"
          onClick={handleSkip}
        >
          ダッシュボードへ
          <ArrowRight className="h-4 w-4 ml-2" />
        </Button>

        <p className="text-xs text-muted-foreground">
          サービス連携やツール設定はダッシュボードから行えます
        </p>
      </div>
    </div>
  )
}
