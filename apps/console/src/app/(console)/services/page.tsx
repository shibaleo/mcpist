"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
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
  getModuleDescription,
} from "@/lib/module-data"
import {
  Link2,
  Loader2,
  ChevronLeft,
  ChevronRight,
  CheckCircle2,
  XCircle,
  Unlink,
  Info,
  ExternalLink,
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
  // 複数の認証方法をサポートするサービス用
  alternativeAuth?: {
    authLabel: string
    helpText?: string
    helpUrl?: string
    authType: "api_key" | "basic" | "oauth"
    extraFields?: AuthConfigField[]
  }
}

const authConfig: Record<string, AuthConfig> = {
  notion: {
    authLabel: "Notion OAuth",
    helpText: "Notionアカウントでログインして、ページへのアクセスを許可します",
    authType: "oauth",
    alternativeAuth: {
      authLabel: "内部インテグレーショントークン",
      helpText: "Notion設定 > マイコネクション > インテグレーションを開発または管理する > 新しいインテグレーションから取得してください",
      helpUrl: "https://www.notion.so/profile/integrations",
      authType: "api_key",
    },
  },
  github: {
    authLabel: "GitHub OAuth",
    helpText: "GitHubアカウントでログインして、リポジトリへのアクセスを許可します",
    authType: "oauth",
    alternativeAuth: {
      authLabel: "Fine-grained Personal Access Token",
      helpText:
        "GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens から発行してください",
      helpUrl: "https://github.com/settings/tokens?type=beta",
      authType: "api_key",
    },
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
  google_tasks: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、タスクへのアクセスを許可します",
    authType: "oauth",
  },
  google_drive: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、Driveへのアクセスを許可します",
    authType: "oauth",
  },
  microsoft_todo: {
    authLabel: "Microsoft OAuth",
    helpText: "Microsoftアカウントでログインして、タスクへのアクセスを許可します",
    authType: "oauth",
  },
  todoist: {
    authLabel: "Todoist OAuth",
    helpText: "Todoistアカウントでログインして、タスクへのアクセスを許可します",
    authType: "oauth",
  },
  trello: {
    authLabel: "Trello OAuth",
    helpText: "Trelloアカウントでログインして、ボードへのアクセスを許可します",
    authType: "oauth",
  },
  asana: {
    authLabel: "Asana OAuth",
    helpText: "Asanaアカウントでログインして、ワークスペースへのアクセスを許可します",
    authType: "oauth",
    alternativeAuth: {
      authLabel: "Personal Access Token",
      helpText: "Asana Settings > Apps > Developer apps > Personal access tokens から発行してください",
      helpUrl: "https://app.asana.com/0/my-apps",
      authType: "api_key",
    },
  },
}

export const dynamic = "force-dynamic"

export default function ServicesPage() {
  const { user } = useAuth()
  const { accentColor } = useAppearance()
  const searchParams = useSearchParams()
  const router = useRouter()
  const accentPreview = accentColors.find((c) => c.id === accentColor)?.preview ?? "#22c55e"

  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [loading, setLoading] = useState(true)
  const carouselRef = useRef<HTMLDivElement>(null)

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

  // ユーザー設定とpreferencesを取得
  const loadSettings = useCallback(async () => {
    try {
      const [userSettings, prefsResponse] = await Promise.all([
        getUserSettings(),
        fetch("/api/user/preferences").then(res => res.json()).catch(() => ({})),
      ])
      setLanguage(userSettings.language)
      // preferred_modules を設定
      const prefs = prefsResponse as UserPreferences
      if (prefs?.preferred_modules && Array.isArray(prefs.preferred_modules)) {
        setPreferredModules(prefs.preferred_modules)
      }
    } catch (error) {
      console.error("Failed to load settings:", error)
    }
  }, [])

  useEffect(() => {
    async function loadData() {
      if (user) {
        await Promise.all([loadConnections(), loadSettings()])
      }
      setLoading(false)
    }
    loadData()
  }, [user, loadConnections, loadSettings])

  // OAuth認可フロー完了後のクエリパラメータを処理
  useEffect(() => {
    const success = searchParams.get("success")
    const error = searchParams.get("error")

    if (success) {
      toast.success(success)
      router.replace("/services")
    } else if (error) {
      toast.error(error)
      router.replace("/services")
    }
  }, [searchParams, router])

  // 接続済みモジュールのIDセット
  const connectedModuleIds = new Set(connections.map((c) => c.module))

  // カルーセルナビゲーション
  const scrollCarousel = (direction: "left" | "right") => {
    if (!carouselRef.current) return
    const scrollAmount = 200
    carouselRef.current.scrollBy({
      left: direction === "left" ? -scrollAmount : scrollAmount,
      behavior: "smooth",
    })
  }

  // 接続関連
  const handleConnect = async (serviceId: string) => {
    const config = authConfig[serviceId]

    // alternativeAuth がある場合は常にダイアログを表示（認証方法を選択させる）
    if (config?.alternativeAuth) {
      setConnectDialog(serviceId)
      setTokenInput("")
      setExtraFields({})
      setConnectionProgress(null)
      return
    }

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

    // 追加フィールドが必須かチェック
    if (config?.extraFields) {
      const missingFields = config.extraFields.filter((f) => !extraFields[f.name])
      if (missingFields.length > 0) {
        toast.error(`${missingFields.map((f) => f.label).join("、")}を入力してください`)
        return
      }
    }

    setSubmitting(true)
    setConnectionProgress({ step: "validating", message: "トークンを検証中..." })

    try {
      // Trello: api_key を username に、token を accessToken に
      // Basic認証: email を username に、domain を metadata に
      const upsertParams: Parameters<typeof upsertTokenWithVerification>[0] = {
        service: connectDialog,
        accessToken: tokenInput,
      }

      if (config?.authType === "basic") {
        upsertParams.username = extraFields.email
        upsertParams.metadata = { domain: extraFields.domain }
      } else if (connectDialog === "trello") {
        // Trello: API Key を username に格納
        upsertParams.username = extraFields.api_key
      }

      await upsertTokenWithVerification(
        upsertParams,
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
          <h1 className="text-2xl font-bold text-foreground">サービス接続</h1>
          <p className="text-muted-foreground mt-1">外部サービスへの接続を管理します</p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  // 接続済みと未接続に分類
  const connectedServices = modules.filter(m => connectedModuleIds.has(m.id))
  const unconnectedServices = modules.filter(m => !connectedModuleIds.has(m.id))

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">サービス接続</h1>
        <p className="text-muted-foreground mt-1">外部サービスへの接続を管理します</p>
      </div>

      {/* 接続済みサービス */}
      {connectedServices.length > 0 && (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-foreground">接続済み</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {connectedServices.map((module) => (
              <Card key={module.id} className="border-success/30">
                <CardHeader className="pb-3">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                      <ModuleIcon icon={getModuleIcon(module.id)} className="h-5 w-5 text-foreground" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-base">{module.name}</CardTitle>
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
                      <CardDescription className="text-xs truncate">
                        {getModuleDescription(module, language)}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="flex-1"
                      onClick={() => handleConnect(module.id)}
                    >
                      <Link2 className="h-3 w-3 mr-1" />
                      更新
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="flex-1"
                      onClick={() => setDisconnectDialog(module.id)}
                    >
                      <Unlink className="h-3 w-3 mr-1" />
                      切断
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* 未接続サービス */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold text-foreground">利用可能なサービス</h2>
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
            {[...unconnectedServices].sort((a, b) => {
              const aPreferred = preferredModules.includes(a.id)
              const bPreferred = preferredModules.includes(b.id)
              if (aPreferred && !bPreferred) return -1
              if (!aPreferred && bPreferred) return 1
              return preferredModules.indexOf(a.id) - preferredModules.indexOf(b.id)
            }).map((module) => {
              const isPreferred = preferredModules.includes(module.id)

              return (
                <div
                  key={module.id}
                  onClick={() => handleConnect(module.id)}
                  className={cn(
                    "flex-shrink-0 w-48 p-4 rounded-xl border-2 transition-all shadow-sm hover:shadow-md cursor-pointer",
                    isPreferred
                      ? "animate-pulse-border border-primary bg-primary/5"
                      : "border-dashed border-muted-foreground/30 bg-muted/30 hover:border-muted-foreground/50"
                  )}
                >
                  <div className="flex flex-col items-center gap-2">
                    <div className="w-12 h-12 rounded-xl flex items-center justify-center bg-muted">
                      <ModuleIcon
                        icon={getModuleIcon(module.id)}
                        className="h-6 w-6 text-muted-foreground"
                      />
                    </div>
                    <div className="text-center w-full">
                      <div className="font-semibold text-sm text-muted-foreground">
                        {module.name}
                      </div>
                      <div className="flex items-center justify-center gap-1 mt-0.5 text-muted-foreground">
                        <span className="text-xs">未接続</span>
                      </div>
                    </div>
                    <div className="w-full mt-1">
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
      </div>

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
                <CheckCircle2 className="h-12 w-12 text-success" />
              ) : connectionProgress.step === "error" ? (
                <XCircle className="h-12 w-12 text-destructive" />
              ) : (
                <Loader2 className="h-12 w-12 animate-spin text-primary" />
              )}
              <p
                className={cn(
                  "text-lg font-medium text-center",
                  connectionProgress.step === "completed" && "text-success",
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
                {/* OAuth + alternativeAuth がある場合: 両方のオプションを表示 */}
                {dialogAuthConfig?.authType === "oauth" && dialogAuthConfig?.alternativeAuth ? (
                  <>
                    {/* OAuth ボタン */}
                    <div className="space-y-2">
                      <Button
                        className="w-full"
                        onClick={async () => {
                          const providerId = getOAuthProviderForService(connectDialog!)
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
                        }}
                      >
                        <ExternalLink className="h-4 w-4 mr-2" />
                        {dialogModule?.name}でログイン
                      </Button>
                      <p className="text-xs text-muted-foreground text-center">{dialogAuthConfig.helpText}</p>
                    </div>

                    {/* 区切り線 */}
                    <div className="relative">
                      <div className="absolute inset-0 flex items-center">
                        <span className="w-full border-t" />
                      </div>
                      <div className="relative flex justify-center text-xs uppercase">
                        <span className="bg-background px-2 text-muted-foreground">または</span>
                      </div>
                    </div>

                    {/* API Key 入力 */}
                    <div className="space-y-2">
                      <Label htmlFor="token-input" className="text-sm font-medium">
                        {dialogAuthConfig.alternativeAuth.authLabel}
                      </Label>
                      <div className="flex gap-2">
                        <Input
                          id="token-input"
                          type="password"
                          value={tokenInput}
                          onChange={(e) => setTokenInput(e.target.value)}
                          placeholder="トークンを入力..."
                          disabled={submitting}
                          className="flex-1"
                        />
                        <Button onClick={handleConnectSubmit} disabled={!tokenInput || submitting}>
                          <Link2 className="h-4 w-4 mr-2" />
                          接続
                        </Button>
                      </div>
                      {dialogAuthConfig.alternativeAuth.helpText && (
                        <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                          <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                          <div className="space-y-1">
                            <p className="text-xs text-muted-foreground">{dialogAuthConfig.alternativeAuth.helpText}</p>
                            {dialogAuthConfig.alternativeAuth.helpUrl && (
                              <a
                                href={dialogAuthConfig.alternativeAuth.helpUrl}
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
                  </>
                ) : dialogAuthConfig?.authType === "oauth" ? (
                  /* OAuth のみの場合 */
                  <div className="space-y-4">
                    <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                      <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                      <p className="text-xs text-muted-foreground">{dialogAuthConfig.helpText}</p>
                    </div>
                    <Button
                      className="w-full"
                      onClick={async () => {
                        const providerId = getOAuthProviderForService(connectDialog!)
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
                      }}
                    >
                      <ExternalLink className="h-4 w-4 mr-2" />
                      {dialogModule?.name}でログイン
                    </Button>
                  </div>
                ) : (
                  /* API Key / Basic 認証の場合 */
                  <>
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
                  </>
                )}
              </div>

              <DialogFooter>
                <Button variant="ghost" onClick={() => setConnectDialog(null)}>
                  キャンセル
                </Button>
                {dialogAuthConfig?.authType !== "oauth" && !dialogAuthConfig?.alternativeAuth && (
                  <Button onClick={handleConnectSubmit} disabled={!tokenInput || submitting}>
                    <Link2 className="h-4 w-4 mr-2" />
                    接続
                  </Button>
                )}
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
