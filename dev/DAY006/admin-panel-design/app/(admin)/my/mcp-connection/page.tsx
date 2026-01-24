"use client"

import { useState } from "react"
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
import { userMcpConnections, type UserMcpConnection } from "@/lib/data"
import { Copy, Check, Key, AlertTriangle, Server, RefreshCw } from "lucide-react"
import { cn } from "@/lib/utils"

function generateToken() {
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
  let result = "mcp_usr_"
  for (let i = 0; i < 24; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

export default function McpConnectionPage() {
  const { user } = useAuth()
  const [connections, setConnections] = useState<UserMcpConnection[]>(userMcpConnections)
  const [confirmDialog, setConfirmDialog] = useState(false)
  const [successDialog, setSuccessDialog] = useState(false)
  const [generatedToken, setGeneratedToken] = useState("")
  const [copiedField, setCopiedField] = useState<string | null>(null)
  const [requestReissueDialog, setRequestReissueDialog] = useState(false)

  const myConnection = connections.find((c) => c.userId === user?.id)
  const isGenerated = myConnection?.status === "active"
  const isRevoked = myConnection?.status === "revoked"

  const handleCopy = async (text: string, field: string) => {
    await navigator.clipboard.writeText(text)
    setCopiedField(field)
    setTimeout(() => setCopiedField(null), 2000)
  }

  const handleGenerateToken = () => {
    const newToken = generateToken()
    setGeneratedToken(newToken)

    setConnections((prev) =>
      prev.map((c) => {
        if (c.userId !== user?.id) return c
        return {
          ...c,
          apiToken: "mcp_usr_****",
          generatedAt: new Date().toISOString().split("T")[0],
          status: "active" as const,
        }
      }),
    )

    setConfirmDialog(false)
    setSuccessDialog(true)
  }

  const configExample = `{
  "mcpServers": {
    "mcpist": {
      "url": "${myConnection?.endpoint || "https://mcp.example.com/u/xxx"}",
      "apiKey": "${generatedToken || "mcp_usr_xxxxxxxxxxxxxxxxxxxxxxxx"}"
    }
  }
}`

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">MCP接続情報</h1>
        <p className="text-muted-foreground mt-1">MCPサーバーに接続するための認証情報を管理します</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
              <Server className="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle>接続情報</CardTitle>
              <CardDescription>MCPクライアントの設定に使用します</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {!isGenerated && !isRevoked && (
            <div className="p-4 bg-secondary/50 rounded-lg space-y-3">
              <div className="flex items-start gap-3">
                <AlertTriangle className="h-5 w-5 text-warning shrink-0 mt-0.5" />
                <div>
                  <p className="font-medium">トークンは一度だけ表示されます</p>
                  <p className="text-sm text-muted-foreground">
                    発行後は再度表示できません。安全な場所に保存してください。
                  </p>
                </div>
              </div>
            </div>
          )}

          <div className="space-y-2">
            <Label>エンドポイント</Label>
            <div className="flex gap-2">
              <Input value={myConnection?.endpoint || ""} readOnly className="font-mono" />
              <Button
                variant="outline"
                size="icon"
                onClick={() => handleCopy(myConnection?.endpoint || "", "endpoint")}
              >
                {copiedField === "endpoint" ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label>APIトークン</Label>
            <div className="flex gap-2">
              <div className="flex-1 relative">
                <Input
                  value={isGenerated || isRevoked ? myConnection?.apiToken || "" : "未発行"}
                  readOnly
                  className={cn("font-mono", !isGenerated && !isRevoked && "text-muted-foreground")}
                />
                {isGenerated && (
                  <Badge className="absolute right-3 top-1/2 -translate-y-1/2 bg-success/20 text-success border-success/30">
                    有効
                  </Badge>
                )}
                {isRevoked && (
                  <Badge className="absolute right-3 top-1/2 -translate-y-1/2 bg-destructive/20 text-destructive border-destructive/30">
                    失効
                  </Badge>
                )}
              </div>
            </div>
            {isGenerated && myConnection?.generatedAt && (
              <p className="text-sm text-muted-foreground">発行日: {myConnection.generatedAt}</p>
            )}
          </div>

          {!isGenerated && !isRevoked && (
            <Button onClick={() => setConfirmDialog(true)} className="w-full">
              <Key className="h-4 w-4 mr-2" />
              トークンを発行する
            </Button>
          )}

          {(isGenerated || isRevoked) && (
            <div className="space-y-4">
              <div className="p-4 bg-secondary/50 rounded-lg">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-warning shrink-0 mt-0.5" />
                  <p className="text-sm text-muted-foreground">
                    トークンを紛失した場合は管理者に連絡して再発行を依頼してください
                  </p>
                </div>
              </div>
              <Button variant="outline" onClick={() => setRequestReissueDialog(true)} className="w-full">
                <RefreshCw className="h-4 w-4 mr-2" />
                再発行を申請する
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Confirm Dialog */}
      <AlertDialog open={confirmDialog} onOpenChange={setConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>トークンを発行しますか？</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="space-y-3">
                <div className="flex items-start gap-2 p-3 bg-warning/10 rounded-lg">
                  <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
                  <div className="text-sm">
                    <p className="font-medium text-warning">注意事項</p>
                    <ul className="list-disc list-inside mt-1 text-muted-foreground space-y-1">
                      <li>トークンは一度だけ表示されます</li>
                      <li>紛失した場合、管理者に連絡して再発行が必要です</li>
                      <li>既存のトークンは無効になります</li>
                    </ul>
                  </div>
                </div>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={handleGenerateToken}>発行する</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Success Dialog */}
      <Dialog open={successDialog} onOpenChange={setSuccessDialog}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Check className="h-5 w-5 text-success" />
              トークンが発行されました
            </DialogTitle>
            <DialogDescription>
              このトークンは一度だけ表示されます。必ずコピーして安全な場所に保存してください。
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>APIトークン</Label>
              <div className="flex gap-2">
                <Input value={generatedToken} readOnly className="font-mono text-sm" />
                <Button variant="outline" size="icon" onClick={() => handleCopy(generatedToken, "token")}>
                  {copiedField === "token" ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <Label>Claude Desktop設定例</Label>
              <div className="relative">
                <pre className="p-4 bg-secondary rounded-lg text-sm font-mono overflow-x-auto whitespace-pre">
                  {configExample}
                </pre>
                <Button
                  variant="ghost"
                  size="sm"
                  className="absolute top-2 right-2"
                  onClick={() => handleCopy(configExample, "config")}
                >
                  {copiedField === "config" ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button onClick={() => setSuccessDialog(false)}>閉じる</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Request Reissue Dialog */}
      <AlertDialog open={requestReissueDialog} onOpenChange={setRequestReissueDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>再発行を申請しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              管理者にトークンの再発行を申請します。承認されると現在のトークンは無効になり、新しいトークンが発行されます。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={() => setRequestReissueDialog(false)}>申請する</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
