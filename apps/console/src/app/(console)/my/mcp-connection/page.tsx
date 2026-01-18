"use client"

import { useState, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { useAuth } from "@/lib/auth-context"
import { Copy, RefreshCw, Eye, EyeOff, Check, Server, Play, CheckCircle2, XCircle, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"

type VerifyStep = {
  name: string
  status: "pending" | "running" | "success" | "error"
  message?: string
}

export default function McpConnectionPage() {
  const { user, isAdmin } = useAuth()
  const [showToken, setShowToken] = useState(false)
  const [copied, setCopied] = useState<string | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [tokenStatus, setTokenStatus] = useState<"not_generated" | "active" | "revoked">("not_generated")

  // Verification state
  const [mcpServerUrl, setMcpServerUrl] = useState(process.env.NEXT_PUBLIC_MCP_SERVER_URL || "http://localhost:8089")
  const [isVerifying, setIsVerifying] = useState(false)
  const [verifySteps, setVerifySteps] = useState<VerifyStep[]>([])
  const [verifyLogs, setVerifyLogs] = useState<string[]>([])

  const endpoint = `https://mcp.mcpist.dev/u/${user?.id?.slice(0, 8) || "..."}`

  const handleGenerateToken = () => {
    const newToken = `mcp_usr_${Math.random().toString(36).substring(2, 15)}`
    setToken(newToken)
    setTokenStatus("active")
  }

  const handleRevokeToken = () => {
    setToken(null)
    setTokenStatus("revoked")
  }

  const handleCopy = async (text: string, type: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(type)
    setTimeout(() => setCopied(null), 2000)
  }

  const maskedToken = token ? `mcp_usr_${"*".repeat(20)}` : null

// OAuth state
  const [oauthMetadata, setOauthMetadata] = useState<{
    authorization_endpoint: string
    token_endpoint: string
    issuer: string
  } | null>(null)
  const [protectedResource, setProtectedResource] = useState<{
    resource: string
    authorization_servers: string[]
  } | null>(null)
  const [accessToken, setAccessToken] = useState<string | null>(null)
  const [codeVerifier, setCodeVerifier] = useState<string | null>(null)

  const addLog = useCallback((message: string) => {
    const timestamp = new Date().toLocaleTimeString()
    setVerifyLogs((prev) => [...prev, `[${timestamp}] ${message}`])
  }, [])

  const updateStep = useCallback((index: number, update: Partial<VerifyStep>) => {
    setVerifySteps((prev) =>
      prev.map((step, i) => (i === index ? { ...step, ...update } : step))
    )
  }, [])

  const verifyConnection = async () => {
    setIsVerifying(true)
    setVerifyLogs([])
    setVerifySteps([
      { name: "ヘルスチェック", status: "pending" },
      { name: "OAuth メタデータ取得", status: "pending" },
      { name: "MCP initialize", status: "pending" },
      { name: "tools/list 取得", status: "pending" },
    ])

    // Step 1: Health check
    updateStep(0, { status: "running" })
    addLog(`MCP Server (${mcpServerUrl}) にヘルスチェック...`)
    try {
      const healthRes = await fetch(`${mcpServerUrl}/health`)
      if (healthRes.ok) {
        updateStep(0, { status: "success", message: "OK" })
        addLog("✓ ヘルスチェック成功")
      } else {
        throw new Error(`Status: ${healthRes.status}`)
      }
    } catch (error) {
      updateStep(0, { status: "error", message: String(error) })
      addLog(`✗ ヘルスチェック失敗: ${error}`)
      setIsVerifying(false)
      return
    }

    // Step 2: OAuth metadata
    updateStep(1, { status: "running" })
    addLog("OAuth メタデータを取得...")
    try {
      const metadataRes = await fetch("/.well-known/oauth-authorization-server")
      if (metadataRes.ok) {
        const metadata = await metadataRes.json()
        updateStep(1, { status: "success", message: `issuer: ${metadata.issuer}` })
        addLog(`✓ OAuth メタデータ取得成功: issuer=${metadata.issuer}`)
      } else {
        throw new Error(`Status: ${metadataRes.status}`)
      }
    } catch (error) {
      updateStep(1, { status: "error", message: String(error) })
      addLog(`✗ OAuth メタデータ取得失敗: ${error}`)
      setIsVerifying(false)
      return
    }

    // Step 3: MCP initialize
    updateStep(2, { status: "running" })
    addLog("MCP initialize リクエスト...")
    try {
      const initRes = await fetch(`${mcpServerUrl}/mcp`, {
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
      const initData = await initRes.json()
      if (initData.result) {
        updateStep(2, { status: "success", message: `v${initData.result.protocolVersion}` })
        addLog(`✓ initialize 成功: serverInfo=${initData.result.serverInfo?.name}`)
      } else if (initData.error) {
        throw new Error(initData.error.message)
      }
    } catch (error) {
      updateStep(2, { status: "error", message: String(error) })
      addLog(`✗ initialize 失敗: ${error}`)
      setIsVerifying(false)
      return
    }

    // Step 4: tools/list
    updateStep(3, { status: "running" })
    addLog("tools/list リクエスト...")
    try {
      const toolsRes = await fetch(`${mcpServerUrl}/mcp`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 2,
          method: "tools/list",
        }),
      })
      const toolsData = await toolsRes.json()
      if (toolsData.result) {
        const toolCount = toolsData.result.tools?.length || 0
        updateStep(3, { status: "success", message: `${toolCount} tools` })
        addLog(`✓ tools/list 成功: ${toolCount} ツール`)
        toolsData.result.tools?.forEach((tool: { name: string }) => {
          addLog(`  - ${tool.name}`)
        })
      } else if (toolsData.error) {
        throw new Error(toolsData.error.message)
      }
    } catch (error) {
      updateStep(3, { status: "error", message: String(error) })
      addLog(`✗ tools/list 失敗: ${error}`)
    }

    setIsVerifying(false)
    addLog("検証完了")
  }

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
    addLog(`Step 1: MCP Server (${mcpServerUrl}/mcp) に接続試行...`)
    try {
      const response = await fetch(`${mcpServerUrl}/mcp`, {
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
        updateStep(0, { status: "success", message: "401 (認証必要)" })
        addLog("✓ 401 Unauthorized - 認証が必要です")
      } else if (response.ok) {
        const data = await response.json()
        if (data.result) {
          updateStep(0, { status: "success", message: "認証なしで接続可能" })
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
        setProtectedResource(metadata)
        updateStep(1, { status: "success", message: metadata.resource })
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
        setOauthMetadata(metadata)
        updateStep(2, { status: "success", message: metadata.issuer })
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
    setCodeVerifier(verifier)

    addLog(`  code_verifier: ${verifier.substring(0, 16)}...`)
    addLog(`  code_challenge: ${challenge.substring(0, 16)}...`)

    const state = generateRandomString(16)
    const redirectUri = `${window.location.origin}/my/mcp-connection/callback`

    // Store in sessionStorage
    sessionStorage.setItem("oauth_state", state)
    sessionStorage.setItem("oauth_verifier", verifier)

    // Use the metadata variable directly since setState is async
    const authServerMetadata = await fetch("/.well-known/oauth-authorization-server").then(r => r.json())
    const authUrl = new URL(authServerMetadata.authorization_endpoint)
    authUrl.searchParams.set("response_type", "code")
    authUrl.searchParams.set("client_id", "mcpist-console")
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
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">MCP接続情報</h1>
        <p className="text-muted-foreground mt-1">MCPクライアントからの接続に必要な情報</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            エンドポイント
          </CardTitle>
          <CardDescription>MCPクライアントの設定に使用するエンドポイントURL</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <Input value={endpoint} readOnly className="font-mono text-sm" />
            <Button
              variant="outline"
              size="icon"
              onClick={() => handleCopy(endpoint, "endpoint")}
            >
              {copied === "endpoint" ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>APIトークン</span>
            {tokenStatus === "active" && (
              <Badge className="bg-green-500/20 text-green-400 border-green-500/30">有効</Badge>
            )}
            {tokenStatus === "revoked" && (
              <Badge className="bg-destructive/20 text-destructive border-destructive/30">無効</Badge>
            )}
            {tokenStatus === "not_generated" && (
              <Badge variant="secondary">未生成</Badge>
            )}
          </CardTitle>
          <CardDescription>
            MCPクライアントからの認証に使用するトークン。セキュリティのため、トークンは一度しか表示されません。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {tokenStatus === "not_generated" ? (
            <Button onClick={handleGenerateToken}>
              <RefreshCw className="h-4 w-4 mr-2" />
              トークンを生成
            </Button>
          ) : tokenStatus === "active" && token ? (
            <>
              <div className="flex items-center gap-2">
                <Input
                  value={showToken ? token : maskedToken || ""}
                  readOnly
                  className="font-mono text-sm"
                />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => setShowToken(!showToken)}
                >
                  {showToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => handleCopy(token, "token")}
                >
                  {copied === "token" ? (
                    <Check className="h-4 w-4 text-success" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" onClick={handleGenerateToken}>
                  <RefreshCw className="h-4 w-4 mr-2" />
                  再生成
                </Button>
                <Button variant="destructive" onClick={handleRevokeToken}>
                  トークンを無効化
                </Button>
              </div>
            </>
          ) : (
            <Button onClick={handleGenerateToken}>
              <RefreshCw className="h-4 w-4 mr-2" />
              新しいトークンを生成
            </Button>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>接続方法</CardTitle>
          <CardDescription>MCPクライアントでの設定例</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="bg-secondary rounded-lg p-4 overflow-x-auto">
            <pre className="text-sm font-mono text-foreground">
{`{
  "mcpServers": {
    "mcpist": {
      "url": "${endpoint}",
      "transport": "sse",
      "headers": {
        "Authorization": "Bearer ${token || "<your-token>"}"
      }
    }
  }
}`}
            </pre>
          </div>
        </CardContent>
      </Card>

      {/* MCP Server Connection Verification - Admin only */}
      {isAdmin && (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Play className="h-5 w-5" />
            MCP サーバー接続検証
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
                <div
                  key={index}
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
      )}
    </div>
  )
}
