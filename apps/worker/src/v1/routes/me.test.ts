import { describe, it, expect } from 'vitest'
import { buildClaims } from './me'

describe('buildClaims', () => {
  it('maps JWT auth to clerk_id', () => {
    const claims = buildClaims({ userId: 'user_clerk_123', type: 'jwt' })
    expect(claims).toEqual({ clerk_id: 'user_clerk_123', email: undefined })
  })

  it('maps API Key auth to user_id', () => {
    const claims = buildClaims({ userId: 'uuid-456', type: 'api_key' })
    expect(claims).toEqual({ user_id: 'uuid-456', email: undefined })
  })

  it('includes email when present (JWT)', () => {
    const claims = buildClaims({ userId: 'user_clerk_123', email: 'test@example.com', type: 'jwt' })
    expect(claims).toEqual({ clerk_id: 'user_clerk_123', email: 'test@example.com' })
  })

  it('includes email when present (API Key)', () => {
    const claims = buildClaims({ userId: 'uuid-456', email: 'test@example.com', type: 'api_key' })
    expect(claims).toEqual({ user_id: 'uuid-456', email: 'test@example.com' })
  })

  it('omits email when not provided', () => {
    const claims = buildClaims({ userId: 'user_clerk_123', type: 'jwt' })
    expect(claims.email).toBeUndefined()
    expect(claims).not.toHaveProperty('user_id')
    expect(claims).toHaveProperty('clerk_id')
  })
})
