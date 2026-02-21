"use client"

import { useEffect, useState, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth/auth-context"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ModuleIcon } from "@/components/module-icon"
import {
  Loader2,
  ExternalLink,
  Eye,
  EyeOff,
  Save,
  Trash2,
  AlertCircle,
  CheckCircle2,
  Settings,
  Cable,
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

// Map provider ID to a representative module name for icon display
const providerIconMap: Record<string, string> = {
  google: "google_calendar",
  microsoft: "microsoft_todo",
  todoist: "todoist",
  atlassian: "jira",
  notion: "notion",
  trello: "trello",
  github: "github",
  asana: "asana",
  airtable: "airtable",
  ticktick: "ticktick",
  dropbox: "dropbox",
}

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
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null)

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
        setSelectedProvider(null)
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

  // Separate configured and not-configured providers
  const configuredProviders = OAUTH_PROVIDERS.filter((p) => {
    const app = apps.find((a) => a.provider === p.id)
    return app?.has_credentials
  })
  const unconfiguredProviders = OAUTH_PROVIDERS.filter((p) => {
    const app = apps.find((a) => a.provider === p.id)
    return !app?.has_credentials
  })

  const dialogProvider = selectedProvider
    ? OAUTH_PROVIDERS.find((p) => p.id === selectedProvider)
    : null
  const dialogApp = selectedProvider
    ? apps.find((a) => a.provider === selectedProvider)
    : null
  const dialogForm = selectedProvider
    ? formStates[selectedProvider] || {
        clientId: "",
        clientSecret: "",
        redirectUri: getDefaultRedirectUri(selectedProvider),
        enabled: true,
      }
    : null

  return (
    <div className="p-6 space-y-6">
      <div className="pl-8 md:pl-0">
        <h1 className="text-2xl font-bold text-foreground">OAuth App Settings</h1>
        <p className="text-muted-foreground mt-1">
          Configure OAuth client credentials for external service integrations
        </p>
      </div>

      {/* Configured Providers */}
      {configuredProviders.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5 text-primary" />
            Configured
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {configuredProviders.map((provider) => (
              <div
                key={provider.id}
                onClick={() => setSelectedProvider(provider.id)}
                className="relative flex items-center gap-3 p-3 rounded-xl border bg-card/70 hover:bg-muted/50 transition-colors cursor-pointer group"
              >
                <div className="relative w-10 h-10 rounded-lg bg-white flex items-center justify-center shrink-0">
                  <ModuleIcon
                    moduleName={providerIconMap[provider.id] || provider.id}
                    className="h-5 w-5 text-foreground"
                  />
                  <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-500 ring-2 ring-card" />
                </div>
                <div className="min-w-0">
                  <span className="font-medium text-sm truncate block">{provider.name}</span>
                  <span className="text-[10px] text-muted-foreground truncate block">
                    {provider.description}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Unconfigured Providers */}
      {unconfiguredProviders.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground flex items-center gap-2">
            <Cable className="h-5 w-5 text-primary" />
            Not Configured
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {unconfiguredProviders.map((provider) => (
              <div
                key={provider.id}
                onClick={() => setSelectedProvider(provider.id)}
                className="flex items-center gap-3 p-3 rounded-xl border bg-card/70 hover:bg-muted/50 transition-colors cursor-pointer"
              >
                <div className="w-10 h-10 rounded-lg bg-white flex items-center justify-center shrink-0">
                  <ModuleIcon
                    moduleName={providerIconMap[provider.id] || provider.id}
                    className="h-5 w-5 text-foreground"
                  />
                </div>
                <div className="min-w-0">
                  <span className="font-medium text-sm truncate block">{provider.name}</span>
                  <span className="text-[10px] text-muted-foreground truncate block">
                    {provider.description}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Edit Dialog */}
      <Dialog open={!!selectedProvider} onOpenChange={(open) => !open && setSelectedProvider(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {dialogProvider && (
                <div className="w-10 h-10 rounded-lg bg-white flex items-center justify-center">
                  <ModuleIcon
                    moduleName={providerIconMap[dialogProvider.id] || dialogProvider.id}
                    className="h-5 w-5 text-foreground"
                  />
                </div>
              )}
              <div>
                <DialogTitle className="flex items-center gap-2">
                  {dialogProvider?.name}
                  {dialogApp?.has_credentials && (
                    <span className="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-600">
                      Configured
                    </span>
                  )}
                </DialogTitle>
                <DialogDescription>{dialogProvider?.description}</DialogDescription>
              </div>
            </div>
          </DialogHeader>

          {selectedProvider && dialogForm && (
            <div className="space-y-4 py-2">
              {/* Message */}
              {messages[selectedProvider] && (
                <div
                  className={`flex items-center gap-2 p-3 rounded-lg ${
                    messages[selectedProvider].type === "success"
                      ? "bg-green-500/10 text-green-600"
                      : "bg-destructive/10 text-destructive"
                  }`}
                >
                  {messages[selectedProvider].type === "success" ? (
                    <CheckCircle2 className="h-4 w-4" />
                  ) : (
                    <AlertCircle className="h-4 w-4" />
                  )}
                  <span className="text-sm">{messages[selectedProvider].text}</span>
                </div>
              )}

              {/* Client ID */}
              <div className="space-y-2">
                <Label htmlFor="dialog-client-id">Client ID</Label>
                <Input
                  id="dialog-client-id"
                  value={dialogForm.clientId}
                  onChange={(e) => updateFormState(selectedProvider, "clientId", e.target.value)}
                  placeholder="Enter client ID"
                  className="font-mono text-sm"
                />
              </div>

              {/* Client Secret */}
              <div className="space-y-2">
                <Label htmlFor="dialog-client-secret">
                  Client Secret
                  {dialogApp?.has_credentials && (
                    <span className="text-xs text-muted-foreground ml-2">(leave empty to keep existing)</span>
                  )}
                </Label>
                <div className="relative">
                  <Input
                    id="dialog-client-secret"
                    type={showSecrets[selectedProvider] ? "text" : "password"}
                    value={dialogForm.clientSecret}
                    onChange={(e) => updateFormState(selectedProvider, "clientSecret", e.target.value)}
                    placeholder={dialogApp?.has_credentials ? "••••••••••••" : "Enter client secret"}
                    className="font-mono text-sm pr-10"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7 p-0"
                    onClick={() =>
                      setShowSecrets({ ...showSecrets, [selectedProvider]: !showSecrets[selectedProvider] })
                    }
                  >
                    {showSecrets[selectedProvider] ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                </div>
              </div>

              {/* Redirect URI */}
              <div className="space-y-2">
                <Label htmlFor="dialog-redirect-uri">Redirect URI</Label>
                <Input
                  id="dialog-redirect-uri"
                  value={dialogForm.redirectUri}
                  onChange={(e) => updateFormState(selectedProvider, "redirectUri", e.target.value)}
                  placeholder="OAuth callback URL"
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  Copy this URL to your OAuth app configuration
                </p>
              </div>

              {/* Developer Console Link */}
              {dialogProvider?.docsUrl && (
                <a
                  href={dialogProvider.docsUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-primary hover:underline flex items-center gap-1"
                >
                  Open Developer Console
                  <ExternalLink className="h-3 w-3" />
                </a>
              )}
            </div>
          )}

          <DialogFooter className="flex-row justify-between sm:justify-between">
            <div>
              {dialogApp?.has_credentials && selectedProvider && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleDelete(selectedProvider)}
                  disabled={deleting[selectedProvider] || saving[selectedProvider]}
                  className="text-destructive hover:text-destructive"
                >
                  {deleting[selectedProvider] ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <>
                      <Trash2 className="h-4 w-4 mr-1" />
                      Delete
                    </>
                  )}
                </Button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={() => setSelectedProvider(null)}>
                Cancel
              </Button>
              {selectedProvider && (
                <Button
                  size="sm"
                  onClick={() => handleSave(selectedProvider)}
                  disabled={saving[selectedProvider] || deleting[selectedProvider]}
                >
                  {saving[selectedProvider] ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <>
                      <Save className="h-4 w-4 mr-1" />
                      Save
                    </>
                  )}
                </Button>
              )}
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
