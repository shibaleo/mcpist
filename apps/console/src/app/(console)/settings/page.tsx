"use client"

import { useTheme } from "next-themes"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  useAppearance,
  accentColors,
  type AccentColorId,
} from "@/lib/appearance-context"
import { Sun, Moon, Monitor, Check, Globe } from "lucide-react"
import { cn } from "@/lib/utils"
import { useEffect, useState } from "react"
import {
  getUserSettings,
  updateUserSettings,
  type Language,
} from "@/lib/user-settings"
import { toast } from "sonner"

// モジュールレベルキャッシュ
let cachedLanguage: Language | null = null

export const dynamic = "force-dynamic"

const languageOptions = [
  { id: "en-US" as Language, name: "English", nativeName: "English" },
  { id: "ja-JP" as Language, name: "Japanese", nativeName: "日本語" },
] as const

export default function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const {
    accentColor,
    setAccentColor,
  } = useAppearance()
  const [mounted, setMounted] = useState(false)
  const [language, setLanguage] = useState<Language>(cachedLanguage ?? "en-US")
  const [savingLanguage, setSavingLanguage] = useState(false)

  useEffect(() => {
    setMounted(true)
    getUserSettings().then((settings) => {
      cachedLanguage = settings.language
      setLanguage(settings.language)
    })
  }, [])

  const handleLanguageChange = async (newLanguage: Language) => {
    if (newLanguage === language) return

    // Optimistic update
    const prevLanguage = language
    setLanguage(newLanguage)
    setSavingLanguage(true)

    try {
      const result = await updateUserSettings({ language: newLanguage })
      if (result.success) {
        cachedLanguage = newLanguage
      } else {
        // Revert on failure
        setLanguage(prevLanguage)
        toast.error(result.error || "保存に失敗しました")
      }
    } catch {
      setLanguage(prevLanguage)
      toast.error("保存に失敗しました")
    } finally {
      setSavingLanguage(false)
    }
  }

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
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">設定</h1>
        <p className="text-muted-foreground mt-1">アプリの設定をカスタマイズします</p>
      </div>

      {/* 言語設定 */}
      <Card className="overflow-hidden">
        <CardHeader>
          <CardTitle className="text-lg flex items-start gap-2">
            <Globe className="h-5 w-5 shrink-0 mt-0.5" />
            <span className="break-all">言語 / Language</span>
          </CardTitle>
          <CardDescription>ツール説明文の言語を選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {languageOptions.map((option) => {
              const isSelected = language === option.id
              return (
                <button
                  key={option.id}
                  onClick={() => handleLanguageChange(option.id)}
                  disabled={savingLanguage}
                  className={cn(
                    "relative flex flex-col items-center gap-1 p-3 rounded-lg border-2 transition-all",
                    isSelected
                      ? "border-success bg-success/10"
                      : "border-border hover:border-muted-foreground",
                    savingLanguage && "opacity-60 pointer-events-none"
                  )}
                >
                  <span
                    className={cn("text-base font-medium", isSelected && "text-success")}
                  >
                    {option.nativeName}
                  </span>
                  <span className="text-xs text-muted-foreground">{option.name}</span>
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

      {/* テーマ設定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">テーマ</CardTitle>
          <CardDescription>ダーク/ライト/システムデフォルトを選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
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
                      : "border-border hover:border-muted-foreground"
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

      {/* アクセントカラー設定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">アクセントカラー</CardTitle>
          <CardDescription>ボタンやリンクの色を選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {accentColors.map((color) => {
              const isSelected = accentColor === color.id
              return (
                <button
                  key={color.id}
                  onClick={() => setAccentColor(color.id)}
                  className={cn(
                    "relative flex flex-col items-center gap-2 p-3 rounded-lg border-2 transition-all",
                    isSelected
                      ? "border-success"
                      : "border-border hover:border-muted-foreground"
                  )}
                >
                  <div
                    className="w-full max-w-10 aspect-square rounded-full mx-auto"
                    style={{ backgroundColor: color.preview || "#888" }}
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
            現在の設定:{" "}
            {languageOptions.find((l) => l.id === language)?.nativeName || "English"} /{" "}
            {themeOptions.find((t) => t.id === theme)?.name || "ダーク"}モード /{" "}
            {accentColors.find((c) => c.id === accentColor)?.name || "グリーン"}アクセント
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
