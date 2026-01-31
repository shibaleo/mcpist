// Tool Settings API - Supabase RPC wrapper
// ユーザーのツール設定をデータベースから取得・保存

import type { SupabaseClient } from "@supabase/supabase-js"
import { createClient } from "@/lib/supabase/client"
import { getModule, isDefaultEnabled } from "./module-data"

// ツール設定の型
export interface ToolSetting {
  module_name: string
  tool_id: string  // Format: {module}:{tool_name} (e.g., "notion:search")
  enabled: boolean
}

// モジュール説明の型
export interface ModuleDescription {
  module_name: string
  description: string
}

// モジュールごとの説明マップ
export type ModuleDescriptionsMap = Record<string, string>

// モジュールごとのツール設定マップ
export type ToolSettingsMap = Record<string, Record<string, boolean>>

// エラークラス
export class ToolSettingsError extends Error {
  constructor(
    message: string,
    public code?: string
  ) {
    super(message)
    this.name = "ToolSettingsError"
  }
}

/**
 * 現在のユーザーのツール設定を取得
 * @param moduleName 特定モジュールのみ取得する場合はモジュール名を指定
 */
export async function getMyToolSettings(moduleName?: string): Promise<ToolSetting[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc("get_my_tool_settings", {
    p_module_name: moduleName,
  })

  if (error) {
    console.error("Failed to get tool settings:", error)
    throw new ToolSettingsError(error.message, error.code)
  }

  return data || []
}

/**
 * ツール設定をマップ形式に変換
 * @param settings ツール設定配列
 * @returns { moduleName: { toolId: enabled } }
 */
export function toToolSettingsMap(settings: ToolSetting[]): ToolSettingsMap {
  const map: ToolSettingsMap = {}

  for (const setting of settings) {
    if (!map[setting.module_name]) {
      map[setting.module_name] = {}
    }
    map[setting.module_name][setting.tool_id] = setting.enabled
  }

  return map
}

/**
 * 現在のユーザーのツール設定を更新
 * @param moduleName モジュール名
 * @param enabledTools 有効にするツール名の配列
 * @param disabledTools 無効にするツール名の配列
 */
export async function upsertMyToolSettings(
  moduleName: string,
  enabledTools: string[],
  disabledTools: string[]
): Promise<{ success: boolean; module: string; enabled_count: number; disabled_count: number }> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc("upsert_my_tool_settings", {
    p_module_name: moduleName,
    p_enabled_tools: enabledTools,
    p_disabled_tools: disabledTools,
  })

  if (error) {
    console.error("Failed to upsert tool settings:", error)
    throw new ToolSettingsError(error.message, error.code)
  }

  // RPCがエラーオブジェクトを返した場合
  const result = data as Record<string, unknown> | null
  if (result?.error) {
    throw new ToolSettingsError(String(result.error))
  }

  return result as { success: boolean; module: string; enabled_count: number; disabled_count: number }
}

/**
 * モジュールのツール設定を一括保存
 * 現在の状態と比較して変更があったもののみ保存
 * @param moduleName モジュール名
 * @param toolStates { toolId: enabled } の形式
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
 * @param supabase Supabaseクライアント（サーバー/クライアント両対応）
 * @param moduleName モジュール名
 */
export async function saveDefaultToolSettings(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  supabase: SupabaseClient<any, any, any>,
  moduleName: string
): Promise<void> {
  const mod = getModule(moduleName)
  if (!mod) {
    console.warn(`[tool-settings] Module not found: ${moduleName}`)
    return
  }

  // 既にツール設定が存在する場合はスキップ（ユーザーのカスタム設定を保持）
  const { data: existing, error: fetchError } = await supabase.rpc("get_my_tool_settings", {
    p_module_name: moduleName,
  })

  if (fetchError) {
    console.error(`[tool-settings] Failed to check existing settings for ${moduleName}:`, fetchError)
    return
  }

  if (existing && existing.length > 0) {
    return
  }

  const enabledTools: string[] = []
  const disabledTools: string[] = []

  for (const tool of mod.tools) {
    if (isDefaultEnabled(tool)) {
      enabledTools.push(tool.id)
    } else {
      disabledTools.push(tool.id)
    }
  }

  const { error } = await supabase.rpc("upsert_my_tool_settings", {
    p_module_name: moduleName,
    p_enabled_tools: enabledTools,
    p_disabled_tools: disabledTools,
  })

  if (error) {
    console.error(`[tool-settings] Failed to save default settings for ${moduleName}:`, error)
    // エラーは投げない（トークン保存は成功しているので）
  }
}

// =============================================================================
// Module Description API
// =============================================================================

/**
 * 現在のユーザーのモジュール説明を全て取得
 */
export async function getMyModuleDescriptions(): Promise<ModuleDescription[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc("get_my_module_descriptions")

  if (error) {
    console.error("Failed to get module descriptions:", error)
    throw new ToolSettingsError(error.message, error.code)
  }

  return data || []
}

/**
 * モジュール説明をマップ形式に変換
 * @param descriptions モジュール説明配列
 * @returns { moduleName: description }
 */
export function toModuleDescriptionsMap(descriptions: ModuleDescription[]): ModuleDescriptionsMap {
  const map: ModuleDescriptionsMap = {}

  for (const desc of descriptions) {
    map[desc.module_name] = desc.description
  }

  return map
}

/**
 * モジュールの説明を更新
 * @param moduleName モジュール名
 * @param description 説明（空文字で削除）
 */
export async function updateModuleDescription(
  moduleName: string,
  description: string
): Promise<{ success: boolean }> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc("upsert_my_module_description", {
    p_module_name: moduleName,
    p_description: description,
  })

  if (error) {
    console.error("Failed to update module description:", error)
    throw new ToolSettingsError(error.message, error.code)
  }

  const result = data as Record<string, unknown> | null
  if (result?.error) {
    throw new ToolSettingsError(String(result.error))
  }

  return { success: true }
}
