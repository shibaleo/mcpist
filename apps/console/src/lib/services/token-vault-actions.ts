"use server"

import { workerFetch } from "@/lib/worker-client"
import { getModule, isDefaultEnabled } from "@/lib/modules/module-data"

interface ServiceConnection {
  module: string
  created_at: string
  updated_at: string
}

export async function listCredentials(): Promise<ServiceConnection[]> {
  return workerFetch<ServiceConnection[]>("GET", "/v1/credentials")
}

export async function upsertCredential(
  module: string,
  credentials: Record<string, unknown>
): Promise<{ success: boolean; module: string }> {
  return workerFetch<{ success: boolean; module: string }>("PUT", "/v1/credentials", {
    module,
    credentials,
  })
}

export async function deleteCredential(
  module: string
): Promise<{ success: boolean }> {
  return workerFetch<{ success: boolean }>("DELETE", `/v1/credentials/${encodeURIComponent(module)}`)
}

export async function saveDefaultToolSettingsAction(
  moduleName: string
): Promise<void> {
  const mod = await getModule(moduleName)
  if (!mod) return

  // Check existing settings
  const existing = await workerFetch<Array<{ tool_id: string }>>(
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
