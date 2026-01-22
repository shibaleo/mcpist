"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"

// 背景色オプション
export const backgroundColors = [
  { id: "black", name: "ブラック", light: "oklch(1 0 0)", dark: "oklch(0.145 0 0)" },
  { id: "slate", name: "スレート", light: "oklch(0.985 0.002 247)", dark: "oklch(0.178 0.014 256)" },
  { id: "zinc", name: "ジンク", light: "oklch(0.985 0 0)", dark: "oklch(0.18 0 0)" },
  { id: "stone", name: "ストーン", light: "oklch(0.985 0.002 75)", dark: "oklch(0.178 0.006 60)" },
] as const

// アクセントカラーオプション
export const accentColors = [
  { id: "green", name: "グリーン", value: "oklch(0.627 0.194 149.214)", preview: "#22c55e" },
  { id: "blue", name: "ブルー", value: "oklch(0.623 0.214 259.815)", preview: "#3b82f6" },
  { id: "purple", name: "パープル", value: "oklch(0.627 0.265 303.9)", preview: "#a855f7" },
  { id: "pink", name: "ピンク", value: "oklch(0.656 0.241 354.308)", preview: "#ec4899" },
  { id: "orange", name: "オレンジ", value: "oklch(0.705 0.213 47.604)", preview: "#f97316" },
  { id: "yellow", name: "イエロー", value: "oklch(0.795 0.184 86.047)", preview: "#eab308" },
] as const

export type BackgroundColorId = (typeof backgroundColors)[number]["id"]
export type AccentColorId = (typeof accentColors)[number]["id"]

interface AppearanceContextType {
  backgroundColor: BackgroundColorId
  accentColor: AccentColorId
  setBackgroundColor: (id: BackgroundColorId) => void
  setAccentColor: (id: AccentColorId) => void
}

const AppearanceContext = createContext<AppearanceContextType | undefined>(undefined)

const STORAGE_KEY_BG = "mcpist-bg-color"
const STORAGE_KEY_ACCENT = "mcpist-accent-color"

export function AppearanceProvider({ children }: { children: ReactNode }) {
  const [backgroundColor, setBackgroundColorState] = useState<BackgroundColorId>("black")
  const [accentColor, setAccentColorState] = useState<AccentColorId>("green")
  const [mounted, setMounted] = useState(false)

  // 初期化時にローカルストレージから読み込み
  useEffect(() => {
    const savedBg = localStorage.getItem(STORAGE_KEY_BG) as BackgroundColorId | null
    const savedAccent = localStorage.getItem(STORAGE_KEY_ACCENT) as AccentColorId | null

    if (savedBg && backgroundColors.some(c => c.id === savedBg)) {
      setBackgroundColorState(savedBg)
    }
    if (savedAccent && accentColors.some(c => c.id === savedAccent)) {
      setAccentColorState(savedAccent)
    }
    setMounted(true)
  }, [])

  // CSSカスタムプロパティを更新
  useEffect(() => {
    if (!mounted) return

    const bgColor = backgroundColors.find(c => c.id === backgroundColor)
    const accent = accentColors.find(c => c.id === accentColor)

    if (bgColor) {
      document.documentElement.style.setProperty("--background-custom", bgColor.dark)
      document.documentElement.style.setProperty("--background-custom-light", bgColor.light)
    }

    if (accent) {
      document.documentElement.style.setProperty("--accent-custom", accent.value)
    }
  }, [backgroundColor, accentColor, mounted])

  const setBackgroundColor = (id: BackgroundColorId) => {
    setBackgroundColorState(id)
    localStorage.setItem(STORAGE_KEY_BG, id)
  }

  const setAccentColor = (id: AccentColorId) => {
    setAccentColorState(id)
    localStorage.setItem(STORAGE_KEY_ACCENT, id)
  }

  return (
    <AppearanceContext.Provider
      value={{
        backgroundColor,
        accentColor,
        setBackgroundColor,
        setAccentColor,
      }}
    >
      {children}
    </AppearanceContext.Provider>
  )
}

export function useAppearance() {
  const context = useContext(AppearanceContext)
  if (!context) {
    throw new Error("useAppearance must be used within an AppearanceProvider")
  }
  return context
}
