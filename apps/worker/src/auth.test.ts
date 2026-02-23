import { describe, it, expect, beforeEach } from 'vitest'
import { getCachedKeyStatus, setCachedKeyStatus, invalidateApiKeyCache } from './auth'

describe('API Key cache', () => {
  beforeEach(() => {
    // Clear cache by invalidating known keys
    invalidateApiKeyCache('test-key-1')
    invalidateApiKeyCache('test-key-2')
  })

  it('returns null for cache miss', () => {
    expect(getCachedKeyStatus('nonexistent')).toBeNull()
  })

  it('returns cached active status after set', () => {
    setCachedKeyStatus('test-key-1', true)
    expect(getCachedKeyStatus('test-key-1')).toBe(true)
  })

  it('returns cached inactive status after set', () => {
    setCachedKeyStatus('test-key-1', false)
    expect(getCachedKeyStatus('test-key-1')).toBe(false)
  })

  it('invalidate removes cached entry', () => {
    setCachedKeyStatus('test-key-1', true)
    invalidateApiKeyCache('test-key-1')
    expect(getCachedKeyStatus('test-key-1')).toBeNull()
  })

  it('caches are independent per key', () => {
    setCachedKeyStatus('test-key-1', true)
    setCachedKeyStatus('test-key-2', false)
    expect(getCachedKeyStatus('test-key-1')).toBe(true)
    expect(getCachedKeyStatus('test-key-2')).toBe(false)
  })
})
