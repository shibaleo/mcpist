/**
 * Direct PostgREST access with service_role key.
 * Used only by Stripe webhooks where no user JWT is available.
 *
 * For all other Worker API calls, use createWorkerClient() from @/lib/worker.
 */

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
