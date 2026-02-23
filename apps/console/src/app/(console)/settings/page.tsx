"use client"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Check, Globe, User } from "lucide-react"
import { cn } from "@/lib/utils"
import { useEffect, useState, useRef } from "react"
import { useAuth } from "@/lib/auth/auth-context"
import {
  getUserSettings,
  updateUserSettings,
  type Language,
  type UserSettings,
} from "@/lib/settings/user-settings"
import { toast } from "sonner"

// モジュールレベルキャッシュ
let cachedLanguage: Language | null = null
let cachedDisplayName: string | null = null

export const dynamic = "force-dynamic"

const languageOptions = [
  { id: "en-US" as Language, name: "English", nativeName: "English" },
  { id: "ja-JP" as Language, name: "Japanese", nativeName: "日本語" },
] as const

export default function SettingsPage() {
  const { user, updateName } = useAuth()
  const [mounted, setMounted] = useState(false)
  const [language, setLanguage] = useState<Language>(cachedLanguage ?? "en-US")
  const [savingLanguage, setSavingLanguage] = useState(false)
  const [displayName, setDisplayName] = useState(cachedDisplayName ?? "")
  const [savingName, setSavingName] = useState(false)
  const nameTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    setMounted(true)
    getUserSettings().then((settings) => {
      cachedLanguage = settings.language
      setLanguage(settings.language)
      cachedDisplayName = settings.display_name
      setDisplayName(settings.display_name)
    })
  }, [])

  const handleLanguageChange = async (newLanguage: Language) => {
    if (newLanguage === language) return

    const prevLanguage = language
    setLanguage(newLanguage)
    setSavingLanguage(true)

    try {
      const result = await updateUserSettings({ language: newLanguage })
      if (result.success) {
        cachedLanguage = newLanguage
      } else {
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

  const handleDisplayNameChange = (value: string) => {
    setDisplayName(value)
    if (nameTimerRef.current) clearTimeout(nameTimerRef.current)
    nameTimerRef.current = setTimeout(async () => {
      const trimmed = value.trim()
      if (!trimmed || trimmed === cachedDisplayName) return
      setSavingName(true)
      try {
        const result = await updateUserSettings({ display_name: trimmed } as Partial<UserSettings>)
        if (result.success) {
          cachedDisplayName = trimmed
          updateName(trimmed)
        } else {
          toast.error(result.error || "保存に失敗しました")
        }
      } catch {
        toast.error("保存に失敗しました")
      } finally {
        setSavingName(false)
      }
    }, 500)
  }

  if (!mounted) {
    return null
  }

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">設定</h1>
        <p className="text-muted-foreground mt-1">アプリの設定をカスタマイズします</p>
      </div>

      {/* 表示名 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-start gap-2">
            <User className="h-5 w-5 shrink-0 mt-0.5" />
            表示名
          </CardTitle>
          <CardDescription>サイドバーに表示される名前</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Input
              value={displayName}
              onChange={(e) => handleDisplayNameChange(e.target.value)}
              placeholder={user?.name || "名前を入力"}
              className="max-w-sm"
            />
            {savingName && (
              <span className="text-xs text-muted-foreground">保存中...</span>
            )}
          </div>
        </CardContent>
      </Card>

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

      {/* 現在の設定表示 */}
      <Card className="bg-secondary/30">
        <CardContent className="p-4">
          <p className="text-sm text-muted-foreground">
            現在の設定:{" "}
            {languageOptions.find((l) => l.id === language)?.nativeName || "English"}
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
