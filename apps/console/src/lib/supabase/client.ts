import { createBrowserClient } from '@supabase/ssr'
import type { Database } from './database.types'

export function createClient() {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
  const supabaseKey = process.env.NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY

  if (!supabaseUrl || !supabaseKey) {
    throw new Error(
      'Missing Supabase environment variables. Please set NEXT_PUBLIC_SUPABASE_URL and NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY in .env.local'
    )
  }

  // ローカル開発環境ではsecure: falseにする必要がある
  const isLocalhost = typeof window !== 'undefined' &&
    (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')

  return createBrowserClient<Database>(supabaseUrl, supabaseKey, {
    cookieOptions: {
      // OAuthリダイレクト後もcookieが送信されるようにする
      sameSite: 'lax',
      secure: !isLocalhost,  // HTTPSの場合のみsecure
      path: '/',
    },
  })
}
