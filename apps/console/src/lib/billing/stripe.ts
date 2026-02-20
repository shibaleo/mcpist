import Stripe from "stripe"

// Stripe client for server-side operations
// Returns null if STRIPE_SECRET_KEY is not configured
export function createStripeClient(): Stripe | null {
  const secretKey = process.env.STRIPE_SECRET_KEY
  if (!secretKey) return null
  return new Stripe(secretKey)
}

// Environment variables for Stripe
export function getStripeConfig() {
  return {
    publishableKey: process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY!,
    plusPriceId: process.env.STRIPE_PLUS_PRICE_ID!,
    webhookSecret: process.env.STRIPE_WEBHOOK_SECRET!,
  }
}
