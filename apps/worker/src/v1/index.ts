/**
 * v1 API サブルーター
 *
 * app.route("/v1", v1) でメインルーターにマウントされる。
 */

import { Hono } from "hono";
import type { Env } from "../types";
import { handleRpcProxy } from "./rpc-proxy";
import { handleMcpProxy } from "./mcp-proxy";
import {
  handleOAuthProtectedResourceMetadata,
  handleOAuthAuthorizationServerMetadata,
} from "./oauth-metadata";

type Bindings = Env;

const v1 = new Hono<{ Bindings: Bindings }>();

// MCP-scoped OAuth metadata (RFC 9728 / 8414)
v1.get("/mcp/.well-known/oauth-protected-resource", (c) =>
  handleOAuthProtectedResourceMetadata(c.req.raw, c.env));
v1.get("/mcp/.well-known/oauth-authorization-server", (c) =>
  handleOAuthAuthorizationServerMetadata(c.env));

// RPC Proxy: Console → Worker → PostgREST
v1.post("/rpc/:name", (c) => {
  const url = new URL(c.req.url);
  return handleRpcProxy(c.req.raw, url, c.env, c.executionCtx);
});

// MCP Proxy: MCP Client → Worker → Go Server
v1.all("/mcp/*", (c) => {
  const url = new URL(c.req.url);
  return handleMcpProxy(c.req.raw, url, c.env, c.executionCtx);
});

export { v1 };
