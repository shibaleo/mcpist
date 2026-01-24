"use client"

import Link from "next/link"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ServiceIcon } from "@/components/service-icon"
import type { Service } from "@/lib/data"
import { cn } from "@/lib/utils"

interface ServiceCardProps {
  service: Service
  onConnect?: () => void
  onDisconnect?: () => void
  onRequest?: () => void
  disabled?: boolean
}

export function ServiceCard({ service, onConnect, onDisconnect, onRequest, disabled }: ServiceCardProps) {
  const statusConfig = {
    connected: {
      badge: "連携済",
      badgeClass: "bg-success/20 text-success border-success/30",
      action: "解除",
      actionVariant: "outline" as const,
      onClick: onDisconnect,
    },
    disconnected: {
      badge: "未連携",
      badgeClass: "bg-warning/20 text-warning border-warning/30",
      action: "連携する",
      actionVariant: "default" as const,
      onClick: onConnect,
    },
    "no-permission": {
      badge: "権限なし",
      badgeClass: "bg-muted text-muted-foreground border-border",
      action: "利用を申請",
      actionVariant: "secondary" as const,
      onClick: onRequest,
    },
  }

  const config = statusConfig[service.status]

  return (
    <Card className={cn("transition-all hover:border-primary/50", disabled && "opacity-60")}>
      <CardContent className="p-4">
        <Link href={`/tools/${service.id}`} className="block">
          <div className="flex items-start gap-4">
            <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center shrink-0">
              <ServiceIcon icon={service.icon} className="h-6 w-6 text-foreground" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <h3 className="font-medium text-foreground truncate">{service.name}</h3>
                <Badge variant="outline" className={cn("text-xs shrink-0", config.badgeClass)}>
                  {config.badge}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground line-clamp-2">{service.description}</p>
            </div>
          </div>
        </Link>
        <div className="mt-4 flex justify-end">
          <Button
            variant={config.actionVariant}
            size="sm"
            onClick={(e) => {
              e.preventDefault()
              config.onClick?.()
            }}
            disabled={disabled}
          >
            {config.action}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
