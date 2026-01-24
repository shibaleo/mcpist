"use client"

import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import type { User } from "@/lib/data"
import { cn } from "@/lib/utils"

interface UserCardProps {
  user: User
  onSelect: () => void
  selected?: boolean
}

export function UserCard({ user, onSelect, selected }: UserCardProps) {
  return (
    <Card
      className={cn("cursor-pointer transition-colors hover:bg-muted/50", selected && "ring-2 ring-primary")}
      onClick={onSelect}
    >
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <Avatar className="h-10 w-10">
            <AvatarImage src={user.avatar || "/placeholder.svg"} />
            <AvatarFallback className="bg-primary text-primary-foreground text-sm">
              {user.name.slice(0, 2)}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1 min-w-0">
            <p className="font-medium text-foreground">{user.name}</p>
            <p className="text-sm text-muted-foreground truncate">{user.email}</p>
          </div>
        </div>
        <div className="mt-3 flex flex-wrap gap-1">
          {user.roles.map((role) => (
            <Badge key={role} variant="secondary" className="text-xs">
              {role}
            </Badge>
          ))}
        </div>
        <p className="mt-2 text-xs text-muted-foreground">最終ログイン: {user.lastLogin}</p>
      </CardContent>
    </Card>
  )
}
