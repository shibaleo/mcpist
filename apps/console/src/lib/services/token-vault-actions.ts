"use server"

import { createWorkerClient } from "@/lib/worker"

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


