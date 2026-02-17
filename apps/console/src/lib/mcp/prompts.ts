"use server"

import { workerFetch } from "@/lib/worker-client"

export interface Prompt {
  id: string
  module_name: string | null
  name: string
  description: string | null
  content: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface UpsertPromptResult {
  success: boolean
  id?: string
  action?: string
  error?: string
}

export interface DeletePromptResult {
  success: boolean
  error?: string
}

export async function listPrompts(moduleName?: string): Promise<Prompt[]> {
  const query = moduleName ? `?module=${encodeURIComponent(moduleName)}` : ""
  return workerFetch<Prompt[]>("GET", `/v1/prompts${query}`)
}

interface GetPromptResponse {
  found: boolean
  id?: string
  module_name?: string | null
  name?: string
  description?: string | null
  content?: string
  enabled?: boolean
  created_at?: string
  updated_at?: string
  error?: string
}

export async function getPrompt(promptId: string): Promise<Prompt | null> {
  const response = await workerFetch<GetPromptResponse>("GET", `/v1/prompts/${promptId}`)

  if (!response || !response.found) {
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
  return workerFetch<UpsertPromptResult>("PUT", "/v1/prompts", {
    name,
    content,
    module_name: moduleName,
    prompt_id: promptId,
    enabled,
    description,
  })
}

export async function deletePrompt(promptId: string): Promise<DeletePromptResult> {
  return workerFetch<DeletePromptResult>("DELETE", `/v1/prompts/${promptId}`)
}
