"use client"

import { useState } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import {
  profiles as initialProfiles,
  type Profile,
  moduleDetails,
  services,
  organizationPlan,
  isPlanSufficient,
  getServiceRequiredPlan,
  getToolRequiredPlan,
  type PlanType,
} from "@/lib/data"
import { useAuth } from "@/lib/auth-context"
import { ServiceIcon } from "@/components/service-icon"
import { Plus, Lock, ChevronDown, Check, Minus, Pencil } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"

export default function ProfilesPage() {
  const { isAdmin } = useAuth()
  const [profileList, setProfileList] = useState<Profile[]>(initialProfiles)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [editProfileId, setEditProfileId] = useState<string | null>(null)
  const [newProfileName, setNewProfileName] = useState("")
  const [newProfileDescription, setNewProfileDescription] = useState("")
  const [expandedServices, setExpandedServices] = useState<string[]>([])

  const currentPlan = organizationPlan.currentPlan

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-[50vh]">
        <p className="text-muted-foreground">このページにアクセスする権限がありません</p>
      </div>
    )
  }

  const getPlanBadgeStyle = (plan: PlanType) => {
    switch (plan) {
      case "pro":
        return "bg-blue-500/20 text-blue-400 border-blue-500/30"
      case "max":
        return "bg-purple-500/20 text-purple-400 border-purple-500/30"
      default:
        return "bg-secondary text-secondary-foreground"
    }
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
    toast.success("プロファイルを作成しました")
  }

  const toggleServiceExpanded = (serviceId: string) => {
    setExpandedServices((prev) =>
      prev.includes(serviceId) ? prev.filter((id) => id !== serviceId) : [...prev, serviceId],
    )
  }

  const handleTogglePermission = (profileId: string, serviceId: string, toolId: string) => {
    const toolPlan = getToolRequiredPlan(serviceId, toolId)
    if (!isPlanSufficient(currentPlan, toolPlan)) return

    setProfileList((prev) =>
      prev.map((profile) => {
        if (profile.id !== profileId) return profile
        const currentTools = profile.modulePermissions[serviceId] || []
        const newTools = currentTools.includes(toolId)
          ? currentTools.filter((id) => id !== toolId)
          : [...currentTools, toolId]
        return {
          ...profile,
          modulePermissions: {
            ...profile.modulePermissions,
            [serviceId]: newTools,
          },
        }
      }),
    )
  }

  const hasPermission = (profile: Profile, serviceId: string, toolId: string) => {
    return profile.modulePermissions[serviceId]?.includes(toolId) || false
  }

  const handleSave = () => {
    toast.success("変更を保存しました")
  }

  // サービスをプランで利用可能なものだけフィルタ
  const availableServices = services.filter((s) => isPlanSufficient(currentPlan, getServiceRequiredPlan(s.id)))
  const lockedServices = services.filter((s) => !isPlanSufficient(currentPlan, getServiceRequiredPlan(s.id)))

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">プロファイル権限設定</h1>
          <p className="text-muted-foreground mt-1">プロファイルごとに利用可能な機能を設定</p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          新規プロファイル
        </Button>
      </div>

      <Card>
        <CardContent className="p-0 overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[200px] sticky left-0 bg-card z-10">サービス / 機能</TableHead>
                {profileList.map((profile) => (
                  <TableHead key={profile.id} className="text-center min-w-[120px]">
                    <div className="flex items-center justify-center gap-1">
                      {profile.name}
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => setEditProfileId(profile.id)}
                      >
                        <Pencil className="h-3 w-3" />
                      </Button>
                    </div>
                  </TableHead>
                ))}
                <TableHead className="w-[80px] text-center">プラン</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {availableServices.map((service) => {
                const module = moduleDetails[service.id]
                if (!module) return null
                const isExpanded = expandedServices.includes(service.id)
                const servicePlan = getServiceRequiredPlan(service.id)

                return (
                  <>
                    <TableRow
                      key={service.id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => toggleServiceExpanded(service.id)}
                    >
                      <TableCell className="sticky left-0 bg-card z-10">
                        <div className="flex items-center gap-2">
                          <ChevronDown className={cn("h-4 w-4 transition-transform", isExpanded && "rotate-180")} />
                          <ServiceIcon icon={service.icon} className="h-4 w-4" />
                          <span className="font-medium">{service.name}</span>
                        </div>
                      </TableCell>
                      {profileList.map((profile) => (
                        <TableCell key={profile.id} className="text-center">
                          <span className="text-muted-foreground text-sm">
                            {profile.modulePermissions[service.id]?.length || 0}/{module.tools.length}
                          </span>
                        </TableCell>
                      ))}
                      <TableCell className="text-center">
                        <Badge className={getPlanBadgeStyle(servicePlan)} variant="outline">
                          {servicePlan.toUpperCase()}
                        </Badge>
                      </TableCell>
                    </TableRow>
                    {isExpanded &&
                      module.tools.map((tool) => {
                        const toolPlan = getToolRequiredPlan(service.id, tool.id)
                        const isToolLocked = !isPlanSufficient(currentPlan, toolPlan)

                        return (
                          <TableRow key={`${service.id}-${tool.id}`} className="bg-muted/30">
                            <TableCell className="sticky left-0 bg-muted/30 z-10 pl-10">
                              <span className="text-sm">{tool.name}</span>
                            </TableCell>
                            {profileList.map((profile) => (
                              <TableCell key={profile.id} className="text-center">
                                {isToolLocked ? (
                                  <Badge className={getPlanBadgeStyle(toolPlan)} variant="outline">
                                    <Lock className="h-3 w-3 mr-1" />
                                    {toolPlan.toUpperCase()}
                                  </Badge>
                                ) : (
                                  <Checkbox
                                    checked={hasPermission(profile, service.id, tool.id)}
                                    onCheckedChange={() => handleTogglePermission(profile.id, service.id, tool.id)}
                                    onClick={(e) => e.stopPropagation()}
                                  />
                                )}
                              </TableCell>
                            ))}
                            <TableCell className="text-center">
                              <Badge className={getPlanBadgeStyle(toolPlan)} variant="outline">
                                {toolPlan.toUpperCase()}
                              </Badge>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                  </>
                )
              })}

              {/* Locked Services Section */}
              {lockedServices.length > 0 && (
                <>
                  <TableRow className="bg-muted/50">
                    <TableCell
                      colSpan={profileList.length + 2}
                      className="text-center text-sm text-muted-foreground py-2"
                    >
                      以下のサービスはプランアップグレードが必要です
                    </TableCell>
                  </TableRow>
                  {lockedServices.map((service) => {
                    const servicePlan = getServiceRequiredPlan(service.id)
                    return (
                      <TableRow key={service.id} className="opacity-50">
                        <TableCell className="sticky left-0 bg-card z-10">
                          <div className="flex items-center gap-2">
                            <Lock className="h-4 w-4 text-muted-foreground" />
                            <ServiceIcon icon={service.icon} className="h-4 w-4" />
                            <span className="font-medium">{service.name}</span>
                          </div>
                        </TableCell>
                        {profileList.map((profile) => (
                          <TableCell key={profile.id} className="text-center">
                            <Minus className="h-4 w-4 mx-auto text-muted-foreground" />
                          </TableCell>
                        ))}
                        <TableCell className="text-center">
                          <Badge className={getPlanBadgeStyle(servicePlan)} variant="outline">
                            <Lock className="h-3 w-3 mr-1" />
                            {servicePlan.toUpperCase()}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Legend */}
      <div className="flex items-center gap-4 text-sm text-muted-foreground">
        <span>凡例:</span>
        <div className="flex items-center gap-1">
          <Check className="h-4 w-4" /> 有効
        </div>
        <div className="flex items-center gap-1">
          <Minus className="h-4 w-4" /> 無効
        </div>
        <div className="flex items-center gap-1">
          <Lock className="h-4 w-4" /> プランアップグレードが必要
        </div>
      </div>

      <div className="flex justify-end">
        <Button onClick={handleSave}>変更を保存</Button>
      </div>

      {/* Create Profile Dialog */}
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

      {/* Edit Profile Dialog */}
      <Dialog open={!!editProfileId} onOpenChange={(open) => !open && setEditProfileId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>プロファイル編集</DialogTitle>
          </DialogHeader>
          {editProfileId && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>プロファイル名</Label>
                <Input
                  value={profileList.find((p) => p.id === editProfileId)?.name || ""}
                  onChange={(e) =>
                    setProfileList((prev) =>
                      prev.map((p) => (p.id === editProfileId ? { ...p, name: e.target.value } : p)),
                    )
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>説明</Label>
                <Textarea
                  value={profileList.find((p) => p.id === editProfileId)?.description || ""}
                  onChange={(e) =>
                    setProfileList((prev) =>
                      prev.map((p) => (p.id === editProfileId ? { ...p, description: e.target.value } : p)),
                    )
                  }
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditProfileId(null)}>
              キャンセル
            </Button>
            <Button
              onClick={() => {
                setEditProfileId(null)
                toast.success("プロファイルを更新しました")
              }}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
