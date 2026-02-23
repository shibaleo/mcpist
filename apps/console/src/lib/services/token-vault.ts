import { validateToken } from './token-validator'
import {
  listCredentials,
  upsertCredential,
  deleteCredential,
} from './token-vault-actions'

// RPC: list_credentials の戻り値に対応
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

// Server Action 経由で取得
export async function getMyConnections(): Promise<ServiceConnection[]> {
  return listCredentials()
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
export function buildCredentials(params: UpsertTokenParams): Record<string, unknown> {
  let authType = 'api_key'

  const isTrello = params.service === 'trello' && params.username

  if (!isTrello && params.username) {
    authType = 'basic'
  } else if (params.refreshToken) {
    authType = 'oauth2'
  }

  const credentials: Record<string, unknown> = {
    auth_type: authType,
  }

  if (isTrello) {
    credentials.api_key = params.username
    credentials.access_token = params.accessToken
  } else if (authType === 'basic') {
    credentials.username = params.username
    credentials.password = params.accessToken
  } else {
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
      credentials.expires_at = Math.floor(params.expiresAt.getTime() / 1000)
    }
  }

  if (params.metadata) {
    credentials.metadata = params.metadata
  }

  return credentials
}

export async function upsertTokenWithVerification(
  params: UpsertTokenParams,
  onProgress: (progress: ConnectionProgress) => void
): Promise<void> {
  // Step 1: トークンを外部APIで検証（最低1秒表示）
  onProgress({ step: 'validating', message: 'トークンを検証中...' })

  let validationExtra: { email?: string; domain?: string; api_key?: string; base_url?: string } | undefined
  if (params.service === 'trello' && params.username) {
    validationExtra = { api_key: params.username }
  } else if (params.service === 'grafana' && params.metadata?.base_url) {
    validationExtra = { base_url: params.metadata.base_url }
  } else if (params.username && params.metadata?.domain) {
    validationExtra = { email: params.username, domain: params.metadata.domain }
  }

  const validationResult = await withMinDelay(
    validateToken(params.service, params.accessToken, validationExtra),
    1000
  )

  if (!validationResult.valid) {
    throw new TokenVaultError(validationResult.error || 'トークンが無効です')
  }

  // Step 2: Vaultへ登録（Server Action 経由）
  onProgress({ step: 'saving', message: 'トークンを保存中...' })

  const credentials = buildCredentials(params)

  await withMinDelay(
    upsertCredential(params.service, credentials),
    1000
  )

  // Step 3: Vaultから取得して検証（Server Action 経由）
  onProgress({ step: 'verifying', message: '接続を確認中...' })

  const connections = await withMinDelay(
    listCredentials(),
    1000
  )

  const savedConnection = connections.find((c: ServiceConnection) => c.module === params.service)
  if (!savedConnection) {
    throw new TokenVaultError('接続の確認に失敗しました')
  }

  // Step 4: デフォルトツール設定はサーバー側の UpsertCredential で自動作成される

  // Step 5: 完了
  onProgress({ step: 'completed', message: '接続完了' })
}

export async function upsertToken(params: UpsertTokenParams): Promise<void> {
  const credentials = buildCredentials(params)
  await upsertCredential(params.service, credentials)
}

export async function deleteToken(service: string): Promise<boolean> {
  const result = await deleteCredential(service)
  return result.success
}
