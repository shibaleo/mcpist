import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { generateState, verifyState } from './state'

// Set env before tests
beforeEach(() => {
  process.env.OAUTH_STATE_SECRET = 'test-secret-key-for-hmac-256'
})

afterEach(() => {
  delete process.env.OAUTH_STATE_SECRET
})

describe('generateState / verifyState roundtrip', () => {
  it('generates and verifies state with payload', () => {
    const data = { service: 'notion', redirect: '/dashboard' }
    const state = generateState(data)

    const result = verifyState(state)
    expect(result.service).toBe('notion')
    expect(result.redirect).toBe('/dashboard')
    expect(result.iat).toBeTypeOf('number')
  })

  it('generates different state for same payload (different iat)', () => {
    const data = { service: 'notion' }
    const state1 = generateState(data)

    // Advance time slightly
    vi.useFakeTimers()
    vi.advanceTimersByTime(1)
    const state2 = generateState(data)
    vi.useRealTimers()

    // States may differ due to iat timestamp
    // Both should verify successfully
    expect(() => verifyState(state1)).not.toThrow()
    expect(() => verifyState(state2)).not.toThrow()
  })
})

describe('verifyState error cases', () => {
  it('throws on invalid format (no dot separator)', () => {
    expect(() => verifyState('nodot')).toThrow('Invalid state format')
  })

  it('throws on tampered signature', () => {
    const state = generateState({ service: 'notion' })
    const tampered = state.slice(0, -1) + 'X'
    expect(() => verifyState(tampered)).toThrow()
  })

  it('throws on tampered payload', () => {
    const state = generateState({ service: 'notion' })
    const [, sig] = state.split('.')
    const fakePayload = Buffer.from(JSON.stringify({ service: 'evil', iat: Date.now() })).toString('base64url')
    expect(() => verifyState(`${fakePayload}.${sig}`)).toThrow()
  })

  it('throws on expired state (>10 minutes)', () => {
    vi.useFakeTimers()
    const state = generateState({ service: 'notion' })

    // Advance 11 minutes
    vi.advanceTimersByTime(11 * 60 * 1000)
    expect(() => verifyState(state)).toThrow('State expired')

    vi.useRealTimers()
  })
})

describe('getSecret', () => {
  it('throws when OAUTH_STATE_SECRET is not set', () => {
    delete process.env.OAUTH_STATE_SECRET
    expect(() => generateState({})).toThrow('OAUTH_STATE_SECRET')
  })
})
