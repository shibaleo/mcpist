/**
 * MCPist API Gateway - Cloudflare Worker (Hono)
 *
 * 責務:
 * 1. JWT / API Key 認証
 * 2. /mcp/* → Go Server プロキシ
 * 3. /rpc/* → PostgREST プロキシ（p_user_id 自動注入）
 * 4. OAuth メタデータ配信
 */

import { Hono } from "hono";
import { cors } from "hono/cors";
import { secureHeaders } from "hono/secure-headers";
import type { Env } from "./types";
import { handleHealthCheck, performScheduledHealthCheck } from "./health";
import { handleInvalidateApiKey } from "./auth";
import { handleRpcProxy } from "./rpc-proxy";
import { handleMcpProxy } from "./mcp-proxy";
import {
  handleOAuthProtectedResourceMetadata,
  handleOAuthAuthorizationServerMetadata,
} from "./oauth-metadata";

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

// --- Routes ---

app.get("/health", (c) => handleHealthCheck(c.env));

// Internal endpoints
app.post("/internal/invalidate-api-key", (c) => {
  const secret = c.req.header("X-Internal-Secret");
  if (!secret || secret !== c.env.INTERNAL_SECRET) {
    return c.json({ error: "Unauthorized" }, 401);
  }
  return handleInvalidateApiKey(c.req.raw, c.env);
});

// OAuth metadata (RFC 9728 / 8414)
app.get("/.well-known/oauth-protected-resource", (c) =>
  handleOAuthProtectedResourceMetadata(c.req.raw, c.env));
app.get("/mcp/.well-known/oauth-protected-resource", (c) =>
  handleOAuthProtectedResourceMetadata(c.req.raw, c.env));
app.get("/.well-known/oauth-authorization-server", (c) =>
  handleOAuthAuthorizationServerMetadata(c.env));
app.get("/mcp/.well-known/oauth-authorization-server", (c) =>
  handleOAuthAuthorizationServerMetadata(c.env));

// RPC Proxy: Console → Worker → PostgREST
app.post("/rpc/:name", (c) => {
  const url = new URL(c.req.url);
  return handleRpcProxy(c.req.raw, url, c.env, c.executionCtx);
});

// MCP Proxy: MCP Client → Worker → Go Server
app.all("/mcp/*", (c) => {
  const url = new URL(c.req.url);
  return handleMcpProxy(c.req.raw, url, c.env, c.executionCtx);
});

export default {
  fetch: app.fetch,
  async scheduled(_event: ScheduledEvent, env: Env, ctx: ExecutionContext): Promise<void> {
    ctx.waitUntil(performScheduledHealthCheck(env));
  },
};
