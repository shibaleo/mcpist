/**
 * /v1/admin 関連ルート（admin ロール必須）
 *
 * GET    /admin/oauth/apps             → list_oauth_apps
 * PUT    /admin/oauth/apps             → upsert_oauth_app
 * DELETE /admin/oauth/apps/:provider   → delete_oauth_app
 * GET    /admin/oauth/consents         → list_all_oauth_consents
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST, callPostgRESTRpc } from "../postgrest";

type Bindings = Env;

const admin = new Hono<{ Bindings: Bindings }>();

/** admin ロールを検証する共通関数 */
async function requireAdmin(
  req: Request,
  env: Env
): Promise<{ userId: string } | Response> {
  const auth = await authenticate(req, env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const rows = await callPostgRESTRpc<{ role?: string }[]>(
    env,
    "get_user_context",
    { p_user_id: auth.userId }
  );
  const context = Array.isArray(rows) ? rows[0] : rows;
  if (context?.role !== "admin") {
    return jsonResponse({ error: "Forbidden" }, 403);
  }

  return { userId: auth.userId };
}

// GET /admin/oauth/apps — list_oauth_apps
admin.get("/oauth/apps", async (c) => {
  const result = await requireAdmin(c.req.raw, c.env);
  if (result instanceof Response) return result;

  return forwardToPostgREST(c.env, "list_oauth_apps", {});
});

// PUT /admin/oauth/apps — upsert_oauth_app
admin.put("/oauth/apps", async (c) => {
  const result = await requireAdmin(c.req.raw, c.env);
  if (result instanceof Response) return result;

  const body = await c.req.json<{
    provider: string;
    client_id: string;
    client_secret: string;
    redirect_uri: string;
    enabled: boolean;
  }>();

  return forwardToPostgREST(c.env, "upsert_oauth_app", {
    p_provider: body.provider,
    p_client_id: body.client_id,
    p_client_secret: body.client_secret,
    p_redirect_uri: body.redirect_uri,
    p_enabled: body.enabled,
  });
});

// DELETE /admin/oauth/apps/:provider — delete_oauth_app
admin.delete("/oauth/apps/:provider", async (c) => {
  const result = await requireAdmin(c.req.raw, c.env);
  if (result instanceof Response) return result;

  return forwardToPostgREST(c.env, "delete_oauth_app", {
    p_provider: c.req.param("provider"),
  });
});

// GET /admin/oauth/consents — list_all_oauth_consents
admin.get("/oauth/consents", async (c) => {
  const result = await requireAdmin(c.req.raw, c.env);
  if (result instanceof Response) return result;

  return forwardToPostgREST(c.env, "list_all_oauth_consents", {});
});

export { admin };
