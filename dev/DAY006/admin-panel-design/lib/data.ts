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
    id: "slack",
    name: "Slack",
    description: "チームコミュニケーション連携",
    icon: "message-square",
    status: "disconnected",
    category: "communication",
  },
  {
    id: "zaim",
    name: "Zaim",
    description: "家計簿データの取得と分析",
    icon: "wallet",
    status: "disconnected",
    category: "finance",
  },
  {
    id: "freee",
    name: "freee",
    description: "会計・経費データの連携",
    icon: "calculator",
    status: "connected",
    category: "finance",
  },
  {
    id: "dropbox",
    name: "Dropbox",
    description: "ファイルストレージ連携",
    icon: "cloud",
    status: "no-permission",
    category: "storage",
  },
  {
    id: "google-drive",
    name: "Google Drive",
    description: "ドライブファイルの管理",
    icon: "hard-drive",
    status: "connected",
    category: "storage",
  },
  {
    id: "trello",
    name: "Trello",
    description: "タスクボードの連携",
    icon: "layout-grid",
    status: "no-permission",
    category: "productivity",
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
    id: "asana",
    name: "Asana",
    description: "タスク管理連携",
    icon: "check-square",
    status: "disconnected",
    category: "productivity",
  },
  {
    id: "moneytree",
    name: "Moneytree",
    description: "資産管理データの取得",
    icon: "trending-up",
    status: "connected",
    category: "finance",
  },
]

export interface User {
  id: string
  name: string
  email: string
  roles: string[]
  lastLogin: string
  avatar?: string
}

export const users: User[] = [
  {
    id: "1",
    name: "山田 太郎",
    email: "yamada@example.com",
    roles: ["管理者", "開発者"],
    lastLogin: "2026-01-15 10:30",
  },
  {
    id: "2",
    name: "佐藤 花子",
    email: "sato@example.com",
    roles: ["開発者"],
    lastLogin: "2026-01-15 09:15",
  },
  {
    id: "3",
    name: "鈴木 一郎",
    email: "suzuki@example.com",
    roles: ["閲覧者"],
    lastLogin: "2026-01-14 18:00",
  },
  {
    id: "4",
    name: "田中 美咲",
    email: "tanaka@example.com",
    roles: ["開発者", "閲覧者"],
    lastLogin: "2026-01-15 08:45",
  },
  {
    id: "5",
    name: "高橋 健太",
    email: "takahashi@example.com",
    roles: ["管理者"],
    lastLogin: "2026-01-13 14:20",
  },
]

export interface Role {
  id: string
  name: string
  description: string
  userCount: number
  permissions: string[]
  services: {
    serviceId: string
    clientId?: string
    clientSecret?: string
    authMethod: "oidc" | "oauth" | "apikey"
  }[]
}

export const roles: Role[] = [
  {
    id: "1",
    name: "管理者",
    description: "全ての機能にアクセス可能",
    userCount: 2,
    permissions: ["tools.manage", "users.manage", "roles.manage", "logs.view"],
    services: [],
  },
  {
    id: "2",
    name: "開発者",
    description: "ツールの連携と利用が可能",
    userCount: 3,
    permissions: ["tools.use", "tools.connect"],
    services: [
      { serviceId: "github", authMethod: "oauth" },
      { serviceId: "notion", authMethod: "oauth" },
    ],
  },
  {
    id: "3",
    name: "閲覧者",
    description: "読み取り専用アクセス",
    userCount: 2,
    permissions: ["tools.view"],
    services: [],
  },
  {
    id: "4",
    name: "経理担当",
    description: "経理関連ツールへのアクセス",
    userCount: 1,
    permissions: ["tools.use"],
    services: [
      { serviceId: "freee", authMethod: "oauth" },
      { serviceId: "zaim", authMethod: "apikey" },
    ],
  },
  {
    id: "5",
    name: "外部パートナー",
    description: "限定的なアクセス権限",
    userCount: 5,
    permissions: ["tools.view"],
    services: [],
  },
]

export const allPermissions = [
  { id: "tools.manage", label: "ツール管理", description: "ツールの追加・削除・設定変更" },
  { id: "tools.connect", label: "ツール連携", description: "ツールの連携・解除" },
  { id: "tools.use", label: "ツール利用", description: "連携済みツールの利用" },
  { id: "tools.view", label: "ツール閲覧", description: "ツール一覧の閲覧" },
  { id: "users.manage", label: "ユーザー管理", description: "ユーザーの追加・削除・編集" },
  { id: "roles.manage", label: "ロール管理", description: "ロールの追加・削除・編集" },
  { id: "logs.view", label: "ログ閲覧", description: "システムログの閲覧" },
]

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
          { name: "calendar_id", type: "string", required: false, description: "カレンダーID (デフォルト: primary)" },
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
          { name: "description", type: "string", required: false, description: "イベントの説明" },
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
          { name: "sort", type: "string", required: false, description: "created, updated, pushed, full_name" },
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
          { name: "body", type: "string", required: false, description: "イシューの本文" },
        ],
        hasPermission: true,
      },
    ],
  },
  slack: {
    id: "slack",
    name: "Slack",
    description: "チームコミュニケーション連携",
    icon: "message-square",
    status: "disconnected",
    tools: [
      {
        id: "send-message",
        name: "send_message",
        description: "チャンネルにメッセージを送信します",
        parameters: [
          { name: "channel", type: "string", required: true, description: "チャンネルID または チャンネル名" },
          { name: "text", type: "string", required: true, description: "メッセージ本文" },
        ],
        hasPermission: true,
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
  zaim: {
    id: "zaim",
    name: "Zaim",
    description: "家計簿データの取得と分析",
    icon: "wallet",
    status: "disconnected",
    tools: [
      {
        id: "get-transactions",
        name: "get_transactions",
        description: "取引履歴を取得します",
        parameters: [
          { name: "start_date", type: "string", required: true, description: "開始日" },
          { name: "end_date", type: "string", required: true, description: "終了日" },
        ],
        hasPermission: true,
      },
    ],
  },
  freee: {
    id: "freee",
    name: "freee",
    description: "会計・経費データの連携",
    icon: "calculator",
    status: "connected",
    tools: [
      {
        id: "get-invoices",
        name: "get_invoices",
        description: "請求書一覧を取得します",
        parameters: [{ name: "status", type: "string", required: false, description: "draft, sent, paid" }],
        hasPermission: true,
      },
    ],
  },
  dropbox: {
    id: "dropbox",
    name: "Dropbox",
    description: "ファイルストレージ連携",
    icon: "cloud",
    status: "no-permission",
    tools: [
      {
        id: "list-files",
        name: "list_files",
        description: "フォルダ内のファイル一覧を取得します",
        parameters: [{ name: "path", type: "string", required: true, description: "フォルダパス" }],
        hasPermission: false,
      },
    ],
  },
  "google-drive": {
    id: "google-drive",
    name: "Google Drive",
    description: "ドライブファイルの管理",
    icon: "hard-drive",
    status: "connected",
    tools: [
      {
        id: "list-files",
        name: "list_files",
        description: "ファイル一覧を取得します",
        parameters: [{ name: "folder_id", type: "string", required: false, description: "フォルダID" }],
        hasPermission: true,
      },
    ],
  },
  trello: {
    id: "trello",
    name: "Trello",
    description: "タスクボードの連携",
    icon: "layout-grid",
    status: "no-permission",
    tools: [
      {
        id: "list-boards",
        name: "list_boards",
        description: "ボード一覧を取得します",
        parameters: [],
        hasPermission: false,
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
  asana: {
    id: "asana",
    name: "Asana",
    description: "タスク管理連携",
    icon: "check-square",
    status: "disconnected",
    tools: [
      {
        id: "list-tasks",
        name: "list_tasks",
        description: "タスク一覧を取得します",
        parameters: [{ name: "project_id", type: "string", required: true, description: "プロジェクトID" }],
        hasPermission: true,
      },
    ],
  },
  moneytree: {
    id: "moneytree",
    name: "Moneytree",
    description: "資産管理データの取得",
    icon: "trending-up",
    status: "connected",
    tools: [
      {
        id: "get-accounts",
        name: "get_accounts",
        description: "口座一覧を取得します",
        parameters: [],
        hasPermission: true,
      },
    ],
  },
}

export interface ApiToken {
  id: string
  name: string
  createdAt: string
  expiresAt: string | null
  lastUsedAt: string | null
  status: "active" | "expired" | "revoked"
  prefix: string
}

export const apiTokens: ApiToken[] = [
  {
    id: "1",
    name: "開発環境用",
    createdAt: "2026-01-10",
    expiresAt: "2026-04-10",
    lastUsedAt: "2026-01-15 09:30",
    status: "active",
    prefix: "mcp_dev_",
  },
  {
    id: "2",
    name: "本番環境用",
    createdAt: "2025-12-01",
    expiresAt: null,
    lastUsedAt: "2026-01-15 10:45",
    status: "active",
    prefix: "mcp_prod_",
  },
  {
    id: "3",
    name: "テスト用",
    createdAt: "2025-10-15",
    expiresAt: "2026-01-01",
    lastUsedAt: "2025-12-28 14:00",
    status: "expired",
    prefix: "mcp_test_",
  },
]

export interface UsageRequest {
  id: string
  userId: string
  userName: string
  userEmail: string
  serviceId: string
  serviceName: string
  toolId?: string
  toolName?: string
  reason: string
  requestedAt: string
  status: "pending" | "approved" | "rejected"
  reviewedAt?: string
  reviewedBy?: string
  rejectionReason?: string
}

export const usageRequests: UsageRequest[] = [
  {
    id: "1",
    userId: "3",
    userName: "鈴木 一郎",
    userEmail: "suzuki@example.com",
    serviceId: "dropbox",
    serviceName: "Dropbox",
    reason: "プロジェクト資料の共有に必要なため",
    requestedAt: "2026-01-15 09:00",
    status: "pending",
  },
  {
    id: "2",
    userId: "4",
    userName: "田中 美咲",
    userEmail: "tanaka@example.com",
    serviceId: "google-calendar",
    serviceName: "Google Calendar",
    toolId: "delete-event",
    toolName: "delete_event",
    reason: "チームのカレンダー整理のため削除権限が必要",
    requestedAt: "2026-01-14 16:30",
    status: "pending",
  },
  {
    id: "3",
    userId: "2",
    userName: "佐藤 花子",
    userEmail: "sato@example.com",
    serviceId: "trello",
    serviceName: "Trello",
    reason: "タスク管理の連携のため",
    requestedAt: "2026-01-13 11:00",
    status: "approved",
    reviewedAt: "2026-01-13 14:00",
    reviewedBy: "山田 太郎",
  },
  {
    id: "4",
    userId: "5",
    userName: "高橋 健太",
    userEmail: "takahashi@example.com",
    serviceId: "jira",
    serviceName: "Jira",
    reason: "開発チームのプロジェクト管理",
    requestedAt: "2026-01-12 10:00",
    status: "rejected",
    reviewedAt: "2026-01-12 15:00",
    reviewedBy: "山田 太郎",
    rejectionReason: "現在Jiraの利用は開発部門のみに限定しています",
  },
]

export interface Profile {
  id: string
  name: string
  description: string
  appliedRoles: string[]
  modulePermissions: Record<string, string[]> // moduleId -> toolIds
}

export const profiles: Profile[] = [
  {
    id: "1",
    name: "開発者標準",
    description: "開発者向けの標準的な権限セット",
    appliedRoles: ["2"],
    modulePermissions: {
      github: ["list-repos", "create-issue"],
      "google-calendar": ["list-events", "create-event"],
      notion: ["search-pages"],
    },
  },
  {
    id: "2",
    name: "閲覧のみ",
    description: "読み取り専用の最小権限セット",
    appliedRoles: ["3", "5"],
    modulePermissions: {
      "google-calendar": ["list-events"],
      github: ["list-repos"],
    },
  },
  {
    id: "3",
    name: "フルアクセス",
    description: "全てのツールへのアクセス権限",
    appliedRoles: ["1"],
    modulePermissions: {
      github: ["list-repos", "create-issue"],
      "google-calendar": ["list-events", "create-event", "delete-event"],
      notion: ["search-pages"],
      slack: ["send-message"],
    },
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
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "notion-client-id-xxx",
          clientSecret: "notion-client-secret-xxx",
          scopes: ["read_content", "update_content"],
        },
      },
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
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "github-client-id-xxx",
          clientSecret: "github-client-secret-xxx",
          scopes: ["repo", "read:user"],
        },
      },
      {
        type: "personal_token",
        enabled: true,
        label: "Personal Access Token",
        helpText: "GitHub Settings > Developer settings > Personal access tokensから発行してください",
      },
    ],
  },
  {
    serviceId: "slack",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "slack-client-id-xxx",
          clientSecret: "slack-client-secret-xxx",
          scopes: ["chat:write", "channels:read"],
        },
      },
    ],
  },
  {
    serviceId: "jira",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "jira-client-id-xxx",
          clientSecret: "jira-client-secret-xxx",
          scopes: ["read:jira-work", "write:jira-work"],
        },
      },
      {
        type: "apikey",
        enabled: true,
        label: "APIキー",
        helpText: "Atlassian管理画面 > APIトークンから発行してください",
      },
    ],
  },
  {
    serviceId: "freee",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "freee-client-id-xxx",
          clientSecret: "freee-client-secret-xxx",
          scopes: ["read", "write"],
        },
      },
    ],
  },
  {
    serviceId: "zaim",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "zaim-client-id-xxx",
          clientSecret: "zaim-client-secret-xxx",
          scopes: ["read"],
        },
      },
    ],
  },
  {
    serviceId: "dropbox",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "dropbox-client-id-xxx",
          clientSecret: "dropbox-client-secret-xxx",
          scopes: ["files.read", "files.write"],
        },
      },
    ],
  },
  {
    serviceId: "google-drive",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "google-drive-client-id-xxx",
          clientSecret: "google-drive-client-secret-xxx",
          scopes: ["https://www.googleapis.com/auth/drive"],
        },
      },
    ],
  },
  {
    serviceId: "trello",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "trello-client-id-xxx",
          clientSecret: "trello-client-secret-xxx",
          scopes: ["read", "write"],
        },
      },
      {
        type: "apikey",
        enabled: true,
        label: "APIキー",
        helpText: "Trello Power-Ups設定からAPIキーを取得してください",
      },
    ],
  },
  {
    serviceId: "asana",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "asana-client-id-xxx",
          clientSecret: "asana-client-secret-xxx",
          scopes: ["default"],
        },
      },
      {
        type: "personal_token",
        enabled: true,
        label: "Personal Access Token",
        helpText: "Asana設定 > アプリ > 開発者アプリの管理から取得してください",
      },
    ],
  },
  {
    serviceId: "moneytree",
    availableMethods: [
      {
        type: "oauth2",
        enabled: true,
        label: "OAuth 2.0",
        oauth: {
          clientId: "moneytree-client-id-xxx",
          clientSecret: "moneytree-client-secret-xxx",
          scopes: ["accounts_read", "transactions_read"],
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

export const userCredentials: UserServiceCredential[] = [
  {
    id: "1",
    userId: "1",
    serviceId: "google-calendar",
    authMethod: "oauth2",
    status: "active",
    connectedAt: "2026-01-10",
  },
  {
    id: "2",
    userId: "1",
    serviceId: "notion",
    authMethod: "integration_token",
    status: "active",
    connectedAt: "2026-01-12",
  },
  {
    id: "3",
    userId: "1",
    serviceId: "github",
    authMethod: "oauth2",
    status: "active",
    connectedAt: "2026-01-08",
  },
  {
    id: "4",
    userId: "2",
    serviceId: "notion",
    authMethod: "oauth2",
    status: "active",
    connectedAt: "2026-01-14",
  },
  {
    id: "5",
    userId: "2",
    serviceId: "github",
    authMethod: "personal_token",
    status: "active",
    connectedAt: "2026-01-11",
  },
  {
    id: "6",
    userId: "3",
    serviceId: "google-calendar",
    authMethod: "oauth2",
    status: "expired",
    connectedAt: "2025-12-01",
  },
]

export interface UserMcpConnection {
  userId: string
  endpoint: string
  apiToken: string | null
  tokenPrefix: string
  generatedAt: string | null
  status: "not_generated" | "active" | "revoked"
}

export const userMcpConnections: UserMcpConnection[] = [
  {
    userId: "1",
    endpoint: "https://mcp.example.com/u/1",
    apiToken: "mcp_usr_****",
    tokenPrefix: "mcp_usr_",
    generatedAt: "2026-01-10",
    status: "active",
  },
  {
    userId: "2",
    endpoint: "https://mcp.example.com/u/2",
    apiToken: null,
    tokenPrefix: "mcp_usr_",
    generatedAt: null,
    status: "not_generated",
  },
  {
    userId: "3",
    endpoint: "https://mcp.example.com/u/3",
    apiToken: null,
    tokenPrefix: "mcp_usr_",
    generatedAt: null,
    status: "not_generated",
  },
  {
    userId: "4",
    endpoint: "https://mcp.example.com/u/4",
    apiToken: "mcp_usr_****",
    tokenPrefix: "mcp_usr_",
    generatedAt: "2026-01-05",
    status: "active",
  },
  {
    userId: "5",
    endpoint: "https://mcp.example.com/u/5",
    apiToken: "mcp_usr_****",
    tokenPrefix: "mcp_usr_",
    generatedAt: "2025-11-20",
    status: "revoked",
  },
]
