"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { ServiceIcon } from "@/components/service-icon"
import { useAuth } from "@/lib/auth-context"
import {
  services,
  moduleDetails,
  organizationPlan,
  userToolPreferences as initialPreferences,
  isPlanSufficient,
  getServiceRequiredPlan,
  getToolRequiredPlan,
  type UserToolPreference,
  type PlanType,
} from "@/lib/data"
import { Lock, ArrowRight, Info } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import Link from "next/link"

export default function MyPreferencesPage() {
  const { user } = useAuth()
  const [preferences, setPreferences] = useState<UserToolPreference[]>(initialPreferences)

  const currentPlan = organizationPlan.currentPlan
  const myPreference = preferences.find((p) => p.userId === user?.id) || {
    userId: user?.id || "",
    enabledTools: {},
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

  const handleToggleTool = (serviceId: string, toolId: string) => {
    const toolPlan = getToolRequiredPlan(serviceId, toolId)
    if (!isPlanSufficient(currentPlan, toolPlan)) return

    setPreferences((prev) => {
      const existing = prev.find((p) => p.userId === user?.id)
      if (existing) {
        const currentTools = existing.enabledTools[serviceId] || []
        const newTools = currentTools.includes(toolId)
          ? currentTools.filter((id) => id !== toolId)
          : [...currentTools, toolId]
        return prev.map((p) =>
          p.userId === user?.id ? { ...p, enabledTools: { ...p.enabledTools, [serviceId]: newTools } } : p,
        )
      } else {
        return [...prev, { userId: user?.id || "", enabledTools: { [serviceId]: [toolId] } }]
      }
    })
  }

  const handleSelectAll = (serviceId: string) => {
    const module = moduleDetails[serviceId]
    if (!module) return

    const availableTools = module.tools.filter((t) =>
      isPlanSufficient(currentPlan, getToolRequiredPlan(serviceId, t.id)),
    )

    setPreferences((prev) => {
      const existing = prev.find((p) => p.userId === user?.id)
      const newTools = availableTools.map((t) => t.id)
      if (existing) {
        return prev.map((p) =>
          p.userId === user?.id ? { ...p, enabledTools: { ...p.enabledTools, [serviceId]: newTools } } : p,
        )
      } else {
        return [...prev, { userId: user?.id || "", enabledTools: { [serviceId]: newTools } }]
      }
    })
  }

  const isToolEnabled = (serviceId: string, toolId: string) => {
    return myPreference.enabledTools[serviceId]?.includes(toolId) || false
  }

  const handleSave = () => {
    toast.success("設定を保存しました")
  }

  // サービスをプランでグループ化
  const availableServices = services.filter((s) => isPlanSufficient(currentPlan, getServiceRequiredPlan(s.id)))
  const lockedServices = services.filter((s) => !isPlanSufficient(currentPlan, getServiceRequiredPlan(s.id)))

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">機能設定</h1>
        <p className="text-muted-foreground mt-1">利用する機能を選択してください</p>
      </div>

      <Card className="bg-secondary/30">
        <CardContent className="p-4">
          <div className="flex items-start gap-3">
            <Info className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
            <p className="text-sm text-muted-foreground">
              管理者が許可した機能の中から、実際に使用する機能を選択できます。
              不要な機能を無効にすることで、AIアシスタントの動作を最適化できます。
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="space-y-6">
        {availableServices.map((service) => {
          const module = moduleDetails[service.id]
          if (!module) return null

          return (
            <Card key={service.id}>
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                      <ServiceIcon icon={service.icon} className="h-5 w-5 text-foreground" />
                    </div>
                    <div>
                      <CardTitle className="text-base">{service.name}</CardTitle>
                      <CardDescription>{service.description}</CardDescription>
                    </div>
                  </div>
                  <Button variant="outline" size="sm" onClick={() => handleSelectAll(service.id)}>
                    全選択
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                {module.tools.map((tool) => {
                  const toolPlan = getToolRequiredPlan(service.id, tool.id)
                  const isToolLocked = !isPlanSufficient(currentPlan, toolPlan)

                  return (
                    <div
                      key={tool.id}
                      className={cn(
                        "flex items-start gap-3 p-3 rounded-lg border",
                        isToolLocked && "opacity-60 bg-muted/50",
                      )}
                    >
                      <Checkbox
                        checked={isToolEnabled(service.id, tool.id)}
                        disabled={isToolLocked}
                        onCheckedChange={() => handleToggleTool(service.id, tool.id)}
                        className="mt-1"
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-sm">{tool.name}</span>
                          {isToolLocked && (
                            <Badge className={getPlanBadgeStyle(toolPlan)}>
                              <Lock className="h-3 w-3 mr-1" />
                              {toolPlan.toUpperCase()}専用
                            </Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {tool.description}
                          {isToolLocked && "（プランアップグレードが必要）"}
                        </p>
                      </div>
                    </div>
                  )
                })}
              </CardContent>
            </Card>
          )
        })}

        {/* Locked Services */}
        {lockedServices.map((service) => {
          const servicePlan = getServiceRequiredPlan(service.id)

          return (
            <Card key={service.id} className="opacity-60">
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center">
                      <Lock className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{service.name}</span>
                        <Badge className={getPlanBadgeStyle(servicePlan)}>
                          <Lock className="h-3 w-3 mr-1" />
                          {servicePlan.toUpperCase()}専用
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        このサービスは{servicePlan.toUpperCase()}プラン以上で利用可能です
                      </p>
                    </div>
                  </div>
                  <Link href="/billing">
                    <Button variant="outline" size="sm">
                      プランをアップグレード
                      <ArrowRight className="h-4 w-4 ml-1" />
                    </Button>
                  </Link>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      <div className="flex justify-end">
        <Button onClick={handleSave}>設定を保存</Button>
      </div>
    </div>
  )
}
