"use server"

import { createWorkerClient } from "@/lib/worker"
import type { ToolSetting, ModuleDescription } from "./tool-settings-types"

export type { ToolSetting, ModuleDescription } from "./tool-settings-types"

/**
 * 現在のユーザーのモジュール設定を一括取得
 */
export async function getModuleConfig(moduleName?: string) {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/me/modules/config")
  if (!data) return []
  // Filter client-side if module specified
  if (moduleName) {
    return data.filter((r) => r.module_name === moduleName)
  }
  return data
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
) {
  const client = await createWorkerClient()
  const { data } = await client.PUT("/v1/me/modules/{name}/tools", {
    params: { path: { name: moduleName } },
    body: {
      enabled_tools: enabledTools,
      disabled_tools: disabledTools,
    },
  })
  return data!
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
    if (r.description != null && !map.has(r.module_name)) {
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
) {
  const client = await createWorkerClient()
  const { data } = await client.PUT("/v1/me/modules/{name}/description", {
    params: { path: { name: moduleName } },
    body: { description },
  })
  return data!
}
