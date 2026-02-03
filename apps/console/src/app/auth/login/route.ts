import { createServerClient } from "@supabase/ssr"
import { cookies } from "next/headers"
import { NextResponse } from "next/server"

const ALLOWED_PROVIDERS = new Set(["google", "github", "azure"])

function normalizeReturnTo(raw: string | null, origin: string): string | null {
  if (!raw) return null
  if (raw.startsWith("/") && !raw.startsWith("//")) return raw
  try {
    const url = new URL(raw)
    if (url.origin === origin) {
      return `${url.pathname}${url.search}${url.hash}`
    }
  } catch {
    // ignore
  }
  return null
}

export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url)
  const provider = searchParams.get("provider")
  const returnTo = normalizeReturnTo(searchParams.get("returnTo"), origin)

  if (!provider || !ALLOWED_PROVIDERS.has(provider)) {
    return NextResponse.redirect(`${origin}/login?error=invalid_provider`)
  }

  const redirectTo = returnTo
    ? `${origin}/auth/callback?returnTo=${encodeURIComponent(returnTo)}`
    : `${origin}/auth/callback`

  const cookieStore = await cookies()
  const cookiesToSet: { name: string; value: string; options?: Record<string, unknown> }[] = []

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY!,
    {
      cookies: {
        getAll() {
          return cookieStore.getAll()
        },
        setAll(cookies) {
          cookies.forEach((cookie) => {
            cookiesToSet.push(cookie)
          })
        },
      },
    }
  )

  const { data, error } = await supabase.auth.signInWithOAuth({
    provider: provider as "google" | "github" | "azure",
    options: {
      redirectTo,
    },
  })

  if (error || !data?.url) {
    const message = error?.message || "oauth_start_failed"
    return NextResponse.redirect(`${origin}/login?error=auth_start_error&error_description=${encodeURIComponent(message)}`)
  }

  const response = NextResponse.redirect(data.url)
  cookiesToSet.forEach(({ name, value, options }) => {
    response.cookies.set(name, value, options as Record<string, unknown>)
  })
  return response
}
