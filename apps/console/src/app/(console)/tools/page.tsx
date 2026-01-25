"use client"

import { useState, useEffect, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { ServiceIcon } from "@/components/service-icon"
import { useAuth } from "@/lib/auth-context"
import { modules, getServiceIcon, type ModuleDef } from "@/lib/module-data"
import { Info, Check, Link2, Loader2, AlertTriangle } from "lucide-react"
import Link from "next/link"
import { getServiceConnections, type ServiceConnection } from "@/lib/credits"

// User tool preferences type
interface UserToolPreference {
  userId: string
  enabledTools: Record<string, string[]> // moduleId -> toolIds
}

export const dynamic = "force-dynamic"

export default function ToolsPage() {
  const { user } = useAuth()
  const [preferences, setPreferences] = useState<UserToolPreference[]>([])
  const [savedModules, setSavedModules] = useState<Record<string, boolean>>({})
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

  // デフォルトで有効なツールを初期設定
  useEffect(() => {
    if (user && preferences.length === 0) {
      const defaultPrefs: Record<string, string[]> = {}
      modules.forEach((mod) => {
        defaultPrefs[mod.id] = mod.tools.filter((t) => t.defaultEnabled).map((t) => t.id)
      })
      setPreferences([{ userId: user.id, enabledTools: defaultPrefs }])
    }
  }, [user, preferences.length])

  const myPreference = preferences.find((p) => p.userId === user?.id) || {
    userId: user?.id || "",
    enabledTools: {},
  }

  const handleToggleTool = (moduleId: string, toolId: string) => {
    setPreferences((prev) => {
      const existing = prev.find((p) => p.userId === user?.id)
      if (existing) {
        const currentTools = existing.enabledTools[moduleId] || []
        const newTools = currentTools.includes(toolId)
          ? currentTools.filter((id) => id !== toolId)
          : [...currentTools, toolId]
        return prev.map((p) =>
          p.userId === user?.id ? { ...p, enabledTools: { ...p.enabledTools, [moduleId]: newTools } } : p,
        )
      } else {
        return [...prev, { userId: user?.id || "", enabledTools: { [moduleId]: [toolId] } }]
      }
    })
  }

  const handleSelectAll = (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    setPreferences((prev) => {
      const existing = prev.find((p) => p.userId === user?.id)
      const newTools = mod.tools.map((t) => t.id)
      if (existing) {
        return prev.map((p) =>
          p.userId === user?.id ? { ...p, enabledTools: { ...p.enabledTools, [moduleId]: newTools } } : p,
        )
      } else {
        return [...prev, { userId: user?.id || "", enabledTools: { [moduleId]: newTools } }]
      }
    })
  }

  const handleSelectDefault = (moduleId: string) => {
    const mod = modules.find((m) => m.id === moduleId)
    if (!mod) return

    setPreferences((prev) => {
      const existing = prev.find((p) => p.userId === user?.id)
      const defaultTools = mod.tools.filter((t) => t.defaultEnabled).map((t) => t.id)
      if (existing) {
        return prev.map((p) =>
          p.userId === user?.id ? { ...p, enabledTools: { ...p.enabledTools, [moduleId]: defaultTools } } : p,
        )
      } else {
        return [...prev, { userId: user?.id || "", enabledTools: { [moduleId]: defaultTools } }]
      }
    })
  }

  const isToolEnabled = (moduleId: string, toolId: string) => {
    return myPreference.enabledTools[moduleId]?.includes(toolId) || false
  }

  const handleSave = (moduleId: string) => {
    // TODO: Save to database via RPC
    setSavedModules((prev) => ({ ...prev, [moduleId]: true }))
    setTimeout(() => {
      setSavedModules((prev) => ({ ...prev, [moduleId]: false }))
    }, 2000)
  }

  // 接続済みサービスのIDセット
  const connectedServiceIds = new Set(connections.map((c) => c.service))

  // 接続済みモジュールのみ表示
  const availableModules = modules.filter((m) => connectedServiceIds.has(m.id))

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

  if (availableModules.length === 0) {
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
            <Link href="/connections">
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
              <span className="text-yellow-500 ml-1">危険なツール</span>
              はデフォルトで無効化されています。
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="space-y-6">
        {availableModules.map((mod) => {
          const enabledCount = myPreference.enabledTools[mod.id]?.length || 0
          const totalCount = mod.tools.length

          return (
            <Card key={mod.id}>
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                      <ServiceIcon icon={getServiceIcon(mod.id)} className="h-5 w-5 text-foreground" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-base">{mod.name}</CardTitle>
                        <Badge variant="secondary" className="text-xs">
                          {enabledCount}/{totalCount}
                        </Badge>
                      </div>
                      <CardDescription>{mod.description}</CardDescription>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={() => handleSelectDefault(mod.id)}>
                      デフォルト
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => handleSelectAll(mod.id)}>
                      全選択
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                {mod.tools.map((tool) => (
                  <div
                    key={tool.id}
                    className={`flex items-start gap-3 p-3 rounded-lg border ${
                      tool.dangerous ? "border-yellow-500/30 bg-yellow-500/5" : ""
                    }`}
                  >
                    <Checkbox
                      checked={isToolEnabled(mod.id, tool.id)}
                      onCheckedChange={() => handleToggleTool(mod.id, tool.id)}
                      className="mt-1"
                    />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm font-mono">{tool.name}</span>
                        {tool.dangerous && (
                          <Badge variant="outline" className="text-yellow-500 border-yellow-500/50 text-xs">
                            <AlertTriangle className="h-3 w-3 mr-1" />
                            危険
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground">{tool.description}</p>
                    </div>
                  </div>
                ))}
                <div className="flex justify-end items-center gap-2 pt-2">
                  {savedModules[mod.id] && (
                    <span className="text-sm text-green-500 flex items-center gap-1">
                      <Check className="h-4 w-4" />
                      保存しました
                    </span>
                  )}
                  <Button size="sm" onClick={() => handleSave(mod.id)}>
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
