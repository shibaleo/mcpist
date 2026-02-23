/**
 * MCPist API Gateway - Cloudflare Worker (Hono)
 *
 * 責務:
 * 1. Clerk JWT / JWT API Key 認証
 * 2. /v1/mcp/* → Go Server プロキシ
 * 3. /v1/me/* → Go Server REST プロキシ
 * 4. OAuth メタデータ配信
 */

import { Hono } from "hono";
import { cors } from "hono/cors";
import { secureHeaders } from "hono/secure-headers";
import type { Env } from "./types";
import { handleHealthCheck, performScheduledHealthCheck } from "./health";
import {
  handleOAuthProtectedResourceMetadata,
  handleOAuthAuthorizationServerMetadata,
} from "./v1/oauth-metadata";
import { handleOpenApiSpec } from "./openapi";
import { getJwksResponse } from "./gateway-token";
import { v1 } from "./v1";

type Bindings = Env;

const app = new Hono<{ Bindings: Bindings }>();

// --- Global Error Handler ---

app.onError((err, c) => {
  console.error(`[Worker] Unhandled error on ${c.req.method} ${c.req.path}:`, err);
  return c.json(
    { error: "Internal server error", message: err.message },
    500
  );
});

// --- Middleware ---

app.use("*", cors());
app.use("*", secureHeaders({
  contentSecurityPolicy: {
    defaultSrc: ["'none'"],
    frameAncestors: ["'none'"],
  },
}));

// --- Unversioned Routes ---

app.get("/health", (c) => handleHealthCheck(c.env));

app.get("/openapi.json", (c) => handleOpenApiSpec(c.req.raw));

// Root-level OAuth metadata (RFC 9728 / 8414)
app.get("/.well-known/oauth-protected-resource", (c) =>
  handleOAuthProtectedResourceMetadata(c.req.raw, c.env));
app.get("/.well-known/oauth-authorization-server", (c) =>
  handleOAuthAuthorizationServerMetadata(c.env));

// Gateway JWKS (Go Server がこの公開鍵で Gateway JWT を検証)
app.get("/.well-known/jwks.json", (c) =>
  getJwksResponse(c.env.GATEWAY_SIGNING_KEY));

// --- Versioned Routes ---

app.route("/v1", v1);

export default {
  fetch: app.fetch,
  async scheduled(_event: ScheduledEvent, env: Env, ctx: ExecutionContext): Promise<void> {
    ctx.waitUntil(performScheduledHealthCheck(env));
  },
};
