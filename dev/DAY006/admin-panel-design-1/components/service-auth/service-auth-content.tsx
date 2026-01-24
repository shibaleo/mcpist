"use client"

import { useState } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { ServiceIcon } from "@/components/service-icon"
import { services, serviceAuthConfigs, type ServiceAuthConfig, type AuthMethodConfig } from "@/lib/data"
import { Search, Eye, EyeOff, Save, X } from "lucide-react"
import { toast } from "sonner"

export function ServiceAuthContent() {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedService, setSelectedService] = useState<string | null>(null)
  const [authConfigs, setAuthConfigs] = useState<ServiceAuthConfig[]>(serviceAuthConfigs)
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({})
  const [scopeInputs, setScopeInputs] = useState<Record<string, string>>({})

  const filteredServices = services.filter(
    (service) =>
      service.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      service.description.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  const selectedServiceData = services.find((s) => s.id === selectedService)
  const selectedAuthConfig = authConfigs.find((c) => c.serviceId === selectedService)

  const getEnabledMethodsCount = (serviceId: string) => {
    const config = authConfigs.find((c) => c.serviceId === serviceId)
    if (!config) return 0
    return config.availableMethods.filter((m) => m.enabled).length
  }

  const handleMethodToggle = (methodType: AuthMethodConfig["type"], enabled: boolean) => {
    setAuthConfigs((prev) =>
      prev.map((config) => {
        if (config.serviceId !== selectedService) return config
        return {
          ...config,
          availableMethods: config.availableMethods.map((method) => {
            if (method.type !== methodType) return method
            return { ...method, enabled }
          }),
        }
      }),
    )
  }

  const handleOAuthUpdate = (
    methodType: AuthMethodConfig["type"],
    field: "clientId" | "clientSecret",
    value: string,
  ) => {
    setAuthConfigs((prev) =>
      prev.map((config) => {
        if (config.serviceId !== selectedService) return config
        return {
          ...config,
          availableMethods: config.availableMethods.map((method) => {
            if (method.type !== methodType || !method.oauth) return method
            return {
              ...method,
              oauth: { ...method.oauth, [field]: value },
            }
          }),
        }
      }),
    )
  }

  const handleScopesUpdate = (methodType: AuthMethodConfig["type"], scopes: string[]) => {
    setAuthConfigs((prev) =>
      prev.map((config) => {
        if (config.serviceId !== selectedService) return config
        return {
          ...config,
          availableMethods: config.availableMethods.map((method) => {
            if (method.type !== methodType || !method.oauth) return method
            return {
              ...method,
              oauth: { ...method.oauth, scopes },
            }
          }),
        }
      }),
    )
  }

  const handleAddScope = (methodType: AuthMethodConfig["type"]) => {
    const key = `${selectedService}-${methodType}`
    const newScope = scopeInputs[key]?.trim()
    if (!newScope) return

    const currentScopes = selectedAuthConfig?.availableMethods.find((m) => m.type === methodType)?.oauth?.scopes || []
    if (!currentScopes.includes(newScope)) {
      handleScopesUpdate(methodType, [...currentScopes, newScope])
    }
    setScopeInputs((prev) => ({ ...prev, [key]: "" }))
  }

  const handleRemoveScope = (methodType: AuthMethodConfig["type"], scopeToRemove: string) => {
    const currentScopes = selectedAuthConfig?.availableMethods.find((m) => m.type === methodType)?.oauth?.scopes || []
    handleScopesUpdate(
      methodType,
      currentScopes.filter((s) => s !== scopeToRemove),
    )
  }

  const handleHelpTextUpdate = (methodType: AuthMethodConfig["type"], value: string) => {
    setAuthConfigs((prev) =>
      prev.map((config) => {
        if (config.serviceId !== selectedService) return config
        return {
          ...config,
          availableMethods: config.availableMethods.map((method) => {
            if (method.type !== methodType) return method
            return { ...method, helpText: value }
          }),
        }
      }),
    )
  }

  const toggleSecretVisibility = (key: string) => {
    setShowSecrets((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const handleSave = () => {
    setSelectedService(null)
    toast.success("設定を保存しました")
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">サービス認証設定</h1>
        <p className="text-muted-foreground mt-1">サービスごとに利用可能な認証方法を設定します</p>
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
          const enabledCount = getEnabledMethodsCount(service.id)
          return (
            <Card
              key={service.id}
              className="cursor-pointer transition-all hover:border-primary/50"
              onClick={() => setSelectedService(service.id)}
            >
              <CardContent className="p-4">
                <div className="flex items-start gap-4">
                  <div className="w-12 h-12 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                    <ServiceIcon icon={service.icon} className="h-6 w-6 text-foreground" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-medium text-foreground truncate">{service.name}</h3>
                    </div>
                    <p className="text-sm text-muted-foreground line-clamp-2">{service.description}</p>
                    <div className="mt-2">
                      <Badge variant="outline" className="text-xs">
                        {enabledCount > 0 ? `${enabledCount}種類の認証方法` : "未設定"}
                      </Badge>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      <Sheet open={!!selectedService} onOpenChange={(open) => !open && setSelectedService(null)}>
        <SheetContent className="w-full sm:max-w-lg overflow-y-auto">
          <SheetHeader>
            <div className="flex items-center gap-3">
              {selectedServiceData && (
                <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                  <ServiceIcon icon={selectedServiceData.icon} className="h-5 w-5 text-foreground" />
                </div>
              )}
              <div>
                <SheetTitle>{selectedServiceData?.name}</SheetTitle>
                <SheetDescription>{selectedServiceData?.description}</SheetDescription>
              </div>
            </div>
          </SheetHeader>

          <div className="mt-6 space-y-6">
            {selectedAuthConfig?.availableMethods.map((method) => (
              <div key={method.type} className="space-y-4 p-4 border rounded-lg">
                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-base font-medium">{method.label}</Label>
                    <p className="text-sm text-muted-foreground">
                      {method.type === "oauth2" && "OAuth 2.0による認証"}
                      {method.type === "apikey" && "APIキーによる認証"}
                      {method.type === "personal_token" && "個人アクセストークンによる認証"}
                      {method.type === "integration_token" && "インテグレーショントークンによる認証"}
                    </p>
                  </div>
                  <Switch
                    checked={method.enabled}
                    onCheckedChange={(checked) => handleMethodToggle(method.type, checked)}
                  />
                </div>

                {method.enabled && (
                  <div className="space-y-4 pt-2">
                    {method.type === "oauth2" && method.oauth && (
                      <>
                        <div className="space-y-2">
                          <Label htmlFor={`${method.type}-client-id`}>Client ID</Label>
                          <Input
                            id={`${method.type}-client-id`}
                            value={method.oauth.clientId}
                            onChange={(e) => handleOAuthUpdate(method.type, "clientId", e.target.value)}
                            placeholder="クライアントIDを入力"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor={`${method.type}-client-secret`}>Client Secret</Label>
                          <div className="relative">
                            <Input
                              id={`${method.type}-client-secret`}
                              type={showSecrets[`${selectedService}-${method.type}`] ? "text" : "password"}
                              value={method.oauth.clientSecret}
                              onChange={(e) => handleOAuthUpdate(method.type, "clientSecret", e.target.value)}
                              placeholder="クライアントシークレットを入力"
                              className="pr-10"
                            />
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              className="absolute right-0 top-0 h-full px-3"
                              onClick={() => toggleSecretVisibility(`${selectedService}-${method.type}`)}
                            >
                              {showSecrets[`${selectedService}-${method.type}`] ? (
                                <EyeOff className="h-4 w-4" />
                              ) : (
                                <Eye className="h-4 w-4" />
                              )}
                            </Button>
                          </div>
                        </div>
                        <div className="space-y-2">
                          <Label>Scopes</Label>
                          <div className="flex flex-wrap gap-2 mb-2">
                            {method.oauth.scopes.map((scope) => (
                              <Badge key={scope} variant="secondary" className="flex items-center gap-1">
                                {scope}
                                <button
                                  type="button"
                                  onClick={() => handleRemoveScope(method.type, scope)}
                                  className="hover:text-destructive"
                                >
                                  <X className="h-3 w-3" />
                                </button>
                              </Badge>
                            ))}
                          </div>
                          <div className="flex gap-2">
                            <Input
                              placeholder="新しいscopeを追加..."
                              value={scopeInputs[`${selectedService}-${method.type}`] || ""}
                              onChange={(e) =>
                                setScopeInputs((prev) => ({
                                  ...prev,
                                  [`${selectedService}-${method.type}`]: e.target.value,
                                }))
                              }
                              onKeyDown={(e) => {
                                if (e.key === "Enter") {
                                  e.preventDefault()
                                  handleAddScope(method.type)
                                }
                              }}
                            />
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() => handleAddScope(method.type)}
                            >
                              追加
                            </Button>
                          </div>
                        </div>
                      </>
                    )}

                    {(method.type === "apikey" ||
                      method.type === "personal_token" ||
                      method.type === "integration_token") && (
                      <div className="space-y-2">
                        <Label htmlFor={`${method.type}-help`}>ユーザー向けヘルプテキスト</Label>
                        <Textarea
                          id={`${method.type}-help`}
                          value={method.helpText || ""}
                          onChange={(e) => handleHelpTextUpdate(method.type, e.target.value)}
                          placeholder="トークンの取得方法をユーザーに説明するテキストを入力..."
                          rows={3}
                        />
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}

            <Button className="w-full" onClick={handleSave}>
              <Save className="h-4 w-4 mr-2" />
              設定を保存
            </Button>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
