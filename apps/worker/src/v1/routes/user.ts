/**
 * /v1/user 関連ルート
 *
 * GET  /user/context     → get_user_context (auth)
 * GET  /user/usage       → get_usage (auth, ?start=&end= クエリ)
 * POST /user/usage       → record_usage (gateway auth, Go Server 用)
 * GET  /user/stripe      → get_stripe_customer_id (auth)
 * PUT  /user/stripe      → link_stripe_customer (auth)
 * PUT  /user/settings    → update_settings (auth)
 * POST /user/onboarding  → complete_user_onboarding (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate, authenticateGateway } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const user = new Hono<{ Bindings: Bindings }>();

// GET /user/context — get_user_context (auth or gateway auth)
user.get("/context", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  const isGateway = authenticateGateway(c.req.raw, c.env);
  if (!auth && !isGateway) return jsonResponse({ error: "Unauthorized" }, 401);

  const userId = auth ? auth.userId : c.req.query("user_id");
  if (!userId) return jsonResponse({ error: "Missing user_id" }, 400);

  return forwardToPostgREST(c.env, "get_user_context", {
    p_user_id: userId,
  });
});

// GET /user/usage — get_usage
user.get("/usage", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "get_usage", {
    p_user_id: auth.userId,
    p_start_date: c.req.query("start"),
    p_end_date: c.req.query("end"),
  });
});

// POST /user/usage — record_usage (Go Server → Worker, gateway auth)
user.post("/usage", async (c) => {
  if (!authenticateGateway(c.req.raw, c.env)) {
    return jsonResponse({ error: "Unauthorized" }, 401);
  }

  const body = await c.req.json<{
    user_id: string;
    meta_tool: string;
    request_id: string;
    details: Array<{ task_id?: string; module: string; tool: string }>;
  }>();

  return forwardToPostgREST(c.env, "record_usage", {
    p_user_id: body.user_id,
    p_meta_tool: body.meta_tool,
    p_request_id: body.request_id,
    p_details: body.details,
  });
});

// GET /user/stripe — get_stripe_customer_id
user.get("/stripe", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "get_stripe_customer_id", {
    p_user_id: auth.userId,
  });
});

// PUT /user/stripe — link_stripe_customer
user.put("/stripe", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{ stripe_customer_id: string }>();

  return forwardToPostgREST(c.env, "link_stripe_customer", {
    p_user_id: auth.userId,
    p_stripe_customer_id: body.stripe_customer_id,
  });
});

// PUT /user/settings — update_settings
user.put("/settings", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{ settings: Record<string, unknown> }>();

  return forwardToPostgREST(c.env, "update_settings", {
    p_user_id: auth.userId,
    p_settings: body.settings,
  });
});

// POST /user/onboarding — complete_user_onboarding
user.post("/onboarding", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{ event_id: string }>();

  return forwardToPostgREST(c.env, "complete_user_onboarding", {
    p_user_id: auth.userId,
    p_event_id: body.event_id,
  });
});

export { user };
