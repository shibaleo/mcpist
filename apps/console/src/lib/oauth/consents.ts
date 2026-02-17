"use server"

import { workerFetch } from "@/lib/worker-client"

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
  const data = await workerFetch<OAuthConsent[]>("GET", "/v1/oauth/consents")
  return data || []
}

/**
 * OAuthコンセントを取り消し
 */
export async function revokeOAuthConsent(consentId: string): Promise<boolean> {
  const data = await workerFetch<{ revoked?: boolean }>("DELETE", `/v1/oauth/consents/${consentId}`)
  return data?.revoked ?? false
}

/**
 * 全ユーザーのOAuthコンセント一覧を取得（管理者用）
 */
export async function listAllOAuthConsents(): Promise<OAuthConsentAdmin[]> {
  const data = await workerFetch<OAuthConsentAdmin[]>("GET", "/v1/admin/oauth/consents")
  return data || []
}
