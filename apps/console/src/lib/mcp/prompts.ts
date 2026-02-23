"use server"

import { createWorkerClient } from "@/lib/worker"
import type { components } from "@/lib/worker"

export type Prompt = components["schemas"]["Prompt"]
export type UpsertPromptResult = components["schemas"]["UpsertPromptResult"]
export type DeletePromptResult = components["schemas"]["DeletePromptResult"]

export async function listPrompts(moduleName?: string): Promise<Prompt[]> {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/me/prompts", {
    params: { query: { module: moduleName } },
  })
  return data!
}

export async function getPrompt(promptId: string): Promise<Prompt | null> {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/me/prompts/{id}", {
    params: { path: { id: promptId } },
  })
  const response = data!

  if (!response.found) {
    return null
  }

  return {
    id: response.id!,
    module_name: response.module_name ?? null,
    name: response.name!,
    description: response.description ?? null,
    content: response.content!,
    enabled: response.enabled!,
    created_at: response.created_at!,
    updated_at: response.updated_at!,
  }
}

export async function upsertPrompt(
  name: string,
  content: string,
  moduleName?: string,
  promptId?: string,
  enabled: boolean = true,
  description?: string
): Promise<UpsertPromptResult> {
  const client = await createWorkerClient()

  if (promptId) {
    // Update existing prompt
    const { data } = await client.PUT("/v1/me/prompts/{id}", {
      params: { path: { id: promptId } },
      body: {
        name,
        content,
        module_name: moduleName,
        enabled,
        description,
      },
    })
    return data!
  }

  // Create new prompt
  const { data } = await client.POST("/v1/me/prompts", {
    body: {
      name,
      content,
      module_name: moduleName,
      enabled,
      description,
    },
  })
  return data!
}

export async function deletePrompt(promptId: string): Promise<DeletePromptResult> {
  const client = await createWorkerClient()
  const { data } = await client.DELETE("/v1/me/prompts/{id}", {
    params: { path: { id: promptId } },
  })
  return data!
}
