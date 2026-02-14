import { NextRequest, NextResponse } from "next/server"
import { createClient } from "@/lib/supabase/server"
import { createStripeClient } from "@/lib/stripe"
import { createAdminClient } from "@/lib/supabase/admin"

/**
 * POST /api/stripe/portal
 * Create a Stripe Customer Portal session for managing subscriptions
 */
export async function POST(request: NextRequest) {
  try {
    const supabase = await createClient()
    const {
      data: { user },
      error: authError,
    } = await supabase.auth.getUser()

    if (authError || !user) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
    }

    const stripe = createStripeClient()
    const adminClient = createAdminClient()

    // Get Stripe Customer ID (RPC returns string directly)
    const { data: stripeCustomerId, error: userError } = await adminClient.rpc(
      "get_stripe_customer_id",
      { p_user_id: user.id }
    )

    if (userError || !stripeCustomerId) {
      return NextResponse.json(
        { error: "No subscription found" },
        { status: 404 }
      )
    }

    const origin = request.headers.get("origin") || process.env.NEXT_PUBLIC_APP_URL

    const session = await stripe.billingPortal.sessions.create({
      customer: stripeCustomerId as string,
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
