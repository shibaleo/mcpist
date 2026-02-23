import { auth } from "@clerk/nextjs/server"
import { createWorkerClient } from "@/lib/worker"
import { NextResponse } from "next/server"

function normalizeReturnTo(raw: string | null, origin: string): string {
  if (!raw) return "/dashboard"
  if (raw.startsWith("/") && !raw.startsWith("//")) return raw
  try {
    const url = new URL(raw)
    if (url.origin === origin) {
      return `${url.pathname}${url.search}${url.hash}`
    }
  } catch {
    // ignore
  }
  return "/dashboard"
}

export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url)
  const returnTo = normalizeReturnTo(searchParams.get("returnTo") || searchParams.get("next"), origin)

  const { userId } = await auth()

  if (!userId) {
    return NextResponse.redirect(`${origin}/login`)
  }

  // Ensure user record exists in DB, then check onboarding state
  try {
    const client = await createWorkerClient()

    // Register (idempotent — creates user if not exists)
    await client.POST("/v1/me/register")

    const { data } = await client.GET("/v1/me/profile")
    const needsOnboarding = data?.account_status === "pre_active"

    if (needsOnboarding && !returnTo.startsWith("/onboarding")) {
      return NextResponse.redirect(`${origin}/onboarding`)
    }
  } catch {
    // If register/profile fetch fails, continue to dashboard
  }

  return NextResponse.redirect(`${origin}${returnTo}`)
}
