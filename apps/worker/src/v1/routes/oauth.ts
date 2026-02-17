/**
 * /v1/oauth 関連ルート
 *
 * GET    /oauth/apps/:provider/credentials → get_oauth_app_credentials (public)
 * GET    /oauth/consents                   → list_oauth_consents (auth)
 * DELETE /oauth/consents/:id               → revoke_oauth_consent (auth)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToPostgREST } from "../postgrest";

type Bindings = Env;

const oauth = new Hono<{ Bindings: Bindings }>();

// GET /oauth/apps/:provider/credentials — get_oauth_app_credentials (public)
oauth.get("/apps/:provider/credentials", async (c) => {
  return forwardToPostgREST(c.env, "get_oauth_app_credentials", {
    p_provider: c.req.param("provider"),
  });
});

// GET /oauth/consents — list_oauth_consents (auth required)
oauth.get("/consents", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "list_oauth_consents", {
    p_user_id: auth.userId,
  });
});

// DELETE /oauth/consents/:id — revoke_oauth_consent (auth required)
oauth.delete("/consents/:id", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  return forwardToPostgREST(c.env, "revoke_oauth_consent", {
    p_user_id: auth.userId,
    p_consent_id: c.req.param("id"),
  });
});

export { oauth };
