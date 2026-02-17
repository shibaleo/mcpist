/**
 * MCPist API Gateway - Cloudflare Worker (Hono)
 *
 * 責務:
 * 1. JWT / API Key 認証
 * 2. /v1/mcp/* → Go Server プロキシ
 * 3. /v1/rpc/* → PostgREST プロキシ（p_user_id 自動注入）
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
import { v1 } from "./v1";

type Bindings = Env;

const app = new Hono<{ Bindings: Bindings }>();

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

// --- Versioned Routes ---

app.route("/v1", v1);

export default {
  fetch: app.fetch,
  async scheduled(_event: ScheduledEvent, env: Env, ctx: ExecutionContext): Promise<void> {
    ctx.waitUntil(performScheduledHealthCheck(env));
  },
};
