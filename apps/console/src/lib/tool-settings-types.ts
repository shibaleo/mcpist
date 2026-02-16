// ツール設定の型
export interface ToolSetting {
  module_name: string
  tool_id: string
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

/**
 * ツール設定をマップ形式に変換
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
 * モジュール説明をマップ形式に変換
 */
export function toModuleDescriptionsMap(descriptions: ModuleDescription[]): ModuleDescriptionsMap {
  const map: ModuleDescriptionsMap = {}
  for (const desc of descriptions) {
    map[desc.module_name] = desc.description
  }
  return map
}
