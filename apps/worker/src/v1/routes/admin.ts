/**
 * /v1/admin 関連ルート（admin ロール必須）
 *
 * GET    /admin/oauth/apps            → list_oauth_apps (Go Server)
 * PUT    /admin/oauth/apps/:provider  → upsert_oauth_app (Go Server)
 * DELETE /admin/oauth/apps/:provider  → delete_oauth_app (Go Server)
 * GET    /admin/oauth/consents        → list_all_oauth_consents (Go Server)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToGoServer } from "../go-server";
import type { GatewayTokenClaims } from "../../gateway-token";

type Bindings = Env;

const admin = new Hono<{ Bindings: Bindings }>();

type AuthOk = { userId: string; email?: string; type: "jwt" | "api_key" };

function buildClaims(auth: AuthOk): GatewayTokenClaims {
  if (auth.type === "api_key") {
    return { user_id: auth.userId, email: auth.email };
  }
  return { clerk_id: auth.userId, email: auth.email };
}

/** 認証 + claims を付けて Go Server にプロキシ */
async function requireAuthAndProxy(
  req: Request,
  env: Env,
  method: string,
  path: string,
  body?: string | null
): Promise<Response> {
  const auth = await authenticate(req, env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToGoServer(env, method, path, buildClaims(auth), body);
}

// GET /admin/oauth/apps — list OAuth apps
admin.get("/oauth/apps", async (c) => {
  return requireAuthAndProxy(c.req.raw, c.env, "GET", "/v1/admin/oauth/apps");
});

// PUT /admin/oauth/apps/:provider — upsert OAuth app
admin.put("/oauth/apps/:provider", async (c) => {
  const provider = c.req.param("provider");
  return requireAuthAndProxy(
    c.req.raw,
    c.env,
    "PUT",
    `/v1/admin/oauth/apps/${provider}`,
    await c.req.text()
  );
});

// DELETE /admin/oauth/apps/:provider — delete OAuth app
admin.delete("/oauth/apps/:provider", async (c) => {
  const provider = c.req.param("provider");
  return requireAuthAndProxy(c.req.raw, c.env, "DELETE", `/v1/admin/oauth/apps/${provider}`);
});

// GET /admin/oauth/consents — list all OAuth consents
admin.get("/oauth/consents", async (c) => {
  return requireAuthAndProxy(c.req.raw, c.env, "GET", "/v1/admin/oauth/consents");
});

export { admin };
