"use client"

import { useState, useEffect, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
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
import { Key, Plus, Trash2, Copy, Check, Loader2, AlertTriangle } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import {
  listApiKeys,
  generateApiKey,
  revokeApiKey,
  ApiKeyError,
  type ApiKey,
  type GenerateApiKeyResult,
} from "@/lib/api-keys"

export default function ApiKeysPage() {
  const { user } = useAuth()
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])
  const [loading, setLoading] = useState(true)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deleteDialogKey, setDeleteDialogKey] = useState<ApiKey | null>(null)
  const [createdKey, setCreatedKey] = useState<GenerateApiKeyResult | null>(null)
  const [copied, setCopied] = useState(false)

  // Create form state
  const [keyName, setKeyName] = useState("")
  const [expiration, setExpiration] = useState<string>("never")
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const loadApiKeys = useCallback(async () => {
    try {
      const keys = await listApiKeys()
      setApiKeys(keys)
    } catch (error) {
      if (error instanceof ApiKeyError) {
        console.error("Failed to load API keys:", error.message)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) {
      loadApiKeys()
    } else {
      setLoading(false)
    }
  }, [user, loadApiKeys])

  const handleCreate = async () => {
    if (!keyName.trim()) {
      toast.error("キー名を入力してください")
      return
    }

    setCreating(true)
    try {
      const expiresInDays = expiration === "never" ? null :
        expiration === "30" ? 30 :
        expiration === "90" ? 90 :
        365

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

  const handleDelete = async () => {
    if (!deleteDialogKey) return

    setDeleting(true)
    try {
      await revokeApiKey(deleteDialogKey.id)
      toast.success("APIキーを削除しました")
      await loadApiKeys()
      setDeleteDialogKey(null)
    } catch (error) {
      if (error instanceof ApiKeyError) {
        toast.error(`削除に失敗しました: ${error.message}`)
      } else {
        toast.error("削除に失敗しました")
      }
    } finally {
      setDeleting(false)
    }
  }

  const handleCopyKey = async (key: string) => {
    await navigator.clipboard.writeText(key)
    setCopied(true)
    toast.success("APIキーをコピーしました")
    setTimeout(() => setCopied(false), 2000)
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

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">API Keys</h1>
          <p className="text-muted-foreground mt-1">
            Claude Code、Cursor などの MCP クライアントで使用する API キーを管理します
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          新規作成
        </Button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : apiKeys.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center">
            <Key className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="font-medium text-foreground mb-2">APIキーがありません</h3>
            <p className="text-sm text-muted-foreground mb-4">
              Claude Code や Cursor から MCPist に接続するには API キーが必要です
            </p>
            <Button onClick={() => setCreateDialogOpen(true)}>
              <Plus className="h-4 w-4 mr-2" />
              APIキーを作成
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {apiKeys.map((apiKey) => (
            <Card
              key={apiKey.id}
              className={cn(
                apiKey.is_expired && "border-warning/50 bg-warning/5"
              )}
            >
              <CardContent className="p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                      <Key className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-medium text-foreground">{apiKey.name}</h3>
                        {apiKey.is_expired && (
                          <Badge variant="outline" className="bg-warning/20 text-warning border-warning/30">
                            <AlertTriangle className="h-3 w-3 mr-1" />
                            期限切れ
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm font-mono text-muted-foreground">
                        {apiKey.key_prefix}
                      </p>
                      <div className="flex gap-4 text-xs text-muted-foreground">
                        <span>作成: {formatDate(apiKey.created_at)}</span>
                        <span>最終使用: {formatLastUsed(apiKey.last_used_at)}</span>
                        {apiKey.expires_at && (
                          <span>有効期限: {formatDate(apiKey.expires_at)}</span>
                        )}
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
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create Dialog */}
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
                  <Label htmlFor="never" className="font-normal">無期限</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="30" id="30days" />
                  <Label htmlFor="30days" className="font-normal">30日</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="90" id="90days" />
                  <Label htmlFor="90days" className="font-normal">90日</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="365" id="1year" />
                  <Label htmlFor="1year" className="font-normal">1年</Label>
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
                <Input
                  value={createdKey?.key || ""}
                  readOnly
                  className="font-mono text-sm"
                />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => createdKey && handleCopyKey(createdKey.key)}
                >
                  {copied ? (
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

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deleteDialogKey} onOpenChange={(open) => !open && setDeleteDialogKey(null)}>
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
    </div>
  )
}
