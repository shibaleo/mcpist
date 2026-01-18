import { createClient } from './supabase/client'
import { validateToken } from './token-validator'

export interface OAuthConnection {
  id: string
  service: string
  token_type: string
  scope: string | null
  expires_at: string | null
  is_expired: boolean
  created_at: string
  updated_at: string
}

export interface UpsertTokenParams {
  service: string
  accessToken: string
  refreshToken?: string
  tokenType?: string
  scope?: string
  expiresAt?: Date
}

export class TokenVaultError extends Error {
  constructor(message: string, public code?: string) {
    super(message)
    this.name = 'TokenVaultError'
  }
}

export async function getMyConnections(): Promise<OAuthConnection[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('get_my_oauth_connections')

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  return data || []
}

export type ConnectionStep = 'validating' | 'saving' | 'verifying' | 'completed' | 'error'

export interface ConnectionProgress {
  step: ConnectionStep
  message: string
}

// 最低表示時間を確保するためのヘルパー
async function withMinDelay<T>(promiseLike: PromiseLike<T>, minDelayMs: number): Promise<T> {
  const [result] = await Promise.all([
    Promise.resolve(promiseLike),
    new Promise((resolve) => setTimeout(resolve, minDelayMs)),
  ])
  return result
}

export async function upsertTokenWithVerification(
  params: UpsertTokenParams,
  onProgress: (progress: ConnectionProgress) => void
): Promise<string> {
  const supabase = createClient()

  // Step 1: トークンを外部APIで検証（最低1秒表示）
  onProgress({ step: 'validating', message: 'トークンを検証中...' })

  const validationResult = await withMinDelay(
    validateToken(params.service, params.accessToken),
    1000
  )

  console.log('[token-vault] Validation result:', JSON.stringify(validationResult))

  if (!validationResult.valid) {
    console.log('[token-vault] Throwing error:', validationResult.error)
    throw new TokenVaultError(validationResult.error || 'トークンが無効です')
  }

  // Step 2: Vaultへ登録（最低1秒表示）
  onProgress({ step: 'saving', message: 'トークンを保存中...' })

  const { data, error } = await withMinDelay(
    supabase.rpc('upsert_oauth_token', {
      p_service: params.service,
      p_access_token: params.accessToken,
      p_refresh_token: params.refreshToken,
      p_token_type: params.tokenType ?? 'Bearer',
      p_scope: params.scope,
      p_expires_at: params.expiresAt?.toISOString(),
    }),
    1000
  )

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  // Step 2: Vaultから取得して検証（最低1秒表示）
  onProgress({ step: 'verifying', message: '接続を確認中...' })

  const { data: connections, error: verifyError } = await withMinDelay(
    supabase.rpc('get_my_oauth_connections'),
    1000
  )

  if (verifyError) {
    throw new TokenVaultError(verifyError.message, verifyError.code)
  }

  const savedConnection = connections?.find((c: OAuthConnection) => c.service === params.service)
  if (!savedConnection) {
    throw new TokenVaultError('接続の確認に失敗しました')
  }

  // Step 3: 完了
  onProgress({ step: 'completed', message: '接続完了' })

  return data
}

export async function upsertToken(params: UpsertTokenParams): Promise<string> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('upsert_oauth_token', {
    p_service: params.service,
    p_access_token: params.accessToken,
    p_refresh_token: params.refreshToken,
    p_token_type: params.tokenType ?? 'Bearer',
    p_scope: params.scope,
    p_expires_at: params.expiresAt?.toISOString(),
  })

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  return data
}

export async function deleteToken(service: string): Promise<boolean> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('delete_oauth_token', {
    p_service: service,
  })

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  return data
}
