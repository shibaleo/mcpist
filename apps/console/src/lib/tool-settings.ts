// Tool Settings API - Supabase RPC wrapper
// ユーザーのツール設定をデータベースから取得・保存

import { createClient } from "@/lib/supabase/client"

// ツール設定の型
export interface ToolSetting {
  module_name: string
  tool_name: string
  enabled: boolean
}

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
    p_module_name: moduleName ?? null,
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
 * @returns { moduleName: { toolName: enabled } }
 */
export function toToolSettingsMap(settings: ToolSetting[]): ToolSettingsMap {
  const map: ToolSettingsMap = {}

  for (const setting of settings) {
    if (!map[setting.module_name]) {
      map[setting.module_name] = {}
    }
    map[setting.module_name][setting.tool_name] = setting.enabled
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
  if (data?.error) {
    throw new ToolSettingsError(data.error)
  }

  return data
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
