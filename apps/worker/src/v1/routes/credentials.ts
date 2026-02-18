/**
 * /v1/credentials 関連ルート
 *
 * GET    /credentials          → list_credentials (auth)
 * GET    /credentials/:module  → get_credential (gateway auth, Go Server 用)
 * PUT    /credentials          → upsert_credential (auth / gateway auth)
 * DELETE /credentials/:module  → delete_credential (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate, authenticateGateway } from "../../auth";
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

// GET /credentials/:module — get_credential (Go Server → Worker, gateway auth)
credentials.get("/:module", async (c) => {
  if (!authenticateGateway(c.req.raw, c.env)) {
    return jsonResponse({ error: "Unauthorized" }, 401);
  }

  const userId = c.req.query("user_id");
  if (!userId) return jsonResponse({ error: "Missing user_id" }, 400);

  return forwardToPostgREST(c.env, "get_credential", {
    p_user_id: userId,
    p_module: c.req.param("module"),
  });
});

// PUT /credentials — upsert_credential (auth or gateway auth)
credentials.put("/", async (c) => {
  // Support both JWT/API key auth (Console) and gateway auth (Go Server)
  const auth = await authenticate(c.req.raw, c.env);
  const isGateway = authenticateGateway(c.req.raw, c.env);
  if (!auth && !isGateway) return jsonResponse({ error: "Unauthorized" }, 401);

  const body = await c.req.json<{
    user_id?: string;
    module: string;
    credentials: Record<string, unknown>;
  }>();

  const userId = auth ? auth.userId : body.user_id;
  if (!userId) return jsonResponse({ error: "Missing user_id" }, 400);

  return forwardToPostgREST(c.env, "upsert_credential", {
    p_user_id: userId,
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
