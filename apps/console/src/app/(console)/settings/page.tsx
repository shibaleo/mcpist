"use client"

import { useTheme } from "next-themes"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import {
  useAppearance,
  backgroundColors,
  accentColors,
  type BackgroundColorId,
  type AccentColorId,
} from "@/lib/appearance-context"
import { Sun, Moon, Monitor, Check, Globe, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { useEffect, useState, useTransition } from "react"
import {
  getUserSettings,
  updateUserSettings,
  type Language,
} from "@/lib/user-settings"

export const dynamic = "force-dynamic"

const languageOptions = [
  { id: "en-US" as Language, name: "English", nativeName: "English" },
  { id: "ja-JP" as Language, name: "Japanese", nativeName: "日本語" },
] as const

export default function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { backgroundColor, accentColor, setBackgroundColor, setAccentColor } = useAppearance()
  const [mounted, setMounted] = useState(false)
  const [language, setLanguage] = useState<Language>("en-US")
  const [originalLanguage, setOriginalLanguage] = useState<Language>("en-US")
  const [isPending, startTransition] = useTransition()
  const [saveMessage, setSaveMessage] = useState<{
    type: "success" | "error"
    text: string
  } | null>(null)

  useEffect(() => {
    setMounted(true)
    // Load user settings
    getUserSettings().then((settings) => {
      setLanguage(settings.language)
      setOriginalLanguage(settings.language)
    })
  }, [])

  const hasChanges = language !== originalLanguage

  const handleSave = () => {
    startTransition(async () => {
      const result = await updateUserSettings({ language })
      if (result.success) {
        setOriginalLanguage(language)
        setSaveMessage({ type: "success", text: "設定を保存しました" })
      } else {
        setSaveMessage({ type: "error", text: result.error || "保存に失敗しました" })
      }
      // Clear message after 3 seconds
      setTimeout(() => setSaveMessage(null), 3000)
    })
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">設定</h1>
          <p className="text-muted-foreground mt-1">アプリの設定をカスタマイズします</p>
        </div>
        <div className="flex items-center gap-3">
          {saveMessage && (
            <span
              className={cn(
                "text-sm",
                saveMessage.type === "success" ? "text-success" : "text-destructive"
              )}
            >
              {saveMessage.text}
            </span>
          )}
          <Button onClick={handleSave} disabled={!hasChanges || isPending}>
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            保存
          </Button>
        </div>
      </div>

      {/* 言語設定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Globe className="h-5 w-5" />
            言語 / Language
          </CardTitle>
          <CardDescription>MCP Serverのツール説明文の言語を選択</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-3 max-w-md">
            {languageOptions.map((option) => {
              const isSelected = language === option.id
              return (
                <button
                  key={option.id}
                  onClick={() => setLanguage(option.id)}
                  className={cn(
                    "relative flex flex-col items-center gap-1 p-4 rounded-lg border-2 transition-all",
                    isSelected
                      ? "border-success bg-success/10"
                      : "border-border hover:border-muted-foreground"
                  )}
                >
                  <span
                    className={cn("text-lg font-medium", isSelected && "text-success")}
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
                      : "border-border hover:border-muted-foreground"
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
                      : "border-border hover:border-muted-foreground"
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
            現在の設定:{" "}
            {languageOptions.find((l) => l.id === language)?.nativeName || "English"} /{" "}
            {themeOptions.find((t) => t.id === theme)?.name || "ダーク"}モード /{" "}
            {backgroundColors.find((c) => c.id === backgroundColor)?.name || "ブラック"}背景 /{" "}
            {accentColors.find((c) => c.id === accentColor)?.name || "グリーン"}アクセント
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
