"use client"

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import type { Role } from "@/lib/data"
import { Users } from "lucide-react"

interface RoleTableProps {
  roles: Role[]
  onRoleSelect: (role: Role) => void
  selectedRoleId?: string
}

export function RoleTable({ roles, onRoleSelect, selectedRoleId }: RoleTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[200px]">ロール名</TableHead>
          <TableHead className="hidden md:table-cell">説明</TableHead>
          <TableHead className="hidden lg:table-cell">権限数</TableHead>
          <TableHead className="text-right">ユーザー数</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {roles.map((role) => (
          <TableRow
            key={role.id}
            className={`cursor-pointer transition-colors hover:bg-muted/50 ${selectedRoleId === role.id ? "bg-muted" : ""}`}
            onClick={() => onRoleSelect(role)}
          >
            <TableCell className="font-medium">{role.name}</TableCell>
            <TableCell className="hidden md:table-cell text-muted-foreground">{role.description}</TableCell>
            <TableCell className="hidden lg:table-cell">
              <Badge variant="secondary">{role.permissions.length}件</Badge>
            </TableCell>
            <TableCell className="text-right">
              <div className="flex items-center justify-end gap-1.5 text-muted-foreground">
                <Users className="h-4 w-4" />
                <span>{role.userCount}</span>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
