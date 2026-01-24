"use client"

import { useState } from "react"
import { services, type Service } from "@/lib/data"
import { ServiceCard } from "@/components/tools/service-card"
import { ConnectDialog } from "@/components/tools/connect-dialog"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Button } from "@/components/ui/button"
import { ChevronDown } from "lucide-react"

export default function ToolsPage() {
  const [serviceList, setServiceList] = useState(services)
  const [selectedService, setSelectedService] = useState<Service | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [unavailableOpen, setUnavailableOpen] = useState(false)

  const availableServices = serviceList.filter((s) => s.status !== "no-permission")
  const unavailableServices = serviceList.filter((s) => s.status === "no-permission")

  const handleConnect = (service: Service) => {
    setSelectedService(service)
    setDialogOpen(true)
  }

  const handleDisconnect = (service: Service) => {
    setServiceList((prev) => prev.map((s) => (s.id === service.id ? { ...s, status: "disconnected" as const } : s)))
  }

  const handleConnectSuccess = () => {
    if (selectedService) {
      setServiceList((prev) =>
        prev.map((s) => (s.id === selectedService.id ? { ...s, status: "connected" as const } : s)),
      )
    }
  }

  const handleRequest = (service: Service) => {
    // In a real app, this would send a request to the admin
    alert(`「${service.name}」の利用申請を送信しました`)
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Tools</h1>
        <p className="text-muted-foreground mt-1">外部サービスの連携状況を管理</p>
      </div>

      {/* Available Services */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {availableServices.map((service) => (
          <ServiceCard
            key={service.id}
            service={service}
            onConnect={() => handleConnect(service)}
            onDisconnect={() => handleDisconnect(service)}
          />
        ))}
      </div>

      {/* Unavailable Services */}
      {unavailableServices.length > 0 && (
        <Collapsible open={unavailableOpen} onOpenChange={setUnavailableOpen}>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" className="flex items-center gap-2 text-muted-foreground hover:text-foreground">
              <ChevronDown
                className={`h-4 w-4 transition-transform ${unavailableOpen ? "transform rotate-180" : ""}`}
              />
              利用できないツール（{unavailableServices.length}件）
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="mt-4">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {unavailableServices.map((service) => (
                <ServiceCard key={service.id} service={service} onRequest={() => handleRequest(service)} disabled />
              ))}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}

      {/* Connect Dialog */}
      <ConnectDialog
        service={selectedService}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSuccess={handleConnectSuccess}
      />
    </div>
  )
}
