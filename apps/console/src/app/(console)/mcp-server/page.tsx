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
import { useAuth } from "@/lib/auth/auth-context"
import {
  Copy,
  Check,
  Server,
  Globe,
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
  Image as ImageIcon,
  BookOpen,
} from "lucide-react"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import {
  listApiKeys,
  generateApiKey,
  type ApiKey,
  type GenerateApiKeyResult,
} from "@/lib/mcp/api-keys"
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

// モジュールレベルキャッシュ
let cachedApiKeys: ApiKey[] | null = null

export default function McpConnectionPage() {
  const { user } = useAuth()
  const accentPreview = "#d07850"
  const [copied, setCopied] = useState<string | null>(null)

  // API Key management state
  const hasCached = cachedApiKeys !== null
  const [apiKeys, setApiKeys] = useState<ApiKey[]>(cachedApiKeys ?? [])
  const [keysLoading, setKeysLoading] = useState(!hasCached)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deleteDialogKey, setDeleteDialogKey] = useState<ApiKey | null>(null)
  const [createdKey, setCreatedKey] = useState<(GenerateApiKeyResult & { display_name: string }) | null>(null)
  const [keyCopied, setKeyCopied] = useState(false)

  // Create form state
  const [keyName, setKeyName] = useState("")
  const [expiration, setExpiration] = useState<string>("90")
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // Setup guide state
  const [guideClient, setGuideClient] = useState<"claude" | "chatgpt">("claude")

  // API Key test state
  const mcpServerUrl = process.env.NEXT_PUBLIC_MCP_SERVER_URL!
  const [isVerifying, setIsVerifying] = useState(false)
  const [verifySteps, setVerifySteps] = useState<VerifyStep[]>([])
  const [testApiKey, setTestApiKey] = useState<string>("")
  const [testLogs, setTestLogs] = useState<LogEntry[]>([])
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set())

  // MCPエンドポイント
  const endpoint = `${mcpServerUrl}/v1/mcp`

  const handleCopy = async (text: string, type: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(type)
    setTimeout(() => setCopied(null), 2000)
  }

  // Load API Keys
  const loadApiKeys = useCallback(async () => {
    try {
      const keys = await listApiKeys()
      cachedApiKeys = keys
      setApiKeys(keys)
    } catch (error) {
      if (error instanceof Error) {
        console.error("Failed to load API keys:", error.message)
      }
    } finally {
      setKeysLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) {
      loadApiKeys()
    } else {
      setKeysLoading(false)
    }
  }, [user, loadApiKeys])

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
      setCreatedKey({ ...result, display_name: keyName.trim() })
      await loadApiKeys()
      setCreateDialogOpen(false)
      setKeyName("")
      setExpiration("90")
    } catch (error) {
      if (error instanceof Error) {
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

    const mcpEndpoint = `${mcpServerUrl}/v1/mcp`
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
        <div className="flex flex-wrap items-center gap-4 pl-8 md:pl-0">
          <h1 className="text-2xl font-bold text-foreground">MCPサーバー</h1>
          <div className="ml-auto">
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
          </div>
          <style>{`
            [data-slot="tabs-trigger"][data-state="active"] {
              background-color: ${accentPreview} !important;
              color: white !important;
            }
          `}</style>
        </div>
        <p className="text-sm text-muted-foreground -mt-4">
          MCPクライアントへの接続を管理
        </p>

        {/* API Key Authentication Tab */}
        <TabsContent value="api-key" className="space-y-6 mt-0">
          {/* API Keys Card */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-4">
            <CardTitle className="flex items-center gap-2">
              <Key className="h-5 w-5" style={{ color: accentPreview }} />
              APIキー
            </CardTitle>
            <div className="ml-auto">
              <Button size="sm" onClick={() => setCreateDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                新規作成
              </Button>
            </div>
          </div>
          <CardDescription>
            Claude Code、Cursor などの MCP クライアントで使用する認証キー
          </CardDescription>
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
              {apiKeys.map((apiKey) => {
                const isExpired = apiKey.expires_at && new Date(apiKey.expires_at) < new Date()
                const isRevoked = !!apiKey.revoked_at
                const daysUntilExpiry = apiKey.expires_at
                  ? Math.ceil((new Date(apiKey.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
                  : null
                const isExpiringSoon = daysUntilExpiry !== null && daysUntilExpiry > 0 && daysUntilExpiry <= 14
                return (
                <div
                  key={apiKey.id}
                  className={cn(
                    "p-3 rounded-lg border overflow-hidden",
                    (isExpired || isRevoked || isExpiringSoon) && "border-warning/50 bg-warning/5"
                  )}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <div className="w-7 h-7 rounded-md bg-secondary flex items-center justify-center shrink-0">
                        <Key className="h-3.5 w-3.5 text-muted-foreground" />
                      </div>
                      <span className="font-medium text-sm truncate">{apiKey.display_name}</span>
                      {isExpired && (
                        <Badge
                          variant="outline"
                          className="bg-warning/20 text-warning border-warning/30 text-xs shrink-0"
                        >
                          <AlertTriangle className="h-3 w-3 mr-1" />
                          期限切れ
                        </Badge>
                      )}
                      {isRevoked && (
                        <Badge
                          variant="outline"
                          className="bg-destructive/20 text-destructive border-destructive/30 text-xs shrink-0"
                        >
                          無効化済み
                        </Badge>
                      )}
                      {isExpiringSoon && (
                        <Badge
                          variant="outline"
                          className="bg-warning/20 text-warning border-warning/30 text-xs shrink-0"
                        >
                          <AlertTriangle className="h-3 w-3 mr-1" />
                          {daysUntilExpiry}日で期限切れ
                        </Badge>
                      )}
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="shrink-0 h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => setDeleteDialogKey(apiKey)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                  <div className="mt-1.5 pl-9 text-xs text-muted-foreground space-y-0.5">
                    <p className="font-mono truncate">{apiKey.key_prefix}</p>
                    <div className="flex flex-wrap gap-x-3">
                      {apiKey.expires_at && <span>有効期限: {formatDate(apiKey.expires_at)}</span>}
                      <span>最終使用: {formatLastUsed(apiKey.last_used_at ?? null)}</span>
                    </div>
                  </div>
                </div>
              )})}

            </div>
          )}
        </CardContent>
      </Card>

      {/* Connection Settings Card (with integrated test) */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-4">
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" style={{ color: accentPreview }} />
              接続設定
            </CardTitle>
            <div className="ml-auto">
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
          </div>
          <CardDescription>
            Claude CodeやCursorなどのMCPクライアントで使用する設定例
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="bg-secondary rounded-lg p-4 overflow-x-auto">
            <pre className="text-sm font-mono text-foreground whitespace-pre">
{`{
  "mcpServers": {
    "mcpist": {
      "type": "http",
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
              <Badge className="bg-success/20 text-success border-success/30 gap-1.5 py-1.5 px-4">
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
                        step.status === "success" && "bg-success/10",
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
                        <CheckCircle2 className="h-4 w-4 text-success" />
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
                <Globe className="h-5 w-5" style={{ color: accentPreview }} />
                エンドポイント
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <div className="flex-1 min-w-0 bg-secondary rounded-lg px-4 py-3 font-mono text-sm break-all">
                  {endpoint}
                </div>
                <Button
                  variant="outline"
                  size="icon"
                  className="shrink-0"
                  onClick={() => handleCopy(endpoint, "oauth-endpoint")}
                >
                  {copied === "oauth-endpoint" ? (
                    <Check className="h-4 w-4 text-success" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground mt-3">
                コネクタでこのURLを指定して、OAuth認可フローを実行すると、MCPistに接続できます。
              </p>
            </CardContent>
          </Card>

        </TabsContent>
      </Tabs>

      {/* Setup Guide */}
      <Card className="mt-8">
        <CardHeader>
          <div className="flex flex-wrap items-center gap-4">
            <CardTitle className="flex items-center gap-2 text-lg">
              <BookOpen className="h-5 w-5" style={{ color: accentPreview }} />
              セットアップガイド
            </CardTitle>
            <div className="ml-auto">
              <Select value={guideClient} onValueChange={(v) => setGuideClient(v as "claude" | "chatgpt")}>
                <SelectTrigger className="w-full sm:w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="claude">Claude Desktop</SelectItem>
                  <SelectItem value="chatgpt">ChatGPT</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-8">
          {guideClient === "claude" ? (
            <div className="space-y-4">
              <ol className="space-y-6 list-none">
                <li className="space-y-2">
                  <p className="font-medium">
                    <span className="inline-flex w-6 h-6 rounded-full bg-primary/10 text-primary items-center justify-center text-xs font-bold mr-2 align-middle">1</span>
                    設定画面を開く
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Claude Desktop の設定 →「Integrations」を開きます。
                  </p>
                  <div className="rounded-lg border border-dashed border-muted-foreground/30 bg-muted/30 h-48 flex items-center justify-center">
                    <div className="text-center text-muted-foreground/50">
                      <ImageIcon className="h-8 w-8 mx-auto mb-2" />
                      <p className="text-xs">スクリーンショット（準備中）</p>
                    </div>
                  </div>
                </li>
                <li className="space-y-2">
                  <p className="font-medium">
                    <span className="inline-flex w-6 h-6 rounded-full bg-primary/10 text-primary items-center justify-center text-xs font-bold mr-2 align-middle">2</span>
                    MCPistのURLを追加
                  </p>
                  <p className="text-sm text-muted-foreground">
                    「Add more integrations」→「Add custom MCP server」を選び、上記のエンドポイント URL を入力します。
                  </p>
                  <div className="rounded-lg border border-dashed border-muted-foreground/30 bg-muted/30 h-48 flex items-center justify-center">
                    <div className="text-center text-muted-foreground/50">
                      <ImageIcon className="h-8 w-8 mx-auto mb-2" />
                      <p className="text-xs">スクリーンショット（準備中）</p>
                    </div>
                  </div>
                </li>
                <li className="space-y-2">
                  <p className="font-medium">
                    <span className="inline-flex w-6 h-6 rounded-full bg-primary/10 text-primary items-center justify-center text-xs font-bold mr-2 align-middle">3</span>
                    OAuth認可を完了
                  </p>
                  <p className="text-sm text-muted-foreground">
                    ブラウザが開きます。MCPistアカウントでログインし、アクセスを許可すると接続完了です。
                  </p>
                  <div className="rounded-lg border border-dashed border-muted-foreground/30 bg-muted/30 h-48 flex items-center justify-center">
                    <div className="text-center text-muted-foreground/50">
                      <ImageIcon className="h-8 w-8 mx-auto mb-2" />
                      <p className="text-xs">スクリーンショット（準備中）</p>
                    </div>
                  </div>
                </li>
              </ol>
            </div>
          ) : (
            <div className="space-y-4">
              <ol className="space-y-6 list-none">
                <li className="space-y-2">
                  <p className="font-medium">
                    <span className="inline-flex w-6 h-6 rounded-full bg-primary/10 text-primary items-center justify-center text-xs font-bold mr-2 align-middle">1</span>
                    チャット画面からMCPを追加
                  </p>
                  <p className="text-sm text-muted-foreground">
                    ChatGPTのチャット画面で、入力欄の下にあるツールアイコン →「Add more tools」→「Add MCP server (Streamable HTTP)」を選びます。
                  </p>
                  <div className="rounded-lg border border-dashed border-muted-foreground/30 bg-muted/30 h-48 flex items-center justify-center">
                    <div className="text-center text-muted-foreground/50">
                      <ImageIcon className="h-8 w-8 mx-auto mb-2" />
                      <p className="text-xs">スクリーンショット（準備中）</p>
                    </div>
                  </div>
                </li>
                <li className="space-y-2">
                  <p className="font-medium">
                    <span className="inline-flex w-6 h-6 rounded-full bg-primary/10 text-primary items-center justify-center text-xs font-bold mr-2 align-middle">2</span>
                    MCPistのURLを入力
                  </p>
                  <p className="text-sm text-muted-foreground">
                    サーバー URL 入力欄に上記のエンドポイント URL を貼り付け、「Add」をクリックします。
                  </p>
                  <div className="rounded-lg border border-dashed border-muted-foreground/30 bg-muted/30 h-48 flex items-center justify-center">
                    <div className="text-center text-muted-foreground/50">
                      <ImageIcon className="h-8 w-8 mx-auto mb-2" />
                      <p className="text-xs">スクリーンショット（準備中）</p>
                    </div>
                  </div>
                </li>
                <li className="space-y-2">
                  <p className="font-medium">
                    <span className="inline-flex w-6 h-6 rounded-full bg-primary/10 text-primary items-center justify-center text-xs font-bold mr-2 align-middle">3</span>
                    OAuth認可を完了
                  </p>
                  <p className="text-sm text-muted-foreground">
                    ブラウザが開きます。MCPistアカウントでログインし、アクセスを許可すると接続完了です。
                  </p>
                  <div className="rounded-lg border border-dashed border-muted-foreground/30 bg-muted/30 h-48 flex items-center justify-center">
                    <div className="text-center text-muted-foreground/50">
                      <ImageIcon className="h-8 w-8 mx-auto mb-2" />
                      <p className="text-xs">スクリーンショット（準備中）</p>
                    </div>
                  </div>
                </li>
              </ol>
            </div>
          )}
        </CardContent>
      </Card>

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
              <Check className="h-5 w-5 text-success" />
              APIキーを作成しました
            </DialogTitle>
            <DialogDescription>
              このキーは一度しか表示されません。安全な場所に保存してください。
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <div className="space-y-2">
              <Label>キー名</Label>
              <p className="text-foreground font-medium">{createdKey?.display_name}</p>
            </div>
            <div className="space-y-2 mt-4">
              <Label>APIキー</Label>
              <div className="flex items-center gap-2">
                <Input value={createdKey?.api_key || ""} readOnly className="font-mono text-sm" />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => createdKey && handleCopyKey(createdKey.api_key)}
                >
                  {keyCopied ? (
                    <Check className="h-4 w-4 text-success" />
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
              「{deleteDialogKey?.display_name}」を削除します。このキーを使用しているアプリケーションは
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

    </div>
  )
}
