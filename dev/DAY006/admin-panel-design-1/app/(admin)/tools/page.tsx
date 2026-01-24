"use client"

import { useState } from "react"
import Link from "next/link"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Checkbox } from "@/components/ui/checkbox"
import { ServiceIcon } from "@/components/service-icon"
import { useAuth } from "@/lib/auth-context"
import {
  services,
  moduleDetails,
  organizationPlan,
  isPlanSufficient,
  getServiceRequiredPlan,
  getToolRequiredPlan,
  type PlanType,
  type Service,
} from "@/lib/data"
import { Lock, Settings, Users, ArrowRight } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"

export default function ToolsPage() {
  const { isAdmin } = useAuth()
  const [selectedService, setSelectedService] = useState<Service | null>(null)
  const [upgradeDialog, setUpgradeDialog] = useState<PlanType | null>(null)

  const currentPlan = organizationPlan.currentPlan

  // サービスをプラン別にグループ化
  const freeServices = services.filter((s) => getServiceRequiredPlan(s.id) === "free")
  const proServices = services.filter((s) => getServiceRequiredPlan(s.id) === "pro")
  const maxServices = services.filter((s) => getServiceRequiredPlan(s.id) === "max")

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

  const handleServiceClick = (service: Service) => {
    const requiredPlan = getServiceRequiredPlan(service.id)
    if (!isPlanSufficient(currentPlan, requiredPlan)) {
      setUpgradeDialog(requiredPlan)
    } else {
      setSelectedService(service)
    }
  }

  const handleUpgrade = () => {
    setUpgradeDialog(null)
    // Redirect to billing page
    window.location.href = "/billing"
  }

  const renderServiceCard = (service: Service, locked: boolean, requiredPlan: PlanType) => {
    const module = moduleDetails[service.id]
    const availableToolsCount =
      module?.tools.filter((t) => isPlanSufficient(currentPlan, getToolRequiredPlan(service.id, t.id))).length || 0
    const totalToolsCount = module?.tools.length || 0

    return (
      <Card
        key={service.id}
        className={cn("cursor-pointer transition-all hover:border-primary/50", locked && "opacity-60")}
        onClick={() => handleServiceClick(service)}
      >
        <CardContent className="p-4">
          <div className="flex items-start gap-4">
            <div
              className={cn(
                "w-12 h-12 rounded-lg flex items-center justify-center shrink-0",
                locked ? "bg-muted" : "bg-secondary",
              )}
            >
              {locked ? (
                <Lock className="h-5 w-5 text-muted-foreground" />
              ) : (
                <ServiceIcon icon={service.icon} className="h-6 w-6 text-foreground" />
              )}
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <h3 className="font-medium text-foreground truncate">{service.name}</h3>
                {locked && <Badge className={getPlanBadgeStyle(requiredPlan)}>{requiredPlan.toUpperCase()}</Badge>}
              </div>
              <p className="text-sm text-muted-foreground line-clamp-2">{service.description}</p>
              {!locked && (
                <p className="text-xs text-muted-foreground mt-2">
                  {availableToolsCount}/{totalToolsCount} 機能利用可能
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  const renderSection = (title: string, serviceList: Service[], requiredPlan: PlanType, showUpgradeButton: boolean) => {
    const isLocked = !isPlanSufficient(currentPlan, requiredPlan)

    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">{title}</h2>
            {requiredPlan !== "free" && (
              <Badge className={getPlanBadgeStyle(requiredPlan)}>{requiredPlan.toUpperCase()}</Badge>
            )}
          </div>
          {showUpgradeButton && isLocked && (
            <Link href="/billing">
              <Button variant="outline" size="sm">
                {requiredPlan.toUpperCase()}にアップグレード
                <ArrowRight className="h-4 w-4 ml-1" />
              </Button>
            </Link>
          )}
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {serviceList.map((service) => renderServiceCard(service, isLocked, requiredPlan))}
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Tools</h1>
        <p className="text-muted-foreground mt-1">組織で利用可能なサービスを管理</p>
      </div>

      {/* Current Plan Info */}
      <Card className="bg-secondary/30">
        <CardContent className="p-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">現在のプラン</p>
                <p className="text-lg font-semibold">{organizationPlan.planName}</p>
              </div>
              <div className="h-8 w-px bg-border" />
              <div className="flex items-center gap-2">
                <Users className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">
                  {organizationPlan.userCount}/{organizationPlan.userLimit}名
                </span>
              </div>
            </div>
            <Link href="/billing">
              <Button variant="outline" size="sm">
                プランを変更
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>

      {/* Free Services */}
      {freeServices.length > 0 && renderSection("利用可能なサービス", freeServices, "free", false)}

      {/* Pro Services */}
      {proServices.length > 0 && renderSection("Proプランで利用可能", proServices, "pro", true)}

      {/* Max Services */}
      {maxServices.length > 0 && renderSection("Maxプランで利用可能", maxServices, "max", true)}

      {/* Service Detail Sheet */}
      <Sheet open={!!selectedService} onOpenChange={(open) => !open && setSelectedService(null)}>
        <SheetContent className="w-full sm:max-w-lg overflow-y-auto">
          {selectedService && (
            <>
              <SheetHeader>
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                    <ServiceIcon icon={selectedService.icon} className="h-5 w-5 text-foreground" />
                  </div>
                  <div>
                    <SheetTitle>{selectedService.name}</SheetTitle>
                    <SheetDescription>{selectedService.description}</SheetDescription>
                  </div>
                </div>
              </SheetHeader>

              <div className="mt-6 space-y-4">
                <h3 className="font-medium">利用可能な機能</h3>
                {moduleDetails[selectedService.id]?.tools.map((tool) => {
                  const toolPlan = getToolRequiredPlan(selectedService.id, tool.id)
                  const isToolLocked = !isPlanSufficient(currentPlan, toolPlan)

                  return (
                    <div
                      key={tool.id}
                      className={cn("p-4 rounded-lg border", isToolLocked && "opacity-60 bg-muted/50")}
                    >
                      <div className="flex items-start gap-3">
                        <Checkbox disabled={isToolLocked} defaultChecked={!isToolLocked} />
                        <div className="flex-1">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{tool.name}</span>
                            {isToolLocked && (
                              <Badge className={getPlanBadgeStyle(toolPlan)}>
                                <Lock className="h-3 w-3 mr-1" />
                                {toolPlan.toUpperCase()}
                              </Badge>
                            )}
                          </div>
                          <p className="text-sm text-muted-foreground mt-1">{tool.description}</p>
                        </div>
                      </div>
                    </div>
                  )
                })}

                <Button
                  className="w-full"
                  onClick={() => {
                    setSelectedService(null)
                    toast.success("設定を保存しました")
                  }}
                >
                  <Settings className="h-4 w-4 mr-2" />
                  設定を保存
                </Button>
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>

      {/* Upgrade Dialog */}
      <AlertDialog open={!!upgradeDialog} onOpenChange={(open) => !open && setUpgradeDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>プランのアップグレードが必要です</AlertDialogTitle>
            <AlertDialogDescription>
              このサービスは{upgradeDialog?.toUpperCase()}プラン以上で利用可能です。 アップグレードしますか？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={handleUpgrade}>アップグレードする</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
