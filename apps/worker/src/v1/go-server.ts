/**
 * Go Server REST クライアント
 *
 * Go Server の REST エンドポイントにプロキシする。
 * Gateway JWT で Worker → Go Server 間を認証。
 */

import type { Env } from "../types";
import { signGatewayToken, type GatewayTokenClaims } from "../gateway-token";

/**
 * Go Server へリクエストをプロキシし、レスポンスをそのまま返す。
 * Gateway JWT に claims（user_id/clerk_id/email）を含める。
 */
export async function forwardToGoServer(
  env: Env,
  method: string,
  path: string,
  claims?: GatewayTokenClaims,
  body?: string | null
): Promise<Response> {
  const token = await signGatewayToken(env.GATEWAY_SIGNING_KEY, claims ?? {});
  const url = `${env.SERVER_URL}${path}`;

  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-Gateway-Token": token,
    },
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
 * ユーザーコンテキストなし（サーバー間通信用）。
 */
export async function callGoServer<T>(
  env: Env,
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const token = await signGatewayToken(env.GATEWAY_SIGNING_KEY, {});
  const url = `${env.SERVER_URL}${path}`;

  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-Gateway-Token": token,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Go Server ${method} ${path} failed (${response.status}): ${text}`);
  }

  return response.json() as Promise<T>;
}
