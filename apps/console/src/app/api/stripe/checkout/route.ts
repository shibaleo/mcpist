import { NextRequest, NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createStripeClient, getStripeConfig } from "@/lib/stripe"
import { createAdminClient } from "@/lib/supabase/admin"

interface StripeCustomerResult {
  stripe_customer_id: string | null
}

/**
 * POST /api/stripe/checkout
 * Create a Stripe Checkout Session for Plus plan subscription
 */
export async function POST(request: NextRequest) {
  try {
    // Get authenticated user
    const supabase = await createClient()
    const {
      data: { user },
      error: authError,
    } = await supabase.auth.getUser()

    if (authError || !user) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const stripe = createStripeClient()
    const config = getStripeConfig()
    const adminClient = createAdminClient()

    // Get or create Stripe Customer
    let stripeCustomerId: string

    // Check if user already has a Stripe Customer ID using RPC
    const { data: userData, error: userError } = await adminClient.rpc(
      "get_stripe_customer_id",
      { p_user_id: user.id }
    ) as { data: StripeCustomerResult | null; error: Error | null }

    if (userError) {
      console.error("[stripe/checkout] Error fetching user:", userError)
      return NextResponse.json(
        { error: "Failed to fetch user data" },
        { status: 500 }
      )
    }

    if (userData?.stripe_customer_id) {
      stripeCustomerId = userData.stripe_customer_id
    } else {
      // Create new Stripe Customer
      const customer = await stripe.customers.create({
        email: user.email,
        metadata: {
          supabase_user_id: user.id,
        },
      })
      stripeCustomerId = customer.id

      // Link Stripe Customer to user
      const { error: linkError } = await adminClient.rpc("link_stripe_customer", {
        p_user_id: user.id,
        p_stripe_customer_id: stripeCustomerId,
      })

      if (linkError) {
        console.error("[stripe/checkout] Error linking customer:", linkError)
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
        user_id: user.id,
        plan_id: "plus",
      },
      subscription_data: {
        metadata: {
          user_id: user.id,
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
