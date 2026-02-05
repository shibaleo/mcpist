import { createClient } from './supabase/client'
import { validateToken } from './token-validator'
import { saveDefaultToolSettings } from './tool-settings'

// RPC: list_my_credentials の戻り値に対応
export interface ServiceConnection {
  module: string
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

// RPC: list_my_credentials を呼び出し
export async function getMyConnections(): Promise<ServiceConnection[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_my_credentials')

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
  // auth_typeを決定
  // - Trello: api_key (api_key + access_token)
  // - Basic認証 (Jira/Confluence): basic (username + password)
  // - OAuth2: oauth2 (refreshTokenあり)
  // - その他: api_key (access_token only)
  let authType = 'api_key'

  // Trello は api_key タイプ（username に api_key が入っている特殊ケース）
  const isTrello = params.service === 'trello' && params.username

  if (!isTrello && params.username) {
    // Basic認証: Jira, Confluence
    authType = 'basic'
  } else if (params.refreshToken) {
    authType = 'oauth2'
  }

  const credentials: Record<string, unknown> = {
    auth_type: authType,
  }

  if (isTrello) {
    // Trello: api_key + access_token
    credentials.api_key = params.username  // api_key は username パラメータで渡される
    credentials.access_token = params.accessToken
  } else if (authType === 'basic') {
    // Basic認証: username(email) + password(token)
    credentials.username = params.username
    credentials.password = params.accessToken
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
      credentials.expires_at = Math.floor(params.expiresAt.getTime() / 1000) // Unix timestamp
    }
  }

  // metadata は認証方式に関わらず保存
  if (params.metadata) {
    credentials.metadata = params.metadata
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

  // 認証方式に応じて追加フィールドを渡す
  let validationExtra: { email?: string; domain?: string; api_key?: string; base_url?: string } | undefined
  if (params.service === 'trello' && params.username) {
    // Trello: username に api_key が入っている
    validationExtra = { api_key: params.username }
  } else if (params.service === 'grafana' && params.metadata?.base_url) {
    // Grafana: base_url が必要
    validationExtra = { base_url: params.metadata.base_url }
  } else if (params.username && params.metadata?.domain) {
    // Basic認証: email + domain
    validationExtra = { email: params.username, domain: params.metadata.domain }
  }

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
  // RPC: upsert_my_credential(p_module, p_credentials)
  onProgress({ step: 'saving', message: 'トークンを保存中...' })

  const credentials = buildCredentials(params)

  const { error } = await withMinDelay(
    supabase.rpc('upsert_my_credential', {
      p_module: params.service,
      p_credentials: credentials as unknown as Record<string, never>,
    }),
    1000
  )

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  // Step 3: Vaultから取得して検証（最低1秒表示）
  // RPC: list_my_credentials
  onProgress({ step: 'verifying', message: '接続を確認中...' })

  const { data: connections, error: verifyError } = await withMinDelay(
    supabase.rpc('list_my_credentials'),
    1000
  )

  if (verifyError) {
    throw new TokenVaultError(verifyError.message, verifyError.code)
  }

  const savedConnection = connections?.find((c: ServiceConnection) => c.module === params.service)
  if (!savedConnection) {
    throw new TokenVaultError('接続の確認に失敗しました')
  }

  // Step 4: デフォルトツール設定を保存
  await saveDefaultToolSettings(supabase, params.service)

  // Step 5: 完了
  onProgress({ step: 'completed', message: '接続完了' })
}

// RPC: upsert_my_credential(p_module, p_credentials)
export async function upsertToken(params: UpsertTokenParams): Promise<void> {
  const supabase = createClient()

  const credentials = buildCredentials(params)

  const { error } = await supabase.rpc('upsert_my_credential', {
    p_module: params.service,
    p_credentials: credentials as unknown as Record<string, never>,
  })

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }
}

// RPC: delete_my_credential(p_module)
export async function deleteToken(service: string): Promise<boolean> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('delete_my_credential', {
    p_module: service,
  })

  if (error) {
    throw new TokenVaultError(error.message, error.code)
  }

  return (data as { deleted?: boolean } | null)?.deleted ?? false
}
