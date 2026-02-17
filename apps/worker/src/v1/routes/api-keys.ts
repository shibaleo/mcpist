/**
 * /v1/api-keys 関連ルート
 *
 * GET    /api-keys      → list_api_keys (auth)
 * POST   /api-keys      → generate_api_key (auth)
 * DELETE /api-keys/:id  → revoke_api_key (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const apiKeys = new Hono<{ Bindings: Bindings }>();

// GET /api-keys — list_api_keys
apiKeys.get("/", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "list_api_keys", {
    p_user_id: auth.userId,
  });
});

// POST /api-keys — generate_api_key
apiKeys.post("/", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{
    display_name: string;
    expires_at?: string;
  }>();

  return forwardToPostgREST(c.env, "generate_api_key", {
    p_user_id: auth.userId,
    p_display_name: body.display_name,
    ...(body.expires_at && { p_expires_at: body.expires_at }),
  });
});

// DELETE /api-keys/:id — revoke_api_key
apiKeys.delete("/:id", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "revoke_api_key", {
    p_user_id: auth.userId,
    p_key_id: c.req.param("id"),
  });
});

export { apiKeys };
