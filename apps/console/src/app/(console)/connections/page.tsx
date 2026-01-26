"use client"

import { useState, useEffect, useCallback } from "react"
import { useSearchParams, useRouter } from "next/navigation"
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
import { Search, Link2, Unlink, Info, CheckCircle2, Loader2, XCircle } from "lucide-react"
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
import { services, getServiceIcon } from "@/lib/module-data"
import { ServiceIcon } from "@/components/service-icon"
import { getOAuthProviderForService, getOAuthAuthorizationUrl, OAuthAppError } from "@/lib/oauth-apps"

// 認証方法の設定（サービスごとの追加情報）
interface AuthConfigField {
  name: string
  label: string
  type: 'text' | 'password' | 'email'
  placeholder: string
}

interface AuthConfig {
  authLabel: string
  helpText?: string
  authType: 'api_key' | 'basic' | 'oauth'
  extraFields?: AuthConfigField[]
}

const authConfig: Record<string, AuthConfig> = {
  notion: {
    authLabel: "内部インテグレーショントークン",
    helpText: "Notion設定 > マイコネクション > インテグレーションを開発または管理する > 新しいインテグレーションから取得してください",
    authType: 'api_key',
  },
  github: {
    authLabel: "Personal Access Token",
    helpText: "GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens から発行してください",
    authType: 'api_key',
  },
  jira: {
    authLabel: "APIトークン",
    helpText: "Atlassian管理画面 > セキュリティ > APIトークンから発行してください",
    authType: 'basic',
    extraFields: [
      { name: 'email', label: 'メールアドレス', type: 'email', placeholder: 'user@example.com' },
      { name: 'domain', label: 'ドメイン', type: 'text', placeholder: 'yourcompany.atlassian.net' },
    ],
  },
  confluence: {
    authLabel: "APIトークン",
    helpText: "Atlassian管理画面 > セキュリティ > APIトークンから発行してください（Jiraと共通のトークンを使用できます）",
    authType: 'basic',
    extraFields: [
      { name: 'email', label: 'メールアドレス', type: 'email', placeholder: 'user@example.com' },
      { name: 'domain', label: 'ドメイン', type: 'text', placeholder: 'yourcompany.atlassian.net' },
    ],
  },
  supabase: {
    authLabel: "Personal Access Token",
    helpText: "Supabase Management APIへ接続するPersonal Access Tokenを取得してください（Dashboard > Account > Access Tokens）",
    authType: 'api_key',
  },
  google_calendar: {
    authLabel: "Google OAuth",
    helpText: "Googleアカウントでログインして、カレンダーへのアクセスを許可します",
    authType: 'oauth',
  },
  microsoft_todo: {
    authLabel: "Microsoft OAuth",
    helpText: "Microsoftアカウントでログインして、タスクへのアクセスを許可します",
    authType: 'oauth',
  },
}

export default function ConnectionsPage() {
  const { user } = useAuth()
  const { accentColor } = useAppearance()
  const searchParams = useSearchParams()
  const router = useRouter()
  const accentPreview = accentColors.find(c => c.id === accentColor)?.preview ?? "#22c55e"
  const [searchQuery, setSearchQuery] = useState("")
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [connectDialog, setConnectDialog] = useState<string | null>(null)
  const [disconnectDialog, setDisconnectDialog] = useState<string | null>(null)
  const [tokenInput, setTokenInput] = useState("")
  const [extraFields, setExtraFields] = useState<Record<string, string>>({})
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

  // OAuth認可フロー完了後のクエリパラメータを処理
  useEffect(() => {
    const success = searchParams.get("success")
    const error = searchParams.get("error")

    if (success) {
      toast.success(success)
      // URLからクエリパラメータを削除
      router.replace("/connections")
    } else if (error) {
      toast.error(error)
      // URLからクエリパラメータを削除
      router.replace("/connections")
    }
  }, [searchParams, router])

  // サービスをフィルタ（services.jsonから読み込んだサービスのみ）
  const filteredServices = services.filter(
    (service) =>
      service.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      service.description.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  // サービスの接続状態を取得
  const getConnectionForService = (serviceId: string) => {
    return connections.find((c) => c.service === serviceId)
  }

  const handleConnect = async (serviceId: string) => {
    const config = authConfig[serviceId]

    // OAuthサービスの場合は認可URLにリダイレクト
    if (config?.authType === 'oauth') {
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
    if (config?.authType === 'basic') {
      const missingFields = config.extraFields?.filter(f => !extraFields[f.name])
      if (missingFields && missingFields.length > 0) {
        toast.error(`${missingFields.map(f => f.label).join('、')}を入力してください`)
        return
      }
    }

    setSubmitting(true)
    setConnectionProgress({ step: 'validating', message: 'トークンを検証中...' })

    try {
      await upsertTokenWithVerification(
        {
          service: connectDialog,
          accessToken: tokenInput,
          // Basic認証の場合、usernameとmetadataを渡す
          ...(config?.authType === 'basic' && {
            username: extraFields.email,
            metadata: { domain: extraFields.domain },
          }),
        },
        (progress) => {
          setConnectionProgress({ ...progress })
        }
      )

      setConnectionProgress({ step: 'completed', message: '接続完了' })

      try {
        await loadConnections()
      } catch {
        // loadConnectionsのエラーは無視
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

  const selectedService = connectDialog ? services.find((s) => s.id === connectDialog) : null
  const selectedAuthConfig = connectDialog ? authConfig[connectDialog] : null

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">サービス接続</h1>
        <p className="text-muted-foreground mt-1">MCPサーバーで利用可能なサービスとの接続を管理します</p>
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
            <Search className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="font-medium text-foreground mb-2">サービスが見つかりません</h3>
            <p className="text-sm text-muted-foreground">
              検索条件を変更してください
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredServices.map((service) => {
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
                    <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                      <ServiceIcon icon={getServiceIcon(service.id)} className="h-6 w-6 text-foreground" />
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
                      <p className="text-xs text-muted-foreground mt-1">API: {service.apiVersion}</p>
                      {connection && (
                        <p className="text-xs text-muted-foreground mt-1">
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
      )}

      {/* Connect Dialog */}
      <Dialog open={!!connectDialog} onOpenChange={(open) => !open && setConnectDialog(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {selectedService && (
                <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                  <ServiceIcon icon={getServiceIcon(selectedService.id)} className="h-5 w-5 text-foreground" />
                </div>
              )}
              <div>
                <DialogTitle>{selectedService?.name}に接続</DialogTitle>
                <DialogDescription>
                  認証情報を入力してください
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
                {/* 追加フィールド（Basic認証用） */}
                {selectedAuthConfig?.extraFields?.map((field) => (
                  <div key={field.name} className="space-y-2">
                    <Label htmlFor={`field-${field.name}`} className="text-sm font-medium">
                      {field.label}
                    </Label>
                    <Input
                      id={`field-${field.name}`}
                      type={field.type}
                      value={extraFields[field.name] || ''}
                      onChange={(e) => setExtraFields(prev => ({ ...prev, [field.name]: e.target.value }))}
                      placeholder={field.placeholder}
                      disabled={submitting}
                    />
                  </div>
                ))}

                {/* トークン入力 */}
                <div className="space-y-2">
                  <Label htmlFor="token-input" className="text-sm font-medium">
                    {selectedAuthConfig?.authLabel || "APIトークン"}
                  </Label>
                  <Input
                    id="token-input"
                    type="password"
                    value={tokenInput}
                    onChange={(e) => setTokenInput(e.target.value)}
                    placeholder="トークンを入力..."
                    disabled={submitting}
                  />
                  {selectedAuthConfig?.helpText && (
                    <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                      <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                      <p className="text-xs text-muted-foreground">
                        {selectedAuthConfig.helpText}
                      </p>
                    </div>
                  )}
                </div>
              </div>

              <DialogFooter>
                <Button variant="ghost" onClick={() => setConnectDialog(null)}>
                  キャンセル
                </Button>
                <Button
                  onClick={handleConnectSubmit}
                  disabled={!tokenInput || submitting}
                >
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
              {disconnectDialog && services.find((s) => s.id === disconnectDialog)?.name}
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
