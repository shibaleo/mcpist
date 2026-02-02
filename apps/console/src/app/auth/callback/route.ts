import { createServerClient } from "@supabase/ssr"
import { createAdminClient } from "@/lib/supabase/admin"
import { cookies } from "next/headers"
import { NextResponse } from "next/server"

export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url)
  const code = searchParams.get("code")
  const errorParam = searchParams.get("error")
  const errorDescription = searchParams.get("error_description")

  // Support both 'next' and 'returnTo' params
  const returnTo =
    searchParams.get("returnTo") || searchParams.get("next") || "/dashboard"

  // Log OAuth error from provider if present
  if (errorParam) {
    console.error("[Auth Callback] OAuth error from provider:", errorParam, errorDescription)
    return NextResponse.redirect(`${origin}/login?error=${errorParam}&error_description=${encodeURIComponent(errorDescription || '')}`)
  }

  if (code) {
    const cookieStore = await cookies()

    // デバッグ: 受信したcookieをログ
    const allCookies = cookieStore.getAll()
    console.log("[Auth Callback] Received cookies:", allCookies.map(c => c.name))
    const pkceVerifier = allCookies.find(c => c.name.includes('code-verifier'))
    console.log("[Auth Callback] PKCE code_verifier cookie present:", !!pkceVerifier)

    // Route Handler用に cookiesToSet を追跡
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
      console.error("[Auth Callback] exchangeCodeForSession error:", error.message, error)
    }

    if (!error) {
      // ユーザー情報を取得してオンボーディング済みか確認
      const { data: { user } } = await supabase.auth.getUser()

      if (user) {
        // account_status を確認（pre_active ならオンボーディング未完了）
        const adminClient = createAdminClient()
        const { data: context } = await adminClient.rpc("get_user_context", {
          p_user_id: user.id,
        })

        // pre_active の場合はオンボーディングへ
        const needsOnboarding = context?.[0]?.account_status === "pre_active"

        if (needsOnboarding && !returnTo.startsWith("/onboarding")) {
          const response = NextResponse.redirect(`${origin}/onboarding`)
          cookiesToSet.forEach(({ name, value, options }) => {
            response.cookies.set(name, value, options as Record<string, unknown>)
          })
          return response
        }
      }

      // If returnTo is a full URL (for OAuth consent flow), redirect there
      const redirectUrl = returnTo.startsWith("http") ? returnTo : `${origin}${returnTo}`
      const response = NextResponse.redirect(redirectUrl)
      // 重要: cookieをレスポンスに設定
      cookiesToSet.forEach(({ name, value, options }) => {
        response.cookies.set(name, value, options as Record<string, unknown>)
      })
      return response
    }
  }

  // エラー時はログインページにリダイレクト
  console.error("[Auth Callback] Redirecting to login with error. code present:", !!code)
  return NextResponse.redirect(`${origin}/login?error=auth_callback_error`)
}
