import { createClient } from '@/lib/supabase/server'
import { NextResponse } from 'next/server'

export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url)
  const code = searchParams.get('code')
  // Support both 'next' and 'returnTo' params
  const returnTo = searchParams.get('returnTo') || searchParams.get('next') || '/dashboard'

  if (code) {
    const supabase = await createClient()
    const { error } = await supabase.auth.exchangeCodeForSession(code)
    if (!error) {
      // If returnTo is a full URL (for OAuth consent flow), redirect there
      if (returnTo.startsWith('http')) {
        return NextResponse.redirect(returnTo)
      }
      return NextResponse.redirect(`${origin}${returnTo}`)
    }
  }

  // エラー時はログインページにリダイレクト
  return NextResponse.redirect(`${origin}/login?error=auth_callback_error`)
}
