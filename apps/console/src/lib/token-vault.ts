import { createClient } from './supabase/client'
import { validateToken } from './token-validator'

// RPC: list_service_connections の戻り値に対応
export interface ServiceConnection {
  id: string
  service: string
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
  // Basic認証用
  username?: string
  metadata?: Record<string, string>
}

export class TokenVaultError extends Error {
  constructor(message: string, public code?: string) {
    super(message)
    this.name = 'TokenVaultError'
  }
}

// RPC: list_service_connections を呼び出し
export async function getMyConnections(): Promise<ServiceConnection[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_service_connections')

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

// Vault JSON形式で credentials を構築
function buildCredentials(params: UpsertTokenParams): Record<string, unknown> {
  // auth_typeを決定: Basic認証 > OAuth2 > API Key
  let authType = 'api_key'
  if (params.username) {
    authType = 'basic'
  } else if (params.refreshToken) {
    authType = 'oauth2'
  }

  const credentials: Record<string, unknown> = {
    _auth_type: authType,
  }

  if (authType === 'basic') {
    // Basic認証: username(email) + password(token)
    credentials.username = params.username
    credentials.password = params.accessToken
    if (params.metadata) {
      credentials._metadata = params.metadata
    }
  } else {
    // OAuth2 / API Key
    credentials.access_token = params.accessToken
    if (params.refreshToken) {
      credentials.refresh_token = params.refreshToken
    }
    if (params.tokenType) {
      credentials.token_type = params.tokenType
    }
    if (params.scope) {
      credentials.scope = params.scope
    }
    if (params.expiresAt) {
      credentials._expires_at = params.expiresAt.toISOString()
    }
  }

  return credentials
}

export async function upsertTokenWithVerification(
  params: UpsertTokenParams,
  onProgress: (progress: ConnectionProgress) => void
): Promise<void> {
  const supabase = createClient()

  // Step 1: トークンを外部APIで検証（最低1秒表示）
  onProgress({ step: 'validating', message: 'トークンを検証中...' })

  // Basic認証の場合は追加フィールドを渡す
  const validationExtra = params.username && params.metadata?.domain
    ? { email: params.username, domain: params.metadata.domain }
    : undefined

  const validationResult = await withMinDelay(
    validateToken(params.service, params.accessToken, validationExtra),
    1000
  )

  console.log('[token-vault] Validation result:', JSON.stringify(validationResult))

  if (!validationResult.valid) {
    console.log('[token-vault] Throwing error:', validationResult.error)
    throw new TokenVaultError(validationResult.error || 'トークンが無効です')
  }

  // Step 2: Vaultへ登録（最低1秒表示）
  // RPC: upsert_service_token(p_service, p_credentials)
  onProgress({ step: 'saving', message: 'トークンを保存中...' })

  const credentials = buildCredentials(params)

  const { error } = await withMinDelay(
    supabase.rpc('upsert_service_token', {
      p_service: params.service,
      p_credentials: credentials as unknown as Record<string, never>,
    }),
    1000
  )

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  // Step 3: Vaultから取得して検証（最低1秒表示）
  // RPC: list_service_connections
  onProgress({ step: 'verifying', message: '接続を確認中...' })

  const { data: connections, error: verifyError } = await withMinDelay(
    supabase.rpc('list_service_connections'),
    1000
  )

  if (verifyError) {
    throw new TokenVaultError(verifyError.message, verifyError.code)
  }

  const savedConnection = connections?.find((c: ServiceConnection) => c.service === params.service)
  if (!savedConnection) {
    throw new TokenVaultError('接続の確認に失敗しました')
  }

  // Step 4: 完了
  onProgress({ step: 'completed', message: '接続完了' })
}

// RPC: upsert_service_token(p_service, p_credentials)
export async function upsertToken(params: UpsertTokenParams): Promise<void> {
  const supabase = createClient()

  const credentials = buildCredentials(params)

  const { error } = await supabase.rpc('upsert_service_token', {
    p_service: params.service,
    p_credentials: credentials as unknown as Record<string, never>,
  })

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }
}

// RPC: delete_service_token(p_service)
export async function deleteToken(service: string): Promise<boolean> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('delete_service_token', {
    p_service: service,
  })

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  return data?.deleted ?? false
}
