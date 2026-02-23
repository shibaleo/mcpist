"use client"

import { useState, useEffect, useCallback } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
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
import { useAuth } from "@/lib/auth/auth-context"
import {
  getModules,
  type ModuleDef,
} from "@/lib/modules/module-data"
import {
  Plug,
  Cable,
  Loader2,
  CircleCheckBig,
  XCircle,
  Info,
  ExternalLink,
  Search,
  Gauge,
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
} from "@/lib/services/token-vault"
import { getOAuthProviderForService, getOAuthAuthorizationUrl, OAuthAppError } from "@/lib/oauth/apps"
import { getUserSettings, type Language } from "@/lib/settings/user-settings"

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

// 外部サービスのレート制限情報（公式ドキュメントベースの推定値）
interface RateLimitInfo {
  rate: string       // e.g. "3 req/s"
  note?: string      // 補足（認証方式やプランで変動する旨など）
}

const rateLimitInfo: Record<string, RateLimitInfo> = {
  notion: { rate: "3 req/s", note: "平均。バーストは短時間で制限される場合あり" },
  github: { rate: "5,000 req/h", note: "認証済みユーザー。Search API は 30 req/min" },
  jira: { rate: "10 req/s" },
  confluence: { rate: "10 req/s" },
  supabase: { rate: "制限なし", note: "Management API。プロジェクト設定に依存" },
  google_calendar: { rate: "500 req/100s" },
  google_tasks: { rate: "500 req/100s" },
  google_drive: { rate: "1,000 req/100s" },
  google_docs: { rate: "300 req/min" },
  google_sheets: { rate: "300 req/min" },
  google_apps_script: { rate: "制限あり", note: "スクリプト実行は 1,500 req/日 (Consumer)" },
  microsoft_todo: { rate: "制限あり", note: "Microsoft Graph: ユーザーあたり 10,000 req/10min" },
  todoist: { rate: "450 req/15min" },
  trello: { rate: "100 req/10s", note: "APIキーごと。300 req/10s (トークンごと)" },
  asana: { rate: "150 req/min" },
  postgresql: { rate: "制限なし", note: "接続先サーバーの設定に依存" },
  airtable: { rate: "5 req/s" },
  ticktick: { rate: "制限あり", note: "公式ドキュメント非公開" },
  dropbox: { rate: "制限あり", note: "エンドポイントにより異なる。約 1,000 req/5min" },
  grafana: { rate: "制限なし", note: "セルフホスト: 制限なし。Cloud: プランに依存" },
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
    authLabel: "Atlassian OAuth",
    helpText: "Atlassianアカウントでログインして、Jiraへのアクセスを許可します",
    authType: "oauth",
    alternativeAuth: {
      authLabel: "APIトークン",
      helpText: "Atlassian管理画面 > セキュリティ > APIトークンから発行してください",
      helpUrl: "https://id.atlassian.com/manage-profile/security/api-tokens",
      authType: "basic",
      extraFields: [
        { name: "email", label: "メールアドレス", type: "email", placeholder: "user@example.com" },
        { name: "domain", label: "ドメイン", type: "text", placeholder: "yourcompany.atlassian.net" },
      ],
    },
  },
  confluence: {
    authLabel: "Atlassian OAuth",
    helpText: "Atlassianアカウントでログインして、Confluenceへのアクセスを許可します",
    authType: "oauth",
    alternativeAuth: {
      authLabel: "APIトークン",
      helpText: "Atlassian管理画面 > セキュリティ > APIトークンから発行してください（Jiraと共通のトークンを使用できます）",
      helpUrl: "https://id.atlassian.com/manage-profile/security/api-tokens",
      authType: "basic",
      extraFields: [
        { name: "email", label: "メールアドレス", type: "email", placeholder: "user@example.com" },
        { name: "domain", label: "ドメイン", type: "text", placeholder: "yourcompany.atlassian.net" },
      ],
    },
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
  google_docs: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、ドキュメントへのアクセスを許可します",
    authType: "oauth",
  },
  google_sheets: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、スプレッドシートへのアクセスを許可します",
    authType: "oauth",
  },
  google_apps_script: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、Apps Scriptプロジェクトへのアクセスを許可します",
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
  postgresql: {
    authLabel: "接続文字列",
    helpText: "PostgreSQL接続文字列を入力してください（例: postgresql://user:password@host:5432/database）。Supabaseの場合: Project Settings > Database > Connection string (URI)",
    helpUrl: "https://supabase.com/docs/guides/database/connecting-to-postgres",
    authType: "api_key",
  },
  airtable: {
    authLabel: "Airtable OAuth",
    helpText: "Airtableアカウントでログインして、ベースへのアクセスを許可します",
    authType: "oauth",
    alternativeAuth: {
      authLabel: "Personal Access Token",
      helpText: "Airtable の Developer Hub > Personal access tokens から発行してください",
      helpUrl: "https://airtable.com/create/tokens",
      authType: "api_key",
    },
  },
  ticktick: {
    authLabel: "TickTick OAuth",
    helpText: "TickTickアカウントでログインして、タスクへのアクセスを許可します",
    authType: "oauth",
  },
  dropbox: {
    authLabel: "Dropbox OAuth",
    helpText: "Dropboxアカウントでログインして、ファイルとフォルダへのアクセスを許可します",
    authType: "oauth",
  },
  grafana: {
    authLabel: "Service Account Token",
    helpText: "Grafana の Administration > Service accounts からトークンを発行してください。Base URLも合わせて設定が必要です",
    helpUrl: "https://grafana.com/docs/grafana/latest/administration/service-accounts/",
    authType: "api_key",
    extraFields: [
      { name: "base_url", label: "Grafana URL", type: "text", placeholder: "https://grafana.example.com" },
    ],
  },
}

// モジュールレベルキャッシュ
let cachedConnections: ServiceConnection[] | null = null
let cachedLanguage: Language | null = null
let cachedPreferredModules: string[] | null = null

export const dynamic = "force-dynamic"

export default function ServicesPage() {
  const { user } = useAuth()
  const searchParams = useSearchParams()
  const router = useRouter()

  const hasCached = cachedConnections !== null
  const [connections, setConnections] = useState<ServiceConnection[]>(cachedConnections ?? [])
  const [loading, setLoading] = useState(!hasCached)
  const [modules, setModules] = useState<ModuleDef[]>([])
  // Language setting
  const [, setLanguage] = useState<Language>(cachedLanguage ?? "ja-JP")

  // User preferences (preferred modules from onboarding)
  const [preferredModules, setPreferredModules] = useState<string[]>(cachedPreferredModules ?? [])

  // Search filter
  const [searchQuery, setSearchQuery] = useState("")

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
      cachedConnections = data
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
      cachedLanguage = userSettings.language
      setLanguage(userSettings.language)
      // preferred_modules を設定
      const prefs = prefsResponse as UserPreferences
      if (prefs?.preferred_modules && Array.isArray(prefs.preferred_modules)) {
        cachedPreferredModules = prefs.preferred_modules
        setPreferredModules(prefs.preferred_modules)
      }
    } catch (error) {
      console.error("Failed to load settings:", error)
    }
  }, [])

  useEffect(() => {
    async function loadData() {
      const [mods] = await Promise.all([
        getModules(),
        ...(user ? [loadConnections(), loadSettings()] : []),
      ])
      setModules(mods)
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

  // 接続関連
  const handleConnect = (serviceId: string) => {
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

    // alternativeAuth がある場合はそちらの extraFields をチェック
    const effectiveAuth = config?.alternativeAuth ?? config
    if (effectiveAuth?.extraFields) {
      const missingFields = effectiveAuth.extraFields.filter((f) => !extraFields[f.name])
      if (missingFields.length > 0) {
        toast.error(`${missingFields.map((f) => f.label).join("、")}を入力してください`)
        return
      }
    }

    setSubmitting(true)
    setConnectionProgress({ step: "validating", message: "トークンを検証中..." })

    try {
      // Basic認証: email を username に、domain を metadata に
      const upsertParams: Parameters<typeof upsertTokenWithVerification>[0] = {
        service: connectDialog,
        accessToken: tokenInput,
      }

      const authType = effectiveAuth?.authType ?? config?.authType
      if (authType === "basic") {
        upsertParams.username = extraFields.email
        upsertParams.metadata = { domain: extraFields.domain }
      } else if (connectDialog === "trello") {
        // Trello: API Key を username に格納
        upsertParams.username = extraFields.api_key
      } else if (connectDialog === "grafana") {
        // Grafana: base_url を metadata に格納
        upsertParams.metadata = { base_url: extraFields.base_url }
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

  const dialogModule = connectDialog ? modules.find((m) => m.name === connectDialog) : null
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

  // 接続済みと未接続に分類（検索フィルタ適用）
  const query = searchQuery.toLowerCase()
  const connectedServices = modules.filter(m => connectedModuleIds.has(m.name) && (!query || m.name.toLowerCase().includes(query)))
  const unconnectedServices = modules.filter(m => !connectedModuleIds.has(m.name) && (!query || m.name.toLowerCase().includes(query)))

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">サービス</h1>
        <p className="text-muted-foreground mt-1">外部サービスへの接続を管理</p>
      </div>

      {/* 検索フィルタ */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="サービスを検索..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* 未接続サービス - Browse connectors 風グリッド */}
      {unconnectedServices.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground flex items-center gap-2">
            <Cable className="h-5 w-5 text-primary" />
            サービスを追加
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {[...unconnectedServices].sort((a, b) => {
              const aPreferred = preferredModules.includes(a.name)
              const bPreferred = preferredModules.includes(b.name)
              if (aPreferred && !bPreferred) return -1
              if (!aPreferred && bPreferred) return 1
              return preferredModules.indexOf(a.name) - preferredModules.indexOf(b.name)
            }).map((module) => (
              <div
                key={module.id}
                onClick={() => handleConnect(module.name)}
                className="flex items-center gap-3 p-3 rounded-xl border bg-card/70 hover:bg-muted/50 transition-colors cursor-pointer"
              >
                <div className="w-10 h-10 rounded-lg bg-white flex items-center justify-center shrink-0">
                  <ModuleIcon moduleName={module.name} className="h-5 w-5 text-foreground" />
                </div>
                <div className="min-w-0">
                  <span className="font-medium text-sm truncate block">{module.name}</span>
                  {rateLimitInfo[module.name] && (
                    <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                      <Gauge className="h-2.5 w-2.5 shrink-0" />
                      {rateLimitInfo[module.name].rate}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 接続済みサービス */}
      {connectedServices.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground flex items-center gap-2">
            <CircleCheckBig className="h-5 w-5 text-primary" />
            接続済みサービス
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {connectedServices.map((module) => (
              <div
                key={module.id}
                onClick={() => setDisconnectDialog(module.name)}
                className="relative flex items-center gap-3 p-3 rounded-xl border bg-card/70 hover:bg-muted/50 transition-colors cursor-pointer group"
              >
                <div className="relative w-10 h-10 rounded-lg bg-white flex items-center justify-center shrink-0">
                  <ModuleIcon moduleName={module.name} className="h-5 w-5 text-foreground" />
                  <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-500 ring-2 ring-card" />
                </div>
                <div className="min-w-0 flex-1">
                  <span className="font-medium text-sm truncate block">{module.name}</span>
                  {rateLimitInfo[module.name] && (
                    <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                      <Gauge className="h-2.5 w-2.5 shrink-0" />
                      {rateLimitInfo[module.name].rate}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Connect Dialog */}
      <Dialog open={!!connectDialog} onOpenChange={(open) => !open && setConnectDialog(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {dialogModule && (
                <div className="w-10 h-10 rounded-lg bg-white flex items-center justify-center">
                  <ModuleIcon moduleName={dialogModule.name} className="h-5 w-5 text-foreground" />
                </div>
              )}
              <div>
                <DialogTitle>{dialogModule?.name}に接続</DialogTitle>
                <DialogDescription>認証情報を入力してください</DialogDescription>
              </div>
            </div>
          </DialogHeader>

          {/* レート制限情報 */}
          {connectDialog && rateLimitInfo[connectDialog] && !connectionProgress && (
            <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
              <Gauge className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
              <div className="text-xs text-muted-foreground">
                <span className="font-medium">レート制限: {rateLimitInfo[connectDialog].rate}</span>
                {rateLimitInfo[connectDialog].note && (
                  <p className="mt-0.5">{rateLimitInfo[connectDialog].note}</p>
                )}
              </div>
            </div>
          )}

          {connectionProgress ? (
            <div className="py-8 flex flex-col items-center justify-center space-y-4">
              {connectionProgress.step === "completed" ? (
                <CircleCheckBig className="h-12 w-12 text-success" />
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

                    {/* Alternative auth 入力 */}
                    <div className="space-y-3">
                      {dialogAuthConfig.alternativeAuth.extraFields?.map((field) => (
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
                          <Plug className="h-4 w-4 mr-2" />
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
                    <Plug className="h-4 w-4 mr-2" />
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
              {disconnectDialog && modules.find((m) => m.name === disconnectDialog)?.name}
              との接続を解除します。この操作は取り消せません。
            </DialogDescription>
          </DialogHeader>
          {disconnectDialog && rateLimitInfo[disconnectDialog] && (
            <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
              <Gauge className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
              <div className="text-xs text-muted-foreground">
                <span className="font-medium">レート制限: {rateLimitInfo[disconnectDialog].rate}</span>
                {rateLimitInfo[disconnectDialog].note && (
                  <p className="mt-0.5">{rateLimitInfo[disconnectDialog].note}</p>
                )}
              </div>
            </div>
          )}
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
