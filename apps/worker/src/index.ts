/**
 * MCPist API Gateway - Cloudflare Worker
 *
 * 責務:
 * 1. JWT署名検証（Supabase JWKS）
 * 2. API Key検証（mpt_*形式）
 * 3. グローバルRate Limit（IP単位）
 * 4. Burst制限（ユーザー単位）
 * 5. X-User-ID付与
 * 6. ヘルスチェック + ロードバランシング
 * 7. MCP Serverへのプロキシ
 */

import * as jose from "jose";

// 型定義
interface Env {
  // KV Namespaces
  RATE_LIMIT: KVNamespace;
  HEALTH_STATE: KVNamespace;

  // バックエンド設定（汎用命名）
  BACKEND_PRIMARY_URL: string;
  BACKEND_PRIMARY_WEIGHT: string;
  BACKEND_SECONDARY_URL: string;
  BACKEND_SECONDARY_WEIGHT: string;

  // Supabase設定
  SUPABASE_URL: string;
  SUPABASE_JWKS_URL: string;
  SUPABASE_SERVICE_ROLE_KEY: string;

  // Rate Limit設定
  RATE_LIMIT_GLOBAL_MAX: string;
  RATE_LIMIT_BURST_MAX: string;

  // Gateway Secret
  GATEWAY_SECRET: string;
}

interface Backend {
  url: string;
  weight: number;
  healthy: boolean;
}

interface HealthState {
  healthy: boolean;
  failureCount: number;
  lastCheck: number;
}

interface AuthResult {
  userId: string;
  type: "jwt" | "api_key";
}

// メインハンドラー
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

    // ヘルスチェックエンドポイント（認証不要）
    if (url.pathname === "/health") {
      return new Response("ok", { status: 200 });
    }

    // MCPエンドポイントのみ処理
    if (!url.pathname.startsWith("/mcp")) {
      return new Response("Not Found", { status: 404 });
    }

    try {
      // 1. 認証（JWT or API Key）
      const authResult = await authenticate(request, env);
      if (!authResult) {
        return jsonResponse({ error: "Unauthorized" }, 401);
      }

      // 2. Rate Limit チェック
      const clientIP = request.headers.get("CF-Connecting-IP") || "unknown";
      const rateLimitResult = await checkRateLimit(
        env,
        clientIP,
        authResult.userId
      );
      if (!rateLimitResult.allowed) {
        return jsonResponse(
          { error: "Rate limit exceeded", retryAfter: rateLimitResult.retryAfter },
          429,
          { "Retry-After": String(rateLimitResult.retryAfter) }
        );
      }

      // 3. バックエンド選択（ロードバランシング）
      const backend = await selectBackend(env);
      if (!backend) {
        return jsonResponse({ error: "Service unavailable" }, 503);
      }

      // 4. プロキシ
      return await proxyRequest(request, backend, authResult, env);
    } catch (error) {
      console.error("Gateway error:", error);
      return jsonResponse({ error: "Internal server error" }, 500);
    }
  },

  // Scheduled handler for health checks
  async scheduled(
    event: ScheduledEvent,
    env: Env,
    ctx: ExecutionContext
  ): Promise<void> {
    ctx.waitUntil(performHealthChecks(env));
  },
};

// === 認証 ===

async function authenticate(
  request: Request,
  env: Env
): Promise<AuthResult | null> {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader) {
    return null;
  }

  // Bearer token
  if (authHeader.startsWith("Bearer ")) {
    const token = authHeader.slice(7);

    // API Key（mpt_*形式）
    if (token.startsWith("mpt_")) {
      return await verifyApiKey(token, env);
    }

    // JWT
    return await verifyJWT(token, env);
  }

  return null;
}

async function verifyJWT(token: string, env: Env): Promise<AuthResult | null> {
  try {
    // JWKSを取得
    const jwks = jose.createRemoteJWKSet(new URL(env.SUPABASE_JWKS_URL));

    // JWT検証
    const { payload } = await jose.jwtVerify(token, jwks, {
      issuer: `${env.SUPABASE_URL}/auth/v1`,
    });

    const userId = payload.sub;
    if (!userId) {
      return null;
    }

    return { userId, type: "jwt" };
  } catch (error) {
    console.error("JWT verification failed:", error);
    return null;
  }
}

interface ApiKeyValidationResult {
  user_id: string;
}

async function verifyApiKey(
  apiKey: string,
  env: Env
): Promise<AuthResult | null> {
  try {
    const url = `${env.SUPABASE_URL}/rest/v1/rpc/validate_api_key`;
    console.log("API Key verification URL:", url);
    console.log("API Key (first 10 chars):", apiKey.substring(0, 10));

    // Supabase RPCでAPI Key検証
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        apikey: env.SUPABASE_SERVICE_ROLE_KEY,
        Authorization: `Bearer ${env.SUPABASE_SERVICE_ROLE_KEY}`,
      },
      body: JSON.stringify({ p_api_key: apiKey, p_service: "mcpist" }),
    });

    console.log("Supabase RPC response status:", response.status);

    if (!response.ok) {
      const errorText = await response.text();
      console.error("Supabase RPC error:", errorText);
      return null;
    }

    const results: ApiKeyValidationResult[] = await response.json();
    console.log("Supabase RPC result:", JSON.stringify(results));

    if (!results || results.length === 0 || !results[0].user_id) {
      return null;
    }

    return { userId: results[0].user_id, type: "api_key" };
  } catch (error) {
    console.error("API Key verification failed:", error);
    return null;
  }
}

// === Rate Limit ===

interface RateLimitResult {
  allowed: boolean;
  retryAfter?: number;
}

async function checkRateLimit(
  env: Env,
  clientIP: string,
  userId: string
): Promise<RateLimitResult> {
  const now = Math.floor(Date.now() / 1000);

  // グローバルRate Limit（IP単位、分単位）
  const globalKey = `global:${clientIP}:${Math.floor(now / 60)}`;
  const globalCount = parseInt((await env.RATE_LIMIT.get(globalKey)) || "0");
  const globalMax = parseInt(env.RATE_LIMIT_GLOBAL_MAX || "1000");

  if (globalCount >= globalMax) {
    return { allowed: false, retryAfter: 60 - (now % 60) };
  }

  // Burst制限（ユーザー単位、秒単位）
  const burstKey = `burst:${userId}:${now}`;
  const burstCount = parseInt((await env.RATE_LIMIT.get(burstKey)) || "0");
  const burstMax = parseInt(env.RATE_LIMIT_BURST_MAX || "5");

  if (burstCount >= burstMax) {
    return { allowed: false, retryAfter: 1 };
  }

  // カウンター更新
  await Promise.all([
    env.RATE_LIMIT.put(globalKey, String(globalCount + 1), {
      expirationTtl: 120,
    }),
    env.RATE_LIMIT.put(burstKey, String(burstCount + 1), { expirationTtl: 60 }),
  ]);

  return { allowed: true };
}

// === ロードバランシング ===

async function selectBackend(env: Env): Promise<Backend | null> {
  const backends: Backend[] = [
    {
      url: env.BACKEND_PRIMARY_URL,
      weight: parseInt(env.BACKEND_PRIMARY_WEIGHT || "50"),
      healthy: true,
    },
    {
      url: env.BACKEND_SECONDARY_URL,
      weight: parseInt(env.BACKEND_SECONDARY_WEIGHT || "50"),
      healthy: true,
    },
  ];

  // ヘルス状態をKVから取得
  for (const backend of backends) {
    const stateKey = `health:${backend.url}`;
    const state = await env.HEALTH_STATE.get<HealthState>(stateKey, "json");
    if (state) {
      backend.healthy = state.healthy;
    }
  }

  // healthy なバックエンドのみフィルタ
  const healthyBackends = backends.filter((b) => b.healthy);
  if (healthyBackends.length === 0) {
    // 全て unhealthy の場合、primary を試す
    return backends[0];
  }

  // 重み付けランダム選択
  const totalWeight = healthyBackends.reduce((sum, b) => sum + b.weight, 0);
  let random = Math.random() * totalWeight;

  for (const backend of healthyBackends) {
    random -= backend.weight;
    if (random <= 0) {
      return backend;
    }
  }

  return healthyBackends[0];
}

// === プロキシ ===

async function proxyRequest(
  request: Request,
  backend: Backend,
  authResult: AuthResult,
  env: Env
): Promise<Response> {
  const url = new URL(request.url);
  const targetUrl = `${backend.url}${url.pathname}${url.search}`;

  // リクエストヘッダーを複製
  const headers = new Headers(request.headers);

  // X-User-ID 付与
  headers.set("X-User-ID", authResult.userId);
  headers.set("X-Auth-Type", authResult.type);

  // Gateway Secret 付与（オリジン直接アクセス防止）
  if (env.GATEWAY_SECRET) {
    headers.set("X-Gateway-Secret", env.GATEWAY_SECRET);
  }

  // Authorization ヘッダーを削除（オリジンには渡さない）
  headers.delete("Authorization");

  // プロキシリクエスト
  const proxyRequest = new Request(targetUrl, {
    method: request.method,
    headers,
    body: request.body,
    redirect: "manual",
  });

  const response = await fetch(proxyRequest);

  // レスポンスヘッダーを複製してCORSを付与
  const responseHeaders = new Headers(response.headers);
  addCORSHeaders(responseHeaders);

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders,
  });
}

// === ヘルスチェック ===

async function performHealthChecks(env: Env): Promise<void> {
  const backends = [
    { url: env.BACKEND_PRIMARY_URL, name: "primary" },
    { url: env.BACKEND_SECONDARY_URL, name: "secondary" },
  ];

  for (const backend of backends) {
    const stateKey = `health:${backend.url}`;
    let state: HealthState = (await env.HEALTH_STATE.get(stateKey, "json")) || {
      healthy: true,
      failureCount: 0,
      lastCheck: 0,
    };

    try {
      const response = await fetch(`${backend.url}/health`, {
        method: "GET",
        signal: AbortSignal.timeout(5000),
      });

      if (response.ok) {
        state = { healthy: true, failureCount: 0, lastCheck: Date.now() };
      } else {
        state.failureCount++;
        state.lastCheck = Date.now();
        if (state.failureCount >= 3) {
          state.healthy = false;
        }
      }
    } catch (error) {
      state.failureCount++;
      state.lastCheck = Date.now();
      if (state.failureCount >= 3) {
        state.healthy = false;
      }
      console.error(`Health check failed for ${backend.name}:`, error);
    }

    await env.HEALTH_STATE.put(stateKey, JSON.stringify(state), {
      expirationTtl: 300,
    });
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
