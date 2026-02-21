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
  {
    id: "trello",
    name: "Trello",
    description: "Trello ボード、カード、チェックリスト など",
    docsUrl: "https://trello.com/power-ups/admin",
  },
  {
    id: "github",
    name: "GitHub",
    description: "GitHub リポジトリ、Issue、PR、Actions など",
    docsUrl: "https://github.com/settings/developers",
  },
  {
    id: "asana",
    name: "Asana",
    description: "Asana ワークスペース、プロジェクト、タスク など",
    docsUrl: "https://app.asana.com/0/developer-console",
  },
  {
    id: "airtable",
    name: "Airtable",
    description: "Airtable ベース、テーブル、レコード など",
    docsUrl: "https://airtable.com/create/oauth",
  },
  {
    id: "ticktick",
    name: "TickTick",
    description: "TickTick タスク・プロジェクト管理",
    docsUrl: "https://developer.ticktick.com/",
  },
  {
    id: "dropbox",
    name: "Dropbox",
    description: "Dropbox ファイル・フォルダ管理",
    docsUrl: "https://www.dropbox.com/developers/apps",
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
  "google-drive": {
    authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    scopes: [
      "https://www.googleapis.com/auth/drive",
    ],
    serviceId: "google_drive",
  },
  "google-docs": {
    authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    scopes: [
      "https://www.googleapis.com/auth/documents",
      "https://www.googleapis.com/auth/drive",
    ],
    serviceId: "google_docs",
  },
  "google-sheets": {
    authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    scopes: [
      "https://www.googleapis.com/auth/spreadsheets",
      "https://www.googleapis.com/auth/drive.readonly",
    ],
    serviceId: "google_sheets",
  },
  "google-apps-script": {
    authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    scopes: [
      "https://www.googleapis.com/auth/script.projects",
      "https://www.googleapis.com/auth/script.deployments",
      "https://www.googleapis.com/auth/script.metrics",
      "https://www.googleapis.com/auth/script.processes",
      "https://www.googleapis.com/auth/script.scriptapp",  // For run_function
      "https://www.googleapis.com/auth/drive.readonly",
    ],
    serviceId: "google_apps_script",
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
      "read:space:confluence",
      "read:page:confluence",
      "write:page:confluence",
      "read:content-details:confluence",
      "write:comment:confluence",
      "read:comment:confluence",
      "read:label:confluence",
      "write:label:confluence",
      "search:confluence",
      "offline_access",
    ],
    serviceId: "confluence",
  },
  notion: {
    authUrl: "https://api.notion.com/v1/oauth/authorize",
    scopes: [],  // Notion はスコープを URL パラメータで指定しない
    serviceId: "notion",
  },
  trello: {
    authUrl: "https://trello.com/1/OAuthAuthorizeToken",  // OAuth 1.0a
    scopes: ["read", "write"],
    serviceId: "trello",
  },
  github: {
    authUrl: "https://github.com/login/oauth/authorize",
    scopes: ["repo", "read:user"],
    serviceId: "github",
  },
  asana: {
    authUrl: "https://app.asana.com/-/oauth_authorize",
    scopes: [],  // Asana doesn't use scope parameter in OAuth authorize request
    serviceId: "asana",
  },
  airtable: {
    authUrl: "https://airtable.com/oauth2/v1/authorize",
    scopes: [
      "data.records:read",
      "data.records:write",
      "schema.bases:read",
      "schema.bases:write",
    ],
    serviceId: "airtable",
  },
  ticktick: {
    authUrl: "https://ticktick.com/oauth/authorize",
    scopes: [
      "tasks:read",
      "tasks:write",
    ],
    serviceId: "ticktick",
  },
  dropbox: {
    authUrl: "https://www.dropbox.com/oauth2/authorize",
    scopes: [
      "files.metadata.read",
      "files.metadata.write",
      "files.content.read",
      "files.content.write",
      "sharing.read",
      "sharing.write",
    ],
    serviceId: "dropbox",
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

  // google-tasks, google-drive は google の authorize を使い、module パラメータで区別
  // atlassian-* は atlassian の authorize を使い、module パラメータで区別
  let apiPath = provider
  if (provider === "google-tasks") {
    apiPath = "google"
    params.set("module", "google_tasks")
  } else if (provider === "google-drive") {
    apiPath = "google"
    params.set("module", "google_drive")
  } else if (provider === "google-docs") {
    apiPath = "google"
    params.set("module", "google_docs")
  } else if (provider === "google-sheets") {
    apiPath = "google"
    params.set("module", "google_sheets")
  } else if (provider === "google-apps-script") {
    apiPath = "google"
    params.set("module", "google_apps_script")
  } else if (provider === "atlassian-jira") {
    apiPath = "atlassian"
    params.set("module", "jira")
  } else if (provider === "atlassian-confluence") {
    apiPath = "atlassian"
    params.set("module", "confluence")
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
