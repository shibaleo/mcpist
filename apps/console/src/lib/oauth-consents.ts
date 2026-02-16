"use server"

import { rpc } from "@/lib/postgrest"
import { getUserId } from "@/lib/auth"

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

/**
 * ユーザー自身のOAuthコンセント一覧を取得
 */
export async function listOAuthConsents(): Promise<OAuthConsent[]> {
  const userId = await getUserId()
  const data = await rpc<OAuthConsent[]>("list_oauth_consents", { p_user_id: userId })
  return data || []
}

/**
 * OAuthコンセントを取り消し
 */
export async function revokeOAuthConsent(consentId: string): Promise<boolean> {
  const userId = await getUserId()
  const data = await rpc<{ revoked?: boolean }>("revoke_oauth_consent", {
    p_user_id: userId,
    p_consent_id: consentId,
  })
  return data?.revoked ?? false
}

/**
 * 全ユーザーのOAuthコンセント一覧を取得（管理者用）
 */
export async function listAllOAuthConsents(): Promise<OAuthConsentAdmin[]> {
  const data = await rpc<OAuthConsentAdmin[]>("list_all_oauth_consents")
  return data || []
}
