// Module and Service data from Go server definitions
// This file provides type-safe access to tools.json and services.json

import toolsData from "./tools.json"
import servicesData from "./services.json"

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

// Types from services.json (same as get_module_schema output)
export interface ServiceDef {
  id: string
  name: string
  descriptions: LocalizedText
  apiVersion: string
}

export interface ServicesExport {
  services: ServiceDef[]
}

// Export typed data
export const modules: ModuleDef[] = (toolsData as ToolsExport).modules
export const services: ServiceDef[] = (servicesData as ServicesExport).services

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
 * Get service description for a specific language
 */
export function getServiceDescription(
  service: ServiceDef,
  lang: string = DEFAULT_LANG
): string {
  return getLocalizedText(service.descriptions, lang)
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

export function getService(serviceId: string): ServiceDef | undefined {
  return services.find((s) => s.id === serviceId)
}

export function getModuleTools(moduleId: string): ToolDef[] {
  return getModule(moduleId)?.tools || []
}

// Map service ID to icon name for UI
export const serviceIcons: Record<string, string> = {
  notion: "file-text",
  github: "github",
  jira: "kanban",
  confluence: "book",
  supabase: "database",
  airtable: "table",
  google_calendar: "calendar",
  microsoft_todo: "check-square",
}

export function getServiceIcon(serviceId: string): string {
  return serviceIcons[serviceId] || "box"
}
