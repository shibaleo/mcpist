// Module data from Go server definitions
// This file provides type-safe access to tools.json

import toolsData from "./tools.json"

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

// Types from tools.json
export interface ToolDef {
  id: string
  name: string
  descriptions: LocalizedText
  annotations: ToolAnnotations
}

/** readOnlyHint: true のツールはデフォルト有効 */
export function isDefaultEnabled(tool: ToolDef): boolean {
  return tool.annotations.readOnlyHint === true
}

/** destructiveHint が true かつ readOnly でないツールは危険表示 */
export function isDangerous(tool: ToolDef): boolean {
  return tool.annotations.readOnlyHint !== true
    && tool.annotations.destructiveHint !== false
}

export interface ModuleDef {
  id: string
  name: string
  descriptions: LocalizedText
  apiVersion: string
  tools: ToolDef[]
}

export interface ToolsExport {
  modules: ModuleDef[]
}

// Export typed data
export const modules: ModuleDef[] = (toolsData as ToolsExport).modules

// Localization helpers
const DEFAULT_LANG = "ja-JP"
const FALLBACK_LANG = "en-US"

/**
 * Get localized text for a given language, falling back to en-US if not found
 */
export function getLocalizedText(
  texts: LocalizedText,
  lang: string = DEFAULT_LANG
): string {
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

// Helper functions
export function getModule(moduleId: string): ModuleDef | undefined {
  return modules.find((m) => m.id === moduleId)
}

export function getModuleTools(moduleId: string): ToolDef[] {
  return getModule(moduleId)?.tools || []
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
  microsoft_todo: "check-square",
  todoist: "check-circle",
  trello: "trello",
}

export function getModuleIcon(moduleId: string): string {
  return moduleIcons[moduleId] || "box"
}
