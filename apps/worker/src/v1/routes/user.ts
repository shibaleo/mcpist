/**
 * /v1/user 関連ルート
 *
 * GET  /user/context     → get_user_context (auth)
 * GET  /user/usage       → get_usage (auth, ?start=&end= クエリ)
 * GET  /user/stripe      → get_stripe_customer_id (auth)
 * PUT  /user/stripe      → link_stripe_customer (auth)
 * PUT  /user/settings    → update_settings (auth)
 * POST /user/onboarding  → complete_user_onboarding (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const user = new Hono<{ Bindings: Bindings }>();

// GET /user/context — get_user_context
user.get("/context", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "get_user_context", {
    p_user_id: auth.userId,
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
