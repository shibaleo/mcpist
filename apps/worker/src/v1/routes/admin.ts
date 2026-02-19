/**
 * /v1/admin 関連ルート（admin ロール必須）
 *
 * PUT    /admin/oauth/apps/:provider  → upsert_oauth_app (Go Server)
 * GET    /admin/oauth/consents        → list_all_oauth_consents (Go Server)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToGoServer } from "../go-server";

type Bindings = Env;

const admin = new Hono<{ Bindings: Bindings }>();

/** 認証 + ヘッダーを付けて Go Server にプロキシ */
async function requireAuthAndProxy(
  req: Request,
  env: Env,
  method: string,
  path: string,
  body?: string | null
): Promise<Response> {
  const auth = await authenticate(req, env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const header: Record<string, string> = auth.type === "api_key"
    ? { "X-User-ID": auth.userId }
    : { "X-Clerk-ID": auth.userId };

  return forwardToGoServer(env, method, path, header, body);
}

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

// GET /admin/oauth/consents — list all OAuth consents
admin.get("/oauth/consents", async (c) => {
  return requireAuthAndProxy(c.req.raw, c.env, "GET", "/v1/admin/oauth/consents");
});

export { admin };
