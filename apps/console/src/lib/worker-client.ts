import { createClient } from "@/lib/supabase/server"

const WORKER_URL = process.env.NEXT_PUBLIC_WORKER_URL || process.env.NEXT_PUBLIC_MCP_SERVER_URL!

export class PostgRESTError extends Error {
  constructor(
    public status: number,
    public body: string
  ) {
    super(`PostgREST error ${status}: ${body}`)
    this.name = "PostgRESTError"
  }
}

/**
 * Worker 経由で RPC を呼び出す（認証付き）。
 * session cookie から JWT を取得し Authorization ヘッダーに付与。
 * Worker が p_user_id を自動注入するため、呼び出し側で渡す必要なし。
 */
export async function rpc<T>(
  name: string,
  params: Record<string, unknown> = {}
): Promise<T> {
  const supabase = await createClient()
  const { data: { session } } = await supabase.auth.getSession()

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  }
  if (session?.access_token) {
    headers["Authorization"] = `Bearer ${session.access_token}`
  }

  const res = await fetch(`${WORKER_URL}/v1/rpc/${name}`, {
    method: "POST",
    headers,
    body: JSON.stringify(params),
  })
  if (!res.ok) throw new PostgRESTError(res.status, await res.text())
  return res.json()
}

/**
 * PostgREST を直接呼び出す（service_role 認証）。
 * Stripe webhook など JWT が存在しないコンテキストで使用。
 */
export async function rpcDirect<T>(
  name: string,
  params: Record<string, unknown> = {}
): Promise<T> {
  const url = process.env.POSTGREST_URL!
  const apiKey = process.env.POSTGREST_API_KEY!

  const res = await fetch(`${url}/rpc/${name}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      apikey: apiKey,
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify(params),
  })
  if (!res.ok) throw new PostgRESTError(res.status, await res.text())
  return res.json()
}
