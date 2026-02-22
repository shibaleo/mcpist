import { ValidationParams, ValidationResult } from './types'

export async function validateTrelloToken(params: ValidationParams): Promise<ValidationResult> {
  const { token, api_key } = params

  if (!api_key) {
    return {
      valid: false,
      error: 'TrelloにはAPI Keyが必要です',
    }
  }

  try {
    const response = await fetch(`https://api.trello.com/1/members/me?key=${api_key}&token=${token}`, {
      method: 'GET',
      headers: {
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
          id: data.id,
          username: data.username,
          fullName: data.fullName,
        },
      }
    }

    if (response.status === 401) {
      return {
        valid: false,
        error: 'API KeyまたはTokenが無効です。正しい認証情報を入力してください。',
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
