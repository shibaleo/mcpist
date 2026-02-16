"use server"

import { rpc } from "@/lib/postgrest"
import { getUserId } from "@/lib/auth"

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
  const userId = await getUserId()
  return rpc<Prompt[]>("list_prompts", {
    p_user_id: userId,
    p_module_name: moduleName,
  })
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
  const userId = await getUserId()
  const response = await rpc<GetPromptResponse>("get_prompt", {
    p_user_id: userId,
    p_prompt_id: promptId,
  })

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
  const userId = await getUserId()
  return rpc<UpsertPromptResult>("upsert_prompt", {
    p_user_id: userId,
    p_name: name,
    p_content: content,
    p_module_name: moduleName,
    p_prompt_id: promptId,
    p_enabled: enabled,
    p_description: description,
  })
}

export async function deletePrompt(promptId: string): Promise<DeletePromptResult> {
  const userId = await getUserId()
  return rpc<DeletePromptResult>("delete_prompt", {
    p_user_id: userId,
    p_prompt_id: promptId,
  })
}
