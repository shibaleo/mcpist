import { NextRequest, NextResponse } from "next/server"
import { createStripeClient, getStripeConfig } from "@/lib/billing/stripe"
import { createWorkerClient } from "@/lib/worker"

/**
 * POST /api/stripe/checkout
 * Create a Stripe Checkout Session for Plus plan subscription
 */
export async function POST(request: NextRequest) {
  try {
    const client = await createWorkerClient()

    // Get authenticated user identity
    const { data: contextRows } = await client.GET("/v1/user/context")
    const userCtx = contextRows?.[0]

    if (!userCtx) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const stripe = createStripeClient()
    const config = getStripeConfig()

    // Get or create Stripe Customer
    let stripeCustomerId: string

    const { data: stripeData } = await client.GET("/v1/user/stripe")

    if (stripeData?.stripe_customer_id) {
      stripeCustomerId = stripeData.stripe_customer_id
    } else {
      // Create new Stripe Customer
      const customer = await stripe.customers.create({
        email: userCtx.email,
        metadata: {
          supabase_user_id: userCtx.user_id,
        },
      })
      stripeCustomerId = customer.id

      // Link Stripe Customer to user
      try {
        await client.PUT("/v1/user/stripe", {
          body: { stripe_customer_id: stripeCustomerId },
        })
      } catch (linkErr) {
        console.error("[stripe/checkout] Error linking customer:", linkErr)
        // Continue anyway - the customer was created in Stripe
      }
    }

    // Get the origin for success/cancel URLs
    const origin = request.headers.get("origin") || process.env.NEXT_PUBLIC_APP_URL

    // Create Checkout Session for subscription
    const session = await stripe.checkout.sessions.create({
      customer: stripeCustomerId,
      line_items: [
        {
          price: config.plusPriceId,
          quantity: 1,
        },
      ],
      mode: "subscription",
      success_url: `${origin}/plans?success=true`,
      cancel_url: `${origin}/plans?canceled=true`,
      metadata: {
        user_id: userCtx.user_id,
        plan_id: "plus",
      },
      subscription_data: {
        metadata: {
          user_id: userCtx.user_id,
          plan_id: "plus",
        },
      },
    })

    return NextResponse.json({ url: session.url })
  } catch (error) {
    console.error("[stripe/checkout] Error:", error)
    return NextResponse.json(
      { error: "Failed to create checkout session" },
      { status: 500 }
    )
  }
}
