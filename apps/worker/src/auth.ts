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

// === JWT API Key 検証 (Go Server JWKS) ===

async function verifyApiKey(apiKey: string, env: Env): Promise<AuthResult | null> {
  try {
    // Strip mpt_ prefix to get the JWT
    const jwt = apiKey.slice(4);

    const jwks = jose.createRemoteJWKSet(new URL(env.API_SERVER_JWKS_URL));
    const { payload } = await jose.jwtVerify(jwt, jwks);

    if (!payload.sub) return null;

    console.log("[Auth] API Key JWT verified");
    return { userId: payload.sub, type: "api_key" };
  } catch (error) {
    console.error("[Auth] API Key verification failed:", error);
    return null;
  }
}

/**
 * Go Server → Worker のサーバー間認証 (X-Gateway-Secret ヘッダー)
 */
export function authenticateGateway(request: Request, env: Env): boolean {
  const secret = request.headers.get("X-Gateway-Secret");
  return !!secret && secret === env.GATEWAY_SECRET;
}
