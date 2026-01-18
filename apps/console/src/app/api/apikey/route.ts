/**
 * API Key Management API
 *
 * POST /api/apikey - Generate new API Key
 * GET /api/apikey - Check if API Key exists
 * DELETE /api/apikey - Revoke API Key
 */

import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@/lib/supabase/server'
import { randomBytes } from 'crypto'

const SERVICE_NAME = 'mcpist'

/**
 * Generate a secure API Key
 * Format: mpt_<32 random hex chars>
 */
function generateApiKey(): string {
  const randomPart = randomBytes(16).toString('hex')
  return `mpt_${randomPart}`
}

/**
 * GET /api/apikey
 * Check if API Key exists for the current user
 */
export async function GET() {
  try {
    const supabase = await createClient()
    const { data: { user }, error: authError } = await supabase.auth.getUser()

    if (authError || !user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Check if user has an API Key
    const { data, error } = await supabase.rpc('get_my_oauth_connections')

    if (error) {
      console.error('Error checking API key:', error)
      return NextResponse.json({ error: 'Failed to check API key' }, { status: 500 })
    }

    const mcpistConnection = data?.find((conn: { service: string }) => conn.service === SERVICE_NAME)

    // Get masked version if exists
    let maskedKey = null
    if (mcpistConnection) {
      const { data: masked } = await supabase.rpc('get_masked_api_key', {
        p_service: SERVICE_NAME,
      })
      maskedKey = masked
    }

    return NextResponse.json({
      exists: !!mcpistConnection,
      masked_key: maskedKey,
      created_at: mcpistConnection?.created_at || null,
      updated_at: mcpistConnection?.updated_at || null,
    })
  } catch (error) {
    console.error('API Key check error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

/**
 * POST /api/apikey
 * Generate a new API Key and store in Vault
 */
export async function POST() {
  try {
    const supabase = await createClient()
    const { data: { user }, error: authError } = await supabase.auth.getUser()

    if (authError || !user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Generate new API Key
    const apiKey = generateApiKey()

    // Store in Vault using existing RPC
    const { data: tokenId, error } = await supabase.rpc('upsert_oauth_token', {
      p_service: SERVICE_NAME,
      p_access_token: apiKey,
      p_token_type: 'ApiKey',
      p_scope: null,
      p_expires_at: null, // No expiration for API Keys
    })

    if (error) {
      console.error('Error storing API key:', error.message, error.details, error.hint)
      return NextResponse.json({ error: 'Failed to store API key', details: error.message }, { status: 500 })
    }

    console.log('API Key stored successfully, token_id:', tokenId)

    // Return the API Key (only shown once)
    return NextResponse.json({
      api_key: apiKey,
      message: 'API Key generated successfully. Save it now - it will not be shown again.',
    })
  } catch (error) {
    console.error('API Key generation error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

/**
 * DELETE /api/apikey
 * Revoke the API Key
 */
export async function DELETE() {
  try {
    const supabase = await createClient()
    const { data: { user }, error: authError } = await supabase.auth.getUser()

    if (authError || !user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Delete using existing RPC
    const { data, error } = await supabase.rpc('delete_oauth_token', {
      p_service: SERVICE_NAME,
    })

    if (error) {
      console.error('Error revoking API key:', error)
      return NextResponse.json({ error: 'Failed to revoke API key' }, { status: 500 })
    }

    return NextResponse.json({
      revoked: data,
      message: data ? 'API Key revoked successfully' : 'No API Key found',
    })
  } catch (error) {
    console.error('API Key revocation error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}
