import { ValidationParams, ValidationResult } from './types'

export async function validateNotionToken(params: ValidationParams): Promise<ValidationResult> {
  const { token } = params

  try {
    const response = await fetch('https://api.notion.com/v1/users/me', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Notion-Version': '2022-06-28',
      },
      signal: AbortSignal.timeout(5000),
      redirect: 'error',
    })

    if (response.ok) {
      const data = await response.json()
      return {
        valid: true,
        details: {
          name: data.name,
          type: data.type,
        },
      }
    }

    if (response.status === 401) {
      return {
        valid: false,
        error: 'トークンが無効です。正しいIntegration Tokenを入力してください。',
      }
    }

    return {
      valid: false,
      error: `API接続エラー (${response.status})`,
    }
  } catch {
    return {
      valid: false,
      error: 'ネットワークエラーが発生しました',
    }
  }
}
