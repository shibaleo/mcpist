/**
 * /v1/plans 関連ルート
 *
 * GET /plans — list plans (public, Go Server)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { forwardToGoServer } from "../go-server";

type Bindings = Env;

const plans = new Hono<{ Bindings: Bindings }>();

// GET /plans — list plans (public, no auth)
plans.get("/", async (c) => {
  return forwardToGoServer(c.env, "GET", "/v1/plans", {});
});

export { plans };
