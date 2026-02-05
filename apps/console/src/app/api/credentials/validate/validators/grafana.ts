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
    const url = `${base_url.replace(/\/+$/, '')}/api/org`
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/json',
      },
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
