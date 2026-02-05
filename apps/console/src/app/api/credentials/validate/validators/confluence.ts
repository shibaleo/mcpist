import { ValidationParams, ValidationResult } from './types'

export async function validateConfluenceToken(params: ValidationParams): Promise<ValidationResult> {
  const { token, email, domain } = params

  if (!email || !domain) {
    return {
      valid: false,
      error: 'Confluenceにはメールアドレスとドメインが必要です',
    }
  }

  try {
    const auth = Buffer.from(`${email}:${token}`).toString('base64')
    const response = await fetch(`https://${domain}/wiki/rest/api/user/current`, {
      method: 'GET',
      headers: {
        'Authorization': `Basic ${auth}`,
        'Accept': 'application/json',
      },
    })

    if (response.ok) {
      const data = await response.json()
      return {
        valid: true,
        details: {
          accountId: data.accountId,
          displayName: data.displayName,
          email: data.email,
        },
      }
    }

    if (response.status === 401) {
      return {
        valid: false,
        error: '認証に失敗しました。メールアドレスとAPIトークンを確認してください。',
      }
    }

    if (response.status === 403) {
      return {
        valid: false,
        error: 'アクセス権限がありません。APIトークンのスコープを確認してください。',
      }
    }

    if (response.status === 404) {
      return {
        valid: false,
        error: 'ドメインが見つかりません。正しいAtlassianドメインを入力してください。',
      }
    }

    return {
      valid: false,
      error: `API接続エラー (${response.status})`,
    }
  } catch {
    return {
      valid: false,
      error: 'ネットワークエラーが発生しました。ドメインが正しいか確認してください。',
    }
  }
}
