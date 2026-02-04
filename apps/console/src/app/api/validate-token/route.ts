import { NextRequest, NextResponse } from 'next/server'

interface ValidationResult {
  valid: boolean
  error?: string
  details?: Record<string, unknown>
}

async function validateSupabaseToken(token: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling Supabase Management API...')
    const response = await fetch('https://api.supabase.com/v1/projects', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    })

    console.log('[validate-token] Supabase API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, projects count:', data.length)
      return {
        valid: true,
        details: {
          projectCount: data.length,
        },
      }
    }

    if (response.status === 401) {
      console.log('[validate-token] Invalid token (401)')
      return {
        valid: false,
        error: 'トークンが無効です。正しいPersonal Access Tokenを入力してください。',
      }
    }

    if (response.status === 403) {
      console.log('[validate-token] Forbidden (403)')
      return {
        valid: false,
        error: 'アクセス権限がありません。Management APIへのアクセス権があるか確認してください。',
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

async function validateGitHubToken(token: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling GitHub API...')
    const response = await fetch('https://api.github.com/user', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
      },
    })

    console.log('[validate-token] GitHub API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, user:', data.login)
      return {
        valid: true,
        details: {
          login: data.login,
          name: data.name,
        },
      }
    }

    if (response.status === 401) {
      console.log('[validate-token] Invalid token (401)')
      return {
        valid: false,
        error: 'トークンが無効です。正しいPersonal Access Tokenを入力してください。',
      }
    }

    if (response.status === 403) {
      console.log('[validate-token] Forbidden (403)')
      return {
        valid: false,
        error: 'アクセス権限がありません。トークンのスコープを確認してください。',
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

async function validateJiraToken(email: string, token: string, domain: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling Jira API...')
    // Basic認証: email:token をBase64エンコード
    const auth = Buffer.from(`${email}:${token}`).toString('base64')
    const response = await fetch(`https://${domain}/rest/api/3/myself`, {
      method: 'GET',
      headers: {
        'Authorization': `Basic ${auth}`,
        'Accept': 'application/json',
      },
    })

    console.log('[validate-token] Jira API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, user:', data.displayName)
      return {
        valid: true,
        details: {
          accountId: data.accountId,
          displayName: data.displayName,
          emailAddress: data.emailAddress,
        },
      }
    }

    if (response.status === 401) {
      console.log('[validate-token] Invalid token (401)')
      return {
        valid: false,
        error: '認証に失敗しました。メールアドレスとAPIトークンを確認してください。',
      }
    }

    if (response.status === 403) {
      console.log('[validate-token] Forbidden (403)')
      return {
        valid: false,
        error: 'アクセス権限がありません。APIトークンのスコープを確認してください。',
      }
    }

    if (response.status === 404) {
      console.log('[validate-token] Not found (404)')
      return {
        valid: false,
        error: 'ドメインが見つかりません。正しいAtlassianドメインを入力してください。',
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
      error: 'ネットワークエラーが発生しました。ドメインが正しいか確認してください。',
    }
  }
}

async function validateConfluenceToken(email: string, token: string, domain: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling Confluence API...')
    // Basic認証: email:token をBase64エンコード
    const auth = Buffer.from(`${email}:${token}`).toString('base64')
    const response = await fetch(`https://${domain}/wiki/rest/api/user/current`, {
      method: 'GET',
      headers: {
        'Authorization': `Basic ${auth}`,
        'Accept': 'application/json',
      },
    })

    console.log('[validate-token] Confluence API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, user:', data.displayName)
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
      console.log('[validate-token] Invalid token (401)')
      return {
        valid: false,
        error: '認証に失敗しました。メールアドレスとAPIトークンを確認してください。',
      }
    }

    if (response.status === 403) {
      console.log('[validate-token] Forbidden (403)')
      return {
        valid: false,
        error: 'アクセス権限がありません。APIトークンのスコープを確認してください。',
      }
    }

    if (response.status === 404) {
      console.log('[validate-token] Not found (404)')
      return {
        valid: false,
        error: 'ドメインが見つかりません。正しいAtlassianドメインを入力してください。',
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
      error: 'ネットワークエラーが発生しました。ドメインが正しいか確認してください。',
    }
  }
}

async function validateTrelloToken(apiKey: string, token: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling Trello API...')
    const response = await fetch(`https://api.trello.com/1/members/me?key=${apiKey}&token=${token}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    })

    console.log('[validate-token] Trello API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, user:', data.username)
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
      console.log('[validate-token] Invalid token (401)')
      return {
        valid: false,
        error: 'API KeyまたはTokenが無効です。正しい認証情報を入力してください。',
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

async function validateGrafanaToken(token: string, baseUrl: string): Promise<ValidationResult> {
  try {
    console.log('[validate-token] Calling Grafana API...')
    const url = `${baseUrl.replace(/\/+$/, '')}/api/org`
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/json',
      },
    })

    console.log('[validate-token] Grafana API response status:', response.status)

    if (response.ok) {
      const data = await response.json()
      console.log('[validate-token] Valid token, org:', data.name)
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
  } catch (error) {
    console.error('[validate-token] Network error:', error)
    return {
      valid: false,
      error: 'ネットワークエラーが発生しました。Grafana URLが正しいか確認してください。',
    }
  }
}

export async function POST(request: NextRequest) {
  try {
    const { service, token, email, domain, api_key, base_url } = await request.json()

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
      case 'github':
        result = await validateGitHubToken(token)
        break
      case 'supabase':
        result = await validateSupabaseToken(token)
        break
      case 'jira':
        if (!email || !domain) {
          return NextResponse.json(
            { valid: false, error: 'Jiraにはメールアドレスとドメインが必要です' },
            { status: 400 }
          )
        }
        result = await validateJiraToken(email, token, domain)
        break
      case 'confluence':
        if (!email || !domain) {
          return NextResponse.json(
            { valid: false, error: 'Confluenceにはメールアドレスとドメインが必要です' },
            { status: 400 }
          )
        }
        result = await validateConfluenceToken(email, token, domain)
        break
      case 'trello':
        if (!api_key) {
          return NextResponse.json(
            { valid: false, error: 'TrelloにはAPI Keyが必要です' },
            { status: 400 }
          )
        }
        result = await validateTrelloToken(api_key, token)
        break
      case 'grafana':
        if (!base_url) {
          return NextResponse.json(
            { valid: false, error: 'GrafanaにはBase URLが必要です' },
            { status: 400 }
          )
        }
        result = await validateGrafanaToken(token, base_url)
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
