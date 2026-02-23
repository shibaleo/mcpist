import crypto from "crypto"

const STATE_MAX_AGE_MS = 10 * 60 * 1000 // 10 minutes

function getSecret(): string {
  const secret = process.env.OAUTH_STATE_SECRET
  if (!secret) {
    throw new Error("OAUTH_STATE_SECRET environment variable is not set")
  }
  return secret
}

/**
 * Generate a signed OAuth state parameter.
 * Format: base64url(JSON payload) + "." + base64url(HMAC-SHA256 signature)
 */
export function generateState(data: Record<string, unknown>): string {
  const payload = { ...data, iat: Date.now() }
  const payloadStr = Buffer.from(JSON.stringify(payload)).toString("base64url")
  const signature = crypto
    .createHmac("sha256", getSecret())
    .update(payloadStr)
    .digest("base64url")
  return `${payloadStr}.${signature}`
}

/**
 * Verify and decode a signed OAuth state parameter.
 * Throws on invalid signature or expired state.
 */
export function verifyState(state: string): Record<string, unknown> {
  const dotIndex = state.lastIndexOf(".")
  if (dotIndex === -1) {
    throw new Error("Invalid state format")
  }

  const payloadStr = state.slice(0, dotIndex)
  const signature = state.slice(dotIndex + 1)

  const expected = crypto
    .createHmac("sha256", getSecret())
    .update(payloadStr)
    .digest("base64url")

  if (!crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected))) {
    throw new Error("Invalid state signature")
  }

  const payload = JSON.parse(Buffer.from(payloadStr, "base64url").toString())

  if (typeof payload.iat !== "number" || Date.now() - payload.iat > STATE_MAX_AGE_MS) {
    throw new Error("State expired")
  }

  return payload
}
