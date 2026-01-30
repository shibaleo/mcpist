import Stripe from "stripe"

// Stripe client for server-side operations
export function createStripeClient() {
  const secretKey = process.env.STRIPE_SECRET_KEY

  if (!secretKey) {
    throw new Error("Missing STRIPE_SECRET_KEY")
  }

  return new Stripe(secretKey)
}

// Environment variables for Stripe
export function getStripeConfig() {
  return {
    publishableKey: process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY!,
    freeCreditPriceId: process.env.STRIPE_FREE_CREDIT_PRICE_ID!,
    freeCreditProductId: process.env.STRIPE_FREE_CREDIT_PROD_ID!,
    webhookSecret: process.env.STRIPE_WEBHOOK_SECRET!,
  }
}
