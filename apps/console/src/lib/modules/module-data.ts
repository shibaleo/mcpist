// Module data fetched from database via list_modules_with_tools RPC
// Replaces the previous static tools.json import

const WORKER_URL = process.env.NEXT_PUBLIC_WORKER_URL || process.env.NEXT_PUBLIC_MCP_SERVER_URL!

// MCP Tool Annotations (MCP spec 2025-11-25)
export interface ToolAnnotations {
  readOnlyHint?: boolean    // default: false - true if tool does not modify its environment
  destructiveHint?: boolean // default: true  - true if tool may perform destructive updates
  idempotentHint?: boolean  // default: false - true if repeated calls have no additional effect
  openWorldHint?: boolean   // default: true  - true if tool interacts with external entities
}

// LocalizedText holds multilingual text
// key: BCP47 language code (en-US, ja-JP)
export type LocalizedText = Record<string, string>

// Types matching the DB tools JSONB structure
export interface ToolDef {
  id: string
  name: string
  descriptions: LocalizedText
  annotations: ToolAnnotations
}

/** destructiveHint が true かつ readOnly でないツールは危険表示 */
export function isDangerous(tool: ToolDef): boolean {
  return tool.annotations?.readOnlyHint !== true
    && tool.annotations?.destructiveHint !== false
}

export interface ModuleDef {
  id: string
  name: string
  status: string
  descriptions: LocalizedText
  tools: ToolDef[]
}

// =============================================================================
// Async module fetch with singleton cache
// =============================================================================

let _modules: ModuleDef[] | null = null
let _fetchPromise: Promise<ModuleDef[]> | null = null

// Row type returned by list_modules_with_tools RPC
interface ModuleRow {
  id: string
  name: string
  status: string
  descriptions: LocalizedText | null
  tools: ToolDef[] | null
}

async function fetchModulesFromDB(): Promise<ModuleDef[]> {
  const res = await fetch(`${WORKER_URL}/v1/modules`, {
    method: "GET",
    headers: { "Accept": "application/json" },
  })

  if (!res.ok) {
    throw new Error(`Failed to fetch modules: ${res.status} ${res.statusText}`)
  }

  const data: ModuleRow[] = await res.json()

  return ((data || []) as ModuleRow[]).map((row) => ({
    id: row.id,
    name: moduleDisplayNames[row.id] || row.name,
    status: row.status,
    descriptions: row.descriptions || {},
    tools: (row.tools || []).map((t: ToolDef) => ({
      ...t,
      annotations: t.annotations || {},
    })),
  }))
}

/**
 * Get all modules (async, fetches from DB on first call, cached afterwards)
 */
export async function getModules(): Promise<ModuleDef[]> {
  if (_modules) return _modules
  if (!_fetchPromise) {
    _fetchPromise = fetchModulesFromDB()
  }
  _modules = await _fetchPromise
  return _modules
}

/**
 * Get a specific module by ID (async)
 */
export async function getModule(moduleId: string): Promise<ModuleDef | undefined> {
  const mods = await getModules()
  return mods.find((m) => m.id === moduleId)
}

/**
 * Get tools for a specific module (async)
 */
export async function getModuleTools(moduleId: string): Promise<ToolDef[]> {
  const mod = await getModule(moduleId)
  return mod?.tools || []
}

// Localization helpers
const DEFAULT_LANG = "ja-JP"
const FALLBACK_LANG = "en-US"

/**
 * Get localized text for a given language, falling back to en-US if not found
 */
export function getLocalizedText(
  texts: LocalizedText | undefined,
  lang: string = DEFAULT_LANG
): string {
  if (!texts) return ""
  return texts[lang] ?? texts[FALLBACK_LANG] ?? ""
}

/**
 * Get module description for a specific language
 */
export function getModuleDescription(
  mod: ModuleDef,
  lang: string = DEFAULT_LANG
): string {
  return getLocalizedText(mod.descriptions, lang)
}

/**
 * Get tool description for a specific language
 */
export function getToolDescription(
  tool: ToolDef,
  lang: string = DEFAULT_LANG
): string {
  return getLocalizedText(tool.descriptions, lang)
}

// Display names for modules (DB stores lowercase IDs)
export const moduleDisplayNames: Record<string, string> = {
  notion: "Notion",
  github: "GitHub",
  jira: "Jira",
  confluence: "Confluence",
  supabase: "Supabase",
  airtable: "Airtable",
  google_calendar: "Google Calendar",
  google_tasks: "Google Tasks",
  google_drive: "Google Drive",
  google_docs: "Google Docs",
  google_sheets: "Google Sheets",
  google_apps_script: "Google Apps Script",
  microsoft_todo: "Microsoft To Do",
  postgresql: "PostgreSQL",
  ticktick: "TickTick",
  todoist: "Todoist",
  trello: "Trello",
  asana: "Asana",
  grafana: "Grafana",
  dropbox: "Dropbox",
}

export function getModuleDisplayName(moduleId: string): string {
  return moduleDisplayNames[moduleId] || moduleId
}

// Map module ID to icon name for UI
export const moduleIcons: Record<string, string> = {
  notion: "file-text",
  github: "github",
  jira: "kanban",
  confluence: "book",
  supabase: "database",
  airtable: "table",
  google_calendar: "calendar",
  google_tasks: "list-todo",
  google_drive: "hard-drive",
  google_docs: "file-text",
  google_sheets: "sheet",
  google_apps_script: "code",
  microsoft_todo: "check-square",
  postgresql: "database",
  ticktick: "check-circle-2",
  todoist: "check-circle",
  trello: "trello",
  asana: "briefcase",
  grafana: "activity",
  dropbox: "cloud",
}

export function getModuleIcon(moduleId: string): string {
  return moduleIcons[moduleId] || "box"
}
