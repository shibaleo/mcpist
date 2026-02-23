import { describe, it, expect } from 'vitest'
import { buildCredentials } from './token-vault'

describe('buildCredentials', () => {
  it('builds API key credentials (default)', () => {
    const result = buildCredentials({
      service: 'grafana',
      accessToken: 'glsa_xxx',
    })
    expect(result).toEqual({
      auth_type: 'api_key',
      access_token: 'glsa_xxx',
    })
  })

  it('builds OAuth2 credentials with refresh token', () => {
    const expiry = new Date('2025-06-01T00:00:00Z')
    const result = buildCredentials({
      service: 'notion',
      accessToken: 'ntn_xxx',
      refreshToken: 'refresh_xxx',
      tokenType: 'bearer',
      scope: 'read write',
      expiresAt: expiry,
    })
    expect(result).toEqual({
      auth_type: 'oauth2',
      access_token: 'ntn_xxx',
      refresh_token: 'refresh_xxx',
      token_type: 'bearer',
      scope: 'read write',
      expires_at: Math.floor(expiry.getTime() / 1000),
    })
  })

  it('builds basic auth credentials', () => {
    const result = buildCredentials({
      service: 'postgresql',
      accessToken: 'mypassword',
      username: 'admin',
    })
    expect(result).toEqual({
      auth_type: 'basic',
      username: 'admin',
      password: 'mypassword',
    })
  })

  it('handles Trello special case (api_key + token)', () => {
    const result = buildCredentials({
      service: 'trello',
      accessToken: 'trello_token',
      username: 'trello_api_key',
    })
    expect(result).toEqual({
      auth_type: 'api_key',
      api_key: 'trello_api_key',
      access_token: 'trello_token',
    })
  })

  it('includes metadata when provided', () => {
    const result = buildCredentials({
      service: 'grafana',
      accessToken: 'glsa_xxx',
      metadata: { base_url: 'https://grafana.example.com' },
    })
    expect(result.metadata).toEqual({ base_url: 'https://grafana.example.com' })
  })
})
