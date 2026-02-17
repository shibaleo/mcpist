"use server"

import { rpc } from "@/lib/worker-client"
import { getModule, isDefaultEnabled } from "@/lib/modules/module-data"

interface ServiceConnection {
  module: string
  created_at: string
  updated_at: string
}

export async function listCredentials(): Promise<ServiceConnection[]> {
  return rpc<ServiceConnection[]>("list_credentials")
}

export async function upsertCredential(
  module: string,
  credentials: Record<string, unknown>
): Promise<{ success: boolean; module: string }> {
  return rpc<{ success: boolean; module: string }>("upsert_credential", {
    p_module: module,
    p_credentials: credentials,
  })
}

export async function deleteCredential(
  module: string
): Promise<{ success: boolean }> {
  return rpc<{ success: boolean }>("delete_credential", {
    p_module: module,
  })
}

export async function saveDefaultToolSettingsAction(
  moduleName: string
): Promise<void> {
  const mod = await getModule(moduleName)
  if (!mod) return

  // Check existing settings
  const existing = await rpc<Array<{ tool_id: string }>>("get_module_config", {
    p_module_name: moduleName,
  })

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

  await rpc("upsert_tool_settings", {
    p_module_name: moduleName,
    p_enabled_tools: enabledTools,
    p_disabled_tools: disabledTools,
  })
}
