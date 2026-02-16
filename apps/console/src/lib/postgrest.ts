const POSTGREST_URL = process.env.POSTGREST_URL!
const POSTGREST_API_KEY = process.env.POSTGREST_API_KEY!

const headers = {
  "Content-Type": "application/json",
  apikey: POSTGREST_API_KEY,
  Authorization: `Bearer ${POSTGREST_API_KEY}`,
}

export class PostgRESTError extends Error {
  constructor(
    public status: number,
    public body: string
  ) {
    super(`PostgREST error ${status}: ${body}`)
    this.name = "PostgRESTError"
  }
}

export async function rpc<T>(
  name: string,
  params: Record<string, unknown> = {}
): Promise<T> {
  const res = await fetch(`${POSTGREST_URL}/rpc/${name}`, {
    method: "POST",
    headers,
    body: JSON.stringify(params),
  })
  if (!res.ok) throw new PostgRESTError(res.status, await res.text())
  return res.json()
}
