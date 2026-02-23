/**
 * /v1/modules 関連ルート
 *
 * GET /modules — list modules (public, Go Server)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { forwardToGoServer } from "../go-server";

type Bindings = Env;

const modules = new Hono<{ Bindings: Bindings }>();

// GET /modules — list modules (public, no auth)
modules.get("/", async (c) => {
  return forwardToGoServer(c.env, "GET", "/v1/modules", {});
});

export { modules };
