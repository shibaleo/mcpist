"use client"

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { roles, type User } from "@/lib/data"
import { X, Plus, ChevronsUpDown } from "lucide-react"
import { useState } from "react"
import { cn } from "@/lib/utils"
import { useMediaQuery } from "@/hooks/use-media-query"

interface UserDetailSheetProps {
  user: User | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onUpdateRoles?: (userId: string, roles: string[]) => void
}

export function UserDetailSheet({ user, open, onOpenChange, onUpdateRoles }: UserDetailSheetProps) {
  const [userRoles, setUserRoles] = useState<string[]>(user?.roles || [])
  const [comboboxOpen, setComboboxOpen] = useState(false)
  const isMobile = useMediaQuery("(max-width: 639px)")

  const availableRoles = roles.map((r) => r.name).filter((r) => !userRoles.includes(r))

  const handleAddRole = (role: string) => {
    const newRoles = [...userRoles, role]
    setUserRoles(newRoles)
    if (user) {
      onUpdateRoles?.(user.id, newRoles)
    }
    setComboboxOpen(false)
  }

  const handleRemoveRole = (role: string) => {
    const newRoles = userRoles.filter((r) => r !== role)
    setUserRoles(newRoles)
    if (user) {
      onUpdateRoles?.(user.id, newRoles)
    }
  }

  // Update local state when user changes
  if (user && user.roles.join(",") !== userRoles.join(",") && !open) {
    setUserRoles(user.roles)
  }

  if (!user) return null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side={isMobile ? "bottom" : "right"} className={cn(isMobile && "h-[90vh]", "w-full sm:max-w-lg")}>
        <SheetHeader>
          <SheetTitle>ユーザー詳細</SheetTitle>
        </SheetHeader>

        <Tabs defaultValue="info" className="mt-6">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="info">基本情報</TabsTrigger>
            <TabsTrigger value="roles">ロール割当</TabsTrigger>
          </TabsList>

          <TabsContent value="info" className="mt-6 space-y-6">
            <div className="flex items-center gap-4">
              <Avatar className="h-16 w-16">
                <AvatarImage src={user.avatar || "/placeholder.svg"} />
                <AvatarFallback className="bg-primary text-primary-foreground text-xl">
                  {user.name.slice(0, 2)}
                </AvatarFallback>
              </Avatar>
              <div>
                <h3 className="text-lg font-semibold">{user.name}</h3>
                <p className="text-sm text-muted-foreground">{user.email}</p>
              </div>
            </div>

            <div className="space-y-4">
              <div className="space-y-2">
                <Label>名前</Label>
                <Input value={user.name} readOnly className="bg-muted" />
              </div>
              <div className="space-y-2">
                <Label>メールアドレス</Label>
                <Input value={user.email} readOnly className="bg-muted" />
              </div>
              <div className="space-y-2">
                <Label>最終ログイン</Label>
                <Input value={user.lastLogin} readOnly className="bg-muted" />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="roles" className="mt-6 space-y-6">
            <div className="space-y-4">
              <Label>現在のロール</Label>
              <div className="flex flex-wrap gap-2">
                {userRoles.map((role) => (
                  <Badge key={role} variant="secondary" className="pl-3 pr-1 py-1.5 text-sm">
                    {role}
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-4 w-4 p-0 ml-2 hover:bg-destructive/20"
                      onClick={() => handleRemoveRole(role)}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </Badge>
                ))}
                {userRoles.length === 0 && (
                  <p className="text-sm text-muted-foreground">ロールが割り当てられていません</p>
                )}
              </div>
            </div>

            <div className="space-y-4">
              <Label>ロールを追加</Label>
              <Popover open={comboboxOpen} onOpenChange={setComboboxOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={comboboxOpen}
                    className="w-full justify-between bg-transparent"
                    disabled={availableRoles.length === 0}
                  >
                    {availableRoles.length === 0 ? "追加可能なロールがありません" : "ロールを選択..."}
                    <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-full p-0" align="start">
                  <Command>
                    <CommandInput placeholder="ロールを検索..." />
                    <CommandList>
                      <CommandEmpty>ロールが見つかりません</CommandEmpty>
                      <CommandGroup>
                        {availableRoles.map((role) => (
                          <CommandItem key={role} value={role} onSelect={() => handleAddRole(role)}>
                            <Plus className="mr-2 h-4 w-4" />
                            {role}
                          </CommandItem>
                        ))}
                      </CommandGroup>
                    </CommandList>
                  </Command>
                </PopoverContent>
              </Popover>
            </div>
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  )
}
