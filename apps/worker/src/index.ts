/**
 * MCPist API Gateway - Cloudflare Worker
 *
 * 責務:
 * 1. JWT署名検証（Supabase JWKS）
 * 2. API Key検証（mpt_*形式）
 * 3. グローバルRate Limit（IP単位）
 * 4. Burst制限（ユーザー単位）
 * 5. X-User-ID付与
 * 6. 負荷ベースLB（Render Primary / Koyeb Failover）
 * 7. MCP Serverへのプロキシ
 *
 * LB戦略:
 * - 主指標: p95レイテンシ（直近50req rolling window）
 * - NORMAL:   p95 < 300ms  → Render 100%
 * - WARMUP:   p95 ≥ 300ms  → Render 100% + Koyeb起動
 * - BALANCE:  p95 ≥ 600ms  → Render 50% / Koyeb 50%
 * - FAILOVER: 致命指標発生  → Koyeb 100%
 *
 * ヒステリシス:
 * - BALANCE → WARMUP: p95 < 500ms
 * - WARMUP → NORMAL:  p95 < 300ms
 */

import * as jose from "jose";

// === 型定義 ===

interface Env {
  // KV Namespaces
  RATE_LIMIT: KVNamespace;
  HEALTH_STATE: KVNamespace;

  // バックエンド設定
  RENDER_URL: string;   // Primary
  KOYEB_URL: string;    // Failover

  // Supabase設定
  SUPABASE_URL: string;
  SUPABASE_JWKS_URL: string;
  SUPABASE_ANON_KEY: string;

  // Rate Limit設定
  RATE_LIMIT_GLOBAL_MAX: string;
  RATE_LIMIT_BURST_MAX: string;

  // Gateway Secret
  GATEWAY_SECRET: string;
}

type LBState = "NORMAL" | "WARMUP" | "BALANCE" | "FAILOVER";
type KoyebState = "sleeping" | "waking" | "ready";

interface Metrics {
  latencies: number[];        // 直近50件のレイテンシ配列
  state: LBState;
  koyebState: KoyebState;
  error5xxCount: number;      // 直近20req中の5xx数
  requestCount: number;       // 直近20req用カウンタ
  lastUpdated: number;
  renderHealthy: boolean;
  koyebHealthy: boolean;
}

interface AuthResult {
  userId: string;
  type: "jwt" | "api_key";
}

interface RateLimitResult {
  allowed: boolean;
  retryAfter?: number;
}

// === 定数 ===

const LATENCY_WINDOW_SIZE = 50;
const ERROR_WINDOW_SIZE = 20;
const FETCH_TIMEOUT_MS = 3000;

// 閾値
const THRESHOLDS = {
  WARMUP: 300,      // p95 ≥ 300ms → WARMUP
  BALANCE: 600,     // p95 ≥ 600ms → BALANCE
  // ヒステリシス
  BALANCE_TO_WARMUP: 500,  // p95 < 500ms → WARMUP
  WARMUP_TO_NORMAL: 300,   // p95 < 300ms → NORMAL
};

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

    // ヘルスチェックエンドポイント（認証不要）
    if (url.pathname === "/health") {
      const metrics = await getMetrics(env);
      const weights = getTrafficWeights(metrics.state, metrics.koyebState);
      return jsonResponse({
        status: "ok",
        traffic: {
          primary: weights.primary,
          failover: weights.failover,
        },
        failoverServerState: metrics.koyebState,
        p95Latency: calculateP95(metrics.latencies),
        backends: {
          primary: { healthy: metrics.renderHealthy },
          failover: { healthy: metrics.koyebHealthy },
        },
      }, 200);
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
      const rateLimitResult = await checkRateLimit(env, clientIP, authResult.userId);
      if (!rateLimitResult.allowed) {
        return jsonResponse(
          { error: "Rate limit exceeded", retryAfter: rateLimitResult.retryAfter },
          429,
          { "Retry-After": String(rateLimitResult.retryAfter) }
        );
      }

      // 3. LBロジック + プロキシ
      return await handleWithLB(request, authResult, env, ctx);
    } catch (error) {
      console.error("Gateway error:", error);
      return jsonResponse({ error: "Internal server error" }, 500);
    }
  },

  // Scheduled handler: Renderヘルスチェック（毎分）
  async scheduled(
    event: ScheduledEvent,
    env: Env,
    ctx: ExecutionContext
  ): Promise<void> {
    ctx.waitUntil(performScheduledHealthCheck(env));
  },
};

// === LB ロジック ===

async function handleWithLB(
  request: Request,
  authResult: AuthResult,
  env: Env,
  ctx: ExecutionContext
): Promise<Response> {
  let metrics = await getMetrics(env);

  // バックエンド選択
  const backend = selectBackend(metrics, env);

  // Koyeb起動が必要な場合（WARMUP状態でsleeping）
  if (metrics.state === "WARMUP" && metrics.koyebState === "sleeping") {
    ctx.waitUntil(wakeKoyeb(env));
    metrics.koyebState = "waking";
  }

  // プロキシ実行
  const start = Date.now();
  let response: Response;
  let latency: number;
  let isFatal = false;
  let is5xx = false;

  try {
    response = await fetchWithTimeout(
      buildProxyRequest(request, backend, authResult, env),
      FETCH_TIMEOUT_MS
    );
    latency = Date.now() - start;

    if (response.status >= 500) {
      is5xx = true;
    }
  } catch (error) {
    // タイムアウトまたはfetch例外 → 致命指標
    latency = Date.now() - start;
    isFatal = true;
    console.error("Fetch failed:", error);

    // FAILOVERへ
    if (metrics.koyebState === "ready" && backend.url !== env.KOYEB_URL) {
      // Koyeb readyならリトライ
      try {
        response = await fetchWithTimeout(
          buildProxyRequest(request, env.KOYEB_URL, authResult, env),
          FETCH_TIMEOUT_MS
        );
      } catch (retryError) {
        return jsonResponse(
          { error: "Service unavailable", retryAfter: 30 },
          503,
          { "Retry-After": "30" }
        );
      }
    } else if (metrics.koyebState !== "ready") {
      // Koyeb not ready → 503 + Koyeb起動
      ctx.waitUntil(wakeKoyeb(env));
      return jsonResponse(
        { error: "Service unavailable", retryAfter: 30 },
        503,
        { "Retry-After": "30" }
      );
    } else {
      return jsonResponse(
        { error: "Service unavailable" },
        503
      );
    }
  }

  // メトリクス更新（非同期）
  ctx.waitUntil(updateMetrics(env, latency, isFatal, is5xx));

  // レスポンス返却
  const responseHeaders = new Headers(response!.headers);
  addCORSHeaders(responseHeaders);
  responseHeaders.set("X-LB-State", metrics.state);
  responseHeaders.set("X-Backend", backend.name);

  return new Response(response!.body, {
    status: response!.status,
    statusText: response!.statusText,
    headers: responseHeaders,
  });
}

interface Backend {
  url: string;
  name: string;
}

function selectBackend(metrics: Metrics, env: Env): Backend {
  switch (metrics.state) {
    case "FAILOVER":
      return { url: env.KOYEB_URL, name: "koyeb" };

    case "BALANCE":
      // 50/50
      if (metrics.koyebState === "ready" && Math.random() < 0.5) {
        return { url: env.KOYEB_URL, name: "koyeb" };
      }
      return { url: env.RENDER_URL, name: "render" };

    case "WARMUP":
    case "NORMAL":
    default:
      return { url: env.RENDER_URL, name: "render" };
  }
}

// === メトリクス管理 ===

async function getMetrics(env: Env): Promise<Metrics> {
  const data = await env.HEALTH_STATE.get<Metrics>("metrics", "json");
  return data || {
    latencies: [],
    state: "NORMAL",
    koyebState: "sleeping",
    error5xxCount: 0,
    requestCount: 0,
    lastUpdated: Date.now(),
    renderHealthy: true,
    koyebHealthy: true,
  };
}

async function updateMetrics(
  env: Env,
  latency: number,
  isFatal: boolean,
  is5xx: boolean
): Promise<void> {
  const metrics = await getMetrics(env);

  // レイテンシ追加（rolling window）
  metrics.latencies.push(latency);
  if (metrics.latencies.length > LATENCY_WINDOW_SIZE) {
    metrics.latencies.shift();
  }

  // 5xxカウント（直近20req）
  metrics.requestCount++;
  if (is5xx) {
    metrics.error5xxCount++;
  }
  if (metrics.requestCount > ERROR_WINDOW_SIZE) {
    // 古いデータをリセット（簡易実装）
    metrics.requestCount = 1;
    metrics.error5xxCount = is5xx ? 1 : 0;
  }

  // 状態遷移
  const p95 = calculateP95(metrics.latencies);
  const prevState = metrics.state;

  if (isFatal || !metrics.renderHealthy) {
    // 致命指標 → FAILOVER
    metrics.state = "FAILOVER";
  } else {
    // p95ベースの状態遷移（ヒステリシス考慮）
    metrics.state = calculateNextState(prevState, p95);
  }

  metrics.lastUpdated = Date.now();
  await env.HEALTH_STATE.put("metrics", JSON.stringify(metrics), {
    expirationTtl: 3600,
  });

  // 補助指標ログ（5xx率が高い場合）
  if (metrics.error5xxCount >= 2 && metrics.requestCount >= 10) {
    console.warn(`High 5xx rate: ${metrics.error5xxCount}/${metrics.requestCount}`);
  }
}

function calculateNextState(currentState: LBState, p95: number): LBState {
  switch (currentState) {
    case "NORMAL":
      if (p95 >= THRESHOLDS.BALANCE) return "BALANCE";
      if (p95 >= THRESHOLDS.WARMUP) return "WARMUP";
      return "NORMAL";

    case "WARMUP":
      if (p95 >= THRESHOLDS.BALANCE) return "BALANCE";
      if (p95 < THRESHOLDS.WARMUP_TO_NORMAL) return "NORMAL";
      return "WARMUP";

    case "BALANCE":
      if (p95 < THRESHOLDS.BALANCE_TO_WARMUP) return "WARMUP";
      return "BALANCE";

    case "FAILOVER":
      // FAILOVERからの復旧はヘルスチェックで行う
      return "FAILOVER";

    default:
      return "NORMAL";
  }
}

function calculateP95(latencies: number[]): number {
  if (latencies.length === 0) return 0;

  const sorted = [...latencies].sort((a, b) => a - b);
  const index = Math.floor(sorted.length * 0.95);
  return sorted[Math.min(index, sorted.length - 1)];
}

function getTrafficWeights(
  state: LBState,
  koyebState: KoyebState
): { primary: number; failover: number } {
  switch (state) {
    case "FAILOVER":
      return { primary: 0, failover: 100 };
    case "BALANCE":
      if (koyebState === "ready") {
        return { primary: 50, failover: 50 };
      }
      return { primary: 100, failover: 0 };
    case "WARMUP":
    case "NORMAL":
    default:
      return { primary: 100, failover: 0 };
  }
}

// === Koyeb 起動 ===

async function wakeKoyeb(env: Env): Promise<void> {
  const metrics = await getMetrics(env);

  if (metrics.koyebState === "ready") {
    return;
  }

  console.log("Waking up Koyeb...");
  metrics.koyebState = "waking";
  await env.HEALTH_STATE.put("metrics", JSON.stringify(metrics), {
    expirationTtl: 3600,
  });

  try {
    const response = await fetch(`${env.KOYEB_URL}/health`, {
      method: "GET",
      signal: AbortSignal.timeout(60000), // コールドスタート待ち
    });

    if (response.ok) {
      metrics.koyebState = "ready";
      metrics.koyebHealthy = true;
      console.log("Koyeb is ready");
    } else {
      metrics.koyebState = "sleeping";
      metrics.koyebHealthy = false;
    }
  } catch (error) {
    console.error("Failed to wake Koyeb:", error);
    metrics.koyebState = "sleeping";
    metrics.koyebHealthy = false;
  }

  await env.HEALTH_STATE.put("metrics", JSON.stringify(metrics), {
    expirationTtl: 3600,
  });
}

// === スケジュールヘルスチェック ===

async function performScheduledHealthCheck(env: Env): Promise<void> {
  const metrics = await getMetrics(env);

  // Renderヘルスチェック（これがウォーム維持にもなる）
  try {
    const response = await fetch(`${env.RENDER_URL}/health`, {
      method: "GET",
      signal: AbortSignal.timeout(5000),
    });

    if (response.ok) {
      metrics.renderHealthy = true;
      // FAILOVERから復旧
      if (metrics.state === "FAILOVER") {
        metrics.state = "NORMAL";
        metrics.latencies = []; // リセット
      }
    } else {
      metrics.renderHealthy = false;
      metrics.state = "FAILOVER";
    }
  } catch (error) {
    console.error("Render health check failed:", error);
    metrics.renderHealthy = false;
    metrics.state = "FAILOVER";
  }

  // Koyebが起動中なら状態確認
  if (metrics.koyebState === "waking" || metrics.state === "BALANCE" || metrics.state === "FAILOVER") {
    try {
      const response = await fetch(`${env.KOYEB_URL}/health`, {
        method: "GET",
        signal: AbortSignal.timeout(5000),
      });

      if (response.ok) {
        metrics.koyebState = "ready";
        metrics.koyebHealthy = true;
      } else {
        metrics.koyebHealthy = false;
      }
    } catch (error) {
      // Koyebがスリープ中は失敗しても問題ない
      if (metrics.state === "FAILOVER") {
        console.log("Koyeb not responding in FAILOVER state, attempting wake...");
      }
    }
  }

  metrics.lastUpdated = Date.now();
  await env.HEALTH_STATE.put("metrics", JSON.stringify(metrics), {
    expirationTtl: 3600,
  });
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
  try {
    const jwks = jose.createRemoteJWKSet(new URL(env.SUPABASE_JWKS_URL));
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
    const response = await fetch(
      `${env.SUPABASE_URL}/rest/v1/rpc/validate_api_key`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          apikey: env.SUPABASE_ANON_KEY,
          Authorization: `Bearer ${env.SUPABASE_ANON_KEY}`,
        },
        body: JSON.stringify({ p_api_key: apiKey, p_service: "mcpist" }),
      }
    );

    if (!response.ok) {
      return null;
    }

    const results: ApiKeyValidationResult[] = await response.json();
    if (!results || results.length === 0 || !results[0].user_id) {
      return null;
    }

    return { userId: results[0].user_id, type: "api_key" };
  } catch (error) {
    console.error("API Key verification error:", error);
    return null;
  }
}

// === Rate Limit ===

async function checkRateLimit(
  env: Env,
  clientIP: string,
  userId: string
): Promise<RateLimitResult> {
  const now = Math.floor(Date.now() / 1000);

  const globalKey = `global:${clientIP}:${Math.floor(now / 60)}`;
  const globalCount = parseInt((await env.RATE_LIMIT.get(globalKey)) || "0");
  const globalMax = parseInt(env.RATE_LIMIT_GLOBAL_MAX || "1000");

  if (globalCount >= globalMax) {
    return { allowed: false, retryAfter: 60 - (now % 60) };
  }

  const burstKey = `burst:${userId}:${now}`;
  const burstCount = parseInt((await env.RATE_LIMIT.get(burstKey)) || "0");
  const burstMax = parseInt(env.RATE_LIMIT_BURST_MAX || "5");

  if (burstCount >= burstMax) {
    return { allowed: false, retryAfter: 1 };
  }

  await Promise.all([
    env.RATE_LIMIT.put(globalKey, String(globalCount + 1), { expirationTtl: 120 }),
    env.RATE_LIMIT.put(burstKey, String(burstCount + 1), { expirationTtl: 60 }),
  ]);

  return { allowed: true };
}

// === プロキシ ===

function buildProxyRequest(
  request: Request,
  backend: string | Backend,
  authResult: AuthResult,
  env: Env
): Request {
  const backendUrl = typeof backend === "string" ? backend : backend.url;
  const url = new URL(request.url);
  const targetUrl = `${backendUrl}${url.pathname}${url.search}`;

  const headers = new Headers(request.headers);
  headers.set("X-User-ID", authResult.userId);
  headers.set("X-Auth-Type", authResult.type);

  if (env.GATEWAY_SECRET) {
    headers.set("X-Gateway-Secret", env.GATEWAY_SECRET);
  }

  headers.delete("Authorization");

  return new Request(targetUrl, {
    method: request.method,
    headers,
    body: request.body,
    redirect: "manual",
  });
}

async function fetchWithTimeout(
  request: Request,
  timeoutMs: number
): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(request, { signal: controller.signal });
    return response;
  } finally {
    clearTimeout(timeoutId);
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
