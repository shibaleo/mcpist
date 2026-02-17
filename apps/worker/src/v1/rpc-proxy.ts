import type { Env } from "../types";
import { authenticate } from "../auth";
import { addCORSToResponse, jsonResponse } from "../http";
import { logRequest, logSecurityEvent } from "../logging";
import { pushRequestLog, pushSecurityEvent } from "../observability";

/** 認証不要の公開 RPC */
const PUBLIC_RPCS = new Set([
  "list_modules_with_tools",
  "get_oauth_app_credentials",
]);

/** admin ロール必須の RPC */
const ADMIN_RPCS = new Set([
  "list_oauth_apps",
  "upsert_oauth_app",
  "delete_oauth_app",
  "list_all_oauth_consents",
]);

/**
 * /rpc/{name} へのリクエストを PostgREST に転送する。
 *
 * - Public RPC: 認証なしで転送
 * - Admin RPC: JWT 認証 + admin role チェック後に転送
 * - User-scoped RPC: JWT 認証後、p_user_id を注入して転送
 */
export async function handleRpcProxy(
  request: Request,
  url: URL,
  env: Env,
  ctx: ExecutionContext
): Promise<Response> {
  if (request.method !== "POST") {
    return jsonResponse({ error: "Method not allowed" }, 405);
  }

  const rpcName = url.pathname.slice("/v1/rpc/".length);
  if (!rpcName) {
    return jsonResponse({ error: "RPC name required" }, 400);
  }

  const requestId = crypto.randomUUID();
  const startTime = Date.now();

  try {
    let body: Record<string, unknown> = {};
    try {
      const text = await request.text();
      if (text) body = JSON.parse(text);
    } catch {
      return jsonResponse({ error: "Invalid JSON body" }, 400);
    }

    // Public RPC: 認証不要
    if (PUBLIC_RPCS.has(rpcName)) {
      const result = await forwardToPostgREST(rpcName, body, env);
      const durationMs = Date.now() - startTime;
      const extra = { rpc: rpcName, auth_type: "none" };
      logRequest(env, requestId, "POST", url.pathname, result.status, durationMs, extra);
      ctx.waitUntil(pushRequestLog(env, requestId, "POST", url.pathname, result.status, durationMs, extra));
      return addCORSToResponse(result, "postgrest");
    }

    // 認証必須
    const authResult = await authenticate(request, env);
    if (!authResult) {
      const extra = { rpc: rpcName, duration_ms: Date.now() - startTime };
      logSecurityEvent(env, requestId, "rpc_auth_failed", extra);
      ctx.waitUntil(pushSecurityEvent(env, requestId, "rpc_auth_failed", extra));
      return jsonResponse({ error: "Unauthorized" }, 401);
    }

    // Admin RPC: role チェック → p_user_id 注入なしで転送
    if (ADMIN_RPCS.has(rpcName)) {
      const rows = await callPostgRESTRpc<{ role?: string }[]>(
        "get_user_context",
        { p_user_id: authResult.userId },
        env
      );
      const context = Array.isArray(rows) ? rows[0] : rows;
      if (context?.role !== "admin") {
        return jsonResponse({ error: "Forbidden" }, 403);
      }
      const result = await forwardToPostgREST(rpcName, body, env);
      const durationMs = Date.now() - startTime;
      const extra = { rpc: rpcName, user_id: authResult.userId, auth_type: authResult.type };
      logRequest(env, requestId, "POST", url.pathname, result.status, durationMs, extra);
      ctx.waitUntil(pushRequestLog(env, requestId, "POST", url.pathname, result.status, durationMs, extra));
      return addCORSToResponse(result, "postgrest");
    }

    // User-scoped RPC: p_user_id を注入して転送
    body.p_user_id = authResult.userId;
    const result = await forwardToPostgREST(rpcName, body, env);
    const durationMs = Date.now() - startTime;
    const extra = { rpc: rpcName, user_id: authResult.userId, auth_type: authResult.type };
    logRequest(env, requestId, "POST", url.pathname, result.status, durationMs, extra);
    ctx.waitUntil(pushRequestLog(env, requestId, "POST", url.pathname, result.status, durationMs, extra));
    return addCORSToResponse(result, "postgrest");
  } catch (error) {
    console.error("[RPC Proxy] Error:", error);
    const durationMs = Date.now() - startTime;
    const extra = { rpc: rpcName, error: error instanceof Error ? error.message : "unknown" };
    logRequest(env, requestId, "POST", url.pathname, 500, durationMs, extra);
    ctx.waitUntil(pushRequestLog(env, requestId, "POST", url.pathname, 500, durationMs, extra));
    return jsonResponse({ error: "Internal server error" }, 500);
  }
}

/** PostgREST に RPC を転送 */
async function forwardToPostgREST(
  rpcName: string,
  body: Record<string, unknown>,
  env: Env
): Promise<Response> {
  return fetch(`${env.POSTGREST_URL}/rpc/${rpcName}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.POSTGREST_API_KEY}`,
      apikey: env.POSTGREST_API_KEY,
    },
    body: JSON.stringify(body),
  });
}

/** PostgREST RPC を呼んで JSON パースする（内部用） */
async function callPostgRESTRpc<T>(
  rpcName: string,
  params: Record<string, unknown>,
  env: Env
): Promise<T> {
  const res = await forwardToPostgREST(rpcName, params, env);
  if (!res.ok) {
    throw new Error(`PostgREST RPC ${rpcName} failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}
