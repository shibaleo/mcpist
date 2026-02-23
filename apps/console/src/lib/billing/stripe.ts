import Stripe from "stripe"

// Stripe client for server-side operations
// Returns null if STRIPE_SECRET_KEY is not configured
export function createStripeClient(): Stripe | null {
  const secretKey = process.env.STRIPE_SECRET_KEY
  if (!secretKey) return null
  return new Stripe(secretKey)
}
