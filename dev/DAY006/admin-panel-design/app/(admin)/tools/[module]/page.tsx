"use client"

import { useParams, useRouter } from "next/navigation"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { ServiceIcon } from "@/components/service-icon"
import { moduleDetails } from "@/lib/data"
import { ArrowLeft, LinkIcon, Unlink, Lock } from "lucide-react"
import { cn } from "@/lib/utils"

export default function ModuleDetailPage() {
  const params = useParams()
  const router = useRouter()
  const moduleId = params.module as string
  const module = moduleDetails[moduleId]

  const [requestDialogOpen, setRequestDialogOpen] = useState(false)
  const [requestTool, setRequestTool] = useState<string | null>(null)
  const [requestReason, setRequestReason] = useState("")
  const [disconnectDialogOpen, setDisconnectDialogOpen] = useState(false)

  if (!module) {
    return (
      <div className="flex flex-col items-center justify-center h-[50vh] gap-4">
        <p className="text-muted-foreground">モジュールが見つかりません</p>
        <Button variant="outline" onClick={() => router.push("/tools")}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          ツール一覧に戻る
        </Button>
      </div>
    )
  }

  const statusConfig = {
    connected: { badge: "連携済", badgeClass: "bg-success/20 text-success border-success/30" },
    disconnected: { badge: "未連携", badgeClass: "bg-warning/20 text-warning border-warning/30" },
    "no-permission": { badge: "権限なし", badgeClass: "bg-muted text-muted-foreground border-border" },
  }

  const handleConnect = () => {
    router.push(`/consent?service=${encodeURIComponent(module.name)}&icon=${module.icon}`)
  }

  const handleDisconnect = () => {
    setDisconnectDialogOpen(false)
    router.push("/tools")
  }

  const handleRequestTool = (toolId: string) => {
    setRequestTool(toolId)
    setRequestDialogOpen(true)
  }

  const handleSubmitRequest = () => {
    setRequestDialogOpen(false)
    setRequestTool(null)
    setRequestReason("")
  }

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={() => router.push("/tools")} className="mb-4">
        <ArrowLeft className="h-4 w-4 mr-2" />
        ツール一覧に戻る
      </Button>

      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <div className="w-16 h-16 rounded-xl bg-secondary flex items-center justify-center">
            <ServiceIcon icon={module.icon} className="h-8 w-8 text-foreground" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-foreground">{module.name}</h1>
              <Badge variant="outline" className={statusConfig[module.status].badgeClass}>
                {statusConfig[module.status].badge}
              </Badge>
            </div>
            <p className="text-muted-foreground mt-1">{module.description}</p>
          </div>
        </div>
        <div>
          {module.status === "connected" ? (
            <Button variant="outline" onClick={() => setDisconnectDialogOpen(true)}>
              <Unlink className="h-4 w-4 mr-2" />
              連携解除
            </Button>
          ) : module.status === "disconnected" ? (
            <Button onClick={handleConnect}>
              <LinkIcon className="h-4 w-4 mr-2" />
              連携する
            </Button>
          ) : null}
        </div>
      </div>

      <div>
        <h2 className="text-lg font-semibold text-foreground mb-4">利用可能なツール</h2>
        <div className="grid gap-4 md:grid-cols-2">
          {module.tools.map((tool) => (
            <Card key={tool.id} className={cn(!tool.hasPermission && "opacity-60")}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <CardTitle className="text-base font-mono">{tool.name}</CardTitle>
                  {!tool.hasPermission && (
                    <Badge variant="outline" className="bg-muted text-muted-foreground border-border">
                      <Lock className="h-3 w-3 mr-1" />
                      権限なし
                    </Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <p className="text-sm text-muted-foreground">{tool.description}</p>
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-2">パラメータ</p>
                  <div className="space-y-2">
                    {tool.parameters.map((param) => (
                      <div key={param.name} className="text-sm bg-secondary/50 p-2 rounded">
                        <div className="flex items-center gap-2">
                          <code className="text-primary">{param.name}</code>
                          <span className="text-muted-foreground text-xs">({param.type})</span>
                          {param.required && (
                            <Badge variant="outline" className="text-xs h-5">
                              必須
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">{param.description}</p>
                      </div>
                    ))}
                  </div>
                </div>
                {!tool.hasPermission && (
                  <Button variant="secondary" size="sm" className="w-full" onClick={() => handleRequestTool(tool.id)}>
                    利用を申請
                  </Button>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      <Dialog open={disconnectDialogOpen} onOpenChange={setDisconnectDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>連携を解除しますか？</DialogTitle>
            <DialogDescription>
              {module.name}との連携を解除すると、このサービスのツールが使用できなくなります。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDisconnectDialogOpen(false)}>
              キャンセル
            </Button>
            <Button variant="destructive" onClick={handleDisconnect}>
              連携解除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={requestDialogOpen} onOpenChange={setRequestDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>利用申請</DialogTitle>
            <DialogDescription>このツールへのアクセス権限を申請します</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>申請理由</Label>
              <Textarea
                placeholder="このツールを利用したい理由を入力してください"
                value={requestReason}
                onChange={(e) => setRequestReason(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRequestDialogOpen(false)}>
              キャンセル
            </Button>
            <Button onClick={handleSubmitRequest} disabled={!requestReason}>
              申請する
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
