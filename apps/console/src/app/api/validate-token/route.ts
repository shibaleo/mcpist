import { NextRequest, NextResponse } from 'next/server'

interface ValidationResult {
  valid: boolean
  error?: string
  details?: Record<string, unknown>
}

async function validateNotionToken(token: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling Notion API...')
    const response = await fetch('https://api.notion.com/v1/users/me', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Notion-Version': '2022-06-28',
      },
    })

    console.log('[validate-token] Notion API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, user:', data.name)
      return {
        valid: true,
        details: {
          name: data.name,
          type: data.type,
        },
      }
    }

    if (response.status === 401) {
      console.log('[validate-token] Invalid token (401)')
      return {
        valid: false,
        error: 'トークンが無効です。正しいIntegration Tokenを入力してください。',
      }
    }

    console.log('[validate-token] API error:', response.status)
    return {
      valid: false,
      error: `API接続エラー (${response.status})`,
    }
  } catch (error) {
    console.error('[validate-token] Network error:', error)
    return {
      valid: false,
      error: 'ネットワークエラーが発生しました',
    }
  }
}

export async function POST(request: NextRequest) {
  try {
    const { service, token } = await request.json()

    console.log('[validate-token] Request received for service:', service)

    if (!service || !token) {
      return NextResponse.json(
        { valid: false, error: 'service と token が必要です' },
        { status: 400 }
      )
    }

    let result: ValidationResult

    switch (service) {
      case 'notion':
        result = await validateNotionToken(token)
        break
      default:
        console.log('[validate-token] Unknown service, skipping validation')
        // For unsupported services, skip validation
        result = { valid: true }
    }

    console.log('[validate-token] Returning result:', JSON.stringify(result))
    return NextResponse.json(result)
  } catch (error) {
    console.error('[validate-token] Error:', error)
    return NextResponse.json(
      { valid: false, error: 'リクエストの処理に失敗しました' },
      { status: 500 }
    )
  }
}
