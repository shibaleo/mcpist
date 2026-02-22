import { ValidationParams, ValidationResult } from './types'

export async function validateSupabaseToken(params: ValidationParams): Promise<ValidationResult> {
  const { token } = params

  try {
    const response = await fetch('https://api.supabase.com/v1/projects', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      signal: AbortSignal.timeout(5000),
      redirect: 'error',
    })

    if (response.ok) {
      const data = await response.json()
      return {
        valid: true,
        details: {
          projectCount: data.length,
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
        error: 'アクセス権限がありません。Management APIへのアクセス権があるか確認してください。',
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
