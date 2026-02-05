"use client"

import { useState, useEffect, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"
import { ModuleIcon } from "@/components/module-icon"
import { useAuth } from "@/lib/auth-context"
import { useAppearance, accentColors } from "@/lib/appearance-context"
import {
  modules,
  getModuleIcon,
  isDefaultEnabled,
  isDangerous,
  getModuleDescription,
  getToolDescription,
} from "@/lib/module-data"
import {
  Check,
  Link2,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  X,
} from "lucide-react"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import {
  getMyConnections,
  type ServiceConnection,
  TokenVaultError,
} from "@/lib/token-vault"
import {
  getMyToolSettings,
  toToolSettingsMap,
  saveModuleToolSettings,
  getMyModuleDescriptions,
  toModuleDescriptionsMap,
  updateModuleDescription,
  ToolSettingsError,
  type ToolSettingsMap,
  type ModuleDescriptionsMap,
} from "@/lib/tool-settings"
import { getUserSettings, type Language } from "@/lib/user-settings"

// モジュールレベルキャッシュ
let cachedToolSettings: ToolSettingsMap | null = null
let cachedConnections: ServiceConnection[] | null = null
let cachedModuleDescriptions: ModuleDescriptionsMap | null = null
let cachedLanguage: Language | null = null

export const dynamic = "force-dynamic"

export default function ToolsPage() {
  const { user } = useAuth()
  const { accentColor } = useAppearance()
  const accentPreview = accentColors.find((c) => c.id === accentColor)?.preview ?? "#22c55e"

  // Tool settings state (loaded from DB)
  const hasCached = cachedToolSettings !== null
  const [toolSettings, setToolSettings] = useState<ToolSettingsMap>(cachedToolSettings ?? {})
  const [localToolSettings, setLocalToolSettings] = useState<ToolSettingsMap>(cachedToolSettings ?? {})
  const [connections, setConnections] = useState<ServiceConnection[]>(cachedConnections ?? [])
  const [loading, setLoading] = useState(!hasCached)
  const [selectedModuleId, setSelectedModuleId] = useState<string | null>(null)

  // Module description state
  const [moduleDescriptions, setModuleDescriptions] = useState<ModuleDescriptionsMap>(cachedModuleDescriptions ?? {})
  const [editingModuleId, setEditingModuleId] = useState<string | null>(null)
  const [editingDescription, setEditingDescription] = useState("")
  const [savingDescription, setSavingDescription] = useState(false)

  // Language setting
  const [language, setLanguage] = useState<Language>(cachedLanguage ?? "ja-JP")

  // 接続済みサービスを取得
  const loadConnections = useCallback(async () => {
    try {
      const data = await getMyConnections()
      cachedConnections = data
      setConnections(data)
    } catch (error) {
      if (error instanceof TokenVaultError) {
        console.error("Failed to load connections:", error.message)
      }
    }
  }, [])

  // ツール設定を取得
  const loadToolSettings = useCallback(async () => {
    try {
      const [settings, descriptions, userSettings] = await Promise.all([
        getMyToolSettings(),
        getMyModuleDescriptions(),
        getUserSettings(),
      ])
      const settingsMap = toToolSettingsMap(settings)
      const descriptionsMap = toModuleDescriptionsMap(descriptions)
      cachedToolSettings = settingsMap
      cachedModuleDescriptions = descriptionsMap
      cachedLanguage = userSettings.language
      setToolSettings(settingsMap)
      setLocalToolSettings(settingsMap)
      setModuleDescriptions(descriptionsMap)
      setLanguage(userSettings.language)
    } catch (error) {
      if (error instanceof ToolSettingsError) {
        console.error("Failed to load tool settings:", error.message)
      }
    }
  }, [])

  useEffect(() => {
    async function loadData() {
      if (user) {
        await Promise.all([loadConnections(), loadToolSettings()])
      }
      setLoading(false)
    }
    loadData()
  }, [user, loadConnections, loadToolSettings])

  // モジュールのローカル設定を取得（DB設定がなければデフォルト値を使用）
  const getModuleToolSettings = useCallback(
    (moduleId: string): Record<string, boolean> => {
      const mod = modules.find((m) => m.id === moduleId)
      if (!mod) return {}

      // ローカル設定があればそれを使用
      if (localToolSettings[moduleId]) {
        return localToolSettings[moduleId]
      }

      // なければデフォルト値を構築
      const defaults: Record<string, boolean> = {}
      mod.tools.forEach((t) => {
        defaults[t.id] = isDefaultEnabled(t)
      })
      return defaults
    },
    [localToolSettings]
  )

  // 接続済みモジュールのIDセット
  const connectedModuleIds = new Set(connections.map((c) => c.module))

  // 接続済みモジュールのみフィルタ
  const connectedModules = modules.filter((m) => connectedModuleIds.has(m.id))

  // 選択中のモジュール情報
  const selectedModule = modules.find((m) => m.id === selectedModuleId)

  // 初回ロード時に最初の接続済みモジュールを選択
  useEffect(() => {
    if (!loading && !selectedModuleId && connectedModules.length > 0) {
      setSelectedModuleId(connectedModules[0].id)
    }
  }, [loading, selectedModuleId, connectedModules])

  // モジュール切り替え時に編集状態をリセット
  useEffect(() => {
    setEditingModuleId(null)
    setEditingDescription("")
  }, [selectedModuleId])

  // ツール設定関連（楽観的更新パターン）
  const handleToggleTool = async (moduleId: string, toolId: string) => {
    const current = getModuleToolSettings(moduleId)
    const newValue = !current[toolId]
    const newSettings = {
      ...current,
      [toolId]: newValue,
    }

    // Optimistic update - 先にUIを更新
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: newSettings,
    }))

    try {
      // サーバーに保存
      await saveModuleToolSettings(moduleId, newSettings)
      // 保存成功後、DBの値も更新
      setToolSettings((prev) => ({
        ...prev,
        [moduleId]: newSettings,
      }))
    } catch (error) {
      // Revert on failure - 失敗した場合は元に戻す
      setLocalToolSettings((prev) => ({
        ...prev,
        [moduleId]: current,
      }))
      if (error instanceof ToolSettingsError) {
        toast.error(`保存に失敗しました: ${error.message}`)
      } else {
        toast.error("ツール設定の保存に失敗しました")
      }
    }
  }

  const handleSelectAll = async (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    const current = getModuleToolSettings(moduleId)
    const allEnabled: Record<string, boolean> = {}
    mod.tools.forEach((t) => {
      allEnabled[t.id] = true
    })

    // Optimistic update
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: allEnabled,
    }))

    try {
      await saveModuleToolSettings(moduleId, allEnabled)
      setToolSettings((prev) => ({
        ...prev,
        [moduleId]: allEnabled,
      }))
    } catch (error) {
      // Revert on failure
      setLocalToolSettings((prev) => ({
        ...prev,
        [moduleId]: current,
      }))
      toast.error("ツール設定の保存に失敗しました")
    }
  }

  const handleDeselectAll = async (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    const current = getModuleToolSettings(moduleId)
    const allDisabled: Record<string, boolean> = {}
    mod.tools.forEach((t) => {
      allDisabled[t.id] = false
    })

    // Optimistic update
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: allDisabled,
    }))

    try {
      await saveModuleToolSettings(moduleId, allDisabled)
      setToolSettings((prev) => ({
        ...prev,
        [moduleId]: allDisabled,
      }))
    } catch (error) {
      // Revert on failure
      setLocalToolSettings((prev) => ({
        ...prev,
        [moduleId]: current,
      }))
      toast.error("ツール設定の保存に失敗しました")
    }
  }

  const handleSelectDefault = async (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    const current = getModuleToolSettings(moduleId)
    const defaults: Record<string, boolean> = {}
    mod.tools.forEach((t) => {
      defaults[t.id] = isDefaultEnabled(t)
    })

    // Optimistic update
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: defaults,
    }))

    try {
      await saveModuleToolSettings(moduleId, defaults)
      setToolSettings((prev) => ({
        ...prev,
        [moduleId]: defaults,
      }))
    } catch (error) {
      // Revert on failure
      setLocalToolSettings((prev) => ({
        ...prev,
        [moduleId]: current,
      }))
      toast.error("ツール設定の保存に失敗しました")
    }
  }

  const isToolEnabled = (moduleId: string, toolId: string) => {
    const settings = getModuleToolSettings(moduleId)
    return settings[toolId] ?? false
  }

  const getEnabledToolCount = (moduleId: string): number => {
    const settings = getModuleToolSettings(moduleId)
    return Object.values(settings).filter(Boolean).length
  }

  // モジュール説明関連（ユーザーが設定したカスタム説明）
  const getUserModuleDescription = (moduleId: string): string | undefined => {
    return moduleDescriptions[moduleId]
  }

  const handleCancelEdit = () => {
    setEditingModuleId(null)
    setEditingDescription("")
  }

  const handleSaveModuleDescription = async (moduleId: string) => {
    setSavingDescription(true)
    try {
      const description = editingDescription.trim()
      await updateModuleDescription(moduleId, description)
      // ローカル状態を更新
      setModuleDescriptions((prev) => {
        const newMap = { ...prev }
        if (description) {
          newMap[moduleId] = description
        } else {
          delete newMap[moduleId]
        }
        return newMap
      })
      setEditingModuleId(null)
      setEditingDescription("")
      toast.success("モジュール説明を保存しました")
    } catch (error) {
      if (error instanceof ToolSettingsError) {
        toast.error(`保存に失敗しました: ${error.message}`)
      } else {
        toast.error("保存に失敗しました")
      }
    } finally {
      setSavingDescription(false)
    }
  }

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div className="pl-8 md:pl-0">
          <h1 className="text-2xl font-bold text-foreground">ツール設定</h1>
          <p className="text-muted-foreground mt-1">接続済みサービスのツールを管理します</p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  // 接続済みサービスがない場合
  if (connectedModules.length === 0) {
    return (
      <div className="p-6 space-y-6">
        <div className="pl-8 md:pl-0">
          <h1 className="text-2xl font-bold text-foreground">ツール設定</h1>
          <p className="text-muted-foreground mt-1">接続済みサービスのツールを管理します</p>
        </div>
        <Card>
          <CardContent className="py-12">
            <div className="text-center text-muted-foreground">
              <Link2 className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <h3 className="text-lg font-semibold mb-2">接続済みサービスがありません</h3>
              <p className="text-sm mb-4">
                サービス接続ページからサービスを接続すると、ツール設定が可能になります
              </p>
              <Button asChild>
                <a href="/services">
                  <Link2 className="h-4 w-4 mr-2" />
                  サービス接続へ
                </a>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">ツール設定</h1>
        <p className="text-muted-foreground mt-1">接続済みサービスのツールを管理します</p>
      </div>

      {/* サービス選択タブ */}
      <div className="flex gap-2 flex-wrap">
        {connectedModules.map((module) => {
          const isSelected = selectedModuleId === module.id
          const enabledCount = getEnabledToolCount(module.id)
          const totalCount = module.tools.length

          return (
            <button
              key={module.id}
              onClick={() => setSelectedModuleId(module.id)}
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-lg border transition-all",
                isSelected
                  ? "border-primary bg-primary/10"
                  : "border-border hover:border-primary/50"
              )}
            >
              <ModuleIcon icon={getModuleIcon(module.id)} className="h-4 w-4" />
              <span className="font-medium text-sm">{module.name}</span>
              <Badge variant="secondary" className="text-xs">
                {enabledCount}/{totalCount}
              </Badge>
            </button>
          )
        })}
      </div>

      {/* 選択されたモジュールの詳細 */}
      {selectedModule && (
        <Card>
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center">
                  <ModuleIcon icon={getModuleIcon(selectedModule.id)} className="h-6 w-6 text-foreground" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <CardTitle className="text-lg">{selectedModule.name}</CardTitle>
                    <Badge
                      style={{
                        backgroundColor: `${accentPreview}20`,
                        color: accentPreview,
                        borderColor: `${accentPreview}30`,
                      }}
                    >
                      <CheckCircle2 className="h-3 w-3 mr-1" />
                      接続済
                    </Badge>
                  </div>
                  <CardDescription>{getModuleDescription(selectedModule, language)}</CardDescription>
                </div>
              </div>
            </div>
          </CardHeader>

          {/* モジュール説明 */}
          <CardContent className="border-t pt-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <h3 className="font-medium text-sm text-foreground">カスタム説明</h3>
                <span className="text-xs text-muted-foreground">
                  {(editingModuleId === selectedModule.id ? editingDescription : getUserModuleDescription(selectedModule.id) || "").length}/256
                </span>
              </div>
              <Textarea
                value={editingModuleId === selectedModule.id ? editingDescription : getUserModuleDescription(selectedModule.id) || ""}
                onChange={(e) => {
                  if (editingModuleId !== selectedModule.id) {
                    setEditingModuleId(selectedModule.id)
                  }
                  setEditingDescription(e.target.value)
                }}
                onFocus={() => {
                  if (editingModuleId !== selectedModule.id) {
                    setEditingModuleId(selectedModule.id)
                    setEditingDescription(getUserModuleDescription(selectedModule.id) || "")
                  }
                }}
                placeholder="このモジュールの使い方や注意点を記述してください（AIへの追加コンテキストとして使用されます）"
                className="min-h-[80px] resize-y"
                maxLength={256}
              />
              {editingModuleId === selectedModule.id && (
                <div className="flex justify-end gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCancelEdit}
                    disabled={savingDescription}
                  >
                    <X className="h-3 w-3 mr-1" />
                    キャンセル
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => handleSaveModuleDescription(selectedModule.id)}
                    disabled={savingDescription}
                  >
                    {savingDescription ? (
                      <>
                        <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                        保存中...
                      </>
                    ) : (
                      <>
                        <Check className="h-3 w-3 mr-1" />
                        保存
                      </>
                    )}
                  </Button>
                </div>
              )}
            </div>
          </CardContent>

          {/* ツール設定 */}
          <CardContent className="space-y-3 border-t pt-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-medium text-sm text-foreground">ツール設定</h3>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => handleSelectDefault(selectedModule.id)}>
                  デフォルト
                </Button>
                <Button variant="outline" size="sm" onClick={() => handleSelectAll(selectedModule.id)}>
                  全選択
                </Button>
                <Button variant="outline" size="sm" onClick={() => handleDeselectAll(selectedModule.id)}>
                  全解除
                </Button>
              </div>
            </div>
            {selectedModule.tools.map((tool) => {
              const dangerous = isDangerous(tool)
              const readOnly = tool.annotations.readOnlyHint === true
              const destructive = tool.annotations.destructiveHint === true
              const idempotent = tool.annotations.idempotentHint === true
              return (
                <div
                  key={tool.id}
                  className={cn(
                    "flex items-center gap-3 p-3 rounded-lg border",
                    dangerous && "border-warning/30 bg-warning/5"
                  )}
                >
                  <Switch
                    checked={isToolEnabled(selectedModule.id, tool.id)}
                    onCheckedChange={() => handleToggleTool(selectedModule.id, tool.id)}
                  />
                  <div className="flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium text-sm font-mono">{tool.name}</span>
                      {readOnly ? (
                        <Badge variant="outline" className="text-info border-info/50 text-xs">
                          ReadOnly
                        </Badge>
                      ) : (
                        <>
                          {destructive && (
                            <Badge variant="outline" className="text-warning border-warning/50 text-xs">
                              <AlertTriangle className="h-3 w-3 mr-1" />
                              Destructive
                            </Badge>
                          )}
                          {idempotent && (
                            <Badge variant="outline" className="text-muted-foreground border-muted-foreground/50 text-xs">
                              Idempotent
                            </Badge>
                          )}
                        </>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">{getToolDescription(tool, language)}</p>
                  </div>
                </div>
              )
            })}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
