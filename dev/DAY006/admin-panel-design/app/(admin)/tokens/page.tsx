"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { apiTokens, type ApiToken } from "@/lib/data"
import { Plus, Copy, Check, Key } from "lucide-react"
import { useMediaQuery } from "@/hooks/use-media-query"

export default function TokensPage() {
  const [tokens, setTokens] = useState<ApiToken[]>(apiTokens)
  const [createOpen, setCreateOpen] = useState(false)
  const [newTokenName, setNewTokenName] = useState("")
  const [newTokenExpiry, setNewTokenExpiry] = useState("90")
  const [generatedToken, setGeneratedToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [revokeToken, setRevokeToken] = useState<ApiToken | null>(null)
  const isMobile = useMediaQuery("(max-width: 768px)")

  const handleCreateToken = () => {
    const token = `mcp_${Math.random().toString(36).substring(2, 15)}_${Math.random().toString(36).substring(2, 15)}`
    setGeneratedToken(token)

    const newToken: ApiToken = {
      id: String(tokens.length + 1),
      name: newTokenName,
      createdAt: new Date().toISOString().split("T")[0],
      expiresAt:
        newTokenExpiry === "none"
          ? null
          : new Date(Date.now() + Number.parseInt(newTokenExpiry) * 24 * 60 * 60 * 1000).toISOString().split("T")[0],
      lastUsedAt: null,
      status: "active",
      prefix: "mcp_",
    }
    setTokens([newToken, ...tokens])
  }

  const handleCopy = () => {
    if (generatedToken) {
      navigator.clipboard.writeText(generatedToken)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleRevoke = () => {
    if (revokeToken) {
      setTokens(tokens.map((t) => (t.id === revokeToken.id ? { ...t, status: "revoked" as const } : t)))
      setRevokeToken(null)
    }
  }

  const handleCloseCreate = () => {
    setCreateOpen(false)
    setNewTokenName("")
    setNewTokenExpiry("90")
    setGeneratedToken(null)
    setCopied(false)
  }

  const statusConfig = {
    active: { label: "有効", class: "bg-success/20 text-success border-success/30" },
    expired: { label: "期限切れ", class: "bg-destructive/20 text-destructive border-destructive/30" },
    revoked: { label: "失効済", class: "bg-muted text-muted-foreground border-border" },
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">APIトークン</h1>
          <p className="text-muted-foreground mt-1">API認証用のトークンを管理します</p>
        </div>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              新規トークン発行
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{generatedToken ? "トークンが発行されました" : "新規トークン発行"}</DialogTitle>
              <DialogDescription>
                {generatedToken
                  ? "このトークンは一度だけ表示されます。安全な場所に保存してください。"
                  : "新しいAPIトークンを発行します"}
              </DialogDescription>
            </DialogHeader>
            {generatedToken ? (
              <div className="space-y-4">
                <div className="p-4 bg-secondary rounded-lg">
                  <div className="flex items-center gap-2">
                    <code className="flex-1 text-sm break-all text-foreground">{generatedToken}</code>
                    <Button variant="ghost" size="icon" onClick={handleCopy}>
                      {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
                    </Button>
                  </div>
                </div>
                <DialogFooter>
                  <Button onClick={handleCloseCreate}>閉じる</Button>
                </DialogFooter>
              </div>
            ) : (
              <>
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="token-name">トークン名</Label>
                    <Input
                      id="token-name"
                      placeholder="例: 開発環境用"
                      value={newTokenName}
                      onChange={(e) => setNewTokenName(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="token-expiry">有効期限</Label>
                    <Select value={newTokenExpiry} onValueChange={setNewTokenExpiry}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="30">30日</SelectItem>
                        <SelectItem value="90">90日</SelectItem>
                        <SelectItem value="365">1年</SelectItem>
                        <SelectItem value="none">無期限</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={handleCloseCreate}>
                    キャンセル
                  </Button>
                  <Button onClick={handleCreateToken} disabled={!newTokenName}>
                    発行する
                  </Button>
                </DialogFooter>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">トークン一覧</CardTitle>
        </CardHeader>
        <CardContent>
          {isMobile ? (
            <div className="space-y-3">
              {tokens.map((token) => (
                <Card key={token.id} className="p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <Key className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium text-foreground">{token.name}</span>
                    </div>
                    <Badge variant="outline" className={statusConfig[token.status].class}>
                      {statusConfig[token.status].label}
                    </Badge>
                  </div>
                  <div className="space-y-1 text-sm text-muted-foreground">
                    <p>作成日: {token.createdAt}</p>
                    <p>有効期限: {token.expiresAt || "無期限"}</p>
                    <p>最終使用: {token.lastUsedAt || "未使用"}</p>
                  </div>
                  {token.status === "active" && (
                    <Button
                      variant="destructive"
                      size="sm"
                      className="mt-3 w-full"
                      onClick={() => setRevokeToken(token)}
                    >
                      失効させる
                    </Button>
                  )}
                </Card>
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名前</TableHead>
                  <TableHead>作成日</TableHead>
                  <TableHead>有効期限</TableHead>
                  <TableHead>最終使用日</TableHead>
                  <TableHead>ステータス</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>{token.createdAt}</TableCell>
                    <TableCell>{token.expiresAt || "無期限"}</TableCell>
                    <TableCell>{token.lastUsedAt || "未使用"}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className={statusConfig[token.status].class}>
                        {statusConfig[token.status].label}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      {token.status === "active" && (
                        <Button variant="destructive" size="sm" onClick={() => setRevokeToken(token)}>
                          失効
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={!!revokeToken} onOpenChange={() => setRevokeToken(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>トークンを失効させますか？</AlertDialogTitle>
            <AlertDialogDescription>
              「{revokeToken?.name}
              」を失効させると、このトークンを使用したAPIアクセスができなくなります。この操作は取り消せません。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleRevoke}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              失効させる
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
