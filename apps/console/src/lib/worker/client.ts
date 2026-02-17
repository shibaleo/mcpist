import createClient, { type Middleware } from "openapi-fetch"
import { createClient as createSupabase } from "@/lib/supabase/server"
import type { paths } from "./types"

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

/** Middleware that throws WorkerAPIError on non-OK responses */
const throwOnError: Middleware = {
  async onResponse({ response }) {
    if (!response.ok) {
      const body = await response.clone().text()
      throw new WorkerAPIError(response.status, body)
    }
    return undefined
  },
}

/**
 * Create a typed Worker API client with JWT auth from the current Supabase session.
 * Call this per-request in server components / API routes.
 */
export async function createWorkerClient() {
  const supabase = await createSupabase()
  const { data: { session } } = await supabase.auth.getSession()

  const client = createClient<paths>({
    baseUrl: WORKER_URL,
    headers: session?.access_token
      ? { Authorization: `Bearer ${session.access_token}` }
      : {},
  })
  client.use(throwOnError)
  return client
}

/**
 * Create a typed Worker API client without auth.
 * For public endpoints (e.g. /v1/modules, /v1/oauth/apps/{provider}/credentials).
 */
export function createPublicWorkerClient() {
  const client = createClient<paths>({ baseUrl: WORKER_URL })
  client.use(throwOnError)
  return client
}
