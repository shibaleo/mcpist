import { createServerClient } from "@supabase/ssr"
import { createAdminClient } from "@/lib/supabase/admin"
import { cookies } from "next/headers"
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
  const requestId = request.headers.get("x-vercel-id") || request.headers.get("x-request-id") || "unknown"
  const code = searchParams.get("code")
  const errorParam = searchParams.get("error")
  const errorDescription = searchParams.get("error_description")

  // Support both 'next' and 'returnTo' params
  const returnTo = normalizeReturnTo(searchParams.get("returnTo") || searchParams.get("next"), origin)

  // Log OAuth error from provider if present
  if (errorParam) {
    console.error(`[Auth Callback] OAuth error from provider (requestId=${requestId}):`, errorParam, errorDescription)
    return NextResponse.redirect(
      `${origin}/login?error=${encodeURIComponent(errorParam)}&error_description=${encodeURIComponent(errorDescription || '')}`
    )
  }

  if (!code) {
    console.error(`[Auth Callback] Missing code (requestId=${requestId})`)
    return NextResponse.redirect(`${origin}/login?error=auth_callback_error&error_description=missing_code`)
  }

  const cookieStore = await cookies()

  // Debug: log received cookies
  const allCookies = cookieStore.getAll()
  console.log(`[Auth Callback] Received cookies (requestId=${requestId}):`, allCookies.map(c => c.name))
  const pkceVerifier = allCookies.find(c => c.name.includes('code-verifier'))
  console.log(`[Auth Callback] PKCE code_verifier cookie present (requestId=${requestId}):`, !!pkceVerifier)

  // Track cookies to set on the response
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

  const { error } = await supabase.auth.exchangeCodeForSession(code)

  if (error) {
    console.error(`[Auth Callback] exchangeCodeForSession error (requestId=${requestId}):`, error.message, error)
    return NextResponse.redirect(
      `${origin}/login?error=auth_callback_error&error_description=${encodeURIComponent(error.message)}`
    )
  }

  // Fetch user and check onboarding state
  const { data: { user } } = await supabase.auth.getUser()

  if (user) {
    // If account_status is pre_active, onboarding is required
    const adminClient = createAdminClient()
    const { data: context } = await adminClient.rpc("get_user_context", {
      p_user_id: user.id,
    })

    // Redirect to onboarding when needed
    const needsOnboarding = context?.[0]?.account_status === "pre_active"

    if (needsOnboarding && !returnTo.startsWith("/onboarding")) {
      const response = NextResponse.redirect(`${origin}/onboarding`)
      cookiesToSet.forEach(({ name, value, options }) => {
        response.cookies.set(name, value, options as Record<string, unknown>)
      })
      return response
    }
  }

  const response = NextResponse.redirect(`${origin}${returnTo}`)
  // Important: set cookies on the response
  cookiesToSet.forEach(({ name, value, options }) => {
    response.cookies.set(name, value, options as Record<string, unknown>)
  })
  return response
}
