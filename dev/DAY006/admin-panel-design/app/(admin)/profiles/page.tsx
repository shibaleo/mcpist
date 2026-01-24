"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { profiles, type Profile, moduleDetails, roles } from "@/lib/data"
import { useAuth } from "@/lib/auth-context"
import { useMediaQuery } from "@/hooks/use-media-query"
import { ServiceIcon } from "@/components/service-icon"
import { Plus, Layers } from "lucide-react"
import { cn } from "@/lib/utils"

export default function ProfilesPage() {
  const { isAdmin } = useAuth()
  const [profileList, setProfileList] = useState<Profile[]>(profiles)
  const [selectedProfile, setSelectedProfile] = useState<Profile | null>(null)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [newProfileName, setNewProfileName] = useState("")
  const [newProfileDescription, setNewProfileDescription] = useState("")
  const isMobile = useMediaQuery("(max-width: 768px)")

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-[50vh]">
        <p className="text-muted-foreground">このページにアクセスする権限がありません</p>
      </div>
    )
  }

  const handleCreateProfile = () => {
    const newProfile: Profile = {
      id: String(profileList.length + 1),
      name: newProfileName,
      description: newProfileDescription,
      appliedRoles: [],
      modulePermissions: {},
    }
    setProfileList([...profileList, newProfile])
    setCreateDialogOpen(false)
    setNewProfileName("")
    setNewProfileDescription("")
    setSelectedProfile(newProfile)
  }

  const handleTogglePermission = (moduleId: string, toolId: string) => {
    if (!selectedProfile) return
    const currentPermissions = selectedProfile.modulePermissions[moduleId] || []
    const newPermissions = currentPermissions.includes(toolId)
      ? currentPermissions.filter((id) => id !== toolId)
      : [...currentPermissions, toolId]

    const updatedProfile = {
      ...selectedProfile,
      modulePermissions: {
        ...selectedProfile.modulePermissions,
        [moduleId]: newPermissions,
      },
    }
    setSelectedProfile(updatedProfile)
    setProfileList(profileList.map((p) => (p.id === updatedProfile.id ? updatedProfile : p)))
  }

  const handleSelectAllModule = (moduleId: string, select: boolean) => {
    if (!selectedProfile) return
    const module = moduleDetails[moduleId]
    if (!module) return

    const updatedProfile = {
      ...selectedProfile,
      modulePermissions: {
        ...selectedProfile.modulePermissions,
        [moduleId]: select ? module.tools.map((t) => t.id) : [],
      },
    }
    setSelectedProfile(updatedProfile)
    setProfileList(profileList.map((p) => (p.id === updatedProfile.id ? updatedProfile : p)))
  }

  const handleToggleRole = (roleId: string) => {
    if (!selectedProfile) return
    const newRoles = selectedProfile.appliedRoles.includes(roleId)
      ? selectedProfile.appliedRoles.filter((id) => id !== roleId)
      : [...selectedProfile.appliedRoles, roleId]

    const updatedProfile = { ...selectedProfile, appliedRoles: newRoles }
    setSelectedProfile(updatedProfile)
    setProfileList(profileList.map((p) => (p.id === updatedProfile.id ? updatedProfile : p)))
  }

  const getRoleNameById = (roleId: string) => roles.find((r) => r.id === roleId)?.name || roleId

  const SheetOrDialog = isMobile ? Dialog : Sheet
  const SheetOrDialogContent = isMobile ? DialogContent : SheetContent

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">プロファイル管理</h1>
          <p className="text-muted-foreground mt-1">権限プロファイルを管理します</p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          新規作成
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">プロファイル一覧</CardTitle>
        </CardHeader>
        <CardContent>
          {isMobile ? (
            <div className="space-y-3">
              {profileList.map((profile) => (
                <Card
                  key={profile.id}
                  className="p-4 cursor-pointer hover:bg-accent/50 transition-colors"
                  onClick={() => setSelectedProfile(profile)}
                >
                  <div className="flex items-start gap-3">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                      <Layers className="h-5 w-5 text-foreground" />
                    </div>
                    <div className="flex-1">
                      <p className="font-medium text-foreground">{profile.name}</p>
                      <p className="text-sm text-muted-foreground line-clamp-1">{profile.description}</p>
                      <p className="text-xs text-muted-foreground mt-1">適用ロール: {profile.appliedRoles.length}件</p>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名前</TableHead>
                  <TableHead>説明</TableHead>
                  <TableHead>適用ロール数</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {profileList.map((profile) => (
                  <TableRow
                    key={profile.id}
                    className="cursor-pointer hover:bg-accent/50"
                    onClick={() => setSelectedProfile(profile)}
                  >
                    <TableCell className="font-medium">{profile.name}</TableCell>
                    <TableCell className="text-muted-foreground">{profile.description}</TableCell>
                    <TableCell>{profile.appliedRoles.length}件</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <SheetOrDialog open={!!selectedProfile} onOpenChange={() => setSelectedProfile(null)}>
        <SheetOrDialogContent className={cn(isMobile ? "max-w-full h-full" : "sm:max-w-xl")}>
          {selectedProfile && (
            <>
              {!isMobile && (
                <SheetHeader>
                  <SheetTitle>{selectedProfile.name}</SheetTitle>
                  <SheetDescription>{selectedProfile.description}</SheetDescription>
                </SheetHeader>
              )}
              {isMobile && (
                <DialogHeader>
                  <DialogTitle>{selectedProfile.name}</DialogTitle>
                  <DialogDescription>{selectedProfile.description}</DialogDescription>
                </DialogHeader>
              )}
              <div className="mt-4">
                <Tabs defaultValue="basic">
                  <TabsList className="w-full">
                    <TabsTrigger value="basic" className="flex-1">
                      基本情報
                    </TabsTrigger>
                    <TabsTrigger value="permissions" className="flex-1">
                      権限設定
                    </TabsTrigger>
                    <TabsTrigger value="roles" className="flex-1">
                      適用ロール
                    </TabsTrigger>
                  </TabsList>
                  <TabsContent value="basic" className="mt-4 space-y-4">
                    <div className="space-y-2">
                      <Label>プロファイル名</Label>
                      <Input value={selectedProfile.name} readOnly />
                    </div>
                    <div className="space-y-2">
                      <Label>説明</Label>
                      <Textarea value={selectedProfile.description} readOnly />
                    </div>
                  </TabsContent>
                  <TabsContent value="permissions" className="mt-4">
                    <Accordion type="multiple" className="w-full">
                      {Object.entries(moduleDetails).map(([moduleId, module]) => {
                        const permissions = selectedProfile.modulePermissions[moduleId] || []
                        return (
                          <AccordionItem key={moduleId} value={moduleId}>
                            <AccordionTrigger className="hover:no-underline">
                              <div className="flex items-center gap-3">
                                <ServiceIcon icon={module.icon} className="h-5 w-5" />
                                <span>{module.name}</span>
                                <Badge variant="secondary" className="ml-2">
                                  {permissions.length}/{module.tools.length}
                                </Badge>
                              </div>
                            </AccordionTrigger>
                            <AccordionContent>
                              <div className="space-y-3 pl-8">
                                <div className="flex gap-2">
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => handleSelectAllModule(moduleId, true)}
                                  >
                                    全選択
                                  </Button>
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => handleSelectAllModule(moduleId, false)}
                                  >
                                    全解除
                                  </Button>
                                </div>
                                {module.tools.map((tool) => (
                                  <div key={tool.id} className="flex items-start gap-3">
                                    <Checkbox
                                      id={`${moduleId}-${tool.id}`}
                                      checked={permissions.includes(tool.id)}
                                      onCheckedChange={() => handleTogglePermission(moduleId, tool.id)}
                                    />
                                    <div>
                                      <label
                                        htmlFor={`${moduleId}-${tool.id}`}
                                        className="text-sm font-medium cursor-pointer"
                                      >
                                        {tool.name}
                                      </label>
                                      <p className="text-xs text-muted-foreground">{tool.description}</p>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </AccordionContent>
                          </AccordionItem>
                        )
                      })}
                    </Accordion>
                  </TabsContent>
                  <TabsContent value="roles" className="mt-4">
                    <div className="space-y-3">
                      <p className="text-sm text-muted-foreground">このプロファイルを適用するロールを選択</p>
                      {roles.map((role) => (
                        <div key={role.id} className="flex items-center gap-3 p-3 bg-secondary/50 rounded-lg">
                          <Checkbox
                            id={`role-${role.id}`}
                            checked={selectedProfile.appliedRoles.includes(role.id)}
                            onCheckedChange={() => handleToggleRole(role.id)}
                          />
                          <div>
                            <label htmlFor={`role-${role.id}`} className="text-sm font-medium cursor-pointer">
                              {role.name}
                            </label>
                            <p className="text-xs text-muted-foreground">{role.description}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </TabsContent>
                </Tabs>
              </div>
            </>
          )}
        </SheetOrDialogContent>
      </SheetOrDialog>

      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新規プロファイル作成</DialogTitle>
            <DialogDescription>新しい権限プロファイルを作成します</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>プロファイル名</Label>
              <Input
                placeholder="例: 営業チーム標準"
                value={newProfileName}
                onChange={(e) => setNewProfileName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>説明</Label>
              <Textarea
                placeholder="このプロファイルの説明を入力"
                value={newProfileDescription}
                onChange={(e) => setNewProfileDescription(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
              キャンセル
            </Button>
            <Button onClick={handleCreateProfile} disabled={!newProfileName}>
              作成
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
