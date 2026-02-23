"use server"

import { createWorkerClient } from "@/lib/worker"
import type { components } from "@/lib/worker"

export type ApiKey = components["schemas"]["ApiKey"]
export type GenerateApiKeyResult = components["schemas"]["GenerateApiKeyResult"]

export async function listApiKeys() {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/me/apikeys")
  return data!
}

export async function generateApiKey(
  name: string,
  expiresInDays: number | null = null
) {
  const client = await createWorkerClient()
  const { data } = await client.POST("/v1/me/apikeys", {
    body: expiresInDays === null
      ? { display_name: name, no_expiry: true }
      : { display_name: name, expires_at: new Date(Date.now() + expiresInDays * 24 * 60 * 60 * 1000).toISOString() },
  })
  return data!
}

export async function revokeApiKey(keyId: string) {
  const client = await createWorkerClient()
  const { data } = await client.DELETE("/v1/me/apikeys/{id}", {
    params: { path: { id: keyId } },
  })
  return data!
}
