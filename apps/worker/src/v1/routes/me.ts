/**
 * /v1/me 関連ルート — Console 向け統合エンドポイント
 *
 * 認証済みユーザーの自分自身に関する操作をすべて /me 以下に集約。
 * Go Server の /v1/me/* REST エンドポイントにプロキシ。
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate, invalidateApiKeyCache } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToGoServer } from "../go-server";
import type { GatewayTokenClaims } from "../../gateway-token";

type Bindings = Env;
const me = new Hono<{ Bindings: Bindings }>();

/** 認証結果の型 */
type AuthOk = { userId: string; email?: string; type: "jwt" | "api_key" };

/** 認証を検証し結果を返す。失敗時は Response を返す */
async function requireAuth(
  req: Request,
  env: Env
): Promise<AuthOk | Response> {
  const auth = await authenticate(req, env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);
  return { userId: auth.userId, email: auth.email, type: auth.type };
}

/**
 * 認証タイプに応じた Gateway JWT claims を生成
 * - Clerk JWT  → clerk_id
 * - API Key    → user_id (mcpist internal UUID)
 */
export function buildClaims(auth: AuthOk): GatewayTokenClaims {
  if (auth.type === "api_key") {
    return { user_id: auth.userId, email: auth.email };
  }
  return { clerk_id: auth.userId, email: auth.email };
}

/** Go Server にプロキシするヘルパー */
function proxy(
  env: Env,
  auth: AuthOk,
  method: string,
  path: string,
  body?: string | null
) {
  return forwardToGoServer(env, method, `/v1/me${path}`, buildClaims(auth), body);
}

// ── Register ────────────────────────────────────────────────────

me.post("/register", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  if (!r.email) return jsonResponse({ error: "email is required for registration" }, 400);
  return forwardToGoServer(c.env, "POST", "/v1/me/register", buildClaims(r));
});

// ── Profile ─────────────────────────────────────────────────────

me.get("/profile", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return forwardToGoServer(c.env, "GET", "/v1/me/profile", buildClaims(r));
});

me.put("/settings", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "PUT", "/settings", await c.req.text());
});

me.post("/onboarding", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "POST", "/onboarding", await c.req.text());
});

// ── Usage ───────────────────────────────────────────────────────

me.get("/usage", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const qs = new URL(c.req.url).search;
  return proxy(c.env, r, "GET", `/usage${qs}`);
});

// ── Stripe ──────────────────────────────────────────────────────

me.get("/stripe", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "GET", "/stripe");
});

me.put("/stripe", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "PUT", "/stripe", await c.req.text());
});

// ── Credentials ─────────────────────────────────────────────────

me.get("/credentials", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "GET", "/credentials");
});

me.put("/credentials/:module", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const module = c.req.param("module");
  return proxy(c.env, r, "PUT", `/credentials/${module}`, await c.req.text());
});

me.delete("/credentials/:module", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const module = c.req.param("module");
  return proxy(c.env, r, "DELETE", `/credentials/${module}`);
});

// ── API Keys ────────────────────────────────────────────────────

me.get("/apikeys", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "GET", "/apikeys");
});

me.post("/apikeys", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "POST", "/apikeys", await c.req.text());
});

me.delete("/apikeys/:id", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const id = c.req.param("id");
  // Bust cache immediately so subsequent requests reject this key
  invalidateApiKeyCache(id);
  return proxy(c.env, r, "DELETE", `/apikeys/${id}`);
});

// ── Prompts ─────────────────────────────────────────────────────

me.get("/prompts", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const query = c.req.query("module") ? `?module=${c.req.query("module")}` : "";
  return proxy(c.env, r, "GET", `/prompts${query}`);
});

me.get("/prompts/:id", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const id = c.req.param("id");
  return proxy(c.env, r, "GET", `/prompts/${id}`);
});

me.post("/prompts", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "POST", "/prompts", await c.req.text());
});

me.put("/prompts/:id", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const id = c.req.param("id");
  return proxy(c.env, r, "PUT", `/prompts/${id}`, await c.req.text());
});

me.delete("/prompts/:id", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const id = c.req.param("id");
  return proxy(c.env, r, "DELETE", `/prompts/${id}`);
});

// ── Module Config ───────────────────────────────────────────────

me.get("/modules/config", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "GET", "/modules/config");
});

me.put("/modules/:name/tools", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const name = c.req.param("name");
  return proxy(c.env, r, "PUT", `/modules/${name}/tools`, await c.req.text());
});

me.put("/modules/:name/description", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const name = c.req.param("name");
  return proxy(c.env, r, "PUT", `/modules/${name}/description`, await c.req.text());
});

// ── OAuth Consents ──────────────────────────────────────────────

me.get("/oauth/consents", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  return proxy(c.env, r, "GET", "/oauth/consents");
});

me.delete("/oauth/consents/:id", async (c) => {
  const r = await requireAuth(c.req.raw, c.env);
  if (r instanceof Response) return r;
  const id = c.req.param("id");
  return proxy(c.env, r, "DELETE", `/oauth/consents/${id}`);
});

export { me };
