"use client"

import { useState, useEffect, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useAuth } from "@/lib/auth-context"
import { modules } from "@/lib/module-data"
import {
  Plus,
  Loader2,
  Pencil,
  Trash2,
  MessageSquareText,
  FileText,
} from "lucide-react"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import {
  listPrompts,
  upsertPrompt,
  deletePrompt,
  type Prompt,
} from "@/lib/prompts"

export const dynamic = "force-dynamic"

export default function PromptsPage() {
  const { user } = useAuth()
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [loading, setLoading] = useState(true)

  // Edit/Create dialog state
  const [editDialog, setEditDialog] = useState<{
    open: boolean
    prompt: Prompt | null // null for create, Prompt for edit
  }>({ open: false, prompt: null })
  const [editForm, setEditForm] = useState({
    name: "",
    content: "",
    moduleName: "",
    enabled: true,
  })
  const [submitting, setSubmitting] = useState(false)

  // Delete dialog state
  const [deleteDialog, setDeleteDialog] = useState<Prompt | null>(null)
  const [deleting, setDeleting] = useState(false)

  // Load prompts
  const loadPrompts = useCallback(async () => {
    try {
      const data = await listPrompts()
      setPrompts(data)
    } catch (error) {
      console.error("Failed to load prompts:", error)
      toast.error("プロンプトの取得に失敗しました")
    }
  }, [])

  useEffect(() => {
    async function loadData() {
      if (user) {
        await loadPrompts()
      }
      setLoading(false)
    }
    loadData()
  }, [user, loadPrompts])

  // Open edit dialog
  const handleEdit = (prompt: Prompt) => {
    setEditForm({
      name: prompt.name,
      content: prompt.content,
      moduleName: prompt.module_name || "",
      enabled: prompt.enabled,
    })
    setEditDialog({ open: true, prompt })
  }

  // Open create dialog
  const handleCreate = () => {
    setEditForm({
      name: "",
      content: "",
      moduleName: "",
      enabled: true,
    })
    setEditDialog({ open: true, prompt: null })
  }

  // Submit edit/create
  const handleSubmit = async () => {
    if (!editForm.name.trim()) {
      toast.error("名前を入力してください")
      return
    }
    if (!editForm.content.trim()) {
      toast.error("内容を入力してください")
      return
    }

    setSubmitting(true)
    try {
      const result = await upsertPrompt(
        editForm.name.trim(),
        editForm.content.trim(),
        editForm.moduleName || undefined,
        editDialog.prompt?.id,
        editForm.enabled
      )

      if (result.success) {
        toast.success(editDialog.prompt ? "プロンプトを更新しました" : "プロンプトを作成しました")
        setEditDialog({ open: false, prompt: null })
        await loadPrompts()
      } else {
        toast.error(result.error || "保存に失敗しました")
      }
    } catch (error) {
      console.error("Failed to save prompt:", error)
      toast.error("保存に失敗しました")
    } finally {
      setSubmitting(false)
    }
  }

  // Toggle enabled status
  const handleToggleEnabled = async (prompt: Prompt) => {
    try {
      const result = await upsertPrompt(
        prompt.name,
        prompt.content,
        prompt.module_name || undefined,
        prompt.id,
        !prompt.enabled
      )

      if (result.success) {
        // Update local state optimistically
        setPrompts((prev) =>
          prev.map((p) => (p.id === prompt.id ? { ...p, enabled: !p.enabled } : p))
        )
      } else {
        toast.error(result.error || "更新に失敗しました")
      }
    } catch (error) {
      console.error("Failed to toggle prompt:", error)
      toast.error("更新に失敗しました")
    }
  }

  // Confirm delete
  const handleDelete = async () => {
    if (!deleteDialog) return

    setDeleting(true)
    try {
      const result = await deletePrompt(deleteDialog.id)

      if (result.success) {
        toast.success("プロンプトを削除しました")
        setDeleteDialog(null)
        await loadPrompts()
      } else {
        toast.error(result.error || "削除に失敗しました")
      }
    } catch (error) {
      console.error("Failed to delete prompt:", error)
      toast.error("削除に失敗しました")
    } finally {
      setDeleting(false)
    }
  }

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">プロンプト</h1>
          <p className="text-muted-foreground mt-1">カスタムプロンプトを管理します</p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">プロンプト</h1>
          <p className="text-muted-foreground mt-1">カスタムプロンプトを管理します</p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="h-4 w-4 mr-2" />
          新規作成
        </Button>
      </div>

      {prompts.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <MessageSquareText className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground text-center">
              プロンプトがありません
              <br />
              <span className="text-sm">「新規作成」ボタンから作成できます</span>
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {prompts.map((prompt) => (
            <Card
              key={prompt.id}
              className={cn(!prompt.enabled && "opacity-60")}
            >
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                      <FileText className="h-5 w-5 text-foreground" />
                    </div>
                    <div>
                      <CardTitle className="text-base">{prompt.name}</CardTitle>
                      <CardDescription className="text-xs">
                        {prompt.module_name ? (
                          <span className="text-primary">{prompt.module_name}</span>
                        ) : (
                          <span className="text-muted-foreground">全モジュール共通</span>
                        )}
                        {" · "}
                        {new Date(prompt.updated_at).toLocaleDateString("ja-JP")}
                      </CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={prompt.enabled}
                      onCheckedChange={() => handleToggleEnabled(prompt)}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleEdit(prompt)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setDeleteDialog(prompt)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <pre className="text-sm text-muted-foreground whitespace-pre-wrap font-mono bg-secondary/30 p-3 rounded-lg max-h-32 overflow-auto">
                  {prompt.content}
                </pre>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Edit/Create Dialog */}
      <Dialog open={editDialog.open} onOpenChange={(open) => !open && setEditDialog({ open: false, prompt: null })}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editDialog.prompt ? "プロンプトを編集" : "新しいプロンプトを作成"}
            </DialogTitle>
            <DialogDescription>
              {editDialog.prompt
                ? "プロンプトの内容を編集します"
                : "AIに渡すカスタムプロンプトを作成します"}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="prompt-name">名前</Label>
              <Input
                id="prompt-name"
                value={editForm.name}
                onChange={(e) => setEditForm((prev) => ({ ...prev, name: e.target.value }))}
                placeholder="daily_tasks"
                disabled={submitting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="prompt-module">対象モジュール（オプション）</Label>
              <Select
                value={editForm.moduleName || "__all__"}
                onValueChange={(value) => setEditForm((prev) => ({ ...prev, moduleName: value === "__all__" ? "" : value }))}
                disabled={submitting}
              >
                <SelectTrigger id="prompt-module">
                  <SelectValue placeholder="全モジュール共通" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">全モジュール共通</SelectItem>
                  {modules.map((mod) => (
                    <SelectItem key={mod.id} value={mod.id}>
                      {mod.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="prompt-content">内容</Label>
              <Textarea
                id="prompt-content"
                value={editForm.content}
                onChange={(e) => setEditForm((prev) => ({ ...prev, content: e.target.value }))}
                placeholder="今日のタスク一覧を取得して、優先度順に整理してください..."
                className="min-h-[150px] font-mono"
                disabled={submitting}
              />
            </div>

            <div className="flex items-center gap-2">
              <Switch
                id="prompt-enabled"
                checked={editForm.enabled}
                onCheckedChange={(checked) => setEditForm((prev) => ({ ...prev, enabled: checked }))}
                disabled={submitting}
              />
              <Label htmlFor="prompt-enabled">有効</Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="ghost" onClick={() => setEditDialog({ open: false, prompt: null })} disabled={submitting}>
              キャンセル
            </Button>
            <Button onClick={handleSubmit} disabled={submitting}>
              {submitting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  保存中...
                </>
              ) : (
                "保存"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deleteDialog} onOpenChange={(open) => !open && setDeleteDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>プロンプトを削除しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              「{deleteDialog?.name}」を削除します。この操作は取り消せません。
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
