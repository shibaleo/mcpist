"use server"

import { createWorkerClient } from "@/lib/worker"
import { getModule, isDefaultEnabled } from "@/lib/modules/module-data"

export async function listCredentials() {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/me/credentials")
  return data!
}

export async function upsertCredential(
  module: string,
  credentials: Record<string, unknown>
) {
  const client = await createWorkerClient()
  const { data } = await client.PUT("/v1/me/credentials/{module}", {
    params: { path: { module } },
    body: { credentials },
  })
  return data!
}

export async function deleteCredential(
  module: string
) {
  const client = await createWorkerClient()
  const { data } = await client.DELETE("/v1/me/credentials/{module}", {
    params: { path: { module } },
  })
  return data!
}

export async function saveDefaultToolSettingsAction(
  moduleName: string
): Promise<void> {
  const mod = await getModule(moduleName)
  if (!mod) return

  // Check existing settings
  const client = await createWorkerClient()
  const { data: existing } = await client.GET("/v1/me/modules/config", {
    params: { query: { module: moduleName } },
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

  await client.PUT("/v1/me/modules/{name}/tools", {
    params: { path: { name: moduleName } },
    body: {
      enabled_tools: enabledTools,
      disabled_tools: disabledTools,
    },
  })
}
