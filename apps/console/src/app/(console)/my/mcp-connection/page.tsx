"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { useAuth } from "@/lib/auth-context"
import { Copy, RefreshCw, Eye, EyeOff, Check, Server } from "lucide-react"
import { cn } from "@/lib/utils"

export default function McpConnectionPage() {
  const { user } = useAuth()
  const [showToken, setShowToken] = useState(false)
  const [copied, setCopied] = useState<string | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [tokenStatus, setTokenStatus] = useState<"not_generated" | "active" | "revoked">("not_generated")

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
    </div>
  )
}
