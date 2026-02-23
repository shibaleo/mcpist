/**
 * v1 API サブルーター
 *
 * app.route("/v1", v1) でメインルーターにマウントされる。
 */

import { Hono } from "hono";
import type { Env } from "../types";
import { handleMcpProxy } from "./mcp-proxy";
import {
  handleOAuthProtectedResourceMetadata,
  handleOAuthAuthorizationServerMetadata,
} from "./oauth-metadata";

// Route handlers
import { modules } from "./routes/modules";
import { plans } from "./routes/plans";
import { me } from "./routes/me";
import { admin } from "./routes/admin";
import { stripe } from "./routes/stripe";
import { oauth } from "./routes/oauth";

type Bindings = Env;

const v1 = new Hono<{ Bindings: Bindings }>();

// MCP-scoped OAuth metadata (RFC 9728 / 8414)
v1.get("/mcp/.well-known/oauth-protected-resource", (c) =>
  handleOAuthProtectedResourceMetadata(c.req.raw, c.env));
v1.get("/mcp/.well-known/oauth-authorization-server", (c) =>
  handleOAuthAuthorizationServerMetadata(c.env));

// RESTful resource routes
v1.route("/modules", modules);
v1.route("/plans", plans);
v1.route("/me", me);
v1.route("/admin", admin);
v1.route("/stripe", stripe);
v1.route("/oauth", oauth);

// MCP Proxy: MCP Client → Worker → Go Server
v1.all("/mcp/*", (c) => {
  const url = new URL(c.req.url);
  return handleMcpProxy(c.req.raw, url, c.env, c.executionCtx);
});

export { v1 };
