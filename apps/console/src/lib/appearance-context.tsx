"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"

// 背景色プリセット
export const backgroundColors = [
  { id: "black", name: "ブラック", preview: "#1a1a1a" },
  { id: "slate", name: "スレート", preview: "#1e293b" },
  { id: "zinc", name: "ジンク", preview: "#27272a" },
  { id: "custom", name: "カスタム", preview: null }, // カスタム色用
] as const

// アクセントカラープリセット
export const accentColors = [
  { id: "green", name: "グリーン", preview: "#22c55e" },
  { id: "blue", name: "ブルー", preview: "#3b82f6" },
  { id: "purple", name: "パープル", preview: "#a855f7" },
  { id: "pink", name: "ピンク", preview: "#ec4899" },
  { id: "orange", name: "オレンジ", preview: "#f97316" },
  { id: "yellow", name: "イエロー", preview: "#eab308" },
  { id: "custom", name: "カスタム", preview: null }, // カスタム色用
] as const

export type BackgroundColorId = (typeof backgroundColors)[number]["id"]
export type AccentColorId = (typeof accentColors)[number]["id"]

// カスタム色の型
export interface CustomColors {
  bgLight?: string
  bgDark?: string
  accent?: string
}

interface AppearanceContextType {
  backgroundColor: BackgroundColorId
  accentColor: AccentColorId
  customColors: CustomColors
  setBackgroundColor: (id: BackgroundColorId) => void
  setAccentColor: (id: AccentColorId) => void
  setCustomColors: (colors: Partial<CustomColors>) => void
}

const AppearanceContext = createContext<AppearanceContextType | undefined>(undefined)

const STORAGE_KEY_BG = "mcpist-bg-color"
const STORAGE_KEY_ACCENT = "mcpist-accent-color"
const STORAGE_KEY_CUSTOM = "mcpist-custom-colors"

export function AppearanceProvider({ children }: { children: ReactNode }) {
  const [backgroundColor, setBackgroundColorState] = useState<BackgroundColorId>("black")
  const [accentColor, setAccentColorState] = useState<AccentColorId>("green")
  const [customColors, setCustomColorsState] = useState<CustomColors>({})
  const [mounted, setMounted] = useState(false)

  // 初期化時にローカルストレージから読み込み
  useEffect(() => {
    const savedBg = localStorage.getItem(STORAGE_KEY_BG) as BackgroundColorId | null
    const savedAccent = localStorage.getItem(STORAGE_KEY_ACCENT) as AccentColorId | null
    const savedCustom = localStorage.getItem(STORAGE_KEY_CUSTOM)

    if (savedBg && backgroundColors.some(c => c.id === savedBg)) {
      setBackgroundColorState(savedBg)
    }
    if (savedAccent && accentColors.some(c => c.id === savedAccent)) {
      setAccentColorState(savedAccent)
    }
    if (savedCustom) {
      try {
        setCustomColorsState(JSON.parse(savedCustom))
      } catch {
        // ignore parse error
      }
    }
    setMounted(true)
  }, [])

  // data属性を設定
  useEffect(() => {
    if (!mounted) return

    // 背景色の設定
    document.documentElement.setAttribute("data-bg-color", backgroundColor)

    // アクセントカラーの設定
    document.documentElement.setAttribute("data-accent-color", accentColor)

    // カスタム色の設定（カスタムが選択されている場合のみ）
    if (backgroundColor === "custom" && customColors.bgLight) {
      document.documentElement.style.setProperty("--custom-bg-light", customColors.bgLight)
    }
    if (backgroundColor === "custom" && customColors.bgDark) {
      document.documentElement.style.setProperty("--custom-bg-dark", customColors.bgDark)
    }
    if (accentColor === "custom" && customColors.accent) {
      document.documentElement.style.setProperty("--custom-accent", customColors.accent)
    }
  }, [backgroundColor, accentColor, customColors, mounted])

  const setBackgroundColor = (id: BackgroundColorId) => {
    setBackgroundColorState(id)
    localStorage.setItem(STORAGE_KEY_BG, id)
  }

  const setAccentColor = (id: AccentColorId) => {
    setAccentColorState(id)
    localStorage.setItem(STORAGE_KEY_ACCENT, id)
  }

  const setCustomColors = (colors: Partial<CustomColors>) => {
    const updated = { ...customColors, ...colors }
    setCustomColorsState(updated)
    localStorage.setItem(STORAGE_KEY_CUSTOM, JSON.stringify(updated))
  }

  return (
    <AppearanceContext.Provider
      value={{
        backgroundColor,
        accentColor,
        customColors,
        setBackgroundColor,
        setAccentColor,
        setCustomColors,
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
