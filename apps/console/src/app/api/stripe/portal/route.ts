import { NextRequest, NextResponse } from "next/server"
import { createStripeClient } from "@/lib/billing/stripe"
import { createWorkerClient } from "@/lib/worker"

/**
 * POST /api/stripe/portal
 * Create a Stripe Customer Portal session for managing subscriptions
 */
export async function POST(request: NextRequest) {
  try {
    const client = await createWorkerClient()

    // Verify user is authenticated (createWorkerClient uses JWT)
    const { data: contextRows } = await client.GET("/v1/user/context")
    if (!contextRows?.[0]) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const stripe = createStripeClient()

    // Get Stripe Customer ID
    const { data } = await client.GET("/v1/user/stripe")
    const stripeCustomerId = data?.stripe_customer_id

    if (!stripeCustomerId) {
      return NextResponse.json(
        { error: "No subscription found" },
        { status: 404 }
      )
    }

    const origin = request.headers.get("origin") || process.env.NEXT_PUBLIC_APP_URL

    const session = await stripe.billingPortal.sessions.create({
      customer: stripeCustomerId,
      return_url: `${origin}/plans`,
    })

    return NextResponse.json({ url: session.url })
  } catch (error) {
    console.error("[stripe/portal] Error:", error)
    return NextResponse.json(
      { error: "Failed to create portal session" },
      { status: 500 }
    )
  }
}
