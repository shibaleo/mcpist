"use client"

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { ServiceIcon } from "@/components/service-icon"
import type { Service } from "@/lib/data"
import { Check } from "lucide-react"
import { useState } from "react"

interface ConnectDialogProps {
  service: Service | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function ConnectDialog({ service, open, onOpenChange, onSuccess }: ConnectDialogProps) {
  const [connecting, setConnecting] = useState(false)
  const [connected, setConnected] = useState(false)

  const handleConnect = async () => {
    setConnecting(true)
    // Simulate OAuth flow
    await new Promise((resolve) => setTimeout(resolve, 1500))
    setConnecting(false)
    setConnected(true)
    setTimeout(() => {
      onSuccess?.()
      setConnected(false)
      onOpenChange(false)
    }, 1500)
  }

  if (!service) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3 mb-2">
            <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center">
              <ServiceIcon icon={service.icon} className="h-6 w-6" />
            </div>
            <div>
              <DialogTitle>{service.name}</DialogTitle>
              <DialogDescription>アカウント連携</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {connected ? (
          <div className="py-8 text-center">
            <div className="w-16 h-16 rounded-full bg-success/20 flex items-center justify-center mx-auto mb-4">
              <Check className="h-8 w-8 text-success" />
            </div>
            <p className="text-foreground font-medium">連携が完了しました</p>
            <p className="text-sm text-muted-foreground mt-1">{service.name}との連携が正常に完了しました。</p>
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {service.name}
              と連携すると、MCPサーバーから以下の操作が可能になります。
            </p>
            <ul className="text-sm text-muted-foreground space-y-2">
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 rounded-full bg-primary" />
                データの読み取り・書き込み
              </li>
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 rounded-full bg-primary" />
                自動同期と通知
              </li>
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 rounded-full bg-primary" />
                APIアクセス
              </li>
            </ul>
            <Button className="w-full" onClick={handleConnect} disabled={connecting}>
              {connecting ? "接続中..." : `${service.name}でログイン`}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
