/**
 * PostgREST RPC 転送ユーティリティ。
 * 各 RESTful ルートハンドラーから呼び出される。
 */

import type { Env } from "../types";
import { addCORSToResponse } from "../http";

/** PostgREST に RPC を転送し、CORS ヘッダーを付与して返す */
export async function forwardToPostgREST(
  env: Env,
  rpcName: string,
  params: Record<string, unknown>
): Promise<Response> {
  const res = await fetch(`${env.POSTGREST_URL}/rpc/${rpcName}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.POSTGREST_API_KEY}`,
      apikey: env.POSTGREST_API_KEY,
    },
    body: JSON.stringify(params),
  });
  return addCORSToResponse(
    new Response(res.body, { status: res.status, headers: res.headers }),
    "postgrest"
  );
}

/** PostgREST RPC を呼んで JSON パースする（内部用、例: admin role チェック） */
export async function callPostgRESTRpc<T>(
  env: Env,
  rpcName: string,
  params: Record<string, unknown>
): Promise<T> {
  const res = await fetch(`${env.POSTGREST_URL}/rpc/${rpcName}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.POSTGREST_API_KEY}`,
      apikey: env.POSTGREST_API_KEY,
    },
    body: JSON.stringify(params),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`PostgREST RPC ${rpcName} failed: ${res.status} ${body}`);
  }
  return res.json() as Promise<T>;
}
