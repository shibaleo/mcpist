import type { Env, AuthResult } from "../types";
import { authenticate } from "../auth";
import { addCORSToResponse, jsonResponse } from "../http";
import { logRequest, logSecurityEvent } from "../logging";
import { pushRequestLog, pushSecurityEvent } from "../observability";

const FETCH_TIMEOUT_MS = 30000;

/**
 * /mcp/* リクエストを Go Server にプロキシする。
 * JWT / API Key 認証 → Primary / Secondary フェイルオーバー。
 */
export async function handleMcpProxy(
  request: Request,
  url: URL,
  env: Env,
  ctx: ExecutionContext
): Promise<Response> {
  const requestId = crypto.randomUUID();
  const startTime = Date.now();

  try {
    const authResult = await authenticate(request, env);
    if (!authResult) {
      const resourceMetadataUrl = `${url.protocol}//${url.host}/v1/mcp/.well-known/oauth-protected-resource`;
      const secExtra = { method: request.method, path: url.pathname, duration_ms: Date.now() - startTime };
      logSecurityEvent(env, requestId, "auth_failed", secExtra);
      ctx.waitUntil(pushSecurityEvent(env, requestId, "auth_failed", secExtra));
      return new Response(JSON.stringify({ error: "Unauthorized" }), {
        status: 401,
        headers: {
          "Content-Type": "application/json",
          "WWW-Authenticate": `Bearer resource_metadata="${resourceMetadataUrl}"`,
          "Access-Control-Allow-Origin": "*",
        },
      });
    }

    const result = await proxyWithFailover(request, requestId, authResult, env);
    const durationMs = Date.now() - startTime;
    const extra = {
      user_id: authResult.userId,
      auth_type: authResult.type,
      backend: result.headers.get("X-Backend") || "unknown",
    };
    logRequest(env, requestId, request.method, url.pathname, result.status, durationMs, extra);
    ctx.waitUntil(pushRequestLog(env, requestId, request.method, url.pathname, result.status, durationMs, extra));
    return result;
  } catch (error) {
    console.error("Gateway error:", error);
    const durationMs = Date.now() - startTime;
    const errExtra = { error: error instanceof Error ? error.message : "unknown" };
    logRequest(env, requestId, request.method, url.pathname, 500, durationMs, errExtra);
    ctx.waitUntil(pushRequestLog(env, requestId, request.method, url.pathname, 500, durationMs, errExtra));
    return jsonResponse({ error: "Internal server error" }, 500);
  }
}

async function proxyWithFailover(
  request: Request,
  requestId: string,
  authResult: AuthResult,
  env: Env
): Promise<Response> {
  try {
    const response = await fetchBackend(request, requestId, env.PRIMARY_API_URL, authResult, env);
    return addCORSToResponse(response, "primary");
  } catch (primaryError) {
    console.error("Primary backend failed:", primaryError);
  }

  try {
    const response = await fetchBackend(request, requestId, env.SECONDARY_API_URL, authResult, env);
    return addCORSToResponse(response, "secondary");
  } catch (secondaryError) {
    console.error("Secondary backend also failed:", secondaryError);
    return jsonResponse(
      { error: "Service unavailable", retryAfter: 30 },
      503,
      { "Retry-After": "30" }
    );
  }
}

async function fetchBackend(
  request: Request,
  requestId: string,
  backendUrl: string,
  authResult: AuthResult,
  env: Env
): Promise<Response> {
  const url = new URL(request.url);
  const backendPath = url.pathname.replace(/^\/v1/, "");
  const targetUrl = `${backendUrl}${backendPath}${url.search}`;

  const headers = new Headers(request.headers);
  headers.set("X-User-ID", authResult.userId);
  headers.set("X-Auth-Type", authResult.type);
  headers.set("X-Request-ID", requestId);
  if (env.GATEWAY_SECRET) {
    headers.set("X-Gateway-Secret", env.GATEWAY_SECRET);
  }
  headers.delete("Authorization");

  const proxyReq = new Request(targetUrl, {
    method: request.method,
    headers,
    body: request.body,
    redirect: "manual",
  });

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);
  try {
    return await fetch(proxyReq, { signal: controller.signal });
  } finally {
    clearTimeout(timeoutId);
  }
}
