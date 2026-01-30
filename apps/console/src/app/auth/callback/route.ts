import { createClient } from "@/lib/supabase/server"
import { createAdminClient } from "@/lib/supabase/admin"
import { NextResponse } from "next/server"

export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url)
  const code = searchParams.get("code")
  // Support both 'next' and 'returnTo' params
  const returnTo =
    searchParams.get("returnTo") || searchParams.get("next") || "/dashboard"

  if (code) {
    const supabase = await createClient()
    const { error } = await supabase.auth.exchangeCodeForSession(code)
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
          return NextResponse.redirect(`${origin}/onboarding`)
        }
      }

      // If returnTo is a full URL (for OAuth consent flow), redirect there
      if (returnTo.startsWith("http")) {
        return NextResponse.redirect(returnTo)
      }
      return NextResponse.redirect(`${origin}${returnTo}`)
    }
  }

  // エラー時はログインページにリダイレクト
  return NextResponse.redirect(`${origin}/login?error=auth_callback_error`)
}
