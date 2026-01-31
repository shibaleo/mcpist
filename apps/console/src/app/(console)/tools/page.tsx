"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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
  ChevronLeft,
  ChevronRight,
  CheckCircle2,
  XCircle,
  Unlink,
  Info,
  ExternalLink,
  X,
} from "lucide-react"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import {
  getMyConnections,
  upsertTokenWithVerification,
  deleteToken,
  type ServiceConnection,
  type ConnectionProgress,
  TokenVaultError,
} from "@/lib/token-vault"
import { getOAuthProviderForService, getOAuthAuthorizationUrl, OAuthAppError } from "@/lib/oauth-apps"
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

// User preferences type
interface UserPreferences {
  preferred_modules?: string[]
  language?: Language
}

// 認証方法の設定
interface AuthConfigField {
  name: string
  label: string
  type: "text" | "password" | "email"
  placeholder: string
}

interface AuthConfig {
  authLabel: string
  helpText?: string
  helpUrl?: string
  authType: "api_key" | "basic" | "oauth"
  extraFields?: AuthConfigField[]
}

const authConfig: Record<string, AuthConfig> = {
  notion: {
    authLabel: "内部インテグレーショントークン",
    helpText:
      "Notion設定 > マイコネクション > インテグレーションを開発または管理する > 新しいインテグレーションから取得してください",
    helpUrl: "https://www.notion.so/profile/integrations",
    authType: "api_key",
  },
  github: {
    authLabel: "Personal Access Token",
    helpText:
      "GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens から発行してください",
    helpUrl: "https://github.com/settings/tokens?type=beta",
    authType: "api_key",
  },
  jira: {
    authLabel: "APIトークン",
    helpText: "Atlassian管理画面 > セキュリティ > APIトークンから発行してください",
    helpUrl: "https://id.atlassian.com/manage-profile/security/api-tokens",
    authType: "basic",
    extraFields: [
      { name: "email", label: "メールアドレス", type: "email", placeholder: "user@example.com" },
      { name: "domain", label: "ドメイン", type: "text", placeholder: "yourcompany.atlassian.net" },
    ],
  },
  confluence: {
    authLabel: "APIトークン",
    helpText:
      "Atlassian管理画面 > セキュリティ > APIトークンから発行してください（Jiraと共通のトークンを使用できます）",
    helpUrl: "https://id.atlassian.com/manage-profile/security/api-tokens",
    authType: "basic",
    extraFields: [
      { name: "email", label: "メールアドレス", type: "email", placeholder: "user@example.com" },
      { name: "domain", label: "ドメイン", type: "text", placeholder: "yourcompany.atlassian.net" },
    ],
  },
  supabase: {
    authLabel: "Personal Access Token",
    helpText:
      "Supabase Management APIへ接続するPersonal Access Tokenを取得してください（Dashboard > Account > Access Tokens）",
    helpUrl: "https://supabase.com/dashboard/account/tokens",
    authType: "api_key",
  },
  google_calendar: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、カレンダーへのアクセスを許可します",
    authType: "oauth",
  },
  microsoft_todo: {
    authLabel: "Microsoft OAuth",
    helpText: "Microsoftアカウントでログインして、タスクへのアクセスを許可します",
    authType: "oauth",
  },
}

export const dynamic = "force-dynamic"

export default function ToolsPage() {
  const { user } = useAuth()
  const { accentColor } = useAppearance()
  const searchParams = useSearchParams()
  const router = useRouter()
  const accentPreview = accentColors.find((c) => c.id === accentColor)?.preview ?? "#22c55e"

  // Tool settings state (loaded from DB)
  const [toolSettings, setToolSettings] = useState<ToolSettingsMap>({})
  const [localToolSettings, setLocalToolSettings] = useState<ToolSettingsMap>({})
  const [savedModules, setSavedModules] = useState<Record<string, boolean>>({})
  const [savingModules, setSavingModules] = useState<Record<string, boolean>>({})
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedModuleId, setSelectedModuleId] = useState<string | null>(null)
  const carouselRef = useRef<HTMLDivElement>(null)

  // Module description state
  const [moduleDescriptions, setModuleDescriptions] = useState<ModuleDescriptionsMap>({})
  const [editingModuleId, setEditingModuleId] = useState<string | null>(null)
  const [editingDescription, setEditingDescription] = useState("")
  const [savingDescription, setSavingDescription] = useState(false)

  // Language setting
  const [language, setLanguage] = useState<Language>("ja-JP")

  // User preferences (preferred modules from onboarding)
  const [preferredModules, setPreferredModules] = useState<string[]>([])

  // Dialog states
  const [connectDialog, setConnectDialog] = useState<string | null>(null)
  const [disconnectDialog, setDisconnectDialog] = useState<string | null>(null)
  const [tokenInput, setTokenInput] = useState("")
  const [extraFields, setExtraFields] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [connectionProgress, setConnectionProgress] = useState<ConnectionProgress | null>(null)

  // 接続済みサービスを取得
  const loadConnections = useCallback(async () => {
    try {
      const data = await getMyConnections()
      setConnections(data)
    } catch (error) {
      if (error instanceof TokenVaultError) {
        console.error("Failed to load connections:", error.message)
      }
    }
  }, [])

  // ツール設定とpreferencesを取得
  const loadToolSettings = useCallback(async () => {
    try {
      const [settings, descriptions, userSettings, prefsResponse] = await Promise.all([
        getMyToolSettings(),
        getMyModuleDescriptions(),
        getUserSettings(),
        fetch("/api/user/preferences").then(res => res.json()).catch(() => ({})),
      ])
      const settingsMap = toToolSettingsMap(settings)
      setToolSettings(settingsMap)
      setLocalToolSettings(settingsMap)
      setModuleDescriptions(toModuleDescriptionsMap(descriptions))
      setLanguage(userSettings.language)
      // preferred_modules を設定
      const prefs = prefsResponse as UserPreferences
      if (prefs?.preferred_modules && Array.isArray(prefs.preferred_modules)) {
        setPreferredModules(prefs.preferred_modules)
      }
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

  // OAuth認可フロー完了後のクエリパラメータを処理
  useEffect(() => {
    const success = searchParams.get("success")
    const error = searchParams.get("error")

    if (success) {
      toast.success(success)
      router.replace("/tools")
    } else if (error) {
      toast.error(error)
      router.replace("/tools")
    }
  }, [searchParams, router])

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

  // モジュールの接続状態を取得
  const getConnectionForModule = (moduleId: string) => {
    return connections.find((c) => c.module === moduleId)
  }

  // 選択中のモジュール情報
  const selectedModule = modules.find((m) => m.id === selectedModuleId)
  const isSelectedConnected = selectedModuleId ? connectedModuleIds.has(selectedModuleId) : false

  // 初回ロード時に最初の接続済みモジュールを選択
  useEffect(() => {
    if (!loading && !selectedModuleId) {
      const firstConnected = modules.find((m) => connectedModuleIds.has(m.id))
      if (firstConnected) {
        setSelectedModuleId(firstConnected.id)
      } else if (modules.length > 0) {
        setSelectedModuleId(modules[0].id)
      }
    }
  }, [loading, selectedModuleId, connectedModuleIds])

  // モジュール切り替え時に編集状態をリセット
  useEffect(() => {
    setEditingModuleId(null)
    setEditingDescription("")
  }, [selectedModuleId])

  // カルーセルナビゲーション
  const scrollCarousel = (direction: "left" | "right") => {
    if (!carouselRef.current) return
    const scrollAmount = 200
    carouselRef.current.scrollBy({
      left: direction === "left" ? -scrollAmount : scrollAmount,
      behavior: "smooth",
    })
  }

  // ツール設定関連
  const handleToggleTool = (moduleId: string, toolId: string) => {
    setLocalToolSettings((prev) => {
      const current = getModuleToolSettings(moduleId)
      return {
        ...prev,
        [moduleId]: {
          ...current,
          [toolId]: !current[toolId],
        },
      }
    })
  }

  const handleSelectAll = (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    const allEnabled: Record<string, boolean> = {}
    mod.tools.forEach((t) => {
      allEnabled[t.id] = true
    })
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: allEnabled,
    }))
  }

  const handleDeselectAll = (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    const allDisabled: Record<string, boolean> = {}
    mod.tools.forEach((t) => {
      allDisabled[t.id] = false
    })
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: allDisabled,
    }))
  }

  const handleSelectDefault = (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    const defaults: Record<string, boolean> = {}
    mod.tools.forEach((t) => {
      defaults[t.id] = isDefaultEnabled(t)
    })
    setLocalToolSettings((prev) => ({
      ...prev,
      [moduleId]: defaults,
    }))
  }

  const isToolEnabled = (moduleId: string, toolId: string) => {
    const settings = getModuleToolSettings(moduleId)
    return settings[toolId] ?? false
  }

  const getEnabledToolCount = (moduleId: string): number => {
    const settings = getModuleToolSettings(moduleId)
    return Object.values(settings).filter(Boolean).length
  }

  const handleSave = async (moduleId: string) => {
    const settings = getModuleToolSettings(moduleId)
    setSavingModules((prev) => ({ ...prev, [moduleId]: true }))

    try {
      await saveModuleToolSettings(moduleId, settings)
      // 保存成功後、DBの値も更新
      setToolSettings((prev) => ({
        ...prev,
        [moduleId]: settings,
      }))
      setSavedModules((prev) => ({ ...prev, [moduleId]: true }))
      setTimeout(() => {
        setSavedModules((prev) => ({ ...prev, [moduleId]: false }))
      }, 2000)
    } catch (error) {
      if (error instanceof ToolSettingsError) {
        toast.error(`保存に失敗しました: ${error.message}`)
      } else {
        toast.error("保存に失敗しました")
      }
    } finally {
      setSavingModules((prev) => ({ ...prev, [moduleId]: false }))
    }
  }

  // モジュール説明関連（ユーザーが設定したカスタム説明）
  const getUserModuleDescription = (moduleId: string): string | undefined => {
    return moduleDescriptions[moduleId]
  }

  const handleEditModuleDescription = (moduleId: string) => {
    setEditingModuleId(moduleId)
    setEditingDescription(getUserModuleDescription(moduleId) || "")
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

  // 接続関連
  const handleConnect = async (serviceId: string) => {
    const config = authConfig[serviceId]

    // OAuthサービスの場合は認可URLにリダイレクト
    if (config?.authType === "oauth") {
      const providerId = getOAuthProviderForService(serviceId)
      if (!providerId) {
        toast.error("OAuth設定が見つかりません")
        return
      }

      try {
        const authUrl = await getOAuthAuthorizationUrl(providerId)
        window.location.href = authUrl
      } catch (error) {
        if (error instanceof OAuthAppError) {
          toast.error(error.message)
        } else {
          toast.error("OAuth認可URLの取得に失敗しました")
        }
      }
      return
    }

    // API Key / Basic認証の場合はダイアログを表示
    setConnectDialog(serviceId)
    setTokenInput("")
    setExtraFields({})
    setConnectionProgress(null)
  }

  const handleConnectionConfirm = () => {
    setConnectDialog(null)
    setTokenInput("")
    setExtraFields({})
    setConnectionProgress(null)
    toast.success("接続が完了しました")
  }

  const handleConnectSubmit = async () => {
    if (!connectDialog || !tokenInput || !user) return

    const config = authConfig[connectDialog]

    // Basic認証の場合、追加フィールドが必須
    if (config?.authType === "basic") {
      const missingFields = config.extraFields?.filter((f) => !extraFields[f.name])
      if (missingFields && missingFields.length > 0) {
        toast.error(`${missingFields.map((f) => f.label).join("、")}を入力してください`)
        return
      }
    }

    setSubmitting(true)
    setConnectionProgress({ step: "validating", message: "トークンを検証中..." })

    try {
      await upsertTokenWithVerification(
        {
          service: connectDialog,
          accessToken: tokenInput,
          ...(config?.authType === "basic" && {
            username: extraFields.email,
            metadata: { domain: extraFields.domain },
          }),
        },
        (progress) => {
          setConnectionProgress({ ...progress })
        }
      )

      setConnectionProgress({ step: "completed", message: "接続完了" })

      try {
        await loadConnections()
      } catch {
        // loadConnectionsのエラーは無視
      }
    } catch (error) {
      let errorMessage = "接続に失敗しました"
      if (error instanceof TokenVaultError) {
        errorMessage = error.message
      } else if (error instanceof Error) {
        errorMessage = error.message
      }
      setConnectionProgress({ step: "error", message: errorMessage })
    } finally {
      setSubmitting(false)
    }
  }

  const handleDisconnect = async () => {
    if (!disconnectDialog || !user) return

    setSubmitting(true)
    try {
      await deleteToken(disconnectDialog)
      toast.success("接続を解除しました")
      await loadConnections()
      setDisconnectDialog(null)
    } catch (error) {
      if (error instanceof TokenVaultError) {
        toast.error(`切断に失敗しました: ${error.message}`)
      } else {
        toast.error("切断に失敗しました")
      }
    } finally {
      setSubmitting(false)
    }
  }

  const dialogModule = connectDialog ? modules.find((m) => m.id === connectDialog) : null
  const dialogAuthConfig = connectDialog ? authConfig[connectDialog] : null

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">サービス & ツール</h1>
          <p className="text-muted-foreground mt-1">サービスの接続とツールの設定を管理します</p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">サービス & ツール</h1>
        <p className="text-muted-foreground mt-1">サービスの接続とツールの設定を管理します</p>
      </div>

      {/* サービスカルーセル */}
      <div className="relative group">
        <Button
          variant="outline"
          size="icon"
          className="absolute -left-3 top-1/2 -translate-y-1/2 z-10 bg-background hover:bg-secondary shadow-lg h-10 w-10 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={() => scrollCarousel("left")}
        >
          <ChevronLeft className="h-5 w-5" />
        </Button>
        <div
          ref={carouselRef}
          className="flex gap-4 overflow-x-auto scrollbar-hide py-4 px-2"
          style={{ scrollbarWidth: "none", msOverflowStyle: "none" }}
        >
          {/* preferredModulesで優先ソート */}
          {[...modules].sort((a, b) => {
            const aPreferred = preferredModules.includes(a.id)
            const bPreferred = preferredModules.includes(b.id)
            if (aPreferred && !bPreferred) return -1
            if (!aPreferred && bPreferred) return 1
            // 同じ優先度なら元の順序を維持
            return preferredModules.indexOf(a.id) - preferredModules.indexOf(b.id)
          }).map((module) => {
            const isConnected = connectedModuleIds.has(module.id)
            const isSelected = selectedModuleId === module.id
            const isPreferred = preferredModules.includes(module.id) && !isConnected
            const moduleDef = modules.find((m) => m.id === module.id)
            const enabledCount = getEnabledToolCount(module.id)
            const totalCount = moduleDef?.tools.length || 0

            return (
              <div
                key={module.id}
                onClick={() => {
                  setSelectedModuleId(module.id)
                  if (!isConnected) {
                    handleConnect(module.id)
                  }
                }}
                className={cn(
                  "flex-shrink-0 w-48 p-4 rounded-xl border-2 transition-all shadow-sm hover:shadow-md cursor-pointer",
                  isSelected
                    ? "border-primary bg-primary/10 shadow-primary/20"
                    : isConnected
                      ? "border-border bg-card hover:border-primary/50"
                      : isPreferred
                        ? "animate-pulse-border border-primary bg-primary/5"
                        : "border-dashed border-muted-foreground/30 bg-muted/30 hover:border-muted-foreground/50"
                )}
              >
                <div className="flex flex-col items-center gap-2">
                  <div
                    className={cn(
                      "w-12 h-12 rounded-xl flex items-center justify-center",
                      isConnected ? "bg-secondary" : "bg-muted"
                    )}
                  >
                    <ModuleIcon
                      icon={getModuleIcon(module.id)}
                      className={cn("h-6 w-6", isConnected ? "text-foreground" : "text-muted-foreground")}
                    />
                  </div>
                  <div className="text-center w-full">
                    <div className={cn("font-semibold text-sm", !isConnected && "text-muted-foreground")}>
                      {module.name}
                    </div>
                    {isConnected ? (
                      <div className="flex items-center justify-center gap-1 mt-0.5">
                        <CheckCircle2 className="h-3 w-3" style={{ color: accentPreview }} />
                        <span className="text-xs text-muted-foreground">
                          {enabledCount}/{totalCount} 有効
                        </span>
                      </div>
                    ) : (
                      <div className="flex items-center justify-center gap-1 mt-0.5 text-muted-foreground">
                        <span className="text-xs">未接続</span>
                      </div>
                    )}
                  </div>
                  {/* 接続/切断ボタン */}
                  <div className="w-full mt-1">
                    {isConnected ? (
                      <Button
                        variant="outline"
                        size="sm"
                        className="w-full h-7 text-xs"
                        onClick={(e) => {
                          e.stopPropagation()
                          setDisconnectDialog(module.id)
                        }}
                      >
                        <Unlink className="h-3 w-3 mr-1" />
                        切断
                      </Button>
                    ) : (
                      <Button
                        size="sm"
                        className="w-full h-7 text-xs"
                        onClick={(e) => {
                          e.stopPropagation()
                          handleConnect(module.id)
                        }}
                      >
                        <Link2 className="h-3 w-3 mr-1" />
                        接続
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
        <Button
          variant="outline"
          size="icon"
          className="absolute -right-3 top-1/2 -translate-y-1/2 z-10 bg-background hover:bg-secondary shadow-lg h-10 w-10 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={() => scrollCarousel("right")}
        >
          <ChevronRight className="h-5 w-5" />
        </Button>
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
                    {isSelectedConnected && (
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
                    )}
                  </div>
                  <CardDescription>{getModuleDescription(selectedModule, language)}</CardDescription>
                </div>
              </div>
              <div className="flex gap-2">
                {isSelectedConnected ? (
                  <>
                    <Button variant="outline" size="sm" onClick={() => handleConnect(selectedModule.id)}>
                      <Link2 className="h-4 w-4 mr-1" />
                      更新
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => setDisconnectDialog(selectedModule.id)}>
                      <Unlink className="h-4 w-4 mr-1" />
                      切断
                    </Button>
                  </>
                ) : (
                  <Button onClick={() => handleConnect(selectedModule.id)}>
                    <Link2 className="h-4 w-4 mr-2" />
                    接続する
                  </Button>
                )}
              </div>
            </div>
          </CardHeader>

          {/* モジュール説明（接続済みの場合のみ） */}
          {isSelectedConnected && selectedModule && (
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
                  className="min-h-[80px] resize-none"
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
          )}

          {/* ツール設定（接続済みの場合のみ） */}
          {isSelectedConnected && selectedModule && (
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
                      dangerous && "border-yellow-500/30 bg-yellow-500/5"
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
                          <Badge variant="outline" className="text-blue-500 border-blue-500/50 text-xs">
                            ReadOnly
                          </Badge>
                        ) : (
                          <>
                            {destructive && (
                              <Badge variant="outline" className="text-yellow-500 border-yellow-500/50 text-xs">
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
              <div className="flex justify-end items-center gap-2 pt-2">
                {savedModules[selectedModule.id] && (
                  <span className="text-sm text-green-500 flex items-center gap-1">
                    <Check className="h-4 w-4" />
                    保存しました
                  </span>
                )}
                <Button
                  size="sm"
                  onClick={() => handleSave(selectedModule.id)}
                  disabled={savingModules[selectedModule.id]}
                >
                  {savingModules[selectedModule.id] ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                      保存中...
                    </>
                  ) : (
                    "設定を保存"
                  )}
                </Button>
              </div>
            </CardContent>
          )}

          {/* 未接続の場合のメッセージ */}
          {!isSelectedConnected && (
            <CardContent className="border-t pt-4">
              <div className="text-center py-8 text-muted-foreground">
                <Link2 className="h-10 w-10 mx-auto mb-3 opacity-50" />
                <p className="text-sm">サービスに接続するとツール設定が可能になります</p>
              </div>
            </CardContent>
          )}
        </Card>
      )}

      {/* Connect Dialog */}
      <Dialog open={!!connectDialog} onOpenChange={(open) => !open && setConnectDialog(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {dialogModule && (
                <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                  <ModuleIcon icon={getModuleIcon(dialogModule.id)} className="h-5 w-5 text-foreground" />
                </div>
              )}
              <div>
                <DialogTitle>{dialogModule?.name}に接続</DialogTitle>
                <DialogDescription>認証情報を入力してください</DialogDescription>
              </div>
            </div>
          </DialogHeader>

          {connectionProgress ? (
            <div className="py-8 flex flex-col items-center justify-center space-y-4">
              {connectionProgress.step === "completed" ? (
                <CheckCircle2 className="h-12 w-12 text-green-500" />
              ) : connectionProgress.step === "error" ? (
                <XCircle className="h-12 w-12 text-destructive" />
              ) : (
                <Loader2 className="h-12 w-12 animate-spin text-primary" />
              )}
              <p
                className={cn(
                  "text-lg font-medium text-center",
                  connectionProgress.step === "completed" && "text-green-500",
                  connectionProgress.step === "error" && "text-destructive"
                )}
              >
                {connectionProgress.step === "error" ? "接続に失敗しました" : connectionProgress.message}
              </p>
              {connectionProgress.step === "error" && (
                <p className="text-sm text-muted-foreground text-center px-4">{connectionProgress.message}</p>
              )}
              {connectionProgress.step === "completed" ? (
                <Button onClick={handleConnectionConfirm} className="mt-4">
                  確認
                </Button>
              ) : connectionProgress.step === "error" ? (
                <Button variant="outline" onClick={() => setConnectionProgress(null)} className="mt-4">
                  再試行
                </Button>
              ) : (
                <p className="text-sm text-muted-foreground">しばらくお待ちください...</p>
              )}
            </div>
          ) : (
            <>
              <div className="space-y-4 py-4">
                {dialogAuthConfig?.extraFields?.map((field) => (
                  <div key={field.name} className="space-y-2">
                    <Label htmlFor={`field-${field.name}`} className="text-sm font-medium">
                      {field.label}
                    </Label>
                    <Input
                      id={`field-${field.name}`}
                      type={field.type}
                      value={extraFields[field.name] || ""}
                      onChange={(e) => setExtraFields((prev) => ({ ...prev, [field.name]: e.target.value }))}
                      placeholder={field.placeholder}
                      disabled={submitting}
                    />
                  </div>
                ))}

                <div className="space-y-2">
                  <Label htmlFor="token-input" className="text-sm font-medium">
                    {dialogAuthConfig?.authLabel || "APIトークン"}
                  </Label>
                  <Input
                    id="token-input"
                    type="password"
                    value={tokenInput}
                    onChange={(e) => setTokenInput(e.target.value)}
                    placeholder="トークンを入力..."
                    disabled={submitting}
                  />
                  {dialogAuthConfig?.helpText && (
                    <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                      <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                      <div className="space-y-1">
                        <p className="text-xs text-muted-foreground">{dialogAuthConfig.helpText}</p>
                        {dialogAuthConfig.helpUrl && (
                          <a
                            href={dialogAuthConfig.helpUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                          >
                            <ExternalLink className="h-3 w-3" />
                            トークンを取得する
                          </a>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              </div>

              <DialogFooter>
                <Button variant="ghost" onClick={() => setConnectDialog(null)}>
                  キャンセル
                </Button>
                <Button onClick={handleConnectSubmit} disabled={!tokenInput || submitting}>
                  <Link2 className="h-4 w-4 mr-2" />
                  接続
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* Disconnect Dialog */}
      <Dialog open={!!disconnectDialog} onOpenChange={(open) => !open && setDisconnectDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>接続を解除しますか？</DialogTitle>
            <DialogDescription>
              {disconnectDialog && modules.find((m) => m.id === disconnectDialog)?.name}
              との接続を解除します。この操作は取り消せません。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDisconnectDialog(null)} disabled={submitting}>
              キャンセル
            </Button>
            <Button variant="destructive" onClick={handleDisconnect} disabled={submitting}>
              {submitting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  切断中...
                </>
              ) : (
                "切断する"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
