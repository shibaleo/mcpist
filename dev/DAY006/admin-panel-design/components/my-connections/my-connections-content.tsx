"use client"

import { useState } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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
import { ServiceIcon } from "@/components/service-icon"
import { useAuth } from "@/lib/auth-context"
import {
  services,
  serviceAuthConfigs,
  userCredentials as initialCredentials,
  type UserServiceCredential,
} from "@/lib/data"
import { Search, Link2, Unlink, ExternalLink, Info } from "lucide-react"
import { cn } from "@/lib/utils"

export function MyConnectionsContent() {
  const { user } = useAuth()
  const [searchQuery, setSearchQuery] = useState("")
  const [credentials, setCredentials] = useState<UserServiceCredential[]>(initialCredentials)
  const [connectDialog, setConnectDialog] = useState<string | null>(null)
  const [disconnectDialog, setDisconnectDialog] = useState<string | null>(null)
  const [selectedAuthMethod, setSelectedAuthMethod] = useState<string>("")
  const [tokenInput, setTokenInput] = useState("")

  const myCredentials = credentials.filter((c) => c.userId === user?.id)

  const availableServices = services.filter((service) => {
    const config = serviceAuthConfigs.find((c) => c.serviceId === service.id)
    return config && config.availableMethods.some((m) => m.enabled)
  })

  const filteredServices = availableServices.filter(
    (service) =>
      service.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      service.description.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  const getCredentialForService = (serviceId: string) => {
    return myCredentials.find((c) => c.serviceId === serviceId)
  }

  const getAuthConfig = (serviceId: string) => {
    return serviceAuthConfigs.find((c) => c.serviceId === serviceId)
  }

  const handleConnect = (serviceId: string) => {
    const config = getAuthConfig(serviceId)
    const enabledMethods = config?.availableMethods.filter((m) => m.enabled) || []
    if (enabledMethods.length > 0) {
      setSelectedAuthMethod(enabledMethods[0].type)
    }
    setConnectDialog(serviceId)
    setTokenInput("")
  }

  const handleConnectSubmit = () => {
    if (!connectDialog || !selectedAuthMethod || !user) return

    const newCredential: UserServiceCredential = {
      id: `new-${Date.now()}`,
      userId: user.id,
      serviceId: connectDialog,
      authMethod: selectedAuthMethod as UserServiceCredential["authMethod"],
      status: "active",
      connectedAt: new Date().toISOString().split("T")[0],
    }
    setCredentials((prev) => [...prev, newCredential])
    setConnectDialog(null)
    setSelectedAuthMethod("")
    setTokenInput("")
  }

  const handleDisconnect = () => {
    if (!disconnectDialog || !user) return
    setCredentials((prev) => prev.filter((c) => !(c.serviceId === disconnectDialog && c.userId === user.id)))
    setDisconnectDialog(null)
  }

  const selectedService = connectDialog ? services.find((s) => s.id === connectDialog) : null
  const selectedConfig = connectDialog ? getAuthConfig(connectDialog) : null

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">マイ接続</h1>
        <p className="text-muted-foreground mt-1">サービスとの接続を管理します</p>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="サービスを検索..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10"
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filteredServices.map((service) => {
          const credential = getCredentialForService(service.id)
          const isConnected = credential?.status === "active"
          const isExpired = credential?.status === "expired"

          return (
            <Card
              key={service.id}
              className={cn("transition-all", isConnected && "border-success/50", isExpired && "border-warning/50")}
            >
              <CardContent className="p-4">
                <div className="flex items-start gap-4">
                  <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                    <ServiceIcon icon={service.icon} className="h-6 w-6 text-foreground" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-medium text-foreground truncate">{service.name}</h3>
                      {isConnected && <Badge className="bg-success/20 text-success border-success/30">接続済</Badge>}
                      {isExpired && <Badge className="bg-warning/20 text-warning border-warning/30">期限切れ</Badge>}
                    </div>
                    <p className="text-sm text-muted-foreground line-clamp-2">{service.description}</p>
                    {credential && (
                      <p className="text-xs text-muted-foreground mt-2">接続日: {credential.connectedAt}</p>
                    )}
                  </div>
                </div>
                <div className="mt-4 flex justify-end gap-2">
                  {isConnected || isExpired ? (
                    <>
                      {isExpired && (
                        <Button variant="default" size="sm" onClick={() => handleConnect(service.id)}>
                          <Link2 className="h-4 w-4 mr-1" />
                          再接続
                        </Button>
                      )}
                      <Button variant="outline" size="sm" onClick={() => setDisconnectDialog(service.id)}>
                        <Unlink className="h-4 w-4 mr-1" />
                        切断
                      </Button>
                    </>
                  ) : (
                    <Button variant="default" size="sm" onClick={() => handleConnect(service.id)}>
                      <Link2 className="h-4 w-4 mr-1" />
                      接続
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      <Dialog open={!!connectDialog} onOpenChange={(open) => !open && setConnectDialog(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {selectedService && (
                <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                  <ServiceIcon icon={selectedService.icon} className="h-5 w-5 text-foreground" />
                </div>
              )}
              <div>
                <DialogTitle>{selectedService?.name}に接続</DialogTitle>
                <DialogDescription>認証方法を選択してください</DialogDescription>
              </div>
            </div>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <RadioGroup value={selectedAuthMethod} onValueChange={setSelectedAuthMethod}>
              {selectedConfig?.availableMethods
                .filter((m) => m.enabled)
                .map((method) => (
                  <div
                    key={method.type}
                    className={cn(
                      "flex items-start space-x-3 p-4 border rounded-lg cursor-pointer transition-colors",
                      selectedAuthMethod === method.type && "border-primary bg-primary/5",
                    )}
                    onClick={() => setSelectedAuthMethod(method.type)}
                  >
                    <RadioGroupItem value={method.type} id={method.type} className="mt-1" />
                    <div className="flex-1">
                      <Label htmlFor={method.type} className="cursor-pointer font-medium">
                        {method.label}
                      </Label>
                      {method.helpText && <p className="text-sm text-muted-foreground mt-1">{method.helpText}</p>}
                    </div>
                  </div>
                ))}
            </RadioGroup>

            {selectedAuthMethod === "oauth2" && (
              <div className="p-4 bg-secondary/50 rounded-lg">
                <p className="text-sm text-muted-foreground">
                  「接続」をクリックすると、{selectedService?.name}
                  の認証画面にリダイレクトされます。
                </p>
              </div>
            )}

            {(selectedAuthMethod === "apikey" ||
              selectedAuthMethod === "personal_token" ||
              selectedAuthMethod === "integration_token") && (
              <div className="space-y-2">
                <Label htmlFor="token-input">
                  {selectedAuthMethod === "apikey" && "APIキー"}
                  {selectedAuthMethod === "personal_token" && "Personal Access Token"}
                  {selectedAuthMethod === "integration_token" && "インテグレーショントークン"}
                </Label>
                <Input
                  id="token-input"
                  type="password"
                  value={tokenInput}
                  onChange={(e) => setTokenInput(e.target.value)}
                  placeholder="トークンを入力..."
                />
                {selectedConfig?.availableMethods.find((m) => m.type === selectedAuthMethod)?.helpText && (
                  <div className="flex items-start gap-2 p-3 bg-secondary/50 rounded-lg">
                    <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                    <p className="text-sm text-muted-foreground">
                      {selectedConfig?.availableMethods.find((m) => m.type === selectedAuthMethod)?.helpText}
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setConnectDialog(null)}>
              キャンセル
            </Button>
            {selectedAuthMethod === "oauth2" ? (
              <Button onClick={handleConnectSubmit}>
                <ExternalLink className="h-4 w-4 mr-2" />
                {selectedService?.name}でログイン
              </Button>
            ) : (
              <Button
                onClick={handleConnectSubmit}
                disabled={
                  !tokenInput &&
                  (selectedAuthMethod === "apikey" ||
                    selectedAuthMethod === "personal_token" ||
                    selectedAuthMethod === "integration_token")
                }
              >
                <Link2 className="h-4 w-4 mr-2" />
                接続
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={!!disconnectDialog} onOpenChange={(open) => !open && setDisconnectDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>接続を解除しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              {disconnectDialog && services.find((s) => s.id === disconnectDialog)?.name}
              との接続を解除します。この操作は取り消せません。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={handleDisconnect} className="bg-destructive text-destructive-foreground">
              切断する
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
