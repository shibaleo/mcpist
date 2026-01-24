"use client"

import { useState } from "react"
import { roles as initialRoles, type Role } from "@/lib/data"
import { RoleTable } from "@/components/roles/role-table"
import { RoleCard } from "@/components/roles/role-card"
import { RoleDetailSheet } from "@/components/roles/role-detail-sheet"
import { CreateRoleDialog } from "@/components/roles/create-role-dialog"
import { Button } from "@/components/ui/button"
import { Plus } from "lucide-react"
import { useMediaQuery } from "@/hooks/use-media-query"

export default function RolesPage() {
  const [roleList, setRoleList] = useState(initialRoles)
  const [selectedRole, setSelectedRole] = useState<Role | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const isMobile = useMediaQuery("(max-width: 767px)")

  const handleRoleSelect = (role: Role) => {
    setSelectedRole(role)
    setSheetOpen(true)
  }

  const handleUpdateRole = (updatedRole: Role) => {
    setRoleList((prev) => prev.map((r) => (r.id === updatedRole.id ? updatedRole : r)))
    setSelectedRole(updatedRole)
  }

  const handleCreateRole = (newRole: Omit<Role, "id">) => {
    const role: Role = {
      ...newRole,
      id: `${Date.now()}`,
    }
    setRoleList((prev) => [...prev, role])
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Roles</h1>
          <p className="text-muted-foreground mt-1">ロールと権限の管理</p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          新規作成
        </Button>
      </div>

      {isMobile ? (
        // Mobile: Card layout
        <div className="space-y-3">
          {roleList.map((role) => (
            <RoleCard
              key={role.id}
              role={role}
              onSelect={() => handleRoleSelect(role)}
              selected={selectedRole?.id === role.id}
            />
          ))}
        </div>
      ) : (
        // Desktop/Tablet: Table layout
        <div className="border border-border rounded-lg">
          <RoleTable roles={roleList} onRoleSelect={handleRoleSelect} selectedRoleId={selectedRole?.id} />
        </div>
      )}

      <RoleDetailSheet
        role={selectedRole}
        open={sheetOpen}
        onOpenChange={setSheetOpen}
        onUpdateRole={handleUpdateRole}
      />

      <CreateRoleDialog open={createDialogOpen} onOpenChange={setCreateDialogOpen} onCreate={handleCreateRole} />
    </div>
  )
}
