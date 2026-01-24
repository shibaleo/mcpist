"use client"

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import type { User } from "@/lib/data"

interface UserTableProps {
  users: User[]
  onUserSelect: (user: User) => void
  selectedUserId?: string
}

export function UserTable({ users, onUserSelect, selectedUserId }: UserTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[250px]">名前</TableHead>
          <TableHead className="hidden md:table-cell">メール</TableHead>
          <TableHead>ロール</TableHead>
          <TableHead className="hidden lg:table-cell text-right">最終ログイン</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((user) => (
          <TableRow
            key={user.id}
            className={`cursor-pointer transition-colors hover:bg-muted/50 ${selectedUserId === user.id ? "bg-muted" : ""}`}
            onClick={() => onUserSelect(user)}
          >
            <TableCell>
              <div className="flex items-center gap-3">
                <Avatar className="h-8 w-8">
                  <AvatarImage src={user.avatar || "/placeholder.svg"} />
                  <AvatarFallback className="bg-primary text-primary-foreground text-xs">
                    {user.name.slice(0, 2)}
                  </AvatarFallback>
                </Avatar>
                <span className="font-medium">{user.name}</span>
              </div>
            </TableCell>
            <TableCell className="hidden md:table-cell text-muted-foreground">{user.email}</TableCell>
            <TableCell>
              <div className="flex flex-wrap gap-1">
                {user.roles.map((role) => (
                  <Badge key={role} variant="secondary" className="text-xs">
                    {role}
                  </Badge>
                ))}
              </div>
            </TableCell>
            <TableCell className="hidden lg:table-cell text-right text-muted-foreground">{user.lastLogin}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
