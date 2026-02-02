export type OAuthApp = {
  provider: string
  redirect_uri: string
  enabled: boolean
  has_credentials: boolean
  client_id: string | null
  created_at: string
  updated_at: string
}

export type UpsertOAuthAppResult = {
  success: boolean
  action: string
  provider: string
}

export type DeleteOAuthAppResult = {
  success: boolean
  provider?: string
  error?: string
  message?: string
}

export class OAuthAppError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "OAuthAppError"
  }
}

// API Route Handler経由でOAuth Apps操作（Vault権限が必要なためサーバー側で処理）
export async function listOAuthApps(): Promise<OAuthApp[]> {
  const response = await fetch("/api/admin/oauth-apps", {
    method: "GET",
    credentials: "include",
  })

  if (!response.ok) {
    const error = await response.json()
    throw new OAuthAppError(error.error || "Failed to list OAuth apps")
  }

  return response.json()
}

export async function upsertOAuthApp(
  provider: string,
  clientId: string,
  clientSecret: string,
  redirectUri: string,
  enabled: boolean = true
): Promise<UpsertOAuthAppResult> {
  const response = await fetch("/api/admin/oauth-apps", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify({
      provider,
      client_id: clientId,
      client_secret: clientSecret,
      redirect_uri: redirectUri,
      enabled,
    }),
  })

  if (!response.ok) {
    const error = await response.json()
    throw new OAuthAppError(error.error || "Failed to upsert OAuth app")
  }

  return response.json()
}

export async function deleteOAuthApp(provider: string): Promise<DeleteOAuthAppResult> {
  const response = await fetch(`/api/admin/oauth-apps?provider=${encodeURIComponent(provider)}`, {
    method: "DELETE",
    credentials: "include",
  })

  if (!response.ok) {
    const error = await response.json()
    throw new OAuthAppError(error.error || "Failed to delete OAuth app")
  }

  return response.json()
}

// Provider display info
export const OAUTH_PROVIDERS = [
  {
    id: "google",
    name: "Google",
    description: "Google Calendar, Gmail, Drive など",
    docsUrl: "https://console.cloud.google.com/apis/credentials",
  },
  {
    id: "microsoft",
    name: "Microsoft",
    description: "Microsoft Todo, Outlook, OneDrive など",
    docsUrl: "https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps",
  },
  {
    id: "todoist",
    name: "Todoist",
    description: "Todoist タスク管理",
    docsUrl: "https://developer.todoist.com/appconsole.html",
  },
  {
    id: "atlassian",
    name: "Atlassian",
    description: "Jira, Confluence など",
    docsUrl: "https://developer.atlassian.com/console/myapps/",
  },
  {
    id: "notion",
    name: "Notion",
    description: "Notion ページ、データベース など",
    docsUrl: "https://www.notion.so/profile/integrations",
  },
] as const

export type OAuthProviderId = (typeof OAUTH_PROVIDERS)[number]["id"]

// デフォルトのRedirect URIを取得
export function getDefaultRedirectUri(provider: string): string {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/api/oauth/${provider}/callback`
  }
  return `https://mcpist.app/api/oauth/${provider}/callback`
}

// OAuth設定情報
export interface OAuthConfig {
  authUrl: string
  scopes: string[]
  serviceId: string  // token-vault で使用するサービスID
}

export const OAUTH_CONFIGS: Record<string, OAuthConfig> = {
  google: {
    authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    scopes: [
      "https://www.googleapis.com/auth/calendar",
      "https://www.googleapis.com/auth/calendar.events",
    ],
    serviceId: "google_calendar",
  },
  "google-tasks": {
    authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    scopes: [
      "https://www.googleapis.com/auth/tasks",
    ],
    serviceId: "google_tasks",
  },
  microsoft: {
    authUrl: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
    scopes: [
      "Tasks.ReadWrite",
      "offline_access",
    ],
    serviceId: "microsoft_todo",
  },
  todoist: {
    authUrl: "https://todoist.com/oauth/authorize",
    scopes: [
      "data:read_write",
      "data:delete",
    ],
    serviceId: "todoist",
  },
  atlassian: {
    authUrl: "https://auth.atlassian.com/authorize",
    scopes: [
      "read:jira-work",
      "write:jira-work",
      "read:jira-user",
      "manage:jira-project",
      "read:confluence-content.all",
      "write:confluence-content",
      "read:confluence-space.summary",
      "offline_access",
    ],
    serviceId: "jira",  // Jira をプライマリとして使用
  },
  "atlassian-jira": {
    authUrl: "https://auth.atlassian.com/authorize",
    scopes: [
      "read:jira-work",
      "write:jira-work",
      "read:jira-user",
      "manage:jira-project",
      "offline_access",
    ],
    serviceId: "jira",
  },
  "atlassian-confluence": {
    authUrl: "https://auth.atlassian.com/authorize",
    scopes: [
      "read:confluence-content.all",
      "write:confluence-content",
      "read:confluence-space.summary",
      "offline_access",
    ],
    serviceId: "confluence",
  },
  notion: {
    authUrl: "https://api.notion.com/v1/oauth/authorize",
    scopes: [],  // Notion はスコープを URL パラメータで指定しない
    serviceId: "notion",
  },
}

// サービスIDからOAuthプロバイダーIDを取得
export function getOAuthProviderForService(serviceId: string): string | null {
  for (const [providerId, config] of Object.entries(OAUTH_CONFIGS)) {
    if (config.serviceId === serviceId) {
      return providerId
    }
  }
  return null
}

// OAuth認可URLを取得するAPI Route経由
export async function getOAuthAuthorizationUrl(
  provider: string,
  returnTo?: string
): Promise<string> {
  const params = new URLSearchParams()
  if (returnTo) {
    params.set("returnTo", returnTo)
  }

  // google-tasks は google の authorize を使い、module パラメータで区別
  // atlassian-* は atlassian の authorize を使い、module パラメータで区別
  let apiPath = provider
  if (provider === "google-tasks") {
    apiPath = "google"
    params.set("module", "google_tasks")
  } else if (provider === "atlassian-jira") {
    apiPath = "atlassian"
    params.set("module", "jira")
  } else if (provider === "atlassian-confluence") {
    apiPath = "atlassian"
    params.set("module", "confluence")
  } else if (provider === "atlassian") {
    params.set("module", "atlassian")
  }

  const url = `/api/oauth/${apiPath}/authorize${params.toString() ? `?${params.toString()}` : ""}`
  const response = await fetch(url, {
    method: "GET",
    credentials: "include",
  })

  if (!response.ok) {
    const error = await response.json()
    throw new OAuthAppError(error.error || "Failed to get authorization URL")
  }

  const data = await response.json()
  return data.authorizationUrl
}
