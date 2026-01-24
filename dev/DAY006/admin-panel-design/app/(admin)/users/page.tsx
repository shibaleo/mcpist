"use client"

import { useState } from "react"
import { users as initialUsers, type User } from "@/lib/data"
import { UserTable } from "@/components/users/user-table"
import { UserCard } from "@/components/users/user-card"
import { UserDetailSheet } from "@/components/users/user-detail-sheet"
import { useMediaQuery } from "@/hooks/use-media-query"

export default function UsersPage() {
  const [userList, setUserList] = useState(initialUsers)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const isMobile = useMediaQuery("(max-width: 767px)")

  const handleUserSelect = (user: User) => {
    setSelectedUser(user)
    setSheetOpen(true)
  }

  const handleUpdateRoles = (userId: string, roles: string[]) => {
    setUserList((prev) => prev.map((u) => (u.id === userId ? { ...u, roles } : u)))
    if (selectedUser?.id === userId) {
      setSelectedUser((prev) => (prev ? { ...prev, roles } : null))
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Users</h1>
        <p className="text-muted-foreground mt-1">ユーザーの管理とロール割当</p>
      </div>

      {isMobile ? (
        // Mobile: Card layout
        <div className="space-y-3">
          {userList.map((user) => (
            <UserCard
              key={user.id}
              user={user}
              onSelect={() => handleUserSelect(user)}
              selected={selectedUser?.id === user.id}
            />
          ))}
        </div>
      ) : (
        // Desktop/Tablet: Table layout
        <div className="border border-border rounded-lg">
          <UserTable users={userList} onUserSelect={handleUserSelect} selectedUserId={selectedUser?.id} />
        </div>
      )}

      <UserDetailSheet
        user={selectedUser}
        open={sheetOpen}
        onOpenChange={setSheetOpen}
        onUpdateRoles={handleUpdateRoles}
      />
    </div>
  )
}
