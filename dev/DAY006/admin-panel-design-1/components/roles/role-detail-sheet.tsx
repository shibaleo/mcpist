"use client"

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  allPermissions,
  services,
  organizationPlan,
  isPlanSufficient,
  getServiceRequiredPlan,
  type Role,
  type PlanType,
} from "@/lib/data"
import { Shield, Users, Lock } from "lucide-react"
import { useState, useEffect } from "react"
import { useMediaQuery } from "@/hooks/use-media-query"
import { ServiceIcon } from "@/components/service-icon"
import { cn } from "@/lib/utils"

interface RoleDetailSheetProps {
  role: Role | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onUpdateRole?: (role: Role) => void
}

export function RoleDetailSheet({ role, open, onOpenChange, onUpdateRole }: RoleDetailSheetProps) {
  const [permissions, setPermissions] = useState<string[]>([])
  const [serviceSettings, setServiceSettings] = useState<Role["services"]>([])
  const [allowedServices, setAllowedServices] = useState<string[]>([])
  const isMobile = useMediaQuery("(max-width: 639px)")

  const currentPlan = organizationPlan.currentPlan

  useEffect(() => {
    if (role) {
      setPermissions(role.permissions)
      setServiceSettings(role.services)
      // Initialize allowed services from existing service settings
      setAllowedServices(role.services.map((s) => s.serviceId))
    }
  }, [role])

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

  const handlePermissionChange = (permissionId: string, checked: boolean) => {
    const newPermissions = checked ? [...permissions, permissionId] : permissions.filter((p) => p !== permissionId)
    setPermissions(newPermissions)
    if (role) {
      onUpdateRole?.({ ...role, permissions: newPermissions })
    }
  }

  const handleServiceSettingChange = (
    serviceId: string,
    field: "clientId" | "clientSecret" | "authMethod",
    value: string,
  ) => {
    const existingIndex = serviceSettings.findIndex((s) => s.serviceId === serviceId)
    let newSettings: Role["services"]

    if (existingIndex >= 0) {
      newSettings = serviceSettings.map((s, i) => (i === existingIndex ? { ...s, [field]: value } : s))
    } else {
      newSettings = [...serviceSettings, { serviceId, [field]: value, authMethod: "oauth" as const }]
    }

    setServiceSettings(newSettings)
    if (role) {
      onUpdateRole?.({ ...role, services: newSettings })
    }
  }

  const handleAllowedServiceToggle = (serviceId: string, checked: boolean) => {
    const requiredPlan = getServiceRequiredPlan(serviceId)
    if (!isPlanSufficient(currentPlan, requiredPlan)) return

    const newAllowed = checked ? [...allowedServices, serviceId] : allowedServices.filter((id) => id !== serviceId)
    setAllowedServices(newAllowed)

    // Update service settings accordingly
    if (role) {
      const newSettings = checked
        ? [...serviceSettings, { serviceId, authMethod: "oauth" as const }]
        : serviceSettings.filter((s) => s.serviceId !== serviceId)
      setServiceSettings(newSettings)
      onUpdateRole?.({ ...role, services: newSettings })
    }
  }

  if (!role) return null

  const content = (
    <>
      <div className="flex items-center gap-4 mb-6">
        <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center">
          <Shield className="h-6 w-6 text-foreground" />
        </div>
        <div>
          <h3 className="text-lg font-semibold">{role.name}</h3>
          <div className="flex items-center gap-1 text-sm text-muted-foreground">
            <Users className="h-4 w-4" />
            <span>{role.userCount}人のユーザー</span>
          </div>
        </div>
      </div>

      <Tabs defaultValue="info" className="flex-1">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="info">基本情報</TabsTrigger>
          <TabsTrigger value="permissions">権限設定</TabsTrigger>
          <TabsTrigger value="service-allow">サービス許可</TabsTrigger>
          <TabsTrigger value="services">認証設定</TabsTrigger>
        </TabsList>

        <TabsContent value="info" className="mt-6 space-y-4">
          <div className="space-y-2">
            <Label>ロール名</Label>
            <Input value={role.name} readOnly className="bg-muted" />
          </div>
          <div className="space-y-2">
            <Label>説明</Label>
            <Input value={role.description} readOnly className="bg-muted" />
          </div>
          <div className="space-y-2">
            <Label>ユーザー数</Label>
            <Input value={`${role.userCount}人`} readOnly className="bg-muted" />
          </div>
        </TabsContent>

        <TabsContent value="permissions" className="mt-6 space-y-4">
          <p className="text-sm text-muted-foreground mb-4">このロールに割り当てる権限を選択してください。</p>
          <div className="space-y-3">
            {allPermissions.map((permission) => (
              <div key={permission.id} className="flex items-start space-x-3 p-3 rounded-lg bg-muted/50">
                <Checkbox
                  id={permission.id}
                  checked={permissions.includes(permission.id)}
                  onCheckedChange={(checked) => handlePermissionChange(permission.id, checked === true)}
                />
                <div className="space-y-1">
                  <label htmlFor={permission.id} className="text-sm font-medium cursor-pointer">
                    {permission.label}
                  </label>
                  <p className="text-xs text-muted-foreground">{permission.description}</p>
                </div>
              </div>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="service-allow" className="mt-6 space-y-4">
          <p className="text-sm text-muted-foreground mb-4">
            このロールのユーザーが接続できるサービスを選択してください。
          </p>
          <div className="space-y-3">
            {services.map((service) => {
              const requiredPlan = getServiceRequiredPlan(service.id)
              const isLocked = !isPlanSufficient(currentPlan, requiredPlan)

              return (
                <div
                  key={service.id}
                  className={cn(
                    "flex items-center space-x-3 p-3 rounded-lg border",
                    isLocked && "opacity-50 bg-muted/30",
                  )}
                >
                  <Checkbox
                    id={`service-${service.id}`}
                    checked={allowedServices.includes(service.id)}
                    disabled={isLocked}
                    onCheckedChange={(checked) => handleAllowedServiceToggle(service.id, checked === true)}
                  />
                  <div className="w-8 h-8 rounded bg-secondary flex items-center justify-center">
                    {isLocked ? (
                      <Lock className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <ServiceIcon icon={service.icon} className="h-4 w-4" />
                    )}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <label
                        htmlFor={`service-${service.id}`}
                        className={cn("text-sm font-medium", !isLocked && "cursor-pointer")}
                      >
                        {service.name}
                      </label>
                      {isLocked && (
                        <Badge className={getPlanBadgeStyle(requiredPlan)}>
                          <Lock className="h-3 w-3 mr-1" />
                          {requiredPlan.toUpperCase()}
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">{service.description}</p>
                  </div>
                </div>
              )
            })}
          </div>
        </TabsContent>

        <TabsContent value="services" className="mt-6 space-y-6">
          <p className="text-sm text-muted-foreground">許可したサービスの認証設定を行います。</p>

          {allowedServices.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <p>サービス許可タブでサービスを選択してください</p>
            </div>
          ) : (
            allowedServices.map((serviceId) => {
              const service = services.find((s) => s.id === serviceId)
              if (!service) return null
              const setting = serviceSettings.find((s) => s.serviceId === serviceId)

              return (
                <div key={serviceId} className="space-y-4 p-4 rounded-lg border border-border">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <ServiceIcon icon={service.icon} className="h-4 w-4" />
                      <span className="font-medium">{service.name}</span>
                      {setting && (
                        <Badge variant="secondary" className="text-xs">
                          設定済
                        </Badge>
                      )}
                    </div>
                  </div>

                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label className="text-xs">認証方式</Label>
                      <Select
                        value={setting?.authMethod || "oauth"}
                        onValueChange={(value) =>
                          handleServiceSettingChange(serviceId, "authMethod", value as "oidc" | "oauth" | "apikey")
                        }
                      >
                        <SelectTrigger className="h-9">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="oidc">OIDC（推奨）</SelectItem>
                          <SelectItem value="oauth">OAuth</SelectItem>
                          <SelectItem value="apikey">APIキー</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label className="text-xs">Client ID</Label>
                      <Input
                        placeholder="クライアントIDを入力"
                        value={setting?.clientId || ""}
                        onChange={(e) => handleServiceSettingChange(serviceId, "clientId", e.target.value)}
                        className="h-9"
                      />
                    </div>

                    <div className="space-y-2">
                      <Label className="text-xs">Client Secret</Label>
                      <Input
                        type="password"
                        placeholder="クライアントシークレットを入力"
                        value={setting?.clientSecret || ""}
                        onChange={(e) => handleServiceSettingChange(serviceId, "clientSecret", e.target.value)}
                        className="h-9"
                      />
                    </div>
                  </div>
                </div>
              )
            })
          )}
        </TabsContent>
      </Tabs>
    </>
  )

  // Mobile: Full screen dialog
  if (isMobile) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-full h-[100dvh] max-h-[100dvh] rounded-none p-6 flex flex-col">
          <DialogHeader>
            <DialogTitle>ロール詳細</DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-auto">{content}</div>
        </DialogContent>
      </Dialog>
    )
  }

  // Desktop/Tablet: Sheet
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg overflow-auto">
        <SheetHeader>
          <SheetTitle>ロール詳細</SheetTitle>
        </SheetHeader>
        <div className="mt-6">{content}</div>
      </SheetContent>
    </Sheet>
  )
}
