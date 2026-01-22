/**
 * MCPist API Gateway - Cloudflare Worker
 *
 * 責務:
 * 1. JWT署名検証（Supabase JWKS）
 * 2. API Key検証（mpt_*形式）
 * 3. グローバルRate Limit（IP単位）
 * 4. Burst制限（ユーザー単位）
 * 5. X-User-ID付与
 * 6. 負荷ベースLB（Primary / Secondary Failover）
 * 7. MCP Serverへのプロキシ
 *
 * LB戦略:
 * - 主指標: p95レイテンシ（直近50req rolling window）
 * - NORMAL:   p95 < 300ms  → Primary 100%
 * - WARMUP:   p95 ≥ 300ms  → Primary 100% + Secondary起動
 * - BALANCE:  p95 ≥ 600ms  → Primary 50% / Secondary 50%
 * - FAILOVER: 致命指標発生  → Secondary 100%
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
  API_KEY_CACHE: KVNamespace;  // APIキーキャッシュ用

  // バックエンド設定
  PRIMARY_API_URL: string;     // Primary API Server
  SECONDARY_API_URL: string;   // Secondary API Server (Failover)

  // Supabase設定
  SUPABASE_URL: string;
  SUPABASE_JWKS_URL: string;
  SUPABASE_ANON_KEY: string;

  // OAuth Mock Server (開発環境用)
  OAUTH_JWKS_URL?: string;

  // Rate Limit設定
  RATE_LIMIT_GLOBAL_MAX: string;
  RATE_LIMIT_BURST_MAX: string;

  // Gateway Secret (Worker → Go Server)
  GATEWAY_SECRET: string;

  // Internal Secret (Console → Worker for /internal/* endpoints)
  INTERNAL_SECRET: string;
}

type LBState = "NORMAL" | "WARMUP" | "BALANCE" | "FAILOVER";
type SecondaryState = "sleeping" | "waking" | "ready";

interface Metrics {
  latencies: number[];        // 直近50件のレイテンシ配列
  state: LBState;
  secondaryState: SecondaryState;
  error5xxCount: number;      // 直近20req中の5xx数
  requestCount: number;       // 直近20req用カウンタ
  lastUpdated: number;
  primaryHealthy: boolean;
  secondaryHealthy: boolean;
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

      // リアルタイムでバックエンドの状態をチェック（順次実行で相互影響を防ぐ）
      const primaryResult = await checkBackendHealth(env.PRIMARY_API_URL);
      const secondaryResult = await checkBackendHealth(env.SECONDARY_API_URL);

      // メトリクスを更新
      metrics.primaryHealthy = primaryResult.healthy;
      metrics.secondaryHealthy = secondaryResult.healthy;
      if (secondaryResult.healthy) {
        metrics.secondaryState = "ready";
      } else {
        metrics.secondaryState = "sleeping";
      }

      const weights = getTrafficWeights(metrics.state, metrics.secondaryState);

      // バックエンド情報を構築（エラー詳細を含む）
      const buildBackendInfo = (result: HealthCheckResult) => {
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
      };

      return jsonResponse({
        status: "ok",
        traffic: {
          primary: weights.primary,
          secondary: weights.failover,
        },
        secondaryServerState: metrics.secondaryState,
        p95Latency: calculateP95(metrics.latencies),
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

  // Secondary起動が必要な場合（WARMUP状態でsleeping）
  if (metrics.state === "WARMUP" && metrics.secondaryState === "sleeping") {
    ctx.waitUntil(wakeSecondary(env));
    metrics.secondaryState = "waking";
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
    if (metrics.secondaryState === "ready" && backend.url !== env.SECONDARY_API_URL) {
      // Secondary readyならリトライ
      try {
        response = await fetchWithTimeout(
          buildProxyRequest(request, env.SECONDARY_API_URL, authResult, env),
          FETCH_TIMEOUT_MS
        );
      } catch (retryError) {
        return jsonResponse(
          { error: "Service unavailable", retryAfter: 30 },
          503,
          { "Retry-After": "30" }
        );
      }
    } else if (metrics.secondaryState !== "ready") {
      // Secondary not ready → 503 + Secondary起動
      ctx.waitUntil(wakeSecondary(env));
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
      return { url: env.SECONDARY_API_URL, name: "secondary" };

    case "BALANCE":
      // 50/50
      if (metrics.secondaryState === "ready" && Math.random() < 0.5) {
        return { url: env.SECONDARY_API_URL, name: "secondary" };
      }
      return { url: env.PRIMARY_API_URL, name: "primary" };

    case "WARMUP":
    case "NORMAL":
    default:
      return { url: env.PRIMARY_API_URL, name: "primary" };
  }
}

// === メトリクス管理 ===

async function getMetrics(env: Env): Promise<Metrics> {
  const data = await env.HEALTH_STATE.get<Metrics>("metrics", "json");
  return data || {
    latencies: [],
    state: "NORMAL",
    secondaryState: "sleeping",
    error5xxCount: 0,
    requestCount: 0,
    lastUpdated: Date.now(),
    primaryHealthy: true,
    secondaryHealthy: true,
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

  if (isFatal || !metrics.primaryHealthy) {
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
  secondaryState: SecondaryState
): { primary: number; failover: number } {
  switch (state) {
    case "FAILOVER":
      return { primary: 0, failover: 100 };
    case "BALANCE":
      if (secondaryState === "ready") {
        return { primary: 50, failover: 50 };
      }
      return { primary: 100, failover: 0 };
    case "WARMUP":
    case "NORMAL":
    default:
      return { primary: 100, failover: 0 };
  }
}

// === バックエンドヘルスチェック（リアルタイム） ===

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
  console.log(`[HealthCheck] Checking: ${healthUrl}`);

  try {
    const response = await fetch(healthUrl, {
      method: "GET",
      signal: AbortSignal.timeout(3000),
    });
    const latencyMs = Date.now() - startTime;
    console.log(`[HealthCheck] ${healthUrl} -> ${response.status} (${latencyMs}ms)`);

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
    console.error(`[HealthCheck] ${healthUrl} -> Error (${errorType}):`, error);
    return { healthy: false, error: errorType, latencyMs };
  }
}

function classifyError(error: unknown): HealthCheckError {
  if (!(error instanceof Error)) {
    return "unknown";
  }

  const message = error.message.toLowerCase();
  const name = error.name;

  // タイムアウト
  if (name === "TimeoutError" || message.includes("timeout") || message.includes("aborted")) {
    return "timeout";
  }

  // DNS解決失敗（workerdの内部エラーも含む）
  if (
    message.includes("dns") ||
    message.includes("enotfound") ||
    message.includes("getaddrinfo") ||
    message.includes("name resolution") ||
    message.includes("internal error")  // workerd内部エラー（DNS解決失敗時に発生）
  ) {
    return "dns_failure";
  }

  // 接続拒否
  if (
    message.includes("econnrefused") ||
    message.includes("connection refused") ||
    message.includes("network connection lost")
  ) {
    return "connection_refused";
  }

  // SSL/TLSエラー（具体的なSSLエラーのみ）
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

// === Secondary Server 起動 ===

async function wakeSecondary(env: Env): Promise<void> {
  const metrics = await getMetrics(env);

  if (metrics.secondaryState === "ready") {
    return;
  }

  console.log("Waking up Secondary...");
  metrics.secondaryState = "waking";
  await env.HEALTH_STATE.put("metrics", JSON.stringify(metrics), {
    expirationTtl: 3600,
  });

  try {
    const response = await fetch(`${env.SECONDARY_API_URL}/health`, {
      method: "GET",
      signal: AbortSignal.timeout(60000), // コールドスタート待ち
    });

    if (response.ok) {
      metrics.secondaryState = "ready";
      metrics.secondaryHealthy = true;
      console.log("Secondary is ready");
    } else {
      metrics.secondaryState = "sleeping";
      metrics.secondaryHealthy = false;
    }
  } catch (error) {
    console.error("Failed to wake Secondary:", error);
    metrics.secondaryState = "sleeping";
    metrics.secondaryHealthy = false;
  }

  await env.HEALTH_STATE.put("metrics", JSON.stringify(metrics), {
    expirationTtl: 3600,
  });
}

// === スケジュールヘルスチェック ===

async function performScheduledHealthCheck(env: Env): Promise<void> {
  const metrics = await getMetrics(env);

  // Primaryヘルスチェック（これがウォーム維持にもなる）
  try {
    const response = await fetch(`${env.PRIMARY_API_URL}/health`, {
      method: "GET",
      signal: AbortSignal.timeout(5000),
    });

    if (response.ok) {
      metrics.primaryHealthy = true;
      // FAILOVERから復旧
      if (metrics.state === "FAILOVER") {
        metrics.state = "NORMAL";
        metrics.latencies = []; // リセット
      }
    } else {
      metrics.primaryHealthy = false;
      metrics.state = "FAILOVER";
    }
  } catch (error) {
    console.error("Primary health check failed:", error);
    metrics.primaryHealthy = false;
    metrics.state = "FAILOVER";
  }

  // Secondaryが起動中なら状態確認
  if (metrics.secondaryState === "waking" || metrics.state === "BALANCE" || metrics.state === "FAILOVER") {
    try {
      const response = await fetch(`${env.SECONDARY_API_URL}/health`, {
        method: "GET",
        signal: AbortSignal.timeout(5000),
      });

      if (response.ok) {
        metrics.secondaryState = "ready";
        metrics.secondaryHealthy = true;
      } else {
        metrics.secondaryHealthy = false;
      }
    } catch (error) {
      // Secondaryがスリープ中は失敗しても問題ない
      if (metrics.state === "FAILOVER") {
        console.log("Secondary not responding in FAILOVER state, attempting wake...");
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
  // 1. Try Supabase JWKS first
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
  } catch (supabaseError) {
    console.log("Supabase JWT verification failed, trying OAuth Mock Server...");
  }

  // 2. Try OAuth Mock Server JWKS (development)
  if (env.OAUTH_JWKS_URL) {
    try {
      const oauthJwks = jose.createRemoteJWKSet(new URL(env.OAUTH_JWKS_URL));
      // OAuth Mock Server doesn't set issuer claim, so skip issuer verification
      const { payload } = await jose.jwtVerify(token, oauthJwks);

      const userId = payload.sub;
      if (!userId) {
        return null;
      }

      console.log("JWT verified via OAuth Mock Server");
      return { userId, type: "jwt" };
    } catch (oauthError) {
      console.error("OAuth Mock Server JWT verification also failed:", oauthError);
    }
  }

  return null;
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
    const cacheReadStart = Date.now();
    const cached = await env.API_KEY_CACHE.get<ApiKeyCacheEntry>(cacheKey, "json");
    const cacheReadLatency = Date.now() - cacheReadStart;

    if (cached) {
      const age = Date.now() - cached.cachedAt;
      const totalLatency = Date.now() - startTime;
      if (age < API_KEY_CACHE_MAX_AGE_MS) {
        // キャッシュヒット（有効期限内）
        console.log(`[APIKey] Cache HIT | total: ${totalLatency}ms, kv_read: ${cacheReadLatency}ms, age: ${Math.round(age / 1000)}s`);
        return { userId: cached.userId, type: "api_key" };
      }
      // ソフト有効期限切れ → バックグラウンドで再検証を行うが、今は古いキャッシュを使用
      console.log(`[APIKey] Cache SOFT-EXPIRED | total: ${totalLatency}ms, kv_read: ${cacheReadLatency}ms, age: ${Math.round(age / 1000)}s`);
      return { userId: cached.userId, type: "api_key" };
    }
  } catch (cacheError) {
    console.error("[APIKey] Cache read error:", cacheError);
    // キャッシュエラーは無視して直接検証へ
  }

  // 2. キャッシュミス → Supabase RPCで検証
  console.log("[APIKey] Cache MISS, validating via Supabase RPC...");
  try {
    const rpcStart = Date.now();
    const response = await fetch(
      `${env.SUPABASE_URL}/rest/v1/rpc/validate_api_key`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          apikey: env.SUPABASE_ANON_KEY,
          Authorization: `Bearer ${env.SUPABASE_ANON_KEY}`,
        },
        body: JSON.stringify({ p_key: apiKey }),
      }
    );
    const rpcLatency = Date.now() - rpcStart;

    if (!response.ok) {
      const totalLatency = Date.now() - startTime;
      console.log(`[APIKey] Validation FAILED (HTTP ${response.status}) | total: ${totalLatency}ms, rpc: ${rpcLatency}ms`);
      return null;
    }

    const result: ApiKeyValidationResult = await response.json();
    if (!result || !result.valid || !result.user_id) {
      const totalLatency = Date.now() - startTime;
      console.log(`[APIKey] Validation FAILED (${result?.error || 'no user_id'}) | total: ${totalLatency}ms, rpc: ${rpcLatency}ms`);
      return null;
    }

    const userId = result.user_id;

    // 3. 検証成功 → KVにキャッシュ
    try {
      const cacheWriteStart = Date.now();
      const cacheEntry: ApiKeyCacheEntry = {
        userId,
        cachedAt: Date.now(),
      };
      await env.API_KEY_CACHE.put(cacheKey, JSON.stringify(cacheEntry), {
        expirationTtl: API_KEY_CACHE_TTL_SECONDS,
      });
      const cacheWriteLatency = Date.now() - cacheWriteStart;
      const totalLatency = Date.now() - startTime;
      console.log(`[APIKey] Validation OK + Cached | total: ${totalLatency}ms, rpc: ${rpcLatency}ms, kv_write: ${cacheWriteLatency}ms`);
    } catch (cacheWriteError) {
      const totalLatency = Date.now() - startTime;
      console.error(`[APIKey] Cache write error (total: ${totalLatency}ms):`, cacheWriteError);
      // キャッシュ書き込みエラーは無視
    }

    return { userId, type: "api_key" };
  } catch (error) {
    const totalLatency = Date.now() - startTime;
    console.error(`[APIKey] Verification error (total: ${totalLatency}ms):`, error);
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
 * key_hash を知っている者のみが無効化できる（=正当な削除者）。
 *
 * Request body: { key_hash: string }
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
