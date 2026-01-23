/**
 * MCPist API Gateway - Cloudflare Worker
 *
 * 責務:
 * 1. JWT署名検証（Supabase JWKS）
 * 2. API Key検証（mpt_*形式）+ KVキャッシュ
 * 3. X-User-ID付与
 * 4. MCP Serverへのプロキシ
 *
 * Note: Rate Limit、LBメトリクスは削除済み（KV消費削減のため）
 * メトリクスはAPI Server側でLokiに送信予定
 */

import * as jose from "jose";

// === 型定義 ===

interface Env {
  // KV Namespaces
  API_KEY_CACHE: KVNamespace;  // APIキーキャッシュ用

  // バックエンド設定
  PRIMARY_API_URL: string;     // Primary API Server
  SECONDARY_API_URL: string;   // Secondary API Server (Failover)

  // Supabase設定
  SUPABASE_URL: string;
  SUPABASE_JWKS_URL: string;
  SUPABASE_PUBLISHABLE_KEY: string;

  // Gateway Secret (Worker → Go Server)
  GATEWAY_SECRET: string;

  // Internal Secret (Console → Worker for /internal/* endpoints)
  INTERNAL_SECRET: string;
}

interface AuthResult {
  userId: string;
  type: "jwt" | "api_key";
}

// === 定数 ===

const FETCH_TIMEOUT_MS = 30000;

// === メインハンドラー ===

export default {
  async fetch(
    request: Request,
    env: Env,
    ctx: ExecutionContext
  ): Promise<Response> {
    const url = new URL(request.url);

    // CORSプリフライト
    if (request.method === "OPTIONS") {
      return handleCORS();
    }

    // ヘルスチェックエンドポイント（認証不要・KV不使用）
    if (url.pathname === "/health") {
      // リアルタイムでバックエンドの状態をチェック
      const [primaryResult, secondaryResult] = await Promise.all([
        checkBackendHealth(env.PRIMARY_API_URL),
        checkBackendHealth(env.SECONDARY_API_URL),
      ]);

      return jsonResponse({
        status: "ok",
        backends: {
          primary: buildBackendInfo(primaryResult),
          secondary: buildBackendInfo(secondaryResult),
        },
      }, 200);
    }

    // 内部サービスエンドポイント（認証必須）
    if (url.pathname.startsWith("/internal/")) {
      // INTERNAL_SECRET による認証
      const internalSecret = request.headers.get("X-Internal-Secret");
      if (!internalSecret || internalSecret !== env.INTERNAL_SECRET) {
        return jsonResponse({ error: "Unauthorized" }, 401);
      }

      // APIキーキャッシュ無効化
      if (url.pathname === "/internal/invalidate-api-key" && request.method === "POST") {
        return await handleInvalidateApiKey(request, env);
      }

      return jsonResponse({ error: "Not Found" }, 404);
    }

    // OAuth Protected Resource Metadata (RFC 9728)
    // MCPクライアント（Claude.ai等）が認可サーバーを発見するために使用
    if (url.pathname === "/mcp/.well-known/oauth-protected-resource") {
      return handleOAuthProtectedResourceMetadata(request, env);
    }

    // OAuth Authorization Server Metadata (RFC 8414)
    if (url.pathname === "/mcp/.well-known/oauth-authorization-server") {
      return handleOAuthAuthorizationServerMetadata(env);
    }

    // MCPエンドポイントのみ処理
    if (!url.pathname.startsWith("/mcp")) {
      return new Response("Not Found", { status: 404 });
    }

    try {
      // 1. 認証（JWT or API Key）
      const authResult = await authenticate(request, env);
      if (!authResult) {
        // WWW-Authenticate ヘッダーでOAuthフローを開始させる (RFC 9728)
        const resourceMetadataUrl = `${url.protocol}//${url.host}/mcp/.well-known/oauth-protected-resource`;
        return new Response(JSON.stringify({ error: "Unauthorized" }), {
          status: 401,
          headers: {
            "Content-Type": "application/json",
            "WWW-Authenticate": `Bearer resource_metadata="${resourceMetadataUrl}"`,
            "Access-Control-Allow-Origin": "*",
          },
        });
      }

      // 2. プロキシ（Primary優先、失敗時Secondary）
      return await proxyRequest(request, authResult, env);
    } catch (error) {
      console.error("Gateway error:", error);
      return jsonResponse({ error: "Internal server error" }, 500);
    }
  },

  // Scheduled handler: バックエンドウォームアップ（5分毎）
  async scheduled(
    event: ScheduledEvent,
    env: Env,
    ctx: ExecutionContext
  ): Promise<void> {
    ctx.waitUntil(performScheduledHealthCheck(env));
  },
};

// === OAuth Metadata Handlers ===

/**
 * OAuth Protected Resource Metadata (RFC 9728)
 * MCPクライアントが認可サーバーを発見するために使用
 */
function handleOAuthProtectedResourceMetadata(request: Request, env: Env): Response {
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;

  const metadata = {
    resource: `${baseUrl}/mcp`,
    authorization_servers: [`${env.SUPABASE_URL}/auth/v1`],
    scopes_supported: ["openid", "profile", "email"],
    bearer_methods_supported: ["header"],
  };

  return new Response(JSON.stringify(metadata), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
    },
  });
}

/**
 * OAuth Authorization Server Metadata (RFC 8414)
 * Supabase Auth のメタデータをプロキシ
 */
async function handleOAuthAuthorizationServerMetadata(env: Env): Promise<Response> {
  try {
    const response = await fetch(
      `${env.SUPABASE_URL}/auth/v1/.well-known/openid-configuration`
    );

    if (response.ok) {
      const metadata = await response.json();
      return new Response(JSON.stringify(metadata), {
        status: 200,
        headers: {
          "Content-Type": "application/json",
          "Cache-Control": "public, max-age=3600",
          "Access-Control-Allow-Origin": "*",
        },
      });
    }
  } catch {
    // Fall through to manual metadata
  }

  // Fallback: 手動構築
  const metadata = {
    issuer: `${env.SUPABASE_URL}/auth/v1`,
    authorization_endpoint: `${env.SUPABASE_URL}/auth/v1/authorize`,
    token_endpoint: `${env.SUPABASE_URL}/auth/v1/token`,
    registration_endpoint: `${env.SUPABASE_URL}/auth/v1/oauth/register`,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code", "refresh_token"],
    code_challenge_methods_supported: ["S256"],
    token_endpoint_auth_methods_supported: ["none"],
    scopes_supported: ["openid", "profile", "email"],
  };

  return new Response(JSON.stringify(metadata), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
      "Access-Control-Allow-Origin": "*",
    },
  });
}

// === プロキシ ===

async function proxyRequest(
  request: Request,
  authResult: AuthResult,
  env: Env
): Promise<Response> {
  // Primary優先
  try {
    const response = await fetchBackend(request, env.PRIMARY_API_URL, authResult, env);
    return addCORSToResponse(response, "primary");
  } catch (primaryError) {
    console.error("Primary backend failed:", primaryError);
  }

  // Primaryが失敗したらSecondaryにフォールバック
  try {
    const response = await fetchBackend(request, env.SECONDARY_API_URL, authResult, env);
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
  backendUrl: string,
  authResult: AuthResult,
  env: Env
): Promise<Response> {
  const url = new URL(request.url);
  const targetUrl = `${backendUrl}${url.pathname}${url.search}`;

  const headers = new Headers(request.headers);
  headers.set("X-User-ID", authResult.userId);
  headers.set("X-Auth-Type", authResult.type);

  if (env.GATEWAY_SECRET) {
    headers.set("X-Gateway-Secret", env.GATEWAY_SECRET);
  }

  headers.delete("Authorization");

  const proxyRequest = new Request(targetUrl, {
    method: request.method,
    headers,
    body: request.body,
    redirect: "manual",
  });

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);

  try {
    return await fetch(proxyRequest, { signal: controller.signal });
  } finally {
    clearTimeout(timeoutId);
  }
}

function addCORSToResponse(response: Response, backend: string): Response {
  const responseHeaders = new Headers(response.headers);
  addCORSHeaders(responseHeaders);
  responseHeaders.set("X-Backend", backend);

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders,
  });
}

// === バックエンドヘルスチェック ===

type HealthCheckError =
  | "timeout"
  | "dns_failure"
  | "connection_refused"
  | "ssl_error"
  | "http_error"
  | "unknown";

interface HealthCheckResult {
  healthy: boolean;
  error?: HealthCheckError;
  statusCode?: number;
  latencyMs?: number;
}

async function checkBackendHealth(url: string): Promise<HealthCheckResult> {
  const healthUrl = `${url}/health`;
  const startTime = Date.now();

  try {
    const response = await fetch(healthUrl, {
      method: "GET",
      signal: AbortSignal.timeout(5000),
    });
    const latencyMs = Date.now() - startTime;

    if (response.ok) {
      return { healthy: true, statusCode: response.status, latencyMs };
    }
    return {
      healthy: false,
      error: "http_error",
      statusCode: response.status,
      latencyMs,
    };
  } catch (error) {
    const latencyMs = Date.now() - startTime;
    const errorType = classifyError(error);
    return { healthy: false, error: errorType, latencyMs };
  }
}

function classifyError(error: unknown): HealthCheckError {
  if (!(error instanceof Error)) {
    return "unknown";
  }

  const message = error.message.toLowerCase();
  const name = error.name;

  if (name === "TimeoutError" || message.includes("timeout") || message.includes("aborted")) {
    return "timeout";
  }

  if (
    message.includes("dns") ||
    message.includes("enotfound") ||
    message.includes("getaddrinfo") ||
    message.includes("name resolution") ||
    message.includes("internal error")
  ) {
    return "dns_failure";
  }

  if (
    message.includes("econnrefused") ||
    message.includes("connection refused") ||
    message.includes("network connection lost")
  ) {
    return "connection_refused";
  }

  if (
    message.includes("ssl handshake") ||
    message.includes("tls handshake") ||
    message.includes("certificate expired") ||
    message.includes("self signed certificate") ||
    message.includes("unable to verify")
  ) {
    return "ssl_error";
  }

  return "unknown";
}

function buildBackendInfo(result: HealthCheckResult): {
  healthy: boolean;
  error?: HealthCheckError;
  statusCode?: number;
  latencyMs?: number;
} {
  const info: {
    healthy: boolean;
    error?: HealthCheckError;
    statusCode?: number;
    latencyMs?: number;
  } = { healthy: result.healthy };
  if (result.error) info.error = result.error;
  if (result.statusCode) info.statusCode = result.statusCode;
  if (result.latencyMs !== undefined) info.latencyMs = result.latencyMs;
  return info;
}

// === スケジュールヘルスチェック（ウォームアップ用） ===

async function performScheduledHealthCheck(env: Env): Promise<void> {
  // 両方のバックエンドにヘルスチェックを送信してウォーム維持
  const [primaryResult, secondaryResult] = await Promise.all([
    checkBackendHealth(env.PRIMARY_API_URL),
    checkBackendHealth(env.SECONDARY_API_URL),
  ]);

  console.log(`[Cron] Health check - Primary: ${primaryResult.healthy}, Secondary: ${secondaryResult.healthy}`);
}

// === 認証 ===

async function authenticate(
  request: Request,
  env: Env
): Promise<AuthResult | null> {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader) {
    return null;
  }

  if (authHeader.startsWith("Bearer ")) {
    const token = authHeader.slice(7);

    // API Key (mpt_xxx format)
    if (token.startsWith("mpt_")) {
      return await verifyApiKey(token, env);
    }

    // JWT (Supabase issued)
    return await verifyJWT(token, env);
  }

  return null;
}

async function verifyJWT(token: string, env: Env): Promise<AuthResult | null> {
  // 1. まずSupabase auth.getUser相当のAPI呼び出しで検証
  // OAuth Serverが発行するトークンはオペークトークンの場合がある
  try {
    const response = await fetch(`${env.SUPABASE_URL}/auth/v1/user`, {
      headers: {
        Authorization: `Bearer ${token}`,
        apikey: env.SUPABASE_PUBLISHABLE_KEY,
      },
    });

    if (response.ok) {
      const user = await response.json() as { id?: string };
      if (user.id) {
        console.log("[Auth] Token verified via Supabase API");
        return { userId: user.id, type: "jwt" };
      }
    }
  } catch (error) {
    console.error("[Auth] Supabase API verification failed:", error);
  }

  // 2. フォールバック: JWT署名検証（従来のSupabase Authトークン用）
  try {
    const jwks = jose.createRemoteJWKSet(new URL(env.SUPABASE_JWKS_URL));
    const { payload } = await jose.jwtVerify(token, jwks, {
      issuer: `${env.SUPABASE_URL}/auth/v1`,
    });

    const userId = payload.sub;
    if (!userId) {
      return null;
    }

    console.log("[Auth] Token verified via JWT signature");
    return { userId, type: "jwt" };
  } catch (error) {
    console.error("[Auth] JWT verification failed:", error);
    return null;
  }
}

interface ApiKeyValidationResult {
  valid: boolean;
  user_id?: string;
  key_name?: string;
  scopes?: string[];
  error?: string;
}

interface ApiKeyCacheEntry {
  userId: string;
  cachedAt: number;
}

// APIキーキャッシュ設定
const API_KEY_CACHE_TTL_SECONDS = 86400; // 1日（KV TTL）
const API_KEY_CACHE_MAX_AGE_MS = 3600000; // 1時間（ソフト有効期限）

/**
 * APIキー検証（KVキャッシュ対応）
 *
 * フロー:
 * 1. KVキャッシュをチェック（ヒット時: 1-5ms）
 * 2. キャッシュミス時: Supabase RPCを呼び出し（10-50ms）
 * 3. 検証成功時: 結果をKVにキャッシュ
 */
async function verifyApiKey(
  apiKey: string,
  env: Env
): Promise<AuthResult | null> {
  const startTime = Date.now();

  // APIキーのSHA-256ハッシュをキャッシュキーとして使用
  const cacheKey = await hashApiKey(apiKey);

  // 1. KVキャッシュをチェック
  try {
    const cached = await env.API_KEY_CACHE.get<ApiKeyCacheEntry>(cacheKey, "json");

    if (cached) {
      const age = Date.now() - cached.cachedAt;
      if (age < API_KEY_CACHE_MAX_AGE_MS) {
        // キャッシュヒット（有効期限内）
        console.log(`[APIKey] Cache HIT | age: ${Math.round(age / 1000)}s`);
        return { userId: cached.userId, type: "api_key" };
      }
      // ソフト有効期限切れ → 古いキャッシュを使用（バックグラウンドで再検証は将来実装）
      console.log(`[APIKey] Cache SOFT-EXPIRED | age: ${Math.round(age / 1000)}s`);
      return { userId: cached.userId, type: "api_key" };
    }
  } catch (cacheError) {
    console.error("[APIKey] Cache read error:", cacheError);
  }

  // 2. キャッシュミス → Supabase RPCで検証
  console.log("[APIKey] Cache MISS, validating via Supabase RPC...");
  try {
    const response = await fetch(
      `${env.SUPABASE_URL}/rest/v1/rpc/validate_api_key`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          apikey: env.SUPABASE_PUBLISHABLE_KEY,
          Authorization: `Bearer ${env.SUPABASE_PUBLISHABLE_KEY}`,
        },
        body: JSON.stringify({ p_key: apiKey }),
      }
    );

    if (!response.ok) {
      console.log(`[APIKey] Validation FAILED (HTTP ${response.status})`);
      return null;
    }

    const result: ApiKeyValidationResult = await response.json();
    if (!result || !result.valid || !result.user_id) {
      console.log(`[APIKey] Validation FAILED (${result?.error || 'no user_id'})`);
      return null;
    }

    const userId = result.user_id;

    // 3. 検証成功 → KVにキャッシュ
    try {
      const cacheEntry: ApiKeyCacheEntry = {
        userId,
        cachedAt: Date.now(),
      };
      await env.API_KEY_CACHE.put(cacheKey, JSON.stringify(cacheEntry), {
        expirationTtl: API_KEY_CACHE_TTL_SECONDS,
      });
      const totalLatency = Date.now() - startTime;
      console.log(`[APIKey] Validation OK + Cached | total: ${totalLatency}ms`);
    } catch (cacheWriteError) {
      console.error("[APIKey] Cache write error:", cacheWriteError);
    }

    return { userId, type: "api_key" };
  } catch (error) {
    console.error("[APIKey] Verification error:", error);
    return null;
  }
}

/**
 * APIキーをSHA-256でハッシュ化（キャッシュキー用）
 */
async function hashApiKey(apiKey: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(apiKey);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, "0")).join("");
}

/**
 * APIキーキャッシュ無効化エンドポイント
 *
 * Console からAPIキー削除時に呼び出され、KVキャッシュを即座に削除する。
 */
async function handleInvalidateApiKey(
  request: Request,
  env: Env
): Promise<Response> {
  try {
    const body = await request.json() as { key_hash?: string };
    const keyHash = body.key_hash;

    if (!keyHash || typeof keyHash !== "string") {
      return jsonResponse({ error: "key_hash is required" }, 400);
    }

    // key_hash の形式検証（64文字の16進数）
    if (!/^[a-f0-9]{64}$/.test(keyHash)) {
      return jsonResponse({ error: "Invalid key_hash format" }, 400);
    }

    // KVキャッシュを削除
    await env.API_KEY_CACHE.delete(keyHash);
    console.log(`[InvalidateKey] Cache deleted for hash: ${keyHash.substring(0, 8)}...`);

    return jsonResponse({ success: true, message: "Cache invalidated" }, 200);
  } catch (error) {
    console.error("[InvalidateKey] Error:", error);
    return jsonResponse({ error: "Invalid request body" }, 400);
  }
}

// === ユーティリティ ===

function handleCORS(): Response {
  return new Response(null, {
    status: 204,
    headers: {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type, Authorization",
      "Access-Control-Max-Age": "86400",
    },
  });
}

function addCORSHeaders(headers: Headers): void {
  headers.set("Access-Control-Allow-Origin", "*");
  headers.set("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
  headers.set("Access-Control-Allow-Headers", "Content-Type, Authorization");
}

function jsonResponse(
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
