import { createClient } from './supabase/client'

export interface OAuthConsent {
  id: string
  client_id: string
  client_name: string | null
  scopes: string
  granted_at: string
}

export interface OAuthConsentAdmin extends OAuthConsent {
  user_id: string
  user_email: string | null
}

export class OAuthConsentError extends Error {
  constructor(message: string, public code?: string) {
    super(message)
    this.name = 'OAuthConsentError'
  }
}

/**
 * ユーザー自身のOAuthコンセント一覧を取得
 */
export async function listOAuthConsents(): Promise<OAuthConsent[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_oauth_consents')

  if (error) {
    throw new OAuthConsentError(error.message, error.code)
  }

  return data || []
}

/**
 * OAuthコンセントを取り消し
 */
export async function revokeOAuthConsent(consentId: string): Promise<boolean> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('revoke_oauth_consent', {
    p_consent_id: consentId,
  })

  if (error) {
    throw new OAuthConsentError(error.message, error.code)
  }

  return data?.revoked ?? false
}

/**
 * 全ユーザーのOAuthコンセント一覧を取得（管理者用）
 */
export async function listAllOAuthConsents(): Promise<OAuthConsentAdmin[]> {
  const supabase = createClient()

  const { data, error } = await supabase.rpc('list_all_oauth_consents')

  if (error) {
    throw new OAuthConsentError(error.message, error.code)
  }

  return data || []
}
