import { NextRequest, NextResponse } from "next/server"
import { createStripeClient, getStripeConfig } from "@/lib/stripe"
import { createAdminClient } from "@/lib/supabase/admin"
import Stripe from "stripe"

/**
 * POST /api/stripe/webhook
 * Handle Stripe webhook events
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
    case "checkout.session.completed": {
      const session = event.data.object as Stripe.Checkout.Session
      await handleCheckoutCompleted(session, event.id)
      break
    }
    default:
      console.log(`[stripe/webhook] Unhandled event type: ${event.type}`)
  }

  return NextResponse.json({ received: true })
}

async function handleCheckoutCompleted(
  session: Stripe.Checkout.Session,
  eventId: string
) {
  console.log(`[stripe/webhook] Processing checkout.session.completed: ${session.id}`)

  const adminClient = createAdminClient()

  // Get user_id from metadata
  const userId = session.metadata?.user_id
  const credits = parseInt(session.metadata?.credits || "0", 10)

  if (!userId) {
    console.error("[stripe/webhook] Missing user_id in session metadata")
    return
  }

  if (credits <= 0) {
    console.error("[stripe/webhook] Invalid credits amount:", credits)
    return
  }

  // Add credits using RPC (handles idempotency)
  const { data, error } = await adminClient.rpc("add_credits", {
    p_user_id: userId,
    p_amount: credits,
    p_credit_type: "paid",
    p_event_id: eventId,
  })

  if (error) {
    console.error("[stripe/webhook] Error adding credits:", error)
    return
  }

  if (data?.success) {
    console.log(
      `[stripe/webhook] Added ${credits} credits to user ${userId}. New balance: ${data.paid_credits}`
    )
  } else if (data?.error === "event_already_processed") {
    console.log(`[stripe/webhook] Event ${eventId} already processed, skipping`)
  } else {
    console.error("[stripe/webhook] Failed to add credits:", data)
  }
}
