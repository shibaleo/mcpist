import { createClient } from './supabase/client'

export interface Prompt {
  id: string
  module_name: string | null
  name: string
  description: string | null  // Short description for prompts/list (MCP spec)
  content: string             // Full content for prompts/get
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

/**
 * Get the list of prompts for the current user
 */
export async function listPrompts(moduleName?: string): Promise<Prompt[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_my_prompts', {
    p_module_name: moduleName
  })

  if (error) {
    console.error('Failed to fetch prompts:', error)
    return []
  }

  return (data || []) as Prompt[]
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

/**
 * Get a single prompt by ID
 */
export async function getPrompt(promptId: string): Promise<Prompt | null> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('get_my_prompt', {
    p_prompt_id: promptId
  })

  if (error) {
    console.error('Failed to fetch prompt:', error)
    return null
  }

  const response = data as unknown as GetPromptResponse
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
    updated_at: response.updated_at!
  }
}

/**
 * Create or update a prompt
 */
export async function upsertPrompt(
  name: string,
  content: string,
  moduleName?: string,
  promptId?: string,
  enabled: boolean = true,
  description?: string
): Promise<UpsertPromptResult> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('upsert_my_prompt', {
    p_name: name,
    p_content: content,
    p_module_name: moduleName,
    p_prompt_id: promptId,
    p_enabled: enabled,
    p_description: description
  })

  if (error) {
    console.error('Failed to upsert prompt:', error)
    return { success: false, error: error.message }
  }

  return data as unknown as UpsertPromptResult
}

/**
 * Delete a prompt by ID
 */
export async function deletePrompt(promptId: string): Promise<DeletePromptResult> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('delete_my_prompt', {
    p_prompt_id: promptId
  })

  if (error) {
    console.error('Failed to delete prompt:', error)
    return { success: false, error: error.message }
  }

  return data as unknown as DeletePromptResult
}
