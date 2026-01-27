"use client"

import { useState, useEffect, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useAuth } from "@/lib/auth-context"
import { useAppearance, accentColors } from "@/lib/appearance-context"
import {
  Copy,
  Check,
  Server,
  Play,
  CheckCircle2,
  XCircle,
  Loader2,
  Key,
  Plus,
  Trash2,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  LogIn,
} from "lucide-react"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import {
  listApiKeys,
  generateApiKey,
  ApiKeyError,
  type ApiKey,
  type GenerateApiKeyResult,
} from "@/lib/api-keys"
import {
  listOAuthConsents,
  revokeOAuthConsent,
  OAuthConsentError,
  type OAuthConsent,
} from "@/lib/oauth-consents"
import { revokeApiKeyAction } from "./actions"

type VerifyStep = {
  name: string
  status: "pending" | "running" | "success" | "error"
  message?: string
  responseJson?: unknown
  responseSize?: number
}

type LogEntry = {
  timestamp: Date
  message: string
}

export default function McpConnectionPage() {
  const { user } = useAuth()
  const { accentColor } = useAppearance()
  const accentPreview = accentColors.find((c) => c.id === accentColor)?.preview ?? "#22c55e"
  const [copied, setCopied] = useState<string | null>(null)

  // API Key management state
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])
  const [keysLoading, setKeysLoading] = useState(true)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deleteDialogKey, setDeleteDialogKey] = useState<ApiKey | null>(null)
  const [createdKey, setCreatedKey] = useState<GenerateApiKeyResult | null>(null)
  const [keyCopied, setKeyCopied] = useState(false)

  // Create form state
  const [keyName, setKeyName] = useState("")
  const [expiration, setExpiration] = useState<string>("never")
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // OAuth Consents state
  const [oauthConsents, setOAuthConsents] = useState<OAuthConsent[]>([])
  const [consentsLoading, setConsentsLoading] = useState(true)
  const [revokeDialogConsent, setRevokeDialogConsent] = useState<OAuthConsent | null>(null)
  const [revoking, setRevoking] = useState(false)

  // API Key test state
  const [mcpServerUrl] = useState(process.env.NEXT_PUBLIC_MCP_SERVER_URL || "http://mcp.localhost")
  const [isVerifying, setIsVerifying] = useState(false)
  const [verifySteps, setVerifySteps] = useState<VerifyStep[]>([])
  const [testApiKey, setTestApiKey] = useState<string>("")
  const [testLogs, setTestLogs] = useState<LogEntry[]>([])
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set())

  // MCPエンドポイント
  const mcpBaseUrl = process.env.NEXT_PUBLIC_MCP_SERVER_URL || "http://localhost:8787"
  const endpoint = `${mcpBaseUrl}/mcp`

  const handleCopy = async (text: string, type: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(type)
    setTimeout(() => setCopied(null), 2000)
  }

  // Load API Keys
  const loadApiKeys = useCallback(async () => {
    try {
      const keys = await listApiKeys()
      setApiKeys(keys)
    } catch (error) {
      if (error instanceof ApiKeyError) {
        console.error("Failed to load API keys:", error.message)
      }
    } finally {
      setKeysLoading(false)
    }
  }, [])

  // Load OAuth Consents
  const loadOAuthConsents = useCallback(async () => {
    try {
      const consents = await listOAuthConsents()
      setOAuthConsents(consents)
    } catch (error) {
      if (error instanceof OAuthConsentError) {
        console.error("Failed to load OAuth consents:", error.message)
      }
    } finally {
      setConsentsLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) {
      loadApiKeys()
      loadOAuthConsents()
    } else {
      setKeysLoading(false)
      setConsentsLoading(false)
    }
  }, [user, loadApiKeys, loadOAuthConsents])

  // Create API Key
  const handleCreate = async () => {
    if (!keyName.trim()) {
      toast.error("キー名を入力してください")
      return
    }

    setCreating(true)
    try {
      const expiresInDays =
        expiration === "never"
          ? null
          : expiration === "30"
            ? 30
            : expiration === "90"
              ? 90
              : 365

      const result = await generateApiKey(keyName.trim(), expiresInDays)
      setCreatedKey(result)
      await loadApiKeys()
      setCreateDialogOpen(false)
      setKeyName("")
      setExpiration("never")
    } catch (error) {
      if (error instanceof ApiKeyError) {
        toast.error(`キーの作成に失敗しました: ${error.message}`)
      } else {
        toast.error("キーの作成に失敗しました")
      }
    } finally {
      setCreating(false)
    }
  }

  // Delete API Key
  const handleDelete = async () => {
    if (!deleteDialogKey) return

    setDeleting(true)
    try {
      const result = await revokeApiKeyAction(deleteDialogKey.id)
      if (result.success) {
        toast.success("APIキーを削除しました")
        await loadApiKeys()
        setDeleteDialogKey(null)
      } else {
        toast.error(`削除に失敗しました: ${result.error}`)
      }
    } catch {
      toast.error("削除に失敗しました")
    } finally {
      setDeleting(false)
    }
  }

  const handleCopyKey = async (key: string) => {
    await navigator.clipboard.writeText(key)
    setKeyCopied(true)
    toast.success("APIキーをコピーしました")
    setTimeout(() => setKeyCopied(false), 2000)
  }

  // Revoke OAuth Consent
  const handleRevokeConsent = async () => {
    if (!revokeDialogConsent) return

    setRevoking(true)
    try {
      const revoked = await revokeOAuthConsent(revokeDialogConsent.id)
      if (revoked) {
        toast.success("セッションを取り消しました")
        await loadOAuthConsents()
        setRevokeDialogConsent(null)
      } else {
        toast.error("取り消しに失敗しました")
      }
    } catch (error) {
      if (error instanceof OAuthConsentError) {
        toast.error(`取り消しに失敗しました: ${error.message}`)
      } else {
        toast.error("取り消しに失敗しました")
      }
    } finally {
      setRevoking(false)
    }
  }

  const formatDate = (dateString: string | null) => {
    if (!dateString) return "なし"
    return new Date(dateString).toLocaleDateString("ja-JP", {
      year: "numeric",
      month: "short",
      day: "numeric",
    })
  }

  const formatLastUsed = (dateString: string | null) => {
    if (!dateString) return "未使用"
    return new Date(dateString).toLocaleDateString("ja-JP", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  // Helper function to add log entry
  const addLog = useCallback((message: string) => {
    setTestLogs((prev) => [...prev, { timestamp: new Date(), message }])
  }, [])

  // Helper function to calculate response size
  const getResponseSize = (data: unknown): number => {
    return new TextEncoder().encode(JSON.stringify(data)).length
  }

  // API Key Connection Test
  const testApiKeyConnection = async () => {
    if (!testApiKey) return

    setIsVerifying(true)
    setTestLogs([])
    setExpandedSteps(new Set())
    setVerifySteps([
      { name: "initialize", status: "pending" },
      { name: "tools/list", status: "pending" },
    ])

    const mcpEndpoint = `${mcpServerUrl}/mcp`
    addLog(`テスト開始 ${mcpEndpoint}`)

    // Step 1: Initialize
    updateStep(0, { status: "running" })
    addLog("リクエスト送信 initialize")
    try {
      const initPayload = {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {
          protocolVersion: "2025-03-26",
          capabilities: {},
          clientInfo: { name: "MCPist Console", version: "1.0.0" },
        },
      }

      const response = await fetch(mcpEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testApiKey}`,
        },
        body: JSON.stringify(initPayload),
      })

      if (response.status === 401) {
        addLog("認証失敗 HTTP 401")
        let errorBody: unknown = null
        let errorSize = 0
        try {
          errorBody = await response.json()
          errorSize = getResponseSize(errorBody)
          addLog(`レスポンス受信 ${errorSize} bytes`)
        } catch {
          // Response might not be JSON
        }
        updateStep(0, {
          status: "error",
          message: "認証失敗 (401)",
          responseJson: errorBody,
          responseSize: errorBody ? errorSize : undefined,
        })
        setIsVerifying(false)
        return
      }

      if (!response.ok) {
        addLog(`HTTPエラー ${response.status}`)
        let errorBody: unknown = null
        let errorSize = 0
        try {
          errorBody = await response.json()
          errorSize = getResponseSize(errorBody)
          addLog(`レスポンス受信 ${errorSize} bytes`)
        } catch {
          // Response might not be JSON
        }
        updateStep(0, {
          status: "error",
          message: `HTTP ${response.status}`,
          responseJson: errorBody,
          responseSize: errorBody ? errorSize : undefined,
        })
        setIsVerifying(false)
        return
      }

      const initData = await response.json()
      const initSize = getResponseSize(initData)
      addLog(`レスポンス受信 ${initSize} bytes`)

      if (initData.result) {
        updateStep(0, {
          status: "success",
          message: `v${initData.result.protocolVersion}`,
          responseJson: initData,
          responseSize: initSize,
        })
        addLog(`initialize成功 v${initData.result.protocolVersion}`)
      } else if (initData.error) {
        updateStep(0, {
          status: "error",
          message: initData.error.message,
          responseJson: initData,
          responseSize: initSize,
        })
        addLog(`initializeエラー ${initData.error.message}`)
        setIsVerifying(false)
        return
      }

      // Step 2: Get tools/list
      updateStep(1, { status: "running" })
      addLog("リクエスト送信 tools/list")
      const toolsRes = await fetch(mcpEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testApiKey}`,
        },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 2,
          method: "tools/list",
        }),
      })

      const toolsData = await toolsRes.json()
      const toolsSize = getResponseSize(toolsData)
      addLog(`レスポンス受信 ${toolsSize} bytes`)

      if (toolsData.result) {
        const toolCount = toolsData.result.tools?.length || 0
        updateStep(1, {
          status: "success",
          message: `${toolCount} tools`,
          responseJson: toolsData,
          responseSize: toolsSize,
        })
        addLog(`tools/list成功 ${toolCount}ツール利用可能`)
      } else if (toolsData.error) {
        updateStep(1, {
          status: "error",
          message: toolsData.error.message,
          responseJson: toolsData,
          responseSize: toolsSize,
        })
        addLog(`tools/listエラー ${toolsData.error.message}`)
      }

      addLog("テスト完了")
    } catch (error) {
      addLog(`接続エラー ${String(error)}`)
      updateStep(0, { status: "error", message: String(error) })
    }

    setIsVerifying(false)
  }

  const updateStep = useCallback((index: number, update: Partial<VerifyStep>) => {
    setVerifySteps((prev) => prev.map((step, i) => (i === index ? { ...step, ...update } : step)))
  }, [])

  return (
    <div className="p-6 space-y-6">
      {/* Auth Method Tabs - wraps everything */}
      <Tabs defaultValue="oauth" className="space-y-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-foreground">MCP接続</h1>
            <p className="text-muted-foreground mt-1">
              MCPクライアントからの接続設定
            </p>
          </div>
          <TabsList className="bg-background p-1" style={{ borderWidth: 1, borderStyle: "solid", borderColor: accentPreview }}>
            <TabsTrigger
              value="oauth"
              className="gap-2 px-4"
            >
              <LogIn className="h-4 w-4" />
              OAuth
            </TabsTrigger>
            <TabsTrigger
              value="api-key"
              className="gap-2 px-4"
            >
              <Key className="h-4 w-4" />
              APIキー
            </TabsTrigger>
          </TabsList>
          <style>{`
            [data-slot="tabs-trigger"][data-state="active"] {
              background-color: ${accentPreview} !important;
              color: white !important;
            }
          `}</style>
        </div>

        {/* API Key Authentication Tab */}
        <TabsContent value="api-key" className="space-y-6 mt-0">
          {/* API Keys Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Key className="h-5 w-5" />
                APIキー
              </CardTitle>
              <CardDescription>
                Claude Code、Cursor などの MCP クライアントで使用する認証キー
              </CardDescription>
            </div>
            <Button size="sm" onClick={() => setCreateDialogOpen(true)}>
              <Plus className="h-4 w-4 mr-2" />
              新規作成
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {keysLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : apiKeys.length === 0 ? (
            <div className="text-center py-8">
              <Key className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
              <p className="text-sm text-muted-foreground mb-3">APIキーがありません</p>
              <Button size="sm" onClick={() => setCreateDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                APIキーを作成
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              {apiKeys.map((apiKey) => (
                <div
                  key={apiKey.id}
                  className={cn(
                    "flex items-center justify-between p-3 rounded-lg border",
                    apiKey.is_expired && "border-warning/50 bg-warning/5"
                  )}
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                      <Key className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{apiKey.name}</span>
                        {apiKey.is_expired && (
                          <Badge
                            variant="outline"
                            className="bg-warning/20 text-warning border-warning/30 text-xs"
                          >
                            <AlertTriangle className="h-3 w-3 mr-1" />
                            期限切れ
                          </Badge>
                        )}
                      </div>
                      <p className="text-xs font-mono text-muted-foreground">{apiKey.key_prefix}</p>
                      <div className="flex gap-3 text-xs text-muted-foreground mt-1">
                        <span>作成: {formatDate(apiKey.created_at)}</span>
                        <span>最終使用: {formatLastUsed(apiKey.last_used_at)}</span>
                      </div>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                    onClick={() => setDeleteDialogKey(apiKey)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Connection Settings Card (with integrated test) */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                接続設定
              </CardTitle>
              <CardDescription>
                Claude CodeやCursorなどのMCPクライアントで使用する設定例
              </CardDescription>
            </div>
            <Button
              variant={testApiKey ? "default" : "outline"}
              size="sm"
              onClick={testApiKeyConnection}
              disabled={isVerifying || !testApiKey}
            >
              {isVerifying ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  テスト中...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  テスト実行
                </>
              )}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="bg-secondary rounded-lg p-4 overflow-x-auto">
            <pre className="text-sm font-mono text-foreground whitespace-pre">
{`{
  "mcpServers": {
    "mcpist": {
      "url": "${endpoint}",
      "headers": {
        "Authorization": "Bearer `}<Input
              value={testApiKey}
              onChange={(e) => setTestApiKey(e.target.value)}
              placeholder="<your-api-key>"
              className="font-mono text-sm h-6 px-2 py-0 inline-flex w-72 bg-primary/10 border-primary/30 focus:border-primary focus:ring-primary/20"
            />{`"
      }
    }
  }
}`}
            </pre>
          </div>

          {/* Result Badge */}
          {verifySteps.length === 2 && verifySteps.every((step) => step.status === "success") && (
            <div className="flex justify-center">
              <Badge className="bg-green-500/20 text-green-600 border-green-500/30 gap-1.5 py-1.5 px-4">
                <CheckCircle2 className="h-4 w-4" />
                利用可能
              </Badge>
            </div>
          )}
          {verifySteps.length > 0 && verifySteps.some((step) => step.status === "error") && (
            <div className="flex justify-center">
              <Badge className="bg-destructive/20 text-destructive border-destructive/30 gap-1.5 py-1.5 px-4">
                <XCircle className="h-4 w-4" />
                利用不可
              </Badge>
            </div>
          )}

          {/* Test Steps */}
          {verifySteps.length > 0 && (
            <div className="space-y-2">
              {verifySteps.map((step, index) => {
                const isExpanded = expandedSteps.has(index)
                const hasResponse = step.responseJson !== undefined

                const toggleExpand = () => {
                  setExpandedSteps((prev) => {
                    const next = new Set(prev)
                    if (next.has(index)) {
                      next.delete(index)
                    } else {
                      next.add(index)
                    }
                    return next
                  })
                }

                return (
                  <div key={index} className="space-y-1">
                    <div
                      className={cn(
                        "flex items-center gap-2 p-2 rounded-lg text-sm",
                        step.status === "success" && "bg-green-500/10",
                        step.status === "error" && "bg-destructive/10",
                        step.status === "running" && "bg-primary/10"
                      )}
                    >
                      {step.status === "pending" && (
                        <div className="h-4 w-4 rounded-full border-2 border-muted" />
                      )}
                      {step.status === "running" && (
                        <Loader2 className="h-4 w-4 animate-spin text-primary" />
                      )}
                      {step.status === "success" && (
                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                      )}
                      {step.status === "error" && <XCircle className="h-4 w-4 text-destructive" />}
                      <span className="flex-1">{step.name}</span>
                      {step.responseSize !== undefined && (
                        <span className="text-xs text-muted-foreground">
                          {step.responseSize} bytes
                        </span>
                      )}
                      {step.message && (
                        <span className="text-xs text-muted-foreground">{step.message}</span>
                      )}
                    </div>
                    {hasResponse && (
                      <Collapsible open={isExpanded} onOpenChange={toggleExpand}>
                        <CollapsibleTrigger asChild>
                          <button className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground ml-6 py-1">
                            {isExpanded ? (
                              <ChevronDown className="h-3 w-3" />
                            ) : (
                              <ChevronRight className="h-3 w-3" />
                            )}
                            Response JSON
                          </button>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <div className="ml-6 mt-1 p-2 bg-secondary rounded-lg overflow-x-auto">
                            <pre className="text-xs font-mono text-foreground whitespace-pre">
                              {JSON.stringify(step.responseJson, null, 2)}
                            </pre>
                          </div>
                        </CollapsibleContent>
                      </Collapsible>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {/* Logs Panel */}
          {testLogs.length > 0 && (
            <div className="mt-4">
              <div className="text-xs font-medium text-muted-foreground mb-2">ログ</div>
              <div className="bg-secondary rounded-lg p-3 max-h-48 overflow-y-auto">
                <div className="space-y-1">
                  {testLogs.map((log, index) => (
                    <div key={index} className="flex gap-2 text-xs font-mono">
                      <span className="text-muted-foreground shrink-0">
                        {log.timestamp.toLocaleTimeString("ja-JP", {
                          hour: "2-digit",
                          minute: "2-digit",
                          second: "2-digit",
                          fractionalSecondDigits: 3,
                        })}
                      </span>
                      <span className="text-foreground">{log.message}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
        </TabsContent>

        {/* OAuth Authentication Tab */}
        <TabsContent value="oauth" className="space-y-6 mt-0">
          {/* Remote MCP Endpoint Card */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                リモートMCP エンドポイント
              </CardTitle>
              <CardDescription>
                Claude DesktopなどのOAuth対応MCPクライアントで指定するURL
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <div className="flex-1 bg-secondary rounded-lg px-4 py-3 font-mono text-sm">
                  {endpoint}
                </div>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => handleCopy(endpoint, "oauth-endpoint")}
                >
                  {copied === "oauth-endpoint" ? (
                    <Check className="h-4 w-4 text-green-500" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground mt-3">
                MCPクライアントでこのURLを指定すると、OAuth認証フローが開始されます。
              </p>
            </CardContent>
          </Card>

          {/* Active Sessions Card */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <LogIn className="h-5 w-5" />
                認可済みクライアント
              </CardTitle>
              <CardDescription>
                OAuth認証で接続を許可したMCPクライアント
              </CardDescription>
            </CardHeader>
            <CardContent>
              {consentsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : oauthConsents.length === 0 ? (
                <div className="text-center py-8">
                  <LogIn className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
                  <p className="text-sm text-muted-foreground mb-2">認可済みのクライアントはありません</p>
                  <p className="text-xs text-muted-foreground max-w-md mx-auto">
                    MCPクライアントからOAuth認証で接続すると、ここに表示されます。
                  </p>
                </div>
              ) : (
                <div className="space-y-3">
                  {oauthConsents.map((consent) => (
                    <div
                      key={consent.id}
                      className="flex items-center justify-between p-3 rounded-lg border"
                    >
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                          <Server className="h-4 w-4 text-muted-foreground" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-sm">
                              {consent.client_name || "Unknown Client"}
                            </span>
                            <Badge variant="outline" className="text-xs bg-green-500/10 text-green-600 border-green-500/30">
                              認可済み
                            </Badge>
                          </div>
                          <div className="flex gap-3 text-xs text-muted-foreground mt-1">
                            <span>スコープ: {consent.scopes}</span>
                            <span>認可日: {formatDate(consent.granted_at)}</span>
                          </div>
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => setRevokeDialogConsent(consent)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

        </TabsContent>
      </Tabs>

      {/* Create API Key Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>APIキーを作成</DialogTitle>
            <DialogDescription>
              新しい API キーを発行します。キーは作成時にのみ表示されます。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="key-name">キー名</Label>
              <Input
                id="key-name"
                placeholder="例: Claude Code"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                disabled={creating}
              />
            </div>
            <div className="space-y-2">
              <Label>有効期限</Label>
              <RadioGroup value={expiration} onValueChange={setExpiration} disabled={creating}>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="never" id="never" />
                  <Label htmlFor="never" className="font-normal">
                    無期限
                  </Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="30" id="30days" />
                  <Label htmlFor="30days" className="font-normal">
                    30日
                  </Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="90" id="90days" />
                  <Label htmlFor="90days" className="font-normal">
                    90日
                  </Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="365" id="1year" />
                  <Label htmlFor="1year" className="font-normal">
                    1年
                  </Label>
                </div>
              </RadioGroup>
            </div>
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setCreateDialogOpen(false)} disabled={creating}>
              キャンセル
            </Button>
            <Button onClick={handleCreate} disabled={creating || !keyName.trim()}>
              {creating ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  作成中...
                </>
              ) : (
                "作成"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Created Key Dialog */}
      <Dialog open={!!createdKey} onOpenChange={(open) => !open && setCreatedKey(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Check className="h-5 w-5 text-green-500" />
              APIキーを作成しました
            </DialogTitle>
            <DialogDescription>
              このキーは一度しか表示されません。安全な場所に保存してください。
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <div className="space-y-2">
              <Label>キー名</Label>
              <p className="text-foreground font-medium">{createdKey?.name}</p>
            </div>
            <div className="space-y-2 mt-4">
              <Label>APIキー</Label>
              <div className="flex items-center gap-2">
                <Input value={createdKey?.key || ""} readOnly className="font-mono text-sm" />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => createdKey && handleCopyKey(createdKey.key)}
                >
                  {keyCopied ? (
                    <Check className="h-4 w-4 text-green-500" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>
            <div className="mt-4 p-3 bg-warning/10 rounded-lg flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 text-warning mt-0.5 shrink-0" />
              <p className="text-xs text-warning">
                このキーは二度と表示されません。今すぐコピーして安全に保管してください。
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setCreatedKey(null)}>完了</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete API Key Confirmation Dialog */}
      <AlertDialog
        open={!!deleteDialogKey}
        onOpenChange={(open) => !open && setDeleteDialogKey(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>APIキーを削除しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              「{deleteDialogKey?.name}」を削除します。このキーを使用しているアプリケーションは
              MCPist に接続できなくなります。この操作は取り消せません。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>キャンセル</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  削除中...
                </>
              ) : (
                "削除"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Revoke OAuth Consent Confirmation Dialog */}
      <AlertDialog
        open={!!revokeDialogConsent}
        onOpenChange={(open) => !open && setRevokeDialogConsent(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>認可を取り消しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              「{revokeDialogConsent?.client_name || "Unknown Client"}」の認可を取り消します。
              このクライアントは再度認証が必要になります。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revoking}>キャンセル</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleRevokeConsent}
              disabled={revoking}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {revoking ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  取り消し中...
                </>
              ) : (
                "取り消し"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
