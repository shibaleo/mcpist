"use server"

import { createWorkerClient } from "@/lib/worker"
import type { components } from "@/lib/worker"

export type OAuthConsent = components["schemas"]["OAuthConsent"]
export type OAuthConsentAdmin = components["schemas"]["OAuthConsentAdmin"]

/**
 * ユーザー自身のOAuthコンセント一覧を取得
 */
export async function listOAuthConsents(): Promise<OAuthConsent[]> {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/me/oauth/consents")
  return data ?? []
}

/**
 * OAuthコンセントを取り消し
 */
export async function revokeOAuthConsent(consentId: string): Promise<boolean> {
  const client = await createWorkerClient()
  const { data } = await client.DELETE("/v1/me/oauth/consents/{id}", {
    params: { path: { id: consentId } },
  })
  return data?.revoked ?? false
}

/**
 * 全ユーザーのOAuthコンセント一覧を取得（管理者用）
 */
export async function listAllOAuthConsents(): Promise<OAuthConsentAdmin[]> {
  const client = await createWorkerClient()
  const { data } = await client.GET("/v1/admin/oauth/consents")
  return data ?? []
}
