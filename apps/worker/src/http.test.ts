import { describe, it, expect } from 'vitest'
import { jsonResponse, addCORSToResponse } from './http'

describe('jsonResponse', () => {
  it('returns JSON body with correct Content-Type', async () => {
    const res = jsonResponse({ ok: true }, 200)
    expect(res.status).toBe(200)
    expect(res.headers.get('Content-Type')).toBe('application/json')
    expect(await res.json()).toEqual({ ok: true })
  })

  it('sets CORS header', () => {
    const res = jsonResponse({ ok: true }, 200)
    expect(res.headers.get('Access-Control-Allow-Origin')).toBe('*')
  })

  it('merges extraHeaders', () => {
    const res = jsonResponse({ ok: true }, 201, { 'X-Custom': 'value' })
    expect(res.status).toBe(201)
    expect(res.headers.get('X-Custom')).toBe('value')
    expect(res.headers.get('Content-Type')).toBe('application/json')
  })

  it('extraHeaders can override defaults', () => {
    const res = jsonResponse({ ok: true }, 200, { 'Content-Type': 'text/plain' })
    expect(res.headers.get('Content-Type')).toBe('text/plain')
  })
})

describe('addCORSToResponse', () => {
  it('adds CORS headers to response', () => {
    const original = new Response('body', {
      status: 200,
      headers: { 'X-Existing': 'keep' },
    })
    const result = addCORSToResponse(original)
    expect(result.headers.get('Access-Control-Allow-Origin')).toBe('*')
    expect(result.headers.get('Access-Control-Allow-Methods')).toContain('GET')
    expect(result.headers.get('Access-Control-Allow-Headers')).toContain('Authorization')
    expect(result.headers.get('X-Existing')).toBe('keep')
  })

  it('preserves status and statusText', () => {
    const original = new Response(null, { status: 404, statusText: 'Not Found' })
    const result = addCORSToResponse(original)
    expect(result.status).toBe(404)
    expect(result.statusText).toBe('Not Found')
  })
})
