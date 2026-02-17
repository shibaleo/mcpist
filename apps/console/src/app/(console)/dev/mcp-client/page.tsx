'use client'

import { useState, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getOrRegisterOAuthClient, getOAuthClientId } from '@/lib/oauth/client'

/**
 * MCP Client Mock - OAuth 2.1 + PKCE Authorization Flow
 *
 * DAY8 itr-clt.md に準拠した認可フロー:
 * 1. MCP Server に接続試行 → 401
 * 2. /.well-known/oauth-protected-resource を取得
 * 3. Authorization Server メタデータを取得
 * 4. PKCE code_verifier / code_challenge 生成
 * 5. 認可リクエスト（/authorize）
 * 6. コールバックで認可コード受け取り
 * 7. トークン交換（/token）
 * 8. JWT で MCP Server にリクエスト
 */

interface OAuthMetadata {
  issuer: string
  authorization_endpoint: string
  token_endpoint: string
  jwks_uri: string
  response_types_supported: string[]
  grant_types_supported: string[]
  code_challenge_methods_supported: string[]
}

interface ProtectedResourceMetadata {
  resource: string
  authorization_servers: string[]
  scopes_supported: string[]
  bearer_methods_supported: string[]
}

interface TokenResponse {
  access_token: string
  token_type: string
  expires_in: number
  refresh_token?: string
}

// PKCE helpers
function generateRandomString(length: number): string {
  const array = new Uint8Array(length)
  crypto.getRandomValues(array)
  return Array.from(array, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(verifier)
  const digest = await crypto.subtle.digest('SHA-256', data)
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

export default function McpClientPage() {
  const [mcpServerUrl, setMcpServerUrl] = useState(process.env.NEXT_PUBLIC_MCP_SERVER_URL!)
  const [consoleUrl, setConsoleUrl] = useState(process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000')
  const [logs, setLogs] = useState<string[]>([])
  const [protectedResource, setProtectedResource] = useState<ProtectedResourceMetadata | null>(null)
  const [authMetadata, setAuthMetadata] = useState<OAuthMetadata | null>(null)
  const [accessToken, setAccessToken] = useState<string | null>(null)
  const [codeVerifier, setCodeVerifier] = useState<string | null>(null)

  const addLog = useCallback((message: string) => {
    const timestamp = new Date().toLocaleTimeString()
    setLogs((prev) => [...prev, `[${timestamp}] ${message}`])
  }, [])

  const clearLogs = useCallback(() => {
    setLogs([])
  }, [])

  // Step 1: Try to connect to MCP Server (expect 401)
  const tryConnect = async () => {
    addLog(`Step 1: MCP Server (${mcpServerUrl}/v1/mcp) に接続試行...`)

    try {
      const response = await fetch(`${mcpServerUrl}/v1/mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'initialize',
          params: {
            protocolVersion: '2025-03-26',
            capabilities: {},
            clientInfo: { name: 'MCP Client Mock', version: '1.0.0' },
          },
        }),
      })

      if (response.status === 401) {
        addLog('✓ 401 Unauthorized を受信（想定通り）')
        addLog('Step 2: /.well-known/oauth-protected-resource を取得...')
        await fetchProtectedResource()
      } else {
        addLog(`⚠ 予期しないステータス: ${response.status}`)
        const text = await response.text()
        addLog(`レスポンス: ${text.substring(0, 200)}`)
      }
    } catch (error) {
      addLog(`❌ エラー: ${error}`)
    }
  }

  // Step 2: Fetch protected resource metadata
  const fetchProtectedResource = async () => {
    try {
      // Note: In production, this would be fetched from MCP Server
      // For local dev, we use Console's endpoint
      const response = await fetch(`${consoleUrl}/.well-known/oauth-protected-resource`)
      const metadata: ProtectedResourceMetadata = await response.json()

      setProtectedResource(metadata)
      addLog(`✓ Protected Resource Metadata 取得:`)
      addLog(`  resource: ${metadata.resource}`)
      addLog(`  authorization_servers: ${metadata.authorization_servers.join(', ')}`)

      addLog('Step 3: Authorization Server Metadata を取得...')
      await fetchAuthMetadata(metadata.authorization_servers[0])
    } catch (error) {
      addLog(`❌ エラー: ${error}`)
    }
  }

  // Step 3: Fetch authorization server metadata
  const fetchAuthMetadata = async (authServerUrl: string) => {
    try {
      // Try oauth-authorization-server first, then openid-configuration
      let response = await fetch(`${authServerUrl}/.well-known/oauth-authorization-server`)
      if (!response.ok) {
        response = await fetch(`${authServerUrl}/.well-known/openid-configuration`)
      }

      const metadata: OAuthMetadata = await response.json()
      setAuthMetadata(metadata)

      addLog(`✓ Authorization Server Metadata 取得:`)
      addLog(`  issuer: ${metadata.issuer}`)
      addLog(`  authorization_endpoint: ${metadata.authorization_endpoint}`)
      addLog(`  token_endpoint: ${metadata.token_endpoint}`)
      addLog(`  code_challenge_methods: ${metadata.code_challenge_methods_supported?.join(', ')}`)
    } catch (error) {
      addLog(`❌ エラー: ${error}`)
    }
  }

  // Step 4 & 5: Start authorization flow with PKCE
  const startAuthFlow = async () => {
    if (!authMetadata) {
      addLog('❌ Authorization Server Metadata がありません')
      return
    }

    addLog('Step 4: PKCE code_verifier / code_challenge を生成...')

    // Generate PKCE values
    const verifier = generateRandomString(32)
    const challenge = await generateCodeChallenge(verifier)
    setCodeVerifier(verifier)

    addLog(`  code_verifier: ${verifier.substring(0, 16)}...`)
    addLog(`  code_challenge: ${challenge.substring(0, 16)}...`)

    // Generate state for CSRF protection
    const state = generateRandomString(16)

    // Get or register OAuth client dynamically
    addLog('Step 5: OAuth Client を取得/登録中...')
    const clientId = await getOrRegisterOAuthClient()
    addLog(`  client_id: ${clientId}`)

    // Build authorization URL
    // FIXME: RFC 8707 Resource Indicators — resource パラメータ未送信 (BL-060)
    const redirectUri = `${consoleUrl}/dev/mcp-client/callback`
    const params = new URLSearchParams({
      response_type: 'code',
      client_id: clientId,
      redirect_uri: redirectUri,
      scope: 'openid profile email',
      code_challenge: challenge,
      code_challenge_method: 'S256',
      state: state,
    })

    const authUrl = `${authMetadata.authorization_endpoint}?${params.toString()}`

    addLog('Step 5: 認可リクエストを開始...')
    addLog(`  redirect_uri: ${redirectUri}`)
    addLog(`  認可URL: ${authUrl.substring(0, 100)}...`)

    // Store state and verifier in sessionStorage for callback
    sessionStorage.setItem('oauth_state', state)
    sessionStorage.setItem('oauth_verifier', verifier)

    // Open authorization URL
    window.open(authUrl, '_blank', 'width=600,height=700')
    addLog('⏳ ブラウザで認証を完了してください...')
  }

  // Step 7: Exchange code for token
  const exchangeToken = async (code: string) => {
    if (!authMetadata || !codeVerifier) {
      addLog('❌ 必要な情報が不足しています')
      return
    }

    addLog('Step 7: トークン交換...')

    try {
      const redirectUri = `${consoleUrl}/dev/mcp-client/callback`
      const response = await fetch(authMetadata.token_endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
          grant_type: 'authorization_code',
          code: code,
          redirect_uri: redirectUri,
          client_id: getOAuthClientId(),
          code_verifier: codeVerifier,
        }),
      })

      if (!response.ok) {
        const error = await response.text()
        addLog(`❌ トークン交換失敗: ${error}`)
        return
      }

      const tokenResponse: TokenResponse = await response.json()
      setAccessToken(tokenResponse.access_token)

      addLog('✓ トークン取得成功!')
      addLog(`  token_type: ${tokenResponse.token_type}`)
      addLog(`  expires_in: ${tokenResponse.expires_in}秒`)
      addLog(`  access_token: ${tokenResponse.access_token.substring(0, 50)}...`)
    } catch (error) {
      addLog(`❌ エラー: ${error}`)
    }
  }

  // Step 8: Call MCP Server with token
  const callMcpServer = async () => {
    if (!accessToken) {
      addLog('❌ Access Token がありません')
      return
    }

    addLog('Step 8: MCP Server に認証済みリクエスト...')

    try {
      const response = await fetch(`${mcpServerUrl}/v1/mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${accessToken}`,
        },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'tools/list',
        }),
      })

      const result = await response.json()
      addLog(`✓ レスポンス受信 (status: ${response.status}):`)
      addLog(JSON.stringify(result, null, 2))
    } catch (error) {
      addLog(`❌ エラー: ${error}`)
    }
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">MCP Client Mock</h1>
          <p className="text-muted-foreground">
            OAuth 2.1 + PKCE 認可フローのテスト
          </p>
        </div>
        <Button variant="outline" onClick={clearLogs}>
          ログをクリア
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>設定</CardTitle>
            <CardDescription>MCP Server と Console の URL</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="mcpServer">MCP Server URL</Label>
              <Input
                id="mcpServer"
                value={mcpServerUrl}
                onChange={(e) => setMcpServerUrl(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="consoleUrl">Console URL</Label>
              <Input
                id="consoleUrl"
                value={consoleUrl}
                onChange={(e) => setConsoleUrl(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        {/* Actions */}
        <Card>
          <CardHeader>
            <CardTitle>認可フロー</CardTitle>
            <CardDescription>ステップごとに実行</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Button onClick={tryConnect} className="w-full">
              Step 1-3: 接続試行 → メタデータ取得
            </Button>
            <Button
              onClick={startAuthFlow}
              disabled={!authMetadata}
              className="w-full"
            >
              Step 4-5: 認可フロー開始 (PKCE)
            </Button>
            <div className="flex gap-2">
              <Input
                placeholder="認可コードを入力..."
                id="authCode"
              />
              <Button
                onClick={() => {
                  const input = document.getElementById('authCode') as HTMLInputElement
                  if (input.value) {
                    exchangeToken(input.value)
                  }
                }}
                disabled={!codeVerifier}
              >
                Step 7: 交換
              </Button>
            </div>
            <Button
              onClick={callMcpServer}
              disabled={!accessToken}
              className="w-full"
              variant="secondary"
            >
              Step 8: MCP Server 呼び出し
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Logs */}
      <Card>
        <CardHeader>
          <CardTitle>ログ</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="bg-muted p-4 rounded-lg text-sm overflow-auto max-h-96 font-mono">
            {logs.length === 0 ? (
              <span className="text-muted-foreground">
                「Step 1-3」ボタンをクリックして開始...
              </span>
            ) : (
              logs.map((log, i) => <div key={i}>{log}</div>)
            )}
          </pre>
        </CardContent>
      </Card>

      {/* Current State */}
      {(protectedResource || authMetadata || accessToken) && (
        <Card>
          <CardHeader>
            <CardTitle>現在の状態</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {protectedResource && (
              <div>
                <h4 className="font-medium mb-2">Protected Resource Metadata</h4>
                <pre className="bg-muted p-2 rounded text-xs">
                  {JSON.stringify(protectedResource, null, 2)}
                </pre>
              </div>
            )}
            {authMetadata && (
              <div>
                <h4 className="font-medium mb-2">Authorization Server Metadata</h4>
                <pre className="bg-muted p-2 rounded text-xs">
                  {JSON.stringify(authMetadata, null, 2)}
                </pre>
              </div>
            )}
            {accessToken && (
              <div>
                <h4 className="font-medium mb-2">Access Token</h4>
                <pre className="bg-muted p-2 rounded text-xs break-all">
                  {accessToken}
                </pre>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
