import * as jose from "jose";
import type { Env, AuthResult } from "./types";
import { callGoServer } from "./v1/go-server";


export async function authenticate(
  request: Request,
  env: Env
): Promise<AuthResult | null> {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader) return null;

  if (authHeader.startsWith("Bearer ")) {
    const token = authHeader.slice(7);

    // API Key (mpt_xxx format) — verify via Go Server JWKS
    if (token.startsWith("mpt_")) {
      return await verifyApiKey(token, env);
    }

    // JWT (Clerk issued)
    return await verifyClerkJWT(token, env);
  }

  return null;
}

// === Clerk JWT 検証 ===

async function verifyClerkJWT(token: string, env: Env): Promise<AuthResult | null> {
  try {
    const jwks = jose.createRemoteJWKSet(new URL(env.CLERK_JWKS_URL));
    const { payload } = await jose.jwtVerify(token, jwks);

    if (!payload.sub) return null;

    // Clerk JWT custom claims may include email
    const email = typeof payload.email === "string" ? payload.email : undefined;
    console.log("[Auth] Clerk JWT verified");
    return { userId: payload.sub, email, type: "jwt" };
  } catch (error) {
    console.error("[Auth] Clerk JWT verification failed:", error);
    return null;
  }
}

// === JWT API Key 検証 (Go Server JWKS + DB 照合 + キャッシュ) ===

interface ApiKeyStatusResponse {
  active: boolean;
  key_id: string;
  user_id: string;
  expires_at?: string | null;
}

// API Key ステータスのインメモリキャッシュ (TTL 5分)
const API_KEY_CACHE_TTL_MS = 5 * 60 * 1000;
const apiKeyStatusCache = new Map<string, { active: boolean; cachedAt: number }>();

export function getCachedKeyStatus(keyId: string): boolean | null {
  const entry = apiKeyStatusCache.get(keyId);
  if (!entry) return null;
  if (Date.now() - entry.cachedAt > API_KEY_CACHE_TTL_MS) {
    apiKeyStatusCache.delete(keyId);
    return null;
  }
  return entry.active;
}

export function setCachedKeyStatus(keyId: string, active: boolean): void {
  apiKeyStatusCache.set(keyId, { active, cachedAt: Date.now() });
}

/** Invalidate cached key status (called on revoke via cache bust API) */
export function invalidateApiKeyCache(keyId: string): void {
  apiKeyStatusCache.delete(keyId);
}

async function verifyApiKey(apiKey: string, env: Env): Promise<AuthResult | null> {
  try {
    // Strip mpt_ prefix to get the JWT
    const jwt = apiKey.slice(4);

    const jwks = jose.createRemoteJWKSet(new URL(env.SERVER_JWKS_URL));
    const { payload } = await jose.jwtVerify(jwt, jwks);

    if (!payload.sub) return null;

    // Extract key_id from JWT claims (kid = api_keys.id)
    const keyId = typeof payload.kid === "string" ? payload.kid : undefined;
    if (!keyId) {
      console.error("[Auth] API Key JWT missing kid claim");
      return null;
    }

    // Check cache first
    const cached = getCachedKeyStatus(keyId);
    if (cached !== null) {
      if (!cached) {
        console.warn("[Auth] API Key revoked (cached):", keyId);
        return null;
      }
      console.log("[Auth] API Key JWT verified (cache hit)");
      return { userId: payload.sub, type: "api_key" };
    }

    // Cache miss — verify via Go Server internal API
    const status = await callGoServer<ApiKeyStatusResponse>(
      env,
      "GET",
      `/v1/internal/apikeys/${keyId}/status`
    );

    setCachedKeyStatus(keyId, status.active);

    if (!status.active) {
      console.warn("[Auth] API Key revoked or inactive:", keyId);
      return null;
    }

    console.log("[Auth] API Key JWT verified + DB check passed");
    return { userId: payload.sub, type: "api_key" };
  } catch (error) {
    console.error("[Auth] API Key verification failed:", error);
    return null;
  }
}
