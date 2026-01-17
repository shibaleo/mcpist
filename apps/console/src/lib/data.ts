// Mock data for the MCP Server Admin

export type ServiceStatus = "connected" | "disconnected" | "no-permission"

export interface Service {
  id: string
  name: string
  description: string
  icon: string
  status: ServiceStatus
  category: string
}

export const services: Service[] = [
  {
    id: "google-calendar",
    name: "Google Calendar",
    description: "カレンダーイベントの管理と同期",
    icon: "calendar",
    status: "connected",
    category: "productivity",
  },
  {
    id: "notion",
    name: "Notion",
    description: "ドキュメントとデータベースの連携",
    icon: "file-text",
    status: "connected",
    category: "productivity",
  },
  {
    id: "github",
    name: "GitHub",
    description: "リポジトリとイシューの管理",
    icon: "github",
    status: "connected",
    category: "development",
  },
  {
    id: "jira",
    name: "Jira",
    description: "プロジェクト管理連携",
    icon: "kanban",
    status: "no-permission",
    category: "development",
  },
  {
    id: "confluence",
    name: "Confluence",
    description: "Wiki・ドキュメント管理",
    icon: "book",
    status: "disconnected",
    category: "productivity",
  },
  {
    id: "microsoft-todo",
    name: "Microsoft To Do",
    description: "タスク・リストの管理",
    icon: "check-square",
    status: "disconnected",
    category: "productivity",
  },
]

export interface AuthMethodConfig {
  type: "oauth2" | "apikey" | "personal_token" | "integration_token"
  enabled: boolean
  label: string
  oauth?: {
    clientId: string
    clientSecret: string
    scopes: string[]
  }
  helpText?: string
}

export interface ServiceAuthConfig {
  serviceId: string
  availableMethods: AuthMethodConfig[]
}

export const serviceAuthConfigs: ServiceAuthConfig[] = [
  {
    serviceId: "google-calendar",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "google-client-id-xxx",
          clientSecret: "google-client-secret-xxx",
          scopes: ["https://www.googleapis.com/auth/calendar"],
        },
      },
    ],
  },
  {
    serviceId: "notion",
    availableMethods: [
      {
        type: "integration_token",
        enabled: true,
        label: "内部インテグレーション",
        helpText: "Notion設定 > インテグレーション > 新しいインテグレーションから取得してください",
      },
    ],
  },
  {
    serviceId: "github",
    availableMethods: [
      {
        type: "personal_token",
        enabled: true,
        label: "Personal Access Token",
        helpText: "GitHub Settings > Developer settings > Personal access tokensから発行してください",
      },
    ],
  },
  {
    serviceId: "jira",
    availableMethods: [
      {
        type: "apikey",
        enabled: true,
        label: "APIキー",
        helpText: "Atlassian管理画面 > APIトークンから発行してください",
      },
    ],
  },
  {
    serviceId: "confluence",
    availableMethods: [
      {
        type: "apikey",
        enabled: true,
        label: "APIキー",
        helpText: "Atlassian管理画面 > APIトークンから発行してください（Jiraと共通）",
      },
    ],
  },
  {
    serviceId: "microsoft-todo",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "microsoft-client-id-xxx",
          clientSecret: "microsoft-client-secret-xxx",
          scopes: ["Tasks.ReadWrite"],
        },
      },
    ],
  },
]

export interface UserServiceCredential {
  id: string
  userId: string
  serviceId: string
  authMethod: "oauth2" | "apikey" | "personal_token" | "integration_token"
  status: "active" | "expired" | "error"
  connectedAt: string
}

export const userCredentials: UserServiceCredential[] = []

export interface UserMcpConnection {
  userId: string
  endpoint: string
  apiToken: string | null
  tokenPrefix: string
  generatedAt: string | null
  status: "not_generated" | "active" | "revoked"
}

export const userMcpConnections: UserMcpConnection[] = []

// ===========================================
// Plan related types and data
// ===========================================

export type PlanType = "free" | "pro" | "max"

export interface PlanInfo {
  id: PlanType
  name: string
  price: number // 月額（円）
  description: string
  features: string[]
}

export const plans: PlanInfo[] = [
  {
    id: "free",
    name: "Free",
    price: 0,
    description: "個人利用や小規模チーム向け",
    features: ["4サービスまで利用可能", "読み取り系機能のみ", "ユーザー5名まで"],
  },
  {
    id: "pro",
    name: "Pro",
    price: 2980,
    description: "成長中のチーム向け",
    features: ["8サービスまで利用可能", "読み取り・書き込み機能", "ユーザー20名まで", "優先サポート"],
  },
  {
    id: "max",
    name: "Max",
    price: 9800,
    description: "大規模組織向け",
    features: ["全サービス利用可能", "全機能（削除含む）", "ユーザー無制限", "専任サポート", "SLA保証"],
  },
]

// 組織の現在のプラン
export interface OrganizationPlan {
  currentPlan: PlanType
  planName: string
  billingCycle: "monthly" | "yearly"
  nextBillingDate: string
  userCount: number
  userLimit: number
}

export const organizationPlan: OrganizationPlan = {
  currentPlan: "free",
  planName: "Free",
  billingCycle: "monthly",
  nextBillingDate: "2026-02-15",
  userCount: 1,
  userLimit: 5,
}

// サービスごとの必要プラン
export interface ServicePlanRequirement {
  serviceId: string
  requiredPlan: PlanType
}

export const servicePlanRequirements: ServicePlanRequirement[] = [
  // Free プラン
  { serviceId: "google-calendar", requiredPlan: "free" },
  { serviceId: "notion", requiredPlan: "free" },
  { serviceId: "github", requiredPlan: "free" },
  { serviceId: "microsoft-todo", requiredPlan: "free" },
  // Pro プラン
  { serviceId: "jira", requiredPlan: "pro" },
  { serviceId: "confluence", requiredPlan: "pro" },
]

// 請求履歴
export interface BillingHistory {
  id: string
  date: string
  plan: string
  amount: number
  status: "paid" | "pending" | "failed"
}

export const billingHistory: BillingHistory[] = []

// ===========================================
// Helper functions
// ===========================================

const planOrder: Record<PlanType, number> = { free: 0, pro: 1, max: 2 }

export function isPlanSufficient(currentPlan: PlanType, requiredPlan: PlanType): boolean {
  return planOrder[currentPlan] >= planOrder[requiredPlan]
}

export function getServiceRequiredPlan(serviceId: string): PlanType {
  return servicePlanRequirements.find((r) => r.serviceId === serviceId)?.requiredPlan || "free"
}

// ===========================================
// Module Details (for preferences page)
// ===========================================

export interface Tool {
  id: string
  name: string
  description: string
  parameters: { name: string; type: string; required: boolean; description: string }[]
  hasPermission: boolean
}

export interface ModuleDetail {
  id: string
  name: string
  description: string
  icon: string
  status: ServiceStatus
  tools: Tool[]
}

export const moduleDetails: Record<string, ModuleDetail> = {
  "google-calendar": {
    id: "google-calendar",
    name: "Google Calendar",
    description: "カレンダーイベントの管理と同期",
    icon: "calendar",
    status: "connected",
    tools: [
      {
        id: "list-events",
        name: "list_events",
        description: "指定した期間のイベント一覧を取得します",
        parameters: [
          { name: "start_date", type: "string", required: true, description: "開始日 (ISO 8601形式)" },
          { name: "end_date", type: "string", required: true, description: "終了日 (ISO 8601形式)" },
        ],
        hasPermission: true,
      },
      {
        id: "create-event",
        name: "create_event",
        description: "新しいイベントを作成します",
        parameters: [
          { name: "title", type: "string", required: true, description: "イベントのタイトル" },
          { name: "start_time", type: "string", required: true, description: "開始時刻 (ISO 8601形式)" },
          { name: "end_time", type: "string", required: true, description: "終了時刻 (ISO 8601形式)" },
        ],
        hasPermission: true,
      },
      {
        id: "delete-event",
        name: "delete_event",
        description: "指定したイベントを削除します",
        parameters: [{ name: "event_id", type: "string", required: true, description: "イベントID" }],
        hasPermission: false,
      },
    ],
  },
  notion: {
    id: "notion",
    name: "Notion",
    description: "ドキュメントとデータベースの連携",
    icon: "file-text",
    status: "connected",
    tools: [
      {
        id: "search-pages",
        name: "search_pages",
        description: "ページを検索します",
        parameters: [{ name: "query", type: "string", required: true, description: "検索クエリ" }],
        hasPermission: true,
      },
    ],
  },
  github: {
    id: "github",
    name: "GitHub",
    description: "リポジトリとイシューの管理",
    icon: "github",
    status: "connected",
    tools: [
      {
        id: "list-repos",
        name: "list_repositories",
        description: "ユーザーのリポジトリ一覧を取得します",
        parameters: [
          { name: "visibility", type: "string", required: false, description: "public, private, all" },
        ],
        hasPermission: true,
      },
      {
        id: "create-issue",
        name: "create_issue",
        description: "新しいイシューを作成します",
        parameters: [
          { name: "repo", type: "string", required: true, description: "リポジトリ名 (owner/repo形式)" },
          { name: "title", type: "string", required: true, description: "イシューのタイトル" },
        ],
        hasPermission: true,
      },
    ],
  },
  jira: {
    id: "jira",
    name: "Jira",
    description: "プロジェクト管理連携",
    icon: "kanban",
    status: "no-permission",
    tools: [
      {
        id: "list-issues",
        name: "list_issues",
        description: "イシュー一覧を取得します",
        parameters: [{ name: "project", type: "string", required: true, description: "プロジェクトキー" }],
        hasPermission: false,
      },
    ],
  },
  confluence: {
    id: "confluence",
    name: "Confluence",
    description: "Wiki・ドキュメント管理",
    icon: "book",
    status: "disconnected",
    tools: [
      {
        id: "search-pages",
        name: "search_pages",
        description: "ページを検索します",
        parameters: [{ name: "query", type: "string", required: true, description: "検索クエリ" }],
        hasPermission: true,
      },
    ],
  },
  "microsoft-todo": {
    id: "microsoft-todo",
    name: "Microsoft To Do",
    description: "タスク・リストの管理",
    icon: "check-square",
    status: "disconnected",
    tools: [
      {
        id: "list-tasks",
        name: "list_tasks",
        description: "タスク一覧を取得します",
        parameters: [{ name: "list_id", type: "string", required: false, description: "リストID" }],
        hasPermission: true,
      },
    ],
  },
}

// 機能（ツール）ごとの必要プラン
export interface ToolPlanRequirement {
  serviceId: string
  toolId: string
  requiredPlan: PlanType
}

export const toolPlanRequirements: ToolPlanRequirement[] = [
  // Google Calendar
  { serviceId: "google-calendar", toolId: "list-events", requiredPlan: "free" },
  { serviceId: "google-calendar", toolId: "create-event", requiredPlan: "pro" },
  { serviceId: "google-calendar", toolId: "delete-event", requiredPlan: "max" },
  // GitHub
  { serviceId: "github", toolId: "list-repos", requiredPlan: "free" },
  { serviceId: "github", toolId: "create-issue", requiredPlan: "pro" },
  // Notion
  { serviceId: "notion", toolId: "search-pages", requiredPlan: "free" },
  // Jira
  { serviceId: "jira", toolId: "list-issues", requiredPlan: "pro" },
  // Confluence
  { serviceId: "confluence", toolId: "search-pages", requiredPlan: "pro" },
  // Microsoft To Do
  { serviceId: "microsoft-todo", toolId: "list-tasks", requiredPlan: "free" },
]

// ユーザー個別の機能有効化設定
export interface UserToolPreference {
  userId: string
  enabledTools: Record<string, string[]> // serviceId -> toolIds
}

export const userToolPreferences: UserToolPreference[] = []

export function getToolRequiredPlan(serviceId: string, toolId: string): PlanType {
  return toolPlanRequirements.find((r) => r.serviceId === serviceId && r.toolId === toolId)?.requiredPlan || "free"
}
