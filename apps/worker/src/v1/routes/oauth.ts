/**
 * /v1/oauth 関連ルート — OAuth App 公開情報エンドポイント
 *
 * OAuth 認可フローで必要な client_id / redirect_uri を返す。
 * Go Server の /v1/oauth/* REST エンドポイントにプロキシ。
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { authenticate } from "../../auth";
import { jsonResponse } from "../../http";
import { forwardToGoServer } from "../go-server";
import type { GatewayTokenClaims } from "../../gateway-token";

type Bindings = Env;
const oauth = new Hono<{ Bindings: Bindings }>();

// GET /v1/oauth/apps/:provider/credentials
oauth.get("/apps/:provider/credentials", async (c) => {
  const auth = await authenticate(c.req.raw, c.env);
  if (!auth) return jsonResponse({ error: "Unauthorized" }, 401);

  const provider = c.req.param("provider");
  const claims: GatewayTokenClaims =
    auth.type === "api_key"
      ? { user_id: auth.userId, email: auth.email }
      : { clerk_id: auth.userId, email: auth.email };

  return forwardToGoServer(
    c.env,
    "GET",
    `/v1/oauth/apps/${provider}/credentials`,
    claims
  );
});

export { oauth };
