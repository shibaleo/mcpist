"use client"

import { useEffect, useState, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth/auth-context"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import {
  Loader2,
  Settings,
  ExternalLink,
  Eye,
  EyeOff,
  Save,
  Trash2,
  AlertCircle,
  CheckCircle2,
} from "lucide-react"
import {
  listOAuthApps,
  upsertOAuthApp,
  deleteOAuthApp,
  OAUTH_PROVIDERS,
  getDefaultRedirectUri,
  type OAuthApp,
  OAuthAppError,
} from "@/lib/oauth/apps"

type FormState = {
  clientId: string
  clientSecret: string
  redirectUri: string
  enabled: boolean
}

type ProviderFormStates = {
  [provider: string]: FormState
}

export default function OAuthAppsPage() {
  const { isAdmin, isLoading: authLoading } = useAuth()
  const router = useRouter()

  const [apps, setApps] = useState<OAuthApp[]>([])
  const [loading, setLoading] = useState(true)
  const [formStates, setFormStates] = useState<ProviderFormStates>({})
  const [showSecrets, setShowSecrets] = useState<{ [key: string]: boolean }>({})
  const [saving, setSaving] = useState<{ [key: string]: boolean }>({})
  const [deleting, setDeleting] = useState<{ [key: string]: boolean }>({})
  const [messages, setMessages] = useState<{ [key: string]: { type: "success" | "error"; text: string } }>({})


  // Initialize form states for all providers
  const initFormStates = useCallback((existingApps: OAuthApp[]) => {
    const states: ProviderFormStates = {}
    for (const provider of OAUTH_PROVIDERS) {
      const existing = existingApps.find((app) => app.provider === provider.id)
      states[provider.id] = {
        clientId: existing?.client_id || "",
        clientSecret: "", // Never pre-fill secret
        redirectUri: existing?.redirect_uri || getDefaultRedirectUri(provider.id),
        enabled: existing?.enabled ?? true,
      }
    }
    setFormStates(states)
  }, [])

  // Load OAuth apps
  const loadApps = useCallback(async () => {
    try {
      setLoading(true)
      const data = await listOAuthApps()
      setApps(data)
      initFormStates(data)
    } catch (error) {
      console.error("Failed to load OAuth apps:", error)
    } finally {
      setLoading(false)
    }
  }, [initFormStates])

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/dashboard")
    }
  }, [authLoading, isAdmin, router])

  useEffect(() => {
    if (isAdmin) {
      loadApps()
    }
  }, [isAdmin, loadApps])

  // Handle form field changes
  const updateFormState = (provider: string, field: keyof FormState, value: string | boolean) => {
    setFormStates((prev) => ({
      ...prev,
      [provider]: {
        ...prev[provider],
        [field]: value,
      },
    }))
    // Clear message when user starts editing
    setMessages((prev) => {
      const newMessages = { ...prev }
      delete newMessages[provider]
      return newMessages
    })
  }

  // Save OAuth app
  const handleSave = async (provider: string) => {
    const form = formStates[provider]
    if (!form.clientId) {
      setMessages({ ...messages, [provider]: { type: "error", text: "Client ID is required" } })
      return
    }

    const existingApp = apps.find((app) => app.provider === provider)
    if (!existingApp?.has_credentials && !form.clientSecret) {
      setMessages({ ...messages, [provider]: { type: "error", text: "Client Secret is required for new apps" } })
      return
    }

    try {
      setSaving({ ...saving, [provider]: true })
      const result = await upsertOAuthApp(
        provider,
        form.clientId,
        form.clientSecret || "UNCHANGED", // Server should handle this
        form.redirectUri,
        form.enabled
      )
      if (result.success) {
        setMessages({
          ...messages,
          [provider]: {
            type: "success",
            text: result.action === "created" ? "Created successfully" : "Updated successfully",
          },
        })
        await loadApps()
      }
    } catch (error) {
      const message = error instanceof OAuthAppError ? error.message : "Failed to save"
      setMessages({ ...messages, [provider]: { type: "error", text: message } })
    } finally {
      setSaving({ ...saving, [provider]: false })
    }
  }

  // Delete OAuth app
  const handleDelete = async (provider: string) => {
    if (!confirm(`Are you sure you want to delete the ${provider} OAuth app configuration?`)) {
      return
    }

    try {
      setDeleting({ ...deleting, [provider]: true })
      const result = await deleteOAuthApp(provider)
      if (result.success) {
        setMessages({ ...messages, [provider]: { type: "success", text: "Deleted successfully" } })
        await loadApps()
      } else {
        setMessages({ ...messages, [provider]: { type: "error", text: result.message || "Failed to delete" } })
      }
    } catch (error) {
      const message = error instanceof OAuthAppError ? error.message : "Failed to delete"
      setMessages({ ...messages, [provider]: { type: "error", text: message } })
    } finally {
      setDeleting({ ...deleting, [provider]: false })
    }
  }

  if (authLoading || loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!isAdmin) {
    return null
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">OAuth App Settings</h1>
        <p className="text-muted-foreground mt-1">
          Configure OAuth client credentials for external service integrations
        </p>
      </div>

      <div className="grid gap-6">
        {OAUTH_PROVIDERS.map((provider) => {
          const existingApp = apps.find((app) => app.provider === provider.id)
          const form = formStates[provider.id] || {
            clientId: "",
            clientSecret: "",
            redirectUri: getDefaultRedirectUri(provider.id),
            enabled: true,
          }
          const isShowingSecret = showSecrets[provider.id] || false
          const isSaving = saving[provider.id] || false
          const isDeleting = deleting[provider.id] || false
          const message = messages[provider.id]

          return (
            <Card key={provider.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center">
                      <Settings className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        {provider.name}
                        {existingApp?.has_credentials ? (
                          <Badge variant="outline" className="bg-green-500/10 text-green-600 border-green-500/30">
                            Configured
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="bg-yellow-500/10 text-yellow-600 border-yellow-500/30">
                            Not Configured
                          </Badge>
                        )}
                      </CardTitle>
                      <CardDescription>{provider.description}</CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id={`${provider.id}-enabled`}
                      checked={form.enabled}
                      onCheckedChange={(checked) => updateFormState(provider.id, "enabled", checked === true)}
                    />
                    <Label htmlFor={`${provider.id}-enabled`} className="text-sm text-muted-foreground cursor-pointer">
                      Enabled
                    </Label>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Message */}
                {message && (
                  <div
                    className={`flex items-center gap-2 p-3 rounded-lg ${
                      message.type === "success"
                        ? "bg-green-500/10 text-green-600"
                        : "bg-destructive/10 text-destructive"
                    }`}
                  >
                    {message.type === "success" ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : (
                      <AlertCircle className="h-4 w-4" />
                    )}
                    <span className="text-sm">{message.text}</span>
                  </div>
                )}

                {/* Client ID */}
                <div className="space-y-2">
                  <Label htmlFor={`${provider.id}-client-id`}>Client ID</Label>
                  <Input
                    id={`${provider.id}-client-id`}
                    value={form.clientId}
                    onChange={(e) => updateFormState(provider.id, "clientId", e.target.value)}
                    placeholder="Enter client ID"
                    className="font-mono text-sm"
                  />
                </div>

                {/* Client Secret */}
                <div className="space-y-2">
                  <Label htmlFor={`${provider.id}-client-secret`}>
                    Client Secret
                    {existingApp?.has_credentials && (
                      <span className="text-xs text-muted-foreground ml-2">(leave empty to keep existing)</span>
                    )}
                  </Label>
                  <div className="relative">
                    <Input
                      id={`${provider.id}-client-secret`}
                      type={isShowingSecret ? "text" : "password"}
                      value={form.clientSecret}
                      onChange={(e) => updateFormState(provider.id, "clientSecret", e.target.value)}
                      placeholder={existingApp?.has_credentials ? "••••••••••••" : "Enter client secret"}
                      className="font-mono text-sm pr-10"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7 p-0"
                      onClick={() => setShowSecrets({ ...showSecrets, [provider.id]: !isShowingSecret })}
                    >
                      {isShowingSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                </div>

                {/* Redirect URI */}
                <div className="space-y-2">
                  <Label htmlFor={`${provider.id}-redirect-uri`}>Redirect URI</Label>
                  <Input
                    id={`${provider.id}-redirect-uri`}
                    value={form.redirectUri}
                    onChange={(e) => updateFormState(provider.id, "redirectUri", e.target.value)}
                    placeholder="OAuth callback URL"
                    className="font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Copy this URL to your OAuth app configuration
                  </p>
                </div>

                {/* Actions */}
                <div className="flex items-center justify-between pt-4 border-t">
                  <a
                    href={provider.docsUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-primary hover:underline flex items-center gap-1"
                  >
                    Open Developer Console
                    <ExternalLink className="h-3 w-3" />
                  </a>
                  <div className="flex items-center gap-2">
                    {existingApp?.has_credentials && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDelete(provider.id)}
                        disabled={isDeleting || isSaving}
                        className="text-destructive hover:text-destructive"
                      >
                        {isDeleting ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <>
                            <Trash2 className="h-4 w-4 mr-1" />
                            Delete
                          </>
                        )}
                      </Button>
                    )}
                    <Button size="sm" onClick={() => handleSave(provider.id)} disabled={isSaving || isDeleting}>
                      {isSaving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <>
                          <Save className="h-4 w-4 mr-1" />
                          Save
                        </>
                      )}
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
