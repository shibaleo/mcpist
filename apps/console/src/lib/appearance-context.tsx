"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"

// アクセントカラープリセット（落ち着いたトーン）
export const accentColors = [
  { id: "green", name: "グリーン", preview: "#4da872" },
  { id: "blue", name: "ブルー", preview: "#5a8fc8" },
  { id: "purple", name: "パープル", preview: "#a070c0" },
  { id: "pink", name: "ピンク", preview: "#c46a88" },
  { id: "orange", name: "オレンジ", preview: "#d07850" },
  { id: "yellow", name: "イエロー", preview: "#b8a050" },
] as const

export type AccentColorId = (typeof accentColors)[number]["id"]

interface AppearanceContextType {
  accentColor: AccentColorId
  setAccentColor: (id: AccentColorId) => void
}

const AppearanceContext = createContext<AppearanceContextType | undefined>(undefined)

const STORAGE_KEY_ACCENT = "mcpist-accent-color"
const STORAGE_KEY_BG = "mcpist-bg-color" // 削除用

export function AppearanceProvider({ children }: { children: ReactNode }) {
  const [accentColor, setAccentColorState] = useState<AccentColorId>("orange")
  const [mounted, setMounted] = useState(false)

  // 初期化時にローカルストレージから読み込み & 古いキーを削除
  useEffect(() => {
    // 古い背景色設定を削除
    localStorage.removeItem(STORAGE_KEY_BG)
    localStorage.removeItem("mcpist-custom-colors")

    const savedAccent = localStorage.getItem(STORAGE_KEY_ACCENT) as AccentColorId | null
    if (savedAccent && accentColors.some(c => c.id === savedAccent)) {
      setAccentColorState(savedAccent)
    }
    setMounted(true)
  }, [])

  // data属性を設定
  useEffect(() => {
    if (!mounted) return

    // アクセントカラーの設定
    document.documentElement.setAttribute("data-accent-color", accentColor)
  }, [accentColor, mounted])

  const setAccentColor = (id: AccentColorId) => {
    setAccentColorState(id)
    localStorage.setItem(STORAGE_KEY_ACCENT, id)
  }

  return (
    <AppearanceContext.Provider
      value={{
        accentColor,
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
