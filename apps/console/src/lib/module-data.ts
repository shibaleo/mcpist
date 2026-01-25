// Module and Service data from Go server definitions
// This file provides type-safe access to tools.json and services.json

import toolsData from "./tools.json"
import servicesData from "./services.json"

// Types from tools.json
export interface ToolDef {
  id: string
  name: string
  description: string
  dangerous: boolean
  defaultEnabled: boolean
}

export interface ModuleDef {
  id: string
  name: string
  description: string
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
  description: string
  apiVersion: string
}

export interface ServicesExport {
  services: ServiceDef[]
}

// Export typed data
export const modules: ModuleDef[] = (toolsData as ToolsExport).modules
export const services: ServiceDef[] = (servicesData as ServicesExport).services

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
  "google-calendar": "calendar",
  "microsoft-todo": "check-square",
}

export function getServiceIcon(serviceId: string): string {
  return serviceIcons[serviceId] || "box"
}
