"use client"

import { useState, useEffect, useCallback } from "react"
import Link from "next/link"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
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
import { useAuth } from "@/lib/auth-context"
import { useAppearance, accentColors } from "@/lib/appearance-context"
import { Search, Link2, Unlink, Info, CheckCircle2, Store, Loader2, XCircle } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import {
  getMyConnections,
  upsertTokenWithVerification,
  deleteToken,
  type ServiceConnection,
  type ConnectionProgress,
  TokenVaultError,
} from "@/lib/token-vault"

// カテゴリ定義（背景色付き）
const categories = [
  { id: "productivity", name: "生産性", bgClass: "bg-blue-500/10" },
  { id: "development", name: "開発", bgClass: "bg-purple-500/10" },
  { id: "communication", name: "コミュニケーション", bgClass: "bg-green-500/10" },
  { id: "storage", name: "ストレージ", bgClass: "bg-amber-500/10" },
  { id: "analytics", name: "分析", bgClass: "bg-cyan-500/10" },
  { id: "marketing", name: "マーケティング", bgClass: "bg-pink-500/10" },
] as const

type CategoryId = (typeof categories)[number]["id"]

// 認証方法の型
type AuthMethod = "oauth2" | "personal_token" | "apikey" | "integration_token"

// 拡張サービス定義（ダミーデータ）
interface ExtendedService {
  id: string
  name: string
  description: string
  category: CategoryId
  authMethod: AuthMethod
  authLabel: string
  helpText?: string
}

const extendedServices: ExtendedService[] = [
  // 生産性
  { id: "google-calendar", name: "Google Calendar", description: "カレンダーイベントの管理と同期", category: "productivity", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "notion", name: "Notion", description: "ドキュメントとデータベースの連携", category: "productivity", authMethod: "integration_token", authLabel: "内部インテグレーション", helpText: "Notion設定 > インテグレーション > 新しいインテグレーションから取得してください" },
  { id: "microsoft-todo", name: "Microsoft To Do", description: "タスク・リストの管理", category: "productivity", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "todoist", name: "Todoist", description: "タスク管理とプロジェクト整理", category: "productivity", authMethod: "personal_token", authLabel: "APIトークン", helpText: "Todoist設定 > 連携 > APIトークンから取得してください" },
  { id: "asana", name: "Asana", description: "チームプロジェクト管理", category: "productivity", authMethod: "personal_token", authLabel: "Personal Access Token", helpText: "Asana開発者コンソールから取得してください" },
  { id: "trello", name: "Trello", description: "カンバンボード形式のタスク管理", category: "productivity", authMethod: "apikey", authLabel: "APIキー", helpText: "Trello Power-Up Admin Portalから取得してください" },
  { id: "airtable", name: "Airtable", description: "スプレッドシート型データベース", category: "productivity", authMethod: "personal_token", authLabel: "Personal Access Token", helpText: "Airtableアカウント設定から取得してください" },
  { id: "clickup", name: "ClickUp", description: "オールインワンプロジェクト管理", category: "productivity", authMethod: "personal_token", authLabel: "APIトークン", helpText: "ClickUp設定 > アプリから取得してください" },
  { id: "monday", name: "Monday.com", description: "ワークマネジメントプラットフォーム", category: "productivity", authMethod: "personal_token", authLabel: "APIトークン", helpText: "Monday.com管理画面 > APIから取得してください" },
  { id: "evernote", name: "Evernote", description: "ノートとメモの管理", category: "productivity", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  // 開発
  { id: "github", name: "GitHub", description: "リポジトリとイシューの管理", category: "development", authMethod: "personal_token", authLabel: "Personal Access Token", helpText: "GitHub Settings > Developer settings > Personal access tokensから発行してください" },
  { id: "jira", name: "Jira", description: "プロジェクト管理連携", category: "development", authMethod: "apikey", authLabel: "APIトークン", helpText: "Atlassian管理画面 > APIトークンから発行してください" },
  { id: "confluence", name: "Confluence", description: "Wiki・ドキュメント管理", category: "development", authMethod: "apikey", authLabel: "APIトークン", helpText: "Atlassian管理画面 > APIトークンから発行してください（Jiraと共通）" },
  { id: "gitlab", name: "GitLab", description: "DevOpsプラットフォーム", category: "development", authMethod: "personal_token", authLabel: "Personal Access Token", helpText: "GitLab User Settings > Access Tokensから取得してください" },
  { id: "bitbucket", name: "Bitbucket", description: "Gitリポジトリホスティング", category: "development", authMethod: "apikey", authLabel: "App Password", helpText: "Bitbucket Personal settings > App passwordsから取得してください" },
  { id: "linear", name: "Linear", description: "モダンなイシュートラッキング", category: "development", authMethod: "personal_token", authLabel: "Personal API Key", helpText: "Linear Settings > API > Personal API keysから取得してください" },
  { id: "sentry", name: "Sentry", description: "エラー監視とパフォーマンス", category: "development", authMethod: "personal_token", authLabel: "Auth Token", helpText: "Sentry User Settings > Auth Tokensから取得してください" },
  { id: "vercel", name: "Vercel", description: "フロントエンドデプロイ", category: "development", authMethod: "personal_token", authLabel: "Access Token", helpText: "Vercel Account Settings > Tokensから取得してください" },
  // コミュニケーション
  { id: "slack", name: "Slack", description: "チームコミュニケーション", category: "communication", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "discord", name: "Discord", description: "コミュニティチャット", category: "communication", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "teams", name: "Microsoft Teams", description: "ビジネスコラボレーション", category: "communication", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "zoom", name: "Zoom", description: "ビデオ会議", category: "communication", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "gmail", name: "Gmail", description: "メール管理", category: "communication", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "outlook", name: "Outlook", description: "メールとカレンダー", category: "communication", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  // ストレージ
  { id: "google-drive", name: "Google Drive", description: "クラウドストレージ", category: "storage", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "dropbox", name: "Dropbox", description: "ファイル同期と共有", category: "storage", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "onedrive", name: "OneDrive", description: "Microsoft クラウドストレージ", category: "storage", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "box", name: "Box", description: "エンタープライズストレージ", category: "storage", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "aws-s3", name: "AWS S3", description: "オブジェクトストレージ", category: "storage", authMethod: "apikey", authLabel: "Access Key", helpText: "AWS IAMコンソールからAccess Key IDとSecret Access Keyを取得してください" },
  // 分析
  { id: "google-analytics", name: "Google Analytics", description: "ウェブ解析", category: "analytics", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "mixpanel", name: "Mixpanel", description: "プロダクト分析", category: "analytics", authMethod: "apikey", authLabel: "API Secret", helpText: "Mixpanel Project Settings > API Secretから取得してください" },
  { id: "amplitude", name: "Amplitude", description: "行動分析プラットフォーム", category: "analytics", authMethod: "apikey", authLabel: "API Key", helpText: "Amplitude Settings > Projectsから取得してください" },
  { id: "hotjar", name: "Hotjar", description: "ヒートマップと録画", category: "analytics", authMethod: "personal_token", authLabel: "Personal Access Token", helpText: "Hotjar Account Settings > Personal Access Tokensから取得してください" },
  { id: "posthog", name: "PostHog", description: "オープンソース分析", category: "analytics", authMethod: "personal_token", authLabel: "Personal API Key", helpText: "PostHog Project Settings > Personal API Keysから取得してください" },
  // マーケティング
  { id: "mailchimp", name: "Mailchimp", description: "メールマーケティング", category: "marketing", authMethod: "apikey", authLabel: "APIキー", helpText: "Mailchimp Account > Extras > API keysから取得してください" },
  { id: "hubspot", name: "HubSpot", description: "CRM・マーケティング", category: "marketing", authMethod: "oauth2", authLabel: "OAuth 2.0" },
  { id: "intercom", name: "Intercom", description: "カスタマーメッセージング", category: "marketing", authMethod: "personal_token", authLabel: "Access Token", helpText: "Intercom Developer Hub > Your apps > Authenticationから取得してください" },
  { id: "zendesk", name: "Zendesk", description: "カスタマーサポート", category: "marketing", authMethod: "apikey", authLabel: "APIトークン", helpText: "Zendesk Admin > Channels > APIから取得してください" },
  { id: "salesforce", name: "Salesforce", description: "CRMプラットフォーム", category: "marketing", authMethod: "oauth2", authLabel: "OAuth 2.0" },
]

// サービスアイコンのマッピング
const serviceIcons: Record<string, string> = {
  "google-calendar": "📅",
  "notion": "📝",
  "github": "🐙",
  "jira": "🎯",
  "confluence": "📚",
  "microsoft-todo": "✅",
  "todoist": "✔️",
  "asana": "📋",
  "trello": "📌",
  "airtable": "📊",
  "clickup": "⚡",
  "monday": "📆",
  "evernote": "🐘",
  "gitlab": "🦊",
  "bitbucket": "🪣",
  "linear": "🔵",
  "sentry": "🔺",
  "vercel": "▲",
  "slack": "💬",
  "discord": "🎮",
  "teams": "👥",
  "zoom": "📹",
  "gmail": "📧",
  "outlook": "📬",
  "google-drive": "📁",
  "dropbox": "📦",
  "onedrive": "☁️",
  "box": "📥",
  "aws-s3": "🪣",
  "google-analytics": "📈",
  "mixpanel": "📊",
  "amplitude": "📉",
  "hotjar": "🔥",
  "posthog": "🦔",
  "mailchimp": "🐵",
  "hubspot": "🧡",
  "intercom": "💬",
  "zendesk": "🎧",
  "salesforce": "☁️",
}

// ユーザーが購入済み（利用可能）なサービス（モック）- 多数追加
const purchasedServices = [
  // 生産性
  "google-calendar", "notion", "microsoft-todo", "todoist", "asana", "trello", "airtable", "clickup", "monday", "evernote",
  // 開発
  "github", "jira", "confluence", "gitlab", "bitbucket", "linear", "sentry", "vercel",
  // コミュニケーション
  "slack", "discord", "teams", "zoom", "gmail", "outlook",
  // ストレージ
  "google-drive", "dropbox", "onedrive", "box", "aws-s3",
  // 分析
  "google-analytics", "mixpanel", "amplitude", "hotjar", "posthog",
  // マーケティング
  "mailchimp", "hubspot", "intercom", "zendesk", "salesforce",
]

export default function MyConnectionsPage() {
  const { user } = useAuth()
  const { accentColor } = useAppearance()
  const accentPreview = accentColors.find(c => c.id === accentColor)?.preview ?? "#22c55e"
  const [searchQuery, setSearchQuery] = useState("")
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [connectDialog, setConnectDialog] = useState<string | null>(null)
  const [disconnectDialog, setDisconnectDialog] = useState<string | null>(null)
  const [tokenInput, setTokenInput] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [connectionProgress, setConnectionProgress] = useState<ConnectionProgress | null>(null)

  // Supabaseから接続情報を取得
  const loadConnections = useCallback(async () => {
    try {
      const data = await getMyConnections()
      setConnections(data)
    } catch (error) {
      if (error instanceof TokenVaultError) {
        console.error("Failed to load connections:", error.message)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) {
      loadConnections()
    } else {
      setLoading(false)
    }
  }, [user, loadConnections])

  // 購入済みのサービスのみ表示（拡張サービスを使用）
  const availableServices = extendedServices.filter((service) =>
    purchasedServices.includes(service.id)
  )

  const filteredServices = availableServices.filter(
    (service) =>
      service.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      service.description.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  // カテゴリごとにサービスをグループ化
  const getServicesByCategory = (categoryId: CategoryId) => {
    return filteredServices.filter((s) => s.category === categoryId)
  }

  // サービスが存在するカテゴリのみ取得
  const activeCategories = categories.filter((cat) => getServicesByCategory(cat.id).length > 0)

  // サービスの接続状態を取得
  const getConnectionForService = (serviceId: string) => {
    return connections.find((c) => c.service === serviceId)
  }

  const handleConnect = (serviceId: string) => {
    setConnectDialog(serviceId)
    setTokenInput("")
    setConnectionProgress(null)
  }

  const handleConnectionConfirm = () => {
    setConnectDialog(null)
    setTokenInput("")
    setConnectionProgress(null)
    toast.success("接続が完了しました")
  }

  const handleConnectSubmit = async () => {
    if (!connectDialog || !tokenInput || !user) return

    setSubmitting(true)
    // 最初に進捗表示を開始
    setConnectionProgress({ step: 'validating', message: 'トークンを検証中...' })

    try {
      await upsertTokenWithVerification(
        {
          service: connectDialog,
          accessToken: tokenInput,
        },
        (progress) => {
          setConnectionProgress({ ...progress })
        }
      )

      // 明示的に完了状態を設定
      setConnectionProgress({ step: 'completed', message: '接続完了' })

      // 接続完了後、接続一覧を更新（エラーでもダイアログは維持）
      try {
        await loadConnections()
      } catch {
        // loadConnectionsのエラーは無視（完了表示は維持）
      }
    } catch (error) {
      console.log('[page] Caught error:', error)
      let errorMessage = '接続に失敗しました'
      if (error instanceof TokenVaultError) {
        errorMessage = error.message
      } else if (error instanceof Error) {
        errorMessage = error.message
      }
      setConnectionProgress({ step: 'error', message: errorMessage })
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

  const selectedService = connectDialog ? extendedServices.find((s) => s.id === connectDialog) : null

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">サービス接続</h1>
        <p className="text-muted-foreground mt-1">利用可能なサービスとの接続を管理します</p>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="サービスを検索..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10"
        />
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : filteredServices.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center">
            <Store className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="font-medium text-foreground mb-2">利用可能なサービスがありません</h3>
            <p className="text-sm text-muted-foreground mb-4">
              マーケットプレイスでサービスを追加してください
            </p>
            <Link href="/marketplace">
              <Button>
                <Store className="h-4 w-4 mr-2" />
                マーケットプレイスへ
              </Button>
            </Link>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-6">
          {activeCategories.map((category) => {
            const categoryServices = getServicesByCategory(category.id)
            return (
              <div key={category.id} className={cn("rounded-xl p-4", category.bgClass)}>
                {/* カテゴリヘッダー */}
                <div className="flex items-center gap-2 mb-3">
                  <h2 className="text-sm font-semibold text-foreground">{category.name}</h2>
                  <Badge variant="secondary" className="text-xs">
                    {categoryServices.length}
                  </Badge>
                </div>

                {/* サービスカード */}
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {categoryServices.map((service) => {
                    const connection = getConnectionForService(service.id)
                    const isConnected = !!connection

                    return (
                      <Card
                        key={service.id}
                        className="transition-all"
                        style={isConnected ? { borderColor: `${accentPreview}80` } : undefined}
                      >
                        <CardContent className="p-4">
                          <div className="flex items-start gap-4">
                            <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center text-2xl shrink-0">
                              {serviceIcons[service.id] || "🔗"}
                            </div>
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2 mb-1 flex-wrap">
                                <h3 className="font-medium text-foreground truncate">{service.name}</h3>
                                {isConnected && (
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
                              <p className="text-sm text-muted-foreground line-clamp-2">{service.description}</p>
                              {connection && (
                                <p className="text-xs text-muted-foreground mt-2">
                                  接続日: {new Date(connection.created_at).toLocaleDateString("ja-JP")}
                                </p>
                              )}
                            </div>
                          </div>
                          <div className="mt-4 flex justify-end gap-2">
                            {isConnected ? (
                              <>
                                <Button variant="outline" size="sm" onClick={() => handleConnect(service.id)}>
                                  <Link2 className="h-4 w-4 mr-1" />
                                  更新
                                </Button>
                                <Button variant="outline" size="sm" onClick={() => setDisconnectDialog(service.id)}>
                                  <Unlink className="h-4 w-4 mr-1" />
                                  切断
                                </Button>
                              </>
                            ) : (
                              <Button variant="default" size="sm" onClick={() => handleConnect(service.id)}>
                                <Link2 className="h-4 w-4 mr-1" />
                                接続
                              </Button>
                            )}
                          </div>
                        </CardContent>
                      </Card>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div className="pt-4 border-t">
        <p className="text-sm text-muted-foreground">
          他のサービスを追加したい場合は
          <Link href="/marketplace" className="text-primary hover:underline mx-1">
            マーケットプレイス
          </Link>
          をご覧ください。
        </p>
      </div>

      {/* Connect Dialog */}
      <Dialog open={!!connectDialog} onOpenChange={(open) => !open && setConnectDialog(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {selectedService && (
                <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center text-xl">
                  {serviceIcons[selectedService.id] || "🔗"}
                </div>
              )}
              <div>
                <DialogTitle>{selectedService?.name}に接続</DialogTitle>
                <DialogDescription>
                  {selectedService?.authMethod === "oauth2"
                    ? "外部サービスの認可画面に移動します"
                    : "認証情報を入力してください"}
                </DialogDescription>
              </div>
            </div>
          </DialogHeader>

          {/* 接続進行中の表示 */}
          {connectionProgress ? (
            <div className="py-8 flex flex-col items-center justify-center space-y-4">
              {connectionProgress.step === 'completed' ? (
                <CheckCircle2 className="h-12 w-12 text-green-500" />
              ) : connectionProgress.step === 'error' ? (
                <XCircle className="h-12 w-12 text-destructive" />
              ) : (
                <Loader2 className="h-12 w-12 animate-spin text-primary" />
              )}
              <p className={cn(
                "text-lg font-medium text-center",
                connectionProgress.step === 'completed' && "text-green-500",
                connectionProgress.step === 'error' && "text-destructive"
              )}>
                {connectionProgress.step === 'error' ? '接続に失敗しました' : connectionProgress.message}
              </p>
              {connectionProgress.step === 'error' && (
                <p className="text-sm text-muted-foreground text-center px-4">
                  {connectionProgress.message}
                </p>
              )}
              {connectionProgress.step === 'completed' ? (
                <Button onClick={handleConnectionConfirm} className="mt-4">
                  確認
                </Button>
              ) : connectionProgress.step === 'error' ? (
                <Button variant="outline" onClick={() => setConnectionProgress(null)} className="mt-4">
                  再試行
                </Button>
              ) : (
                <p className="text-sm text-muted-foreground">
                  しばらくお待ちください...
                </p>
              )}
            </div>
          ) : (
            <>
              <div className="space-y-4 py-4">
                {/* OAuth2.0認可 */}
                <div className="space-y-2">
                  <Label className="text-sm font-medium">OAuth 2.0で接続</Label>
                  <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                    <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                    <p className="text-xs text-muted-foreground">
                      外部の認可画面に移動し、アカウントを連携します
                    </p>
                  </div>
                  <Button
                    className="w-full"
                    disabled
                  >
                    <Link2 className="h-4 w-4 mr-2" />
                    認可を開始（準備中）
                  </Button>
                </div>

                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">または</span>
                  </div>
                </div>

                {/* トークン入力 */}
                <div className="space-y-2">
                  <Label htmlFor="token-input" className="text-sm font-medium">
                    {selectedService?.authLabel || "APIトークン"}で接続
                  </Label>
                  <Input
                    id="token-input"
                    type="password"
                    value={tokenInput}
                    onChange={(e) => setTokenInput(e.target.value)}
                    placeholder="トークンを入力..."
                    disabled={submitting}
                  />
                  {selectedService?.helpText && (
                    <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                      <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                      <p className="text-xs text-muted-foreground">
                        {selectedService.helpText}
                      </p>
                    </div>
                  )}
                  <Button
                    className="w-full"
                    onClick={handleConnectSubmit}
                    disabled={!tokenInput || submitting}
                  >
                    <Link2 className="h-4 w-4 mr-2" />
                    トークンで接続
                  </Button>
                </div>
              </div>

              <DialogFooter>
                <Button variant="ghost" onClick={() => setConnectDialog(null)}>
                  キャンセル
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
              {disconnectDialog && extendedServices.find((s) => s.id === disconnectDialog)?.name}
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
