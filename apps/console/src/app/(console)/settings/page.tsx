"use client"

import { useTheme } from "next-themes"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import {
  useAppearance,
  backgroundColors,
  accentColors,
  type BackgroundColorId,
  type AccentColorId,
} from "@/lib/appearance-context"
import { Sun, Moon, Monitor, Check } from "lucide-react"
import { cn } from "@/lib/utils"
import { useEffect, useState } from "react"

export const dynamic = "force-dynamic"

export default function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { backgroundColor, accentColor, setBackgroundColor, setAccentColor } = useAppearance()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) {
    return null
  }

  const themeOptions = [
    { id: "light", name: "ライト", icon: Sun },
    { id: "dark", name: "ダーク", icon: Moon },
    { id: "system", name: "システム", icon: Monitor },
  ] as const

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">設定</h1>
        <p className="text-muted-foreground mt-1">アプリの外見をカスタマイズします</p>
      </div>

      {/* テーマ設定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">テーマ</CardTitle>
          <CardDescription>ダーク/ライト/システムデフォルトを選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-3">
            {themeOptions.map((option) => {
              const Icon = option.icon
              const isSelected = theme === option.id
              return (
                <button
                  key={option.id}
                  onClick={() => setTheme(option.id)}
                  className={cn(
                    "flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all",
                    isSelected
                      ? "border-success bg-success/10"
                      : "border-border hover:border-muted-foreground",
                  )}
                >
                  <Icon className={cn("h-6 w-6", isSelected && "text-success")} />
                  <span className={cn("text-sm font-medium", isSelected && "text-success")}>
                    {option.name}
                  </span>
                </button>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {/* 背景色設定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">背景色</CardTitle>
          <CardDescription>アプリの背景色を選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-3">
            {backgroundColors.map((color) => {
              const isSelected = backgroundColor === color.id
              return (
                <button
                  key={color.id}
                  onClick={() => setBackgroundColor(color.id as BackgroundColorId)}
                  className={cn(
                    "relative flex flex-col items-center gap-2 p-3 rounded-lg border-2 transition-all",
                    isSelected
                      ? "border-success"
                      : "border-border hover:border-muted-foreground",
                  )}
                >
                  <div
                    className="w-10 h-10 rounded-full border border-border"
                    style={{ backgroundColor: color.dark }}
                  />
                  <span className="text-xs font-medium">{color.name}</span>
                  {isSelected && (
                    <div className="absolute top-1 right-1">
                      <Check className="h-4 w-4 text-success" />
                    </div>
                  )}
                </button>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {/* アクセントカラー設定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">アクセントカラー</CardTitle>
          <CardDescription>ボタンやリンクの色を選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-6 gap-3">
            {accentColors.map((color) => {
              const isSelected = accentColor === color.id
              return (
                <button
                  key={color.id}
                  onClick={() => setAccentColor(color.id as AccentColorId)}
                  className={cn(
                    "relative flex flex-col items-center gap-2 p-3 rounded-lg border-2 transition-all",
                    isSelected
                      ? "border-success"
                      : "border-border hover:border-muted-foreground",
                  )}
                >
                  <div
                    className="w-10 h-10 rounded-full"
                    style={{ backgroundColor: color.preview }}
                  />
                  <span className="text-xs font-medium">{color.name}</span>
                  {isSelected && (
                    <div className="absolute top-1 right-1">
                      <Check className="h-4 w-4 text-success" />
                    </div>
                  )}
                </button>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {/* 現在の設定表示 */}
      <Card className="bg-secondary/30">
        <CardContent className="p-4">
          <p className="text-sm text-muted-foreground">
            現在の設定: {themeOptions.find((t) => t.id === theme)?.name || "ダーク"}モード /
            {" "}{backgroundColors.find((c) => c.id === backgroundColor)?.name || "ブラック"}背景 /
            {" "}{accentColors.find((c) => c.id === accentColor)?.name || "グリーン"}アクセント
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
