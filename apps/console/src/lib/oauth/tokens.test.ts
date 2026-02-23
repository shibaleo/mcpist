import { describe, it, expect } from 'vitest'
import { isTokenExpired, getTimeUntilExpiration, formatTimeUntilExpiration } from './tokens'
import type { TokenData } from './tokens'

describe('isTokenExpired', () => {
  it('returns true when tokens is null', () => {
    expect(isTokenExpired(null)).toBe(true)
  })

  it('returns true when token is already expired', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() - 1000,
      scope: '',
    }
    expect(isTokenExpired(tokens)).toBe(true)
  })

  it('returns true when token expires within buffer (default 5 min)', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + 2 * 60 * 1000, // 2 min from now
      scope: '',
    }
    expect(isTokenExpired(tokens)).toBe(true)
  })

  it('returns false when token has plenty of time', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + 10 * 60 * 1000, // 10 min from now
      scope: '',
    }
    expect(isTokenExpired(tokens)).toBe(false)
  })

  it('respects custom buffer', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + 30 * 1000, // 30 sec from now
      scope: '',
    }
    expect(isTokenExpired(tokens, 0)).toBe(false)
    expect(isTokenExpired(tokens, 60 * 1000)).toBe(true)
  })
})

describe('getTimeUntilExpiration', () => {
  it('returns -1 for null', () => {
    expect(getTimeUntilExpiration(null)).toBe(-1)
  })

  it('returns positive value for future expiry', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + 60000,
      scope: '',
    }
    const result = getTimeUntilExpiration(tokens)
    expect(result).toBeGreaterThan(0)
    expect(result).toBeLessThanOrEqual(60000)
  })

  it('returns negative value for expired token', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() - 1000,
      scope: '',
    }
    expect(getTimeUntilExpiration(tokens)).toBeLessThan(0)
  })
})

describe('formatTimeUntilExpiration', () => {
  it('returns "期限切れ" for null', () => {
    expect(formatTimeUntilExpiration(null)).toBe('期限切れ')
  })

  it('returns "期限切れ" for expired token', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() - 1000,
      scope: '',
    }
    expect(formatTimeUntilExpiration(tokens)).toBe('期限切れ')
  })

  it('formats hours and minutes', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + (2 * 60 * 60 + 30 * 60) * 1000, // 2h 30min
      scope: '',
    }
    const result = formatTimeUntilExpiration(tokens)
    expect(result).toMatch(/^2時間30分$/)
  })

  it('formats minutes and seconds', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + (5 * 60 + 10) * 1000, // 5min 10sec
      scope: '',
    }
    const result = formatTimeUntilExpiration(tokens)
    expect(result).toMatch(/^5分\d+秒$/)
  })

  it('formats seconds only', () => {
    const tokens: TokenData = {
      accessToken: 'abc',
      refreshToken: null,
      expiresAt: Date.now() + 45 * 1000, // 45 sec
      scope: '',
    }
    const result = formatTimeUntilExpiration(tokens)
    expect(result).toMatch(/^\d+秒$/)
  })
})
