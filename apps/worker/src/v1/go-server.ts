/**
 * Go Server REST クライアント
 *
 * PostgREST RPC を置換し、Go Server の REST エンドポイントにプロキシする。
 */

import type { Env } from "../types";
import { jsonResponse } from "../http";

/**
 * Go Server へリクエストをプロキシし、レスポンスをそのまま返す。
 * CORS ヘッダーを追加。
 */
export async function forwardToGoServer(
  env: Env,
  method: string,
  path: string,
  headers?: Record<string, string>,
  body?: string | null
): Promise<Response> {
  const url = `${env.PRIMARY_API_URL}${path}`;

  const reqHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Gateway-Secret": env.GATEWAY_SECRET,
    ...headers,
  };

  const response = await fetch(url, {
    method,
    headers: reqHeaders,
    body: body ?? undefined,
  });

  // Add CORS headers
  const respHeaders = new Headers(response.headers);
  respHeaders.set("Access-Control-Allow-Origin", "*");

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: respHeaders,
  });
}

/**
 * Go Server の REST エンドポイントを呼び出し、JSON レスポンスをパースして返す。
 */
export async function callGoServer<T>(
  env: Env,
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const url = `${env.PRIMARY_API_URL}${path}`;

  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-Gateway-Secret": env.GATEWAY_SECRET,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Go Server ${method} ${path} failed (${response.status}): ${text}`);
  }

  return response.json() as Promise<T>;
}
