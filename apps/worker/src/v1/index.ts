/**
 * v1 API サブルーター
 *
 * app.route("/v1", v1) でメインルーターにマウントされる。
 * RESTful ルーティングで各リソースを個別に定義。
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
import { apiKeys } from "./routes/api-keys";
import { prompts } from "./routes/prompts";
import { credentials } from "./routes/credentials";
import { user } from "./routes/user";
import { oauth } from "./routes/oauth";
import { admin } from "./routes/admin";

type Bindings = Env;

const v1 = new Hono<{ Bindings: Bindings }>();

// MCP-scoped OAuth metadata (RFC 9728 / 8414)
v1.get("/mcp/.well-known/oauth-protected-resource", (c) =>
  handleOAuthProtectedResourceMetadata(c.req.raw, c.env));
v1.get("/mcp/.well-known/oauth-authorization-server", (c) =>
  handleOAuthAuthorizationServerMetadata(c.env));

// RESTful resource routes
v1.route("/modules", modules);
v1.route("/api-keys", apiKeys);
v1.route("/prompts", prompts);
v1.route("/credentials", credentials);
v1.route("/user", user);
v1.route("/oauth", oauth);
v1.route("/admin", admin);

// MCP Proxy: MCP Client → Worker → Go Server
v1.all("/mcp/*", (c) => {
  const url = new URL(c.req.url);
  return handleMcpProxy(c.req.raw, url, c.env, c.executionCtx);
});

export { v1 };
