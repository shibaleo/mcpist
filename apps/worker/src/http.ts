/**
 * プロキシレスポンスに CORS ヘッダーを付与するユーティリティ。
 * CORS プリフライト・セキュリティヘッダーは Hono ミドルウェアが担当。
 * ここではプロキシ先から返った生 Response を加工する用途に限定。
 */

export function addCORSToResponse(response: Response, backend: string): Response {
  const headers = new Headers(response.headers);
  headers.set("Access-Control-Allow-Origin", "*");
  headers.set("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
  headers.set("Access-Control-Allow-Headers", "Content-Type, Authorization");
  headers.set("X-Backend", backend);

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  });
}

export function jsonResponse(
  data: object,
  status: number,
  extraHeaders?: Record<string, string>
): Response {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "Access-Control-Allow-Origin": "*",
    ...extraHeaders,
  };

  return new Response(JSON.stringify(data), { status, headers });
}
