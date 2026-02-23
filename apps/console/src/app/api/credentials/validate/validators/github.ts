import { ValidationParams, ValidationResult } from './types'

export async function validateGitHubToken(params: ValidationParams): Promise<ValidationResult> {
  const { token } = params

  try {
    const response = await fetch('https://api.github.com/user', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
      },
      signal: AbortSignal.timeout(5000),
      redirect: 'error',
    })

    if (response.ok) {
      const data = await response.json()
      return {
        valid: true,
        details: {
          login: data.login,
          name: data.name,
        },
      }
    }

    if (response.status === 401) {
      return {
        valid: false,
        error: 'トークンが無効です。正しいPersonal Access Tokenを入力してください。',
      }
    }

    if (response.status === 403) {
      return {
        valid: false,
        error: 'アクセス権限がありません。トークンのスコープを確認してください。',
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
