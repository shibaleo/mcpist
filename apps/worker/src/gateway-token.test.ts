import { describe, it, expect } from 'vitest'
import * as jose from 'jose'
import { signGatewayToken, getJwksResponse } from './gateway-token'

// Generate a valid 32-byte Ed25519 seed as base64 for testing
const TEST_SEED_BASE64 = btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(32))))

describe('signGatewayToken', () => {
  it('produces a valid JWT with EdDSA algorithm', async () => {
    const token = await signGatewayToken(TEST_SEED_BASE64, { clerk_id: 'user_123' })
    expect(typeof token).toBe('string')
    expect(token.split('.')).toHaveLength(3)

    // Decode header to verify algorithm
    const header = jose.decodeProtectedHeader(token)
    expect(header.alg).toBe('EdDSA')
    expect(header.kid).toBe('mcpist-gateway-v1')
  })

  it('sets iss=mcpist-gateway and exp ~30s', async () => {
    const token = await signGatewayToken(TEST_SEED_BASE64, { clerk_id: 'user_123' })
    const payload = jose.decodeJwt(token)
    expect(payload.iss).toBe('mcpist-gateway')
    expect(payload.iat).toBeDefined()
    expect(payload.exp).toBeDefined()
    // exp should be ~30s after iat
    expect(payload.exp! - payload.iat!).toBe(30)
  })

  it('includes clerk_id in claims', async () => {
    const token = await signGatewayToken(TEST_SEED_BASE64, { clerk_id: 'user_123', email: 'a@b.com' })
    const payload = jose.decodeJwt(token)
    expect(payload.clerk_id).toBe('user_123')
    expect(payload.email).toBe('a@b.com')
  })

  it('includes user_id for API key auth', async () => {
    const token = await signGatewayToken(TEST_SEED_BASE64, { user_id: 'uuid-789' })
    const payload = jose.decodeJwt(token)
    expect(payload.user_id).toBe('uuid-789')
    expect(payload).not.toHaveProperty('clerk_id')
  })

  it('filters out undefined claims', async () => {
    const token = await signGatewayToken(TEST_SEED_BASE64, { clerk_id: 'user_123' })
    const payload = jose.decodeJwt(token)
    expect(payload).not.toHaveProperty('user_id')
    expect(payload).not.toHaveProperty('email')
  })
})

describe('getJwksResponse', () => {
  it('returns JWKS JSON with public key only', async () => {
    const res = await getJwksResponse(TEST_SEED_BASE64)
    expect(res.status).toBe(200)
    expect(res.headers.get('Content-Type')).toBe('application/json')
    expect(res.headers.get('Cache-Control')).toContain('max-age=3600')

    const body = await res.json() as { keys: jose.JWK[] }
    expect(body.keys).toHaveLength(1)

    const key = body.keys[0]
    expect(key.kty).toBe('OKP')
    expect(key.crv).toBe('Ed25519')
    expect(key.kid).toBe('mcpist-gateway-v1')
    expect(key.use).toBe('sig')
    expect(key.alg).toBe('EdDSA')
    // Public key only — no private key (d) should be exposed
    expect(key).not.toHaveProperty('d')
    expect(key.x).toBeDefined()
  })

  it('JWT can be verified with JWKS public key', async () => {
    const token = await signGatewayToken(TEST_SEED_BASE64, { clerk_id: 'user_verify' })
    const jwksRes = await getJwksResponse(TEST_SEED_BASE64)
    const { keys } = await jwksRes.json() as { keys: jose.JWK[] }

    const publicKey = await jose.importJWK(keys[0], 'EdDSA')
    const { payload } = await jose.jwtVerify(token, publicKey)
    expect(payload.clerk_id).toBe('user_verify')
    expect(payload.iss).toBe('mcpist-gateway')
  })
})
