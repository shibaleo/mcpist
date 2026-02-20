/**
 * /v1/stripe 関連ルート
 *
 * POST /stripe/webhook → Go Server /v1/stripe/webhook にプロキシ
 * Stripe signature 検証は Go Server 側で実施。
 */

import { Hono } from "hono";
import type { Env } from "../../types";

type Bindings = Env;

const stripe = new Hono<{ Bindings: Bindings }>();

// POST /stripe/webhook — Forward to Go Server (signature verification done there)
stripe.post("/webhook", async (c) => {
  const body = await c.req.text();
  const signature = c.req.header("stripe-signature") || "";

  const url = `${c.env.SERVER_URL}/v1/stripe/webhook`;
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Stripe-Signature": signature,
    },
    body,
  });

  const respHeaders = new Headers(response.headers);
  respHeaders.set("Access-Control-Allow-Origin", "*");

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: respHeaders,
  });
});

export { stripe };
