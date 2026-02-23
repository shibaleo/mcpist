"use client"

import { useEffect, useState, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth/auth-context"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Users, Activity, Server, CreditCard, Play, Loader2, CheckCircle2, XCircle, ChevronDown, ChevronRight, Copy, Check, LogIn } from "lucide-react"
import { cn } from "@/lib/utils"
import { getOrRegisterOAuthClient } from "@/lib/oauth/client"
import {
  listAllOAuthConsents,
  type OAuthConsentAdmin,
} from "@/lib/oauth/consents"

type VerifyStep = {
  name: string
  status: "pending" | "running" | "success" | "error"
  message?: string
  response?: unknown
}

// モックデータ
const adminStats = {
  totalUsers: 42,
  activeConnections: 156,
  totalApiCalls: 12847,
  monthlyRevenue: 128000,
}

export default function AdminPage() {
  const { isAdmin, isLoading } = useAuth()
  const router = useRouter()

  // OAuth Consents state
  const [oauthConsents, setOAuthConsents] = useState<OAuthConsentAdmin[]>([])
  const [consentsLoading, setConsentsLoading] = useState(true)

  // Load OAuth Consents
  const loadOAuthConsents = useCallback(async () => {
    try {
      const consents = await listAllOAuthConsents()
      setOAuthConsents(consents)
    } catch (error) {
      console.error("Failed to load OAuth consents:", error instanceof Error ? error.message : error)
    } finally {
      setConsentsLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!isLoading && !isAdmin) {
      router.push("/dashboard")
    }
  }, [isLoading, isAdmin, router])

  useEffect(() => {
    if (isAdmin) {
      loadOAuthConsents()
    }
  }, [isAdmin, loadOAuthConsents])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (!isAdmin) {
    return null
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY" }).format(price)
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">管理者パネル</h1>
        <p className="text-muted-foreground mt-1">
          システム全体の統計情報を確認できます
        </p>
      </div>

      <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Users */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Users className="h-4 w-4" />
              登録ユーザー数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{adminStats.totalUsers}</div>
            <p className="text-xs text-muted-foreground mt-1">ユーザー</p>
          </CardContent>
        </Card>

        {/* Active Connections */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Server className="h-4 w-4" />
              アクティブ接続
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{adminStats.activeConnections}</div>
            <p className="text-xs text-muted-foreground mt-1">接続</p>
          </CardContent>
        </Card>

        {/* API Calls */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Activity className="h-4 w-4" />
              API呼び出し
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{adminStats.totalApiCalls.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1">今月</p>
          </CardContent>
        </Card>

        {/* Monthly Revenue */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <CreditCard className="h-4 w-4" />
              月間売上
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{formatPrice(adminStats.monthlyRevenue)}</div>
            <p className="text-xs text-muted-foreground mt-1">今月</p>
          </CardContent>
        </Card>
      </div>

      {/* OAuth Authentication Flow Verification */}
      <OAuthVerificationCard />

      {/* OAuth Consents List */}
      <OAuthConsentsCard
        consents={oauthConsents}
        loading={consentsLoading}
        onRefresh={loadOAuthConsents}
      />

      {/* Placeholder for future admin features */}
      <Card>
        <CardHeader>
          <CardTitle>管理機能</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            将来的にはユーザー管理、システム設定、ログ閲覧などの機能が追加される予定です。
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

// OAuth Consents List Card Component
function OAuthConsentsCard({
  consents,
  loading,
  onRefresh,
}: {
  consents: OAuthConsentAdmin[]
  loading: boolean
  onRefresh: () => void
}) {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("ja-JP", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <LogIn className="h-5 w-5" />
              OAuth認可済みクライアント
            </CardTitle>
            <CardDescription>
              全ユーザーのOAuth認可状況を確認できます
            </CardDescription>
          </div>
          <Button variant="outline" size="sm" onClick={onRefresh} disabled={loading}>
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              "更新"
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : consents.length === 0 ? (
          <div className="text-center py-8">
            <LogIn className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">認可済みのクライアントはありません</p>
          </div>
        ) : (
          <div className="space-y-3">
            {consents.map((consent) => (
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
                      <span>ユーザー: {consent.user_email || consent.user_id}</span>
                      <span>スコープ: {consent.scopes}</span>
                      <span>認可日: {formatDate(consent.granted_at)}</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// JSON Response Viewer Component
function JsonResponseViewer({ data, label }: { data: unknown; label: string }) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [copied, setCopied] = useState(false)

  const jsonString = JSON.stringify(data, null, 2)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(jsonString)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="mt-2 border rounded-lg overflow-hidden">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-2 bg-muted/50 hover:bg-muted text-left text-xs"
      >
        <span className="flex items-center gap-1 font-medium">
          {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          {label}
        </span>
        <span className="text-muted-foreground">
          {jsonString.length} bytes
        </span>
      </button>
      {isExpanded && (
        <div className="relative">
          <button
            onClick={handleCopy}
            className="absolute top-2 right-2 p-1 rounded bg-background/80 hover:bg-background border text-muted-foreground hover:text-foreground"
            title="Copy JSON"
          >
            {copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
          </button>
          <pre className="p-3 text-xs font-mono overflow-x-auto bg-secondary/50 max-h-64 overflow-y-auto">
            {jsonString}
          </pre>
        </div>
      )}
    </div>
  )
}

// OAuth Authentication Flow Verification Component
function OAuthVerificationCard() {
  const [mcpServerUrl, setMcpServerUrl] = useState(
    process.env.NEXT_PUBLIC_MCP_SERVER_URL!
  )
  const [isVerifying, setIsVerifying] = useState(false)
  const [verifySteps, setVerifySteps] = useState<VerifyStep[]>([])
  const [verifyLogs, setVerifyLogs] = useState<string[]>([])

  const addLog = useCallback((message: string) => {
    const timestamp = new Date().toLocaleTimeString()
    setVerifyLogs((prev) => [...prev, `[${timestamp}] ${message}`])
  }, [])

  const updateStep = useCallback((index: number, update: Partial<VerifyStep>) => {
    setVerifySteps((prev) =>
      prev.map((step, i) => (i === index ? { ...step, ...update } : step))
    )
  }, [])

  // PKCE helpers
  const generateRandomString = (length: number): string => {
    const array = new Uint8Array(length)
    crypto.getRandomValues(array)
    return Array.from(array, (byte) => byte.toString(16).padStart(2, '0')).join('')
  }

  const generateCodeChallenge = async (verifier: string): Promise<string> => {
    const encoder = new TextEncoder()
    const data = encoder.encode(verifier)
    const digest = await crypto.subtle.digest('SHA-256', data)
    return btoa(String.fromCharCode(...new Uint8Array(digest)))
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '')
  }

  // OAuth 2.1 + PKCE Authorization Flow
  const startOAuthFlow = async () => {
    setIsVerifying(true)
    setVerifyLogs([])
    setVerifySteps([
      { name: "MCP Server 接続", status: "pending" },
      { name: "Protected Resource メタデータ", status: "pending" },
      { name: "Authorization Server メタデータ", status: "pending" },
      { name: "認可リクエスト", status: "pending" },
    ])

    // Step 1: Try MCP Server (expect 401 or success)
    updateStep(0, { status: "running" })
    const oauthMcpEndpoint = `${mcpServerUrl}/v1/mcp`
    addLog(`Step 1: MCP Server (${oauthMcpEndpoint}) に接続試行...`)
    try {
      const response = await fetch(oauthMcpEndpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "initialize",
          params: {
            protocolVersion: "2025-03-26",
            capabilities: {},
            clientInfo: { name: "MCPist Console", version: "1.0.0" },
          },
        }),
      })

      if (response.status === 401) {
        const responseText = await response.text()
        let responseData: unknown = responseText
        try { responseData = JSON.parse(responseText) } catch { /* keep as text */ }
        updateStep(0, { status: "success", message: "401 (認証必要)", response: responseData })
        addLog("✓ 401 Unauthorized - 認証が必要です")
      } else if (response.ok) {
        const data = await response.json()
        if (data.result) {
          updateStep(0, { status: "success", message: "認証なしで接続可能", response: data })
          addLog("✓ 認証なしで接続成功 (開発モード)")
        }
      } else {
        throw new Error(`Status: ${response.status}`)
      }
    } catch (error) {
      updateStep(0, { status: "error", message: String(error) })
      addLog(`✗ 接続失敗: ${error}`)
      setIsVerifying(false)
      return
    }

    // Step 2: Get Protected Resource Metadata
    updateStep(1, { status: "running" })
    addLog("Step 2: Protected Resource メタデータを取得...")
    try {
      const response = await fetch("/.well-known/oauth-protected-resource")
      if (response.ok) {
        const metadata = await response.json()
        updateStep(1, { status: "success", message: metadata.resource, response: metadata })
        addLog(`✓ resource: ${metadata.resource}`)
        addLog(`  authorization_servers: ${metadata.authorization_servers.join(", ")}`)
      } else {
        throw new Error(`Status: ${response.status}`)
      }
    } catch (error) {
      updateStep(1, { status: "error", message: String(error) })
      addLog(`✗ 取得失敗: ${error}`)
      setIsVerifying(false)
      return
    }

    // Step 3: Get Authorization Server Metadata
    updateStep(2, { status: "running" })
    addLog("Step 3: Authorization Server メタデータを取得...")
    try {
      const response = await fetch("/.well-known/oauth-authorization-server")
      if (response.ok) {
        const metadata = await response.json()
        updateStep(2, { status: "success", message: metadata.issuer, response: metadata })
        addLog(`✓ issuer: ${metadata.issuer}`)
        addLog(`  authorization_endpoint: ${metadata.authorization_endpoint}`)
        addLog(`  token_endpoint: ${metadata.token_endpoint}`)
      } else {
        throw new Error(`Status: ${response.status}`)
      }
    } catch (error) {
      updateStep(2, { status: "error", message: String(error) })
      addLog(`✗ 取得失敗: ${error}`)
      setIsVerifying(false)
      return
    }

    // Step 4: Start Authorization Request
    updateStep(3, { status: "running" })
    addLog("Step 4: PKCE code_verifier / code_challenge を生成...")

    const verifier = generateRandomString(32)
    const challenge = await generateCodeChallenge(verifier)

    addLog(`  code_verifier: ${verifier.substring(0, 16)}...`)
    addLog(`  code_challenge: ${challenge.substring(0, 16)}...`)

    const state = generateRandomString(16)
    const redirectUri = `${window.location.origin}/oauth/callback`

    // Store in sessionStorage
    sessionStorage.setItem("oauth_state", state)
    sessionStorage.setItem("oauth_verifier", verifier)

    // Get or register OAuth client dynamically
    const clientId = await getOrRegisterOAuthClient()
    addLog(`  client_id: ${clientId}`)

    // Use the metadata variable directly since setState is async
    const authServerMetadata = await fetch("/.well-known/oauth-authorization-server").then(r => r.json())
    const authUrl = new URL(authServerMetadata.authorization_endpoint)
    authUrl.searchParams.set("response_type", "code")
    authUrl.searchParams.set("client_id", clientId)
    authUrl.searchParams.set("redirect_uri", redirectUri)
    authUrl.searchParams.set("scope", "openid profile email")
    authUrl.searchParams.set("code_challenge", challenge)
    authUrl.searchParams.set("code_challenge_method", "S256")
    authUrl.searchParams.set("state", state)

    addLog(`認可 URL: ${authUrl.toString().substring(0, 100)}...`)
    addLog("⏳ ブラウザで認証を行ってください...")

    updateStep(3, { status: "success", message: "認可フロー開始" })
    setIsVerifying(false)

    // Navigate to auth URL in same window
    window.location.href = authUrl.toString()
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Play className="h-5 w-5" />
          OAuth 認証フロー検証
        </CardTitle>
        <CardDescription>
          OAuth 2.1 + PKCE 認可フローを使用して MCP Server への接続を検証します
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-2">
          <Input
            value={mcpServerUrl}
            onChange={(e) => setMcpServerUrl(e.target.value)}
            placeholder="MCP Server URL"
            className="font-mono text-sm"
          />
          <Button onClick={startOAuthFlow} disabled={isVerifying}>
            {isVerifying ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                検証中...
              </>
            ) : (
              <>
                <Play className="h-4 w-4 mr-2" />
                接続の検証を開始
              </>
            )}
          </Button>
        </div>

        {/* Verification Steps */}
        {verifySteps.length > 0 && (
          <div className="space-y-2">
            {verifySteps.map((step, index) => (
              <div key={index}>
                <div
                  className={cn(
                    "flex items-center gap-2 p-2 rounded-lg",
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
                  {step.status === "error" && (
                    <XCircle className="h-4 w-4 text-destructive" />
                  )}
                  <span className="flex-1 text-sm">{step.name}</span>
                  {step.message && (
                    <span className="text-xs text-muted-foreground">
                      {step.message}
                    </span>
                  )}
                </div>
                {step.response !== undefined && (
                  <div className="ml-6">
                    <JsonResponseViewer data={step.response} label="Response JSON" />
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Logs */}
        {verifyLogs.length > 0 && (
          <div className="bg-secondary rounded-lg p-4 max-h-64 overflow-y-auto">
            <pre className="text-xs font-mono text-foreground whitespace-pre-wrap">
              {verifyLogs.join("\n")}
            </pre>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
