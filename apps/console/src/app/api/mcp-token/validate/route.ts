/**
 * MCP Token Validation API
 *
 * Go Server から呼び出され、MCP トークンを検証する
 */
import { createClient } from '@supabase/supabase-js'
import { NextRequest, NextResponse } from 'next/server'

const INTERNAL_SERVICE_KEY = process.env.INTERNAL_SERVICE_KEY

/**
 * SHA-256 ハッシュを生成
 */
async function sha256(text: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(text)
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('')
}

export async function POST(req: NextRequest) {
  // Internal service key check
  const internalKey = req.headers.get('X-Internal-Service-Key')
  if (INTERNAL_SERVICE_KEY && internalKey !== INTERNAL_SERVICE_KEY) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  let body: { token?: string }
  try {
    body = await req.json()
  } catch {
    return NextResponse.json({ error: 'Invalid JSON' }, { status: 400 })
  }

  const { token } = body
  if (!token || token.length !== 64) {
    return NextResponse.json({ error: 'Invalid token format' }, { status: 400 })
  }

  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL || process.env.SUPABASE_URL
  const supabaseServiceKey = process.env.SUPABASE_SERVICE_ROLE_KEY

  if (!supabaseUrl || !supabaseServiceKey) {
    return NextResponse.json({ error: 'Server configuration error' }, { status: 500 })
  }

  const supabase = createClient(supabaseUrl, supabaseServiceKey)

  // Hash the token and validate
  const tokenHash = await sha256(token)

  const { data, error } = await supabase.rpc('validate_mcp_token', {
    p_token_hash: tokenHash,
  })

  if (error) {
    console.error('Token validation error:', error)
    return NextResponse.json({ valid: false, error: error.message }, { status: 500 })
  }

  if (data && data.length > 0 && data[0].is_valid) {
    // Update last_used_at (async, ignore errors)
    supabase.rpc('update_mcp_token_last_used', {
      p_token_id: data[0].token_id,
    }).then(() => {})

    return NextResponse.json({
      valid: true,
      user_id: data[0].user_id,
    })
  }

  return NextResponse.json({ valid: false }, { status: 200 })
}
