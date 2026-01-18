import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@supabase/supabase-js'

/**
 * Token Vault API
 * Provides tokens to MCP Server (Go) for external API calls
 *
 * Security: Uses Service Role Key for server-to-server communication
 */

interface TokenRequest {
  user_id: string
  service: string
}

interface TokenResponse {
  oauth_token?: string
  long_term_token?: string
}

interface ErrorResponse {
  error: string
}

// Create Supabase admin client with service role key
function createAdminClient() {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL
  const serviceRoleKey = process.env.SUPABASE_SERVICE_ROLE_KEY

  if (!supabaseUrl || !serviceRoleKey) {
    throw new Error('Missing Supabase configuration')
  }

  return createClient(supabaseUrl, serviceRoleKey, {
    auth: {
      autoRefreshToken: false,
      persistSession: false,
    },
  })
}

// Verify internal service key (simple shared secret for server-to-server auth)
function verifyInternalKey(request: NextRequest): boolean {
  const authHeader = request.headers.get('Authorization')
  if (!authHeader) return false

  const token = authHeader.replace('Bearer ', '')
  const internalKey = process.env.INTERNAL_SERVICE_KEY

  // If no internal key is configured, deny all requests
  if (!internalKey) {
    console.warn('[token-vault] INTERNAL_SERVICE_KEY not configured')
    return false
  }

  return token === internalKey
}

export async function POST(request: NextRequest): Promise<NextResponse<TokenResponse | ErrorResponse>> {
  try {
    // Verify internal service authentication
    if (!verifyInternalKey(request)) {
      console.log('[token-vault] Unauthorized request')
      return NextResponse.json(
        { error: 'Unauthorized' },
        { status: 401 }
      )
    }

    const body: TokenRequest = await request.json()
    const { user_id, service } = body

    if (!user_id || !service) {
      return NextResponse.json(
        { error: 'user_id and service are required' },
        { status: 400 }
      )
    }

    console.log(`[token-vault] Getting token for user=${user_id} service=${service}`)

    const supabase = createAdminClient()

    // Call the RPC function to get decrypted token
    const { data, error } = await supabase.rpc('get_service_token', {
      p_user_id: user_id,
      p_service: service,
    })

    if (error) {
      console.error('[token-vault] RPC error:', error)
      return NextResponse.json(
        { error: error.message },
        { status: 500 }
      )
    }

    // data is an array with one row, or empty
    if (!data || data.length === 0 || !data[0].oauth_token) {
      console.log(`[token-vault] No token found for user=${user_id} service=${service}`)
      return NextResponse.json(
        { error: 'Token not found' },
        { status: 404 }
      )
    }

    console.log(`[token-vault] Token retrieved for user=${user_id} service=${service}`)

    return NextResponse.json({
      oauth_token: data[0].oauth_token,
      long_term_token: data[0].long_term_token,
    })
  } catch (error) {
    console.error('[token-vault] Error:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}
