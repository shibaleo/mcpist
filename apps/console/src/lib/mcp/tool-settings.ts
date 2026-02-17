"use server"

import { workerFetch } from "@/lib/worker-client"
import { getModule, isDefaultEnabled } from "@/lib/modules/module-data"
import type { ToolSetting, ModuleDescription } from "./tool-settings-types"

export type { ToolSetting, ModuleDescription } from "./tool-settings-types"

// モジュール設定の取得結果型 (get_module_config)
interface ModuleConfigRow {
  module_name: string
  description: string | null
  tool_id: string
  enabled: boolean
}

/**
 * 現在のユーザーのモジュール設定を一括取得
 */
export async function getModuleConfig(moduleName?: string): Promise<ModuleConfigRow[]> {
  const query = moduleName ? `?module=${encodeURIComponent(moduleName)}` : ""
  return workerFetch<ModuleConfigRow[]>("GET", `/v1/modules/config${query}`)
}

/**
 * 現在のユーザーのツール設定を取得
 */
export async function getMyToolSettings(moduleName?: string): Promise<ToolSetting[]> {
  const rows = await getModuleConfig(moduleName)
  return rows.map((r) => ({
    module_name: r.module_name,
    tool_id: r.tool_id,
    enabled: r.enabled,
  }))
}

/**
 * 現在のユーザーのツール設定を更新
 */
export async function upsertMyToolSettings(
  moduleName: string,
  enabledTools: string[],
  disabledTools: string[]
): Promise<{ success: boolean; enabled_count: number; disabled_count: number }> {
  return workerFetch<{ success: boolean; enabled_count: number; disabled_count: number }>(
    "PUT",
    `/v1/modules/${encodeURIComponent(moduleName)}/tools`,
    {
      enabled_tools: enabledTools,
      disabled_tools: disabledTools,
    }
  )
}

/**
 * モジュールのツール設定を一括保存
 */
export async function saveModuleToolSettings(
  moduleName: string,
  toolStates: Record<string, boolean>
): Promise<void> {
  const enabledTools: string[] = []
  const disabledTools: string[] = []

  for (const [toolId, enabled] of Object.entries(toolStates)) {
    if (enabled) {
      enabledTools.push(toolId)
    } else {
      disabledTools.push(toolId)
    }
  }

  await upsertMyToolSettings(moduleName, enabledTools, disabledTools)
}

/**
 * モジュールのデフォルトツール設定を保存
 * サービス接続時に呼び出す（トークン保存後）
 * @deprecated Use saveDefaultToolSettingsAction from token-vault-actions.ts instead
 */
export async function saveDefaultToolSettings(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  _supabase: any,
  moduleName: string
): Promise<void> {
  const mod = await getModule(moduleName)
  if (!mod) return

  const existing = await workerFetch<ModuleConfigRow[]>(
    "GET",
    `/v1/modules/config?module=${encodeURIComponent(moduleName)}`
  )

  if (existing && existing.length > 0) return

  const enabledTools: string[] = []
  const disabledTools: string[] = []

  for (const tool of mod.tools) {
    if (isDefaultEnabled(tool)) {
      enabledTools.push(tool.id)
    } else {
      disabledTools.push(tool.id)
    }
  }

  await workerFetch("PUT", `/v1/modules/${encodeURIComponent(moduleName)}/tools`, {
    enabled_tools: enabledTools,
    disabled_tools: disabledTools,
  })
}

// =============================================================================
// Module Description API
// =============================================================================

/**
 * 現在のユーザーのモジュール説明を全て取得
 */
export async function getMyModuleDescriptions(): Promise<ModuleDescription[]> {
  const rows = await getModuleConfig()
  const map = new Map<string, string>()
  for (const r of rows) {
    if (r.description && !map.has(r.module_name)) {
      map.set(r.module_name, r.description)
    }
  }
  return Array.from(map.entries()).map(([module_name, description]) => ({
    module_name,
    description,
  }))
}

/**
 * モジュールの説明を更新
 */
export async function updateModuleDescription(
  moduleName: string,
  description: string
): Promise<{ success: boolean }> {
  return workerFetch<{ success: boolean }>(
    "PUT",
    `/v1/modules/${encodeURIComponent(moduleName)}/description`,
    { description }
  )
}
