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
import { useAuth } from "@/lib/auth/auth-context"
import {
  Plus,
  Loader2,
  Pencil,
  Trash2,
  MessageSquareText,
} from "lucide-react"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import {
  listPrompts,
  upsertPrompt,
  deletePrompt,
  type Prompt,
} from "@/lib/mcp/prompts"

// モジュールレベルキャッシュ
let cachedPrompts: Prompt[] | null = null

export const dynamic = "force-dynamic"

export default function PromptsPage() {
  const { user } = useAuth()
  const hasCached = cachedPrompts !== null
  const [prompts, setPrompts] = useState<Prompt[]>(cachedPrompts ?? [])
  const [loading, setLoading] = useState(!hasCached)

  // Edit/Create dialog state
  const [editDialog, setEditDialog] = useState<{
    open: boolean
    prompt: Prompt | null // null for create, Prompt for edit
  }>({ open: false, prompt: null })
  const [editForm, setEditForm] = useState({
    name: "",
    description: "",
    content: "",
  })
  const [submitting, setSubmitting] = useState(false)

  // Delete dialog state
  const [deleteDialog, setDeleteDialog] = useState<Prompt | null>(null)
  const [deleting, setDeleting] = useState(false)

  // Load prompts
  const loadPrompts = useCallback(async () => {
    try {
      const data = await listPrompts()
      cachedPrompts = data
      setPrompts(data)
    } catch (error) {
      console.error("Failed to load prompts:", error)
      toast.error("テンプレートの取得に失敗しました")
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
      description: prompt.description || "",
      content: prompt.content,
    })
    setEditDialog({ open: true, prompt })
  }

  // Open create dialog
  const handleCreate = () => {
    setEditForm({
      name: "",
      description: "",
      content: "",
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
        undefined, // moduleName is not used
        editDialog.prompt?.id,
        editDialog.prompt?.enabled ?? true,
        editForm.description.trim() || undefined
      )

      if (result.success) {
        toast.success(editDialog.prompt ? "テンプレートを更新しました" : "テンプレートを作成しました")
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

  // Toggle enabled status (directly from list)
  const handleToggleEnabled = async (prompt: Prompt) => {
    // Optimistic update
    setPrompts((prev) =>
      prev.map((p) => (p.id === prompt.id ? { ...p, enabled: !p.enabled } : p))
    )

    try {
      const result = await upsertPrompt(
        prompt.name,
        prompt.content,
        undefined,
        prompt.id,
        !prompt.enabled,
        prompt.description || undefined
      )

      if (!result.success) {
        // Revert on failure
        setPrompts((prev) =>
          prev.map((p) => (p.id === prompt.id ? { ...p, enabled: prompt.enabled } : p))
        )
        toast.error(result.error || "更新に失敗しました")
      }
    } catch (error) {
      // Revert on error
      setPrompts((prev) =>
        prev.map((p) => (p.id === prompt.id ? { ...p, enabled: prompt.enabled } : p))
      )
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
        toast.success("テンプレートを削除しました")
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
        <div className="pl-8 md:pl-0">
          <h1 className="text-2xl font-bold text-foreground">テンプレート</h1>
          <p className="text-muted-foreground mt-1">カスタムテンプレートを管理します</p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <div className="flex flex-wrap items-center gap-4">
          <h1 className="text-2xl font-bold text-foreground">テンプレート</h1>
          <div className="ml-auto">
            <Button onClick={handleCreate}>
              <Plus className="h-4 w-4 mr-2" />
              新規作成
            </Button>
          </div>
        </div>
        <p className="text-muted-foreground mt-1">カスタムテンプレートを管理します</p>
      </div>

      {prompts.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <MessageSquareText className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground text-center">
              テンプレートがありません
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
              className={cn("py-2 cursor-pointer hover:bg-accent/50 transition-colors", !prompt.enabled && "opacity-60")}
              onClick={() => handleEdit(prompt)}
            >
              <CardHeader className="py-2">
                <div className="flex items-start gap-3 overflow-hidden">
                  <Switch
                    checked={prompt.enabled}
                    onCheckedChange={() => handleToggleEnabled(prompt)}
                    onClick={(e) => e.stopPropagation()}
                    className="shrink-0 mt-0.5"
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <CardTitle className="text-sm font-medium truncate">{prompt.name}</CardTitle>
                      <CardDescription className="text-xs shrink-0">
                        {new Date(prompt.updated_at).toLocaleDateString("ja-JP")}
                      </CardDescription>
                    </div>
                    <p className="text-xs text-muted-foreground truncate mt-0.5">
                      {prompt.description || prompt.content.replace(/\n/g, " ")}
                    </p>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleEdit(prompt)
                      }}
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={(e) => {
                        e.stopPropagation()
                        setDeleteDialog(prompt)
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5 text-destructive" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      {/* Edit/Create Dialog */}
      <Dialog open={editDialog.open} onOpenChange={(open) => !open && setEditDialog({ open: false, prompt: null })}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editDialog.prompt ? "テンプレートを編集" : "新しいテンプレートを作成"}
            </DialogTitle>
            <DialogDescription>
              {editDialog.prompt
                ? "テンプレートの内容を編集します"
                : "AIに渡すカスタムテンプレートを作成します"}
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
              <Label htmlFor="prompt-description">説明（任意）</Label>
              <Input
                id="prompt-description"
                value={editForm.description}
                onChange={(e) => setEditForm((prev) => ({ ...prev, description: e.target.value }))}
                placeholder="今日のタスクを取得します"
                disabled={submitting}
              />
              <p className="text-xs text-muted-foreground">
                MCPクライアントに表示される短い説明文です
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="prompt-content">内容</Label>
              <Textarea
                id="prompt-content"
                value={editForm.content}
                onChange={(e) => setEditForm((prev) => ({ ...prev, content: e.target.value }))}
                placeholder="今日のタスク一覧を取得して、優先度順に整理してください..."
                className="min-h-[200px] font-mono"
                disabled={submitting}
              />
              <p className="text-xs text-muted-foreground">
                実際にAIに渡されるプロンプトの内容です
              </p>
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
            <AlertDialogTitle>テンプレートを削除しますか？</AlertDialogTitle>
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
