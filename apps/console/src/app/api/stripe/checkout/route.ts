import { NextRequest, NextResponse } from "next/server"
import { createStripeClient } from "@/lib/billing/stripe"
import { createWorkerClient, createPublicWorkerClient } from "@/lib/worker"

/**
 * POST /api/stripe/checkout
 * Create a Stripe Checkout Session for Plus plan subscription
 */
export async function POST(request: NextRequest) {
  try {
    const client = await createWorkerClient()

    // Get authenticated user profile
    const { data: profile } = await client.GET("/v1/me/profile")

    if (!profile) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const stripe = createStripeClient()
    if (!stripe) {
      return NextResponse.json(
        { error: "Stripe is not configured" },
        { status: 503 }
      )
    }
    // Get Plus plan's Stripe Price ID from plans API
    const publicClient = createPublicWorkerClient()
    const { data: plans } = await publicClient.GET("/v1/plans")
    const plusPlan = plans?.find((p) => p.id === "plus")
    if (!plusPlan?.stripe_price_id) {
      return NextResponse.json(
        { error: "Plus plan not configured" },
        { status: 503 }
      )
    }

    // Get or create Stripe Customer
    let stripeCustomerId: string

    const { data: stripeData } = await client.GET("/v1/me/stripe")

    if (stripeData?.stripe_customer_id) {
      stripeCustomerId = stripeData.stripe_customer_id
    } else {
      // Create new Stripe Customer
      const customer = await stripe.customers.create({
        email: profile.email,
        metadata: {
          user_id: profile.user_id,
        },
      })
      stripeCustomerId = customer.id

      // Link Stripe Customer to user (must succeed before creating checkout)
      await client.PUT("/v1/me/stripe", {
        body: { stripe_customer_id: stripeCustomerId },
      })
    }

    // Get the origin for success/cancel URLs
    const origin = request.headers.get("origin") || process.env.NEXT_PUBLIC_APP_URL

    // Create Checkout Session for subscription
    const session = await stripe.checkout.sessions.create({
      customer: stripeCustomerId,
      line_items: [
        {
          price: plusPlan.stripe_price_id,
          quantity: 1,
        },
      ],
      mode: "subscription",
      success_url: `${origin}/plans?success=true`,
      cancel_url: `${origin}/plans?canceled=true`,
      metadata: {
        user_id: profile.user_id,
        plan_id: "plus",
      },
      subscription_data: {
        metadata: {
          user_id: profile.user_id,
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
