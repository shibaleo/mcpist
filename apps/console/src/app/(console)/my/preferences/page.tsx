"use client"

import { useState, useEffect, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { ServiceIcon } from "@/components/service-icon"
import { useAuth } from "@/lib/auth-context"
import { services, moduleDetails, type UserToolPreference } from "@/lib/data"
import { Info, Check, Link2, Loader2 } from "lucide-react"
import Link from "next/link"
import { getServiceConnections, type ServiceConnection } from "@/lib/credits"

export const dynamic = "force-dynamic"

export default function MyPreferencesPage() {
  const { user } = useAuth()
  const [preferences, setPreferences] = useState<UserToolPreference[]>([])
  const [savedServices, setSavedServices] = useState<Record<string, boolean>>({})
  const [connections, setConnections] = useState<ServiceConnection[]>([])
  const [loading, setLoading] = useState(true)

  // 接続済みサービスを取得
  const loadConnections = useCallback(async () => {
    try {
      const data = await getServiceConnections()
      setConnections(data)
    } catch (error) {
      console.error("Failed to load connections:", error)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) {
      loadConnections()
    } else {
      setLoading(false)
    }
  }, [user, loadConnections])

  const myPreference = preferences.find((p) => p.userId === user?.id) || {
    userId: user?.id || "",
    enabledTools: {},
  }

  const handleToggleTool = (serviceId: string, toolId: string) => {
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

    setPreferences((prev) => {
      const existing = prev.find((p) => p.userId === user?.id)
      const newTools = module.tools.map((t) => t.id)
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

  const handleSave = (serviceId: string) => {
    setSavedServices((prev) => ({ ...prev, [serviceId]: true }))
    setTimeout(() => {
      setSavedServices((prev) => ({ ...prev, [serviceId]: false }))
    }, 2000)
  }

  // 接続済みサービスのIDセット
  const connectedServiceIds = new Set(connections.map((c) => c.service))

  // 接続済みサービスのみ表示
  const availableServices = services.filter((s) => connectedServiceIds.has(s.id))

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">ツール設定</h1>
          <p className="text-muted-foreground mt-1">利用するツールを選択してください</p>
        </div>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  if (availableServices.length === 0) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">ツール設定</h1>
          <p className="text-muted-foreground mt-1">
            サービスごとに呼び出せるツールをカスタマイズできます。呼び出すツールが少ないほど、LLMのコンテキストを節約できます。
          </p>
        </div>

        <Card>
          <CardContent className="p-8 text-center">
            <Link2 className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="font-medium text-foreground mb-2">サービスを接続してください</h3>
            <p className="text-sm text-muted-foreground mb-4">
              サービス連携からサービスを接続すると、ツール設定が可能になります
            </p>
            <Link href="/my/connections">
              <Button>
                <Link2 className="h-4 w-4 mr-2" />
                サービス連携へ
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">ツール設定</h1>
        <p className="text-muted-foreground mt-1">利用するツールを選択してください</p>
      </div>

      <Card className="bg-secondary/30">
        <CardContent className="p-4">
          <div className="flex items-start gap-3">
            <Info className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
            <p className="text-sm text-muted-foreground">
              サービスごとに呼び出せるツールをカスタマイズできます。呼び出すツールが少ないほど、LLMのコンテキストを節約できます。
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
                {module.tools.map((tool) => (
                  <div key={tool.id} className="flex items-start gap-3 p-3 rounded-lg border">
                    <Checkbox
                      checked={isToolEnabled(service.id, tool.id)}
                      onCheckedChange={() => handleToggleTool(service.id, tool.id)}
                      className="mt-1"
                    />
                    <div className="flex-1">
                      <span className="font-medium text-sm">{tool.name}</span>
                      <p className="text-sm text-muted-foreground">{tool.description}</p>
                    </div>
                  </div>
                ))}
                <div className="flex justify-end items-center gap-2 pt-2">
                  {savedServices[service.id] && (
                    <span className="text-sm text-green-500 flex items-center gap-1">
                      <Check className="h-4 w-4" />
                      保存しました
                    </span>
                  )}
                  <Button size="sm" onClick={() => handleSave(service.id)}>
                    設定を保存
                  </Button>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
