"use client"

import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import type { Role } from "@/lib/data"
import { Users, Shield } from "lucide-react"
import { cn } from "@/lib/utils"

interface RoleCardProps {
  role: Role
  onSelect: () => void
  selected?: boolean
}

export function RoleCard({ role, onSelect, selected }: RoleCardProps) {
  return (
    <Card
      className={cn("cursor-pointer transition-colors hover:bg-muted/50", selected && "ring-2 ring-primary")}
      onClick={onSelect}
    >
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center shrink-0">
            <Shield className="h-5 w-5 text-foreground" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="font-medium text-foreground">{role.name}</p>
            <p className="text-sm text-muted-foreground line-clamp-1">{role.description}</p>
          </div>
        </div>
        <div className="mt-3 flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            {role.permissions.length}件の権限
          </Badge>
          <div className="flex items-center gap-1 text-sm text-muted-foreground">
            <Users className="h-4 w-4" />
            <span>{role.userCount}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
