/**
 * /v1/stripe 関連ルート
 *
 * POST /stripe/webhook  → Stripe webhook イベント処理 (Stripe signature 検証、JWT 不要)
 */

import { Hono } from "hono";
import type { Env } from "../../types";
import { jsonResponse } from "../../http";
import { callPostgRESTRpc } from "../postgrest";

type Bindings = Env;

const stripe = new Hono<{ Bindings: Bindings }>();

// ── Stripe signature verification (Web Crypto API) ──────────────

async function verifyStripeSignature(
  payload: string,
  sigHeader: string,
  secret: string,
  toleranceSec = 300
): Promise<boolean> {
  const pairs = sigHeader.split(",").map((p) => p.trim().split("=", 2));
  const timestamp = pairs.find(([k]) => k === "t")?.[1];
  const signatures = pairs.filter(([k]) => k === "v1").map(([, v]) => v);

  if (!timestamp || signatures.length === 0) return false;

  // Replay attack protection
  const ts = parseInt(timestamp, 10);
  if (Math.abs(Date.now() / 1000 - ts) > toleranceSec) return false;

  const signedPayload = `${timestamp}.${payload}`;
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const mac = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(signedPayload)
  );
  const expected = Array.from(new Uint8Array(mac))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  return signatures.some((sig) => timingSafeEqual(expected, sig));
}

/** Constant-time string comparison */
function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return result === 0;
}

// ── Event types ─────────────────────────────────────────────────

interface StripeInvoice {
  id: string;
  customer: string | { id: string } | null;
  subscription_details?: { metadata?: Record<string, string> };
  metadata?: Record<string, string>;
}

interface StripeSubscription {
  id: string;
  metadata?: Record<string, string>;
}

interface StripeEvent {
  id: string;
  type: string;
  data: { object: Record<string, unknown> };
}

// ── Handlers ────────────────────────────────────────────────────

async function activateSubscription(
  env: Env,
  userId: string,
  planId: string,
  eventId: string
): Promise<void> {
  try {
    const result = await callPostgRESTRpc<{
      success: boolean;
      already_processed?: boolean;
      plan_id?: string;
      error?: string;
    }>(env, "activate_subscription", {
      p_user_id: userId,
      p_plan_id: planId,
      p_event_id: eventId,
    });

    if (result?.success) {
      if (result.already_processed) {
        console.log(`[stripe/webhook] Event ${eventId} already processed, skipping`);
      } else {
        console.log(`[stripe/webhook] Activated plan '${result.plan_id}' for user ${userId}`);
      }
    } else {
      console.error("[stripe/webhook] Failed to activate subscription:", result);
    }
  } catch (err) {
    console.error("[stripe/webhook] Error activating subscription:", err);
  }
}

async function handleInvoicePaid(
  env: Env,
  invoice: StripeInvoice,
  eventId: string
): Promise<void> {
  console.log(`[stripe/webhook] Processing invoice.paid: ${invoice.id}`);

  let userId =
    invoice.subscription_details?.metadata?.user_id ||
    invoice.metadata?.user_id;

  // Fallback: resolve user from Stripe customer ID
  if (!userId) {
    const customerId =
      typeof invoice.customer === "string"
        ? invoice.customer
        : invoice.customer?.id;

    if (!customerId) {
      console.error("[stripe/webhook] Missing customer ID in invoice");
      return;
    }

    try {
      const foundUserId = await callPostgRESTRpc<string>(
        env,
        "get_user_by_stripe_customer",
        { p_stripe_customer_id: customerId }
      );

      if (!foundUserId) {
        console.error(`[stripe/webhook] Could not find user for customer ${customerId}`);
        return;
      }

      console.log(`[stripe/webhook] Resolved user ${foundUserId} from customer ${customerId}`);
      userId = foundUserId;
    } catch (lookupErr) {
      console.error(`[stripe/webhook] Could not find user for customer ${customerId}:`, lookupErr);
      return;
    }
  }

  await activateSubscription(env, userId, "plus", eventId);
}

async function handleSubscriptionDeleted(
  env: Env,
  subscription: StripeSubscription,
  eventId: string
): Promise<void> {
  console.log(`[stripe/webhook] Processing customer.subscription.deleted: ${subscription.id}`);

  const userId = subscription.metadata?.user_id;

  if (!userId) {
    console.error("[stripe/webhook] Missing user_id in subscription metadata");
    return;
  }

  await activateSubscription(env, userId, "free", eventId);
}

// ── Route ───────────────────────────────────────────────────────

// POST /stripe/webhook — Handle Stripe webhook events
stripe.post("/webhook", async (c) => {
  const body = await c.req.text();
  const signature = c.req.header("stripe-signature");

  if (!signature) {
    console.error("[stripe/webhook] Missing stripe-signature header");
    return jsonResponse({ error: "Missing signature" }, 400);
  }

  const isValid = await verifyStripeSignature(
    body,
    signature,
    c.env.STRIPE_WEBHOOK_SECRET
  );

  if (!isValid) {
    console.error("[stripe/webhook] Signature verification failed");
    return jsonResponse({ error: "Webhook signature verification failed" }, 400);
  }

  let event: StripeEvent;
  try {
    event = JSON.parse(body);
  } catch {
    return jsonResponse({ error: "Invalid JSON" }, 400);
  }

  console.log(`[stripe/webhook] Received event: ${event.type} (${event.id})`);

  switch (event.type) {
    case "invoice.paid": {
      const invoice = event.data.object as unknown as StripeInvoice;
      await handleInvoicePaid(c.env, invoice, event.id);
      break;
    }
    case "customer.subscription.deleted": {
      const subscription = event.data.object as unknown as StripeSubscription;
      await handleSubscriptionDeleted(c.env, subscription, event.id);
      break;
    }
    default:
      console.log(`[stripe/webhook] Unhandled event type: ${event.type}`);
  }

  return jsonResponse({ received: true }, 200);
});

export { stripe };
