/**
 * /v1/modules 関連ルート
 *
 * GET  /modules            → list_modules_with_tools (public)
 * GET  /modules/config     → get_module_config (auth, ?module= クエリ)
 * PUT  /modules/:name/tools       → upsert_tool_settings (auth)
 * PUT  /modules/:name/description → upsert_module_description (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const modules = new Hono<{ Bindings: Bindings }>();

// GET /modules — list_modules_with_tools (public, no auth)
modules.get("/", async (c) => {
  return forwardToPostgREST(c.env, "list_modules_with_tools", {});
});

// GET /modules/config — get_module_config (auth required)
modules.get("/config", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const params: Record<string, unknown> = { p_user_id: auth.userId };
  const moduleName = c.req.query("module");
  if (moduleName) params.p_module_name = moduleName;

  return forwardToPostgREST(c.env, "get_module_config", params);
});

// PUT /modules/:name/tools — upsert_tool_settings (auth required)
modules.put("/:name/tools", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{
    enabled_tools: string[];
    disabled_tools: string[];
  }>();

  return forwardToPostgREST(c.env, "upsert_tool_settings", {
    p_user_id: auth.userId,
    p_module_name: c.req.param("name"),
    p_enabled_tools: body.enabled_tools,
    p_disabled_tools: body.disabled_tools,
  });
});

// PUT /modules/:name/description — upsert_module_description (auth required)
modules.put("/:name/description", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{ description: string }>();

  return forwardToPostgREST(c.env, "upsert_module_description", {
    p_user_id: auth.userId,
    p_module_name: c.req.param("name"),
    p_description: body.description,
  });
});

export { modules };
