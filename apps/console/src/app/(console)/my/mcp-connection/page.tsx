"use client"

import { useState, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Copy, Check, Server, Play, CheckCircle2, XCircle, Loader2, Key, ChevronDown, ChevronRight, ExternalLink } from "lucide-react"
import { cn } from "@/lib/utils"
import Link from "next/link"

type VerifyStep = {
  name: string
  status: "pending" | "running" | "success" | "error"
  message?: string
}

export default function McpConnectionPage() {
  const [copied, setCopied] = useState<string | null>(null)
  const [isApiKeyOpen, setIsApiKeyOpen] = useState(false)

  // API Key test state
  const [mcpServerUrl] = useState(process.env.NEXT_PUBLIC_MCP_SERVER_URL || "http://mcp.localhost")
  const [isVerifying, setIsVerifying] = useState(false)
  const [verifySteps, setVerifySteps] = useState<VerifyStep[]>([])
  const [testApiKey, setTestApiKey] = useState<string>("")

  // MCPエンドポイント: Worker経由でアクセス
  // Worker (8787) が認証・Rate Limit・LBを処理してGo Server (8089)にプロキシ
  const mcpBaseUrl = process.env.NEXT_PUBLIC_MCP_SERVER_URL || "http://localhost:8787"
  const endpoint = `${mcpBaseUrl}/mcp`

  const handleCopy = async (text: string, type: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(type)
    setTimeout(() => setCopied(null), 2000)
  }

  // API Key Connection Test
  const testApiKeyConnection = async () => {
    if (!testApiKey) return

    setIsVerifying(true)
    setVerifySteps([
      { name: "MCP Server 接続", status: "pending" },
      { name: "initialize", status: "pending" },
      { name: "tools/list", status: "pending" },
    ])

    // Step 1: Connect to MCP Server with API Key
    updateStep(0, { status: "running" })
    try {
      const mcpEndpoint = `${mcpServerUrl}/mcp`
      const response = await fetch(mcpEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${testApiKey}`,
        },
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
        updateStep(0, { status: "error", message: "認証失敗 (401)" })
        setIsVerifying(false)
        return
      }

      if (!response.ok) {
        throw new Error(`Status: ${response.status}`)
      }

      updateStep(0, { status: "success", message: "接続成功" })

      // Step 2: Check initialize response
      updateStep(1, { status: "running" })
      const initData = await response.json()
      if (initData.result) {
        updateStep(1, { status: "success", message: `v${initData.result.protocolVersion}` })
      } else if (initData.error) {
        updateStep(1, { status: "error", message: initData.error.message })
        setIsVerifying(false)
        return
      }

      // Step 3: Get tools/list
      updateStep(2, { status: "running" })
      const toolsRes = await fetch(mcpEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${testApiKey}`,
        },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 2,
          method: "tools/list",
        }),
      })

      const toolsData = await toolsRes.json()
      if (toolsData.result) {
        const toolCount = toolsData.result.tools?.length || 0
        updateStep(2, { status: "success", message: `${toolCount} tools` })
      } else if (toolsData.error) {
        updateStep(2, { status: "error", message: toolsData.error.message })
      }
    } catch (error) {
      updateStep(0, { status: "error", message: String(error) })
    }

    setIsVerifying(false)
  }

  const updateStep = useCallback((index: number, update: Partial<VerifyStep>) => {
    setVerifySteps((prev) =>
      prev.map((step, i) => (i === index ? { ...step, ...update } : step))
    )
  }, [])

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

      {/* API Key認証と接続方法 */}
      <Collapsible open={isApiKeyOpen} onOpenChange={setIsApiKeyOpen}>
        <Card>
          <CardHeader>
            <CollapsibleTrigger asChild>
              <button className="flex items-center justify-between w-full text-left">
                <CardTitle className="flex items-center gap-2">
                  <Key className="h-5 w-5" />
                  API Key と接続方法
                </CardTitle>
                <div className="flex items-center gap-2">
                  {isApiKeyOpen ? (
                    <ChevronDown className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  )}
                </div>
              </button>
            </CollapsibleTrigger>
            <CardDescription>
              Claude CodeやCursorなどのMCPクライアントで使用するAPI Keyと設定例
            </CardDescription>
          </CardHeader>
          <CollapsibleContent>
            <CardContent className="space-y-4">
              {/* Link to API Keys page */}
              <div className="p-4 bg-secondary/50 rounded-lg border">
                <p className="text-sm text-muted-foreground mb-3">
                  API Keyは「API Keys」ページで管理できます。新しいキーの生成や既存キーの無効化が可能です。
                </p>
                <Button asChild variant="outline">
                  <Link href="/my/api-keys">
                    <Key className="h-4 w-4 mr-2" />
                    API Keys管理
                    <ExternalLink className="h-3 w-3 ml-2" />
                  </Link>
                </Button>
              </div>

              {/* 接続設定例 */}
              <div className="border-t pt-4 mt-4">
                <h4 className="text-sm font-medium mb-2">MCPクライアント設定例</h4>
                <div className="bg-secondary rounded-lg p-4 overflow-x-auto">
                  <pre className="text-sm font-mono text-foreground whitespace-pre">{`{
  "mcpServers": {
    "mcpist": {
      "type": "sse",
      "url": "${endpoint}",
      "headers": {
        "Authorization": "Bearer `}<Input
                    value={testApiKey}
                    onChange={(e) => setTestApiKey(e.target.value)}
                    placeholder="<your-api-key>"
                    className="inline-flex w-72 h-6 px-1 py-0 text-xs font-mono bg-white dark:bg-zinc-700 border rounded align-middle"
                  />{`"
      }
    }
  }
}`}</pre>
                </div>

                <div className="flex justify-end mt-4">
                  <Button
                    variant="outline"
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
                        接続テスト
                      </>
                    )}
                  </Button>
                </div>

                {/* Test Steps */}
                {verifySteps.length > 0 && (
                  <div className="space-y-2 mt-4">
                    {verifySteps.map((step, index) => (
                      <div
                        key={index}
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
                        {step.status === "error" && (
                          <XCircle className="h-4 w-4 text-destructive" />
                        )}
                        <span className="flex-1">{step.name}</span>
                        {step.message && (
                          <span className="text-xs text-muted-foreground">
                            {step.message}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </CardContent>
          </CollapsibleContent>
        </Card>
      </Collapsible>
    </div>
  )
}
