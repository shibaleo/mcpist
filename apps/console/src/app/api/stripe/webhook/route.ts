import { NextRequest, NextResponse } from "next/server"
import { createStripeClient, getStripeConfig } from "@/lib/stripe"
import { rpc } from "@/lib/worker-client"
import Stripe from "stripe"

/**
 * POST /api/stripe/webhook
 * Handle Stripe webhook events for subscription billing
 */
export async function POST(request: NextRequest) {
  const stripe = createStripeClient()
  const config = getStripeConfig()

  // Get the raw body for signature verification
  const body = await request.text()
  const signature = request.headers.get("stripe-signature")

  if (!signature) {
    console.error("[stripe/webhook] Missing stripe-signature header")
    return NextResponse.json({ error: "Missing signature" }, { status: 400 })
  }

  let event: Stripe.Event

  try {
    event = stripe.webhooks.constructEvent(body, signature, config.webhookSecret)
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error"
    console.error("[stripe/webhook] Signature verification failed:", message)
    return NextResponse.json(
      { error: `Webhook signature verification failed: ${message}` },
      { status: 400 }
    )
  }

  console.log(`[stripe/webhook] Received event: ${event.type} (${event.id})`)

  // Handle the event
  switch (event.type) {
    case "invoice.paid": {
      const invoice = event.data.object as Stripe.Invoice
      await handleInvoicePaid(invoice, event.id)
      break
    }
    case "customer.subscription.deleted": {
      const subscription = event.data.object as Stripe.Subscription
      await handleSubscriptionDeleted(subscription, event.id)
      break
    }
    default:
      console.log(`[stripe/webhook] Unhandled event type: ${event.type}`)
  }

  return NextResponse.json({ received: true })
}

/**
 * Call activate_subscription RPC and log result
 */
async function activateSubscription(
  userId: string,
  planId: string,
  eventId: string
) {
  try {
    const result = await rpc<{
      success: boolean
      already_processed?: boolean
      plan_id?: string
      error?: string
    }>("activate_subscription", {
      p_user_id: userId,
      p_plan_id: planId,
      p_event_id: eventId,
    })

    if (result?.success) {
      if (result.already_processed) {
        console.log(`[stripe/webhook] Event ${eventId} already processed, skipping`)
      } else {
        console.log(`[stripe/webhook] Activated plan '${result.plan_id}' for user ${userId}`)
      }
    } else {
      console.error("[stripe/webhook] Failed to activate subscription:", result)
    }
  } catch (err) {
    console.error(`[stripe/webhook] Error activating subscription:`, err)
  }
}

/**
 * Handle invoice.paid — activate/maintain subscription plan
 */
async function handleInvoicePaid(
  invoice: Stripe.Invoice,
  eventId: string
) {
  console.log(`[stripe/webhook] Processing invoice.paid: ${invoice.id}`)

  // Get user_id from subscription metadata or invoice metadata
  let userId = invoice.subscription_details?.metadata?.user_id
    || invoice.metadata?.user_id

  // Fallback: resolve user from Stripe customer ID
  if (!userId) {
    const customerId = typeof invoice.customer === "string"
      ? invoice.customer
      : invoice.customer?.id

    if (!customerId) {
      console.error("[stripe/webhook] Missing customer ID in invoice")
      return
    }

    try {
      const foundUserId = await rpc<string>(
        "get_user_by_stripe_customer",
        { p_stripe_customer_id: customerId }
      )

      if (!foundUserId) {
        console.error(`[stripe/webhook] Could not find user for customer ${customerId}`)
        return
      }

      console.log(`[stripe/webhook] Resolved user ${foundUserId} from customer ${customerId}`)
      userId = foundUserId
    } catch (lookupErr) {
      console.error(`[stripe/webhook] Could not find user for customer ${customerId}:`, lookupErr)
      return
    }
  }

  await activateSubscription(userId, "plus", eventId)
}

/**
 * Handle customer.subscription.deleted — downgrade to free plan
 */
async function handleSubscriptionDeleted(
  subscription: Stripe.Subscription,
  eventId: string
) {
  console.log(`[stripe/webhook] Processing customer.subscription.deleted: ${subscription.id}`)

  const userId = subscription.metadata?.user_id

  if (!userId) {
    console.error("[stripe/webhook] Missing user_id in subscription metadata")
    return
  }

  await activateSubscription(userId, "free", eventId)
}
