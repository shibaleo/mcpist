import { ValidationParams, ValidationResult } from './types'

export async function validateGrafanaToken(params: ValidationParams): Promise<ValidationResult> {
  const { token, base_url } = params

  if (!base_url) {
    return {
      valid: false,
      error: 'GrafanaにはBase URLが必要です',
    }
  }

  try {
    const parsed = new URL(base_url)
    if (parsed.protocol !== 'https:') {
      return { valid: false, error: 'Base URLはhttpsのみ対応しています。' }
    }
    const hostname = parsed.hostname.toLowerCase()
    if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1'
        || hostname.startsWith('10.') || hostname.startsWith('192.168.')
        || /^172\.(1[6-9]|2\d|3[01])\./.test(hostname)
        || hostname === '169.254.169.254' || hostname.endsWith('.internal')) {
      return { valid: false, error: 'プライベートアドレスは指定できません。' }
    }
  } catch {
    return { valid: false, error: 'Base URLの形式が不正です。' }
  }

  try {
    const url = `${base_url.replace(/\/+$/, '')}/api/org`
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/json',
      },
      signal: AbortSignal.timeout(5000),
      redirect: 'error',
    })

    if (response.ok) {
      const data = await response.json()
      return {
        valid: true,
        details: {
          orgId: data.id,
          orgName: data.name,
        },
      }
    }

    if (response.status === 401) {
      return {
        valid: false,
        error: 'Service Account Tokenが無効です。正しいトークンを入力してください。',
      }
    }

    return {
      valid: false,
      error: `API接続エラー (${response.status})`,
    }
  } catch {
    return {
      valid: false,
      error: 'ネットワークエラーが発生しました。Grafana URLが正しいか確認してください。',
    }
  }
}
