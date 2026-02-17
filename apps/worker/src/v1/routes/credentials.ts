/**
 * /v1/credentials 関連ルート
 *
 * GET    /credentials          → list_credentials (auth)
 * PUT    /credentials          → upsert_credential (auth)
 * DELETE /credentials/:module  → delete_credential (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const credentials = new Hono<{ Bindings: Bindings }>();

// GET /credentials — list_credentials
credentials.get("/", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "list_credentials", {
    p_user_id: auth.userId,
  });
});

// PUT /credentials — upsert_credential
credentials.put("/", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{
    module: string;
    credentials: Record<string, unknown>;
  }>();

  return forwardToPostgREST(c.env, "upsert_credential", {
    p_user_id: auth.userId,
    p_module: body.module,
    p_credentials: body.credentials,
  });
});

// DELETE /credentials/:module — delete_credential
credentials.delete("/:module", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "delete_credential", {
    p_user_id: auth.userId,
    p_module: c.req.param("module"),
  });
});

export { credentials };
