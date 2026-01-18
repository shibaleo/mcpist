/**
 * Consent API - Generate Authorization Code
 *
 * POST /api/auth/consent
 *
 * Called from the consent page after user approves the authorization request.
 * Generates an authorization code and stores it in the database.
 */

import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@/lib/supabase/server'
import { storeAuthorizationCode } from '../lib/codes'
import { generateAuthorizationCode } from '../lib/pkce'

interface ConsentRequest {
  client_id: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: string
  scope: string
  state: string
  user_id: string
}

export async function POST(request: NextRequest) {
  try {
    const body: ConsentRequest = await request.json()

    // Validate required fields
    if (!body.client_id || !body.redirect_uri || !body.code_challenge || !body.state) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      )
    }

    // Verify user is authenticated
    const supabase = await createClient()
    const { data: { user } } = await supabase.auth.getUser()

    if (!user) {
      return NextResponse.json(
        { error: 'User not authenticated' },
        { status: 401 }
      )
    }

    // Verify user_id matches authenticated user
    if (body.user_id !== user.id) {
      return NextResponse.json(
        { error: 'User ID mismatch' },
        { status: 403 }
      )
    }

    // Generate authorization code
    const code = generateAuthorizationCode()

    // Store in database
    await storeAuthorizationCode({
      code,
      user_id: user.id,
      client_id: body.client_id,
      redirect_uri: body.redirect_uri,
      code_challenge: body.code_challenge,
      code_challenge_method: body.code_challenge_method || 'S256',
      scope: body.scope || null,
      state: body.state,
    })

    return NextResponse.json({ code })
  } catch (error) {
    console.error('Consent API error:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}
