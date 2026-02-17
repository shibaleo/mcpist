import { createClient } from "@/lib/supabase/server"

const WORKER_URL = process.env.NEXT_PUBLIC_WORKER_URL || process.env.NEXT_PUBLIC_MCP_SERVER_URL!

export class WorkerAPIError extends Error {
  constructor(
    public status: number,
    public body: string
  ) {
    super(`Worker API error ${status}: ${body}`)
    this.name = "WorkerAPIError"
  }
}

/** @deprecated Use WorkerAPIError instead */
export const PostgRESTError = WorkerAPIError

type HttpMethod = "GET" | "POST" | "PUT" | "DELETE"

/**
 * Worker RESTful API を呼び出す（JWT 認証付き）。
 * session cookie から JWT を取得し Authorization ヘッダーに付与。
 */
export async function workerFetch<T>(
  method: HttpMethod,
  path: string,
  body?: Record<string, unknown>
): Promise<T> {
  const supabase = await createClient()
  const { data: { session } } = await supabase.auth.getSession()

  const headers: Record<string, string> = {}
  if (session?.access_token) {
    headers["Authorization"] = `Bearer ${session.access_token}`
  }
  if (body) {
    headers["Content-Type"] = "application/json"
  }

  const res = await fetch(`${WORKER_URL}${path}`, {
    method,
    headers,
    ...(body && { body: JSON.stringify(body) }),
  })
  if (!res.ok) throw new WorkerAPIError(res.status, await res.text())
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
  if (!res.ok) throw new WorkerAPIError(res.status, await res.text())
  return res.json()
}
