/**
 * Token Validator Types
 */

export interface ValidationResult {
  valid: boolean
  error?: string
  details?: Record<string, unknown>
}

export interface ValidationParams {
  token: string
  email?: string
  domain?: string
  api_key?: string
  base_url?: string
}

export type ValidatorFunction = (params: ValidationParams) => Promise<ValidationResult>
