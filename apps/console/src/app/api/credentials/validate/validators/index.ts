/**
 * Token Validators Registry
 *
 * Validators are only needed for manually entered tokens (API Key / Basic auth).
 * OAuth 2.0 modules don't need validation as tokens are issued by the auth provider.
 */

import { ValidationParams, ValidationResult, ValidatorFunction } from './types'
import { validateNotionToken } from './notion'
import { validateGitHubToken } from './github'
import { validateSupabaseToken } from './supabase'
import { validateJiraToken } from './jira'
import { validateConfluenceToken } from './confluence'
import { validateTrelloToken } from './trello'
import { validateGrafanaToken } from './grafana'

export type { ValidationResult, ValidationParams, ValidatorFunction }

export const validators: Record<string, ValidatorFunction> = {
  notion: validateNotionToken,
  github: validateGitHubToken,
  supabase: validateSupabaseToken,
  jira: validateJiraToken,
  confluence: validateConfluenceToken,
  trello: validateTrelloToken,
  grafana: validateGrafanaToken,
}

export const requiredParams: Record<string, string[]> = {
  jira: ['email', 'domain'],
  confluence: ['email', 'domain'],
  trello: ['api_key'],
  grafana: ['base_url'],
}

/**
 * Get validator for a service
 * Returns undefined for OAuth 2.0 modules (validation skipped)
 */
export function getValidator(service: string): ValidatorFunction | undefined {
  return validators[service]
}

/**
 * Get required params for a service
 */
export function getRequiredParams(service: string): string[] {
  return requiredParams[service] || []
}
