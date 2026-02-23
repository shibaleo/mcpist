import { describe, it, expect } from 'vitest'
import { classifyError, buildBackendInfo } from './health'

describe('classifyError', () => {
  it('returns timeout for TimeoutError', () => {
    const err = new Error('request timed out')
    err.name = 'TimeoutError'
    expect(classifyError(err)).toBe('timeout')
  })

  it('returns timeout for abort message', () => {
    expect(classifyError(new Error('The operation was aborted'))).toBe('timeout')
  })

  it('returns dns_failure for DNS errors', () => {
    expect(classifyError(new Error('getaddrinfo ENOTFOUND example.com'))).toBe('dns_failure')
  })

  it('returns dns_failure for name resolution', () => {
    expect(classifyError(new Error('Name resolution failed'))).toBe('dns_failure')
  })

  it('returns dns_failure for internal error', () => {
    expect(classifyError(new Error('internal error'))).toBe('dns_failure')
  })

  it('returns connection_refused for ECONNREFUSED', () => {
    expect(classifyError(new Error('connect ECONNREFUSED 127.0.0.1:8080'))).toBe('connection_refused')
  })

  it('returns connection_refused for network connection lost', () => {
    expect(classifyError(new Error('network connection lost'))).toBe('connection_refused')
  })

  it('returns ssl_error for SSL handshake failure', () => {
    expect(classifyError(new Error('SSL handshake failed'))).toBe('ssl_error')
  })

  it('returns ssl_error for certificate expired', () => {
    expect(classifyError(new Error('certificate expired'))).toBe('ssl_error')
  })

  it('returns ssl_error for self signed certificate', () => {
    expect(classifyError(new Error('self signed certificate'))).toBe('ssl_error')
  })

  it('returns unknown for non-Error values', () => {
    expect(classifyError('string error')).toBe('unknown')
    expect(classifyError(42)).toBe('unknown')
    expect(classifyError(null)).toBe('unknown')
  })

  it('returns unknown for unrecognized error', () => {
    expect(classifyError(new Error('something unexpected'))).toBe('unknown')
  })
})

describe('buildBackendInfo', () => {
  it('includes healthy flag', () => {
    const info = buildBackendInfo({ healthy: true })
    expect(info).toEqual({ healthy: true })
  })

  it('includes error when present', () => {
    const info = buildBackendInfo({ healthy: false, error: 'timeout' })
    expect(info).toEqual({ healthy: false, error: 'timeout' })
  })

  it('includes statusCode when present', () => {
    const info = buildBackendInfo({ healthy: true, statusCode: 200, latencyMs: 42 })
    expect(info).toEqual({ healthy: true, statusCode: 200, latencyMs: 42 })
  })

  it('omits undefined fields', () => {
    const info = buildBackendInfo({ healthy: false, error: 'dns_failure', latencyMs: 100 })
    expect(info).toEqual({ healthy: false, error: 'dns_failure', latencyMs: 100 })
    expect(info).not.toHaveProperty('statusCode')
  })
})
