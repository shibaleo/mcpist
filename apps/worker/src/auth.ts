import * as jose from "jose";
import type { Env, AuthResult } from "./types";


export async function authenticate(
  request: Request,
  env: Env
): Promise<AuthResult | null> {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader) return null;

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

// === JWT 検証 ===

async function verifyJWT(token: string, env: Env): Promise<AuthResult | null> {
  // 1. OAuth Server 発行トークン: /auth/v1/oauth/userinfo で検証
  try {
    const response = await fetch(`${env.SUPABASE_URL}/auth/v1/oauth/userinfo`, {
      headers: { Authorization: `Bearer ${token}` },
    });

    if (response.ok) {
      const userInfo = await response.json() as { sub?: string };
      if (userInfo.sub) {
        console.log("[Auth] Token verified via OAuth userinfo");
        return { userId: userInfo.sub, type: "jwt" };
      }
    }
  } catch (error) {
    console.error("[Auth] OAuth userinfo verification failed:", error);
  }

  // 2. 従来の Supabase Auth トークン: /auth/v1/user で検証
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

  // 3. フォールバック: JWT 署名検証
  try {
    const jwks = jose.createRemoteJWKSet(new URL(env.SUPABASE_JWKS_URL));
    const { payload } = await jose.jwtVerify(token, jwks, {
      issuer: `${env.SUPABASE_URL}/auth/v1`,
    });

    if (!payload.sub) return null;

    console.log("[Auth] Token verified via JWT signature");
    return { userId: payload.sub, type: "jwt" };
  } catch (error) {
    console.error("[Auth] JWT verification failed:", error);
    return null;
  }
}

// === API Key 検証 ===

interface LookupUserByKeyHashResult {
  valid: boolean;
  user_id?: string;
  error?: string;
}

interface ApiKeyCacheEntry {
  userId: string;
  cachedAt: number;
}

const API_KEY_CACHE_TTL_SECONDS = 86400;   // 1日（KV TTL）
const API_KEY_CACHE_MAX_AGE_MS = 3600000;  // 1時間（ソフト有効期限）

/**
 * API Key 検証（KV キャッシュ対応）
 *
 * フロー:
 * 1. SHA-256 ハッシュ計算
 * 2. KV キャッシュチェック
 * 3. キャッシュミス時: PostgREST RPC で検証
 * 4. 検証成功時: KV にキャッシュ
 */
async function verifyApiKey(apiKey: string, env: Env): Promise<AuthResult | null> {
  const startTime = Date.now();
  const keyHash = await hashApiKey(apiKey);

  // 1. KV キャッシュチェック
  try {
    const cached = await env.API_KEY_CACHE.get<ApiKeyCacheEntry>(keyHash, "json");
    if (cached) {
      const age = Date.now() - cached.cachedAt;
      if (age < API_KEY_CACHE_MAX_AGE_MS) {
        console.log(`[APIKey] Cache HIT | age: ${Math.round(age / 1000)}s`);
      } else {
        console.log(`[APIKey] Cache SOFT-EXPIRED | age: ${Math.round(age / 1000)}s`);
      }
      return { userId: cached.userId, type: "api_key" };
    }
  } catch (cacheError) {
    console.error("[APIKey] Cache read error:", cacheError);
  }

  // 2. キャッシュミス → PostgREST RPC で検証
  console.log("[APIKey] Cache MISS, validating via PostgREST RPC...");
  try {
    const response = await fetch(`${env.POSTGREST_URL}/rpc/lookup_user_by_key_hash`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${env.POSTGREST_API_KEY}`,
      },
      body: JSON.stringify({ p_key_hash: keyHash }),
    });

    if (!response.ok) {
      console.log(`[APIKey] Validation FAILED (HTTP ${response.status})`);
      return null;
    }

    const result: LookupUserByKeyHashResult = await response.json();
    if (!result?.valid || !result.user_id) {
      console.log(`[APIKey] Validation FAILED (${result?.error || "no user_id"})`);
      return null;
    }

    const userId = result.user_id;

    // 3. KV にキャッシュ
    try {
      await env.API_KEY_CACHE.put(
        keyHash,
        JSON.stringify({ userId, cachedAt: Date.now() } satisfies ApiKeyCacheEntry),
        { expirationTtl: API_KEY_CACHE_TTL_SECONDS }
      );
      console.log(`[APIKey] Validation OK + Cached | total: ${Date.now() - startTime}ms`);
    } catch (cacheWriteError) {
      console.error("[APIKey] Cache write error:", cacheWriteError);
    }

    return { userId, type: "api_key" };
  } catch (error) {
    console.error("[APIKey] Verification error:", error);
    return null;
  }
}

async function hashApiKey(apiKey: string): Promise<string> {
  const data = new TextEncoder().encode(apiKey);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  return Array.from(new Uint8Array(hashBuffer))
    .map(b => b.toString(16).padStart(2, "0"))
    .join("");
}