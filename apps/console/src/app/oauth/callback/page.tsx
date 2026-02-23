'use client'

import { Suspense } from 'react'
import { useEffect, useState, useRef, useCallback } from 'react'
import { useSearchParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CheckCircle2, XCircle, Loader2, Play, ChevronDown, ChevronRight, Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getOAuthClientId } from '@/lib/oauth/client'

type TestStep = {
  name: string
  status: 'pending' | 'running' | 'success' | 'error'
  message?: string
  response?: unknown
}

// JSON Response Viewer Component
function JsonResponseViewer({ data, label }: { data: unknown; label: string }) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [copied, setCopied] = useState(false)

  const jsonString = JSON.stringify(data, null, 2)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(jsonString)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="mt-2 border rounded-lg overflow-hidden">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-2 bg-muted/50 hover:bg-muted text-left text-xs"
      >
        <span className="flex items-center gap-1 font-medium">
          {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          {label}
        </span>
        <span className="text-muted-foreground">
          {jsonString.length} bytes
        </span>
      </button>
      {isExpanded && (
        <div className="relative">
          <button
            onClick={handleCopy}
            className="absolute top-2 right-2 p-1 rounded bg-background/80 hover:bg-background border text-muted-foreground hover:text-foreground"
            title="Copy JSON"
          >
            {copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
          </button>
          <pre className="p-3 text-xs font-mono overflow-x-auto bg-secondary/50 max-h-64 overflow-y-auto">
            {jsonString}
          </pre>
        </div>
      )}
    </div>
  )
}

function CallbackContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing')
  const [message, setMessage] = useState('認可コードを処理中...')
  const isProcessingRef = useRef(false)

  // MCP Connection Test state
  const [isTesting, setIsTesting] = useState(false)
  const [testSteps, setTestSteps] = useState<TestStep[]>([])
  const [testComplete, setTestComplete] = useState(false)

  const code = searchParams.get('code')
  const state = searchParams.get('state')
  const error = searchParams.get('error')
  const errorDescription = searchParams.get('error_description')

  useEffect(() => {
    // Prevent double execution in React Strict Mode
    // Check both ref (for same render cycle) and sessionStorage (for re-mounts)
    if (isProcessingRef.current) {
      console.log('[OAuth Callback] Already processing (ref check)')
      return
    }

    // Check if this specific code was already processed - do this SYNCHRONOUSLY before setting ref
    if (code) {
      const processedCode = sessionStorage.getItem('oauth_processed_code')
      if (processedCode === code) {
        console.log('[OAuth Callback] Code already processed:', code.substring(0, 8) + '...')
        // Already processed this code, check if we have a token
        const existingToken = sessionStorage.getItem('mcp_access_token')
        if (existingToken) {
          setStatus('success')
          setMessage('')
        }
        return
      }
      // Mark this code as being processed IMMEDIATELY (synchronous)
      sessionStorage.setItem('oauth_processed_code', code)
      console.log('[OAuth Callback] Marked code as processing:', code.substring(0, 8) + '...')
    }

    isProcessingRef.current = true

    const processCallback = async () => {
      if (error) {
        setStatus('error')
        setMessage(`${error}: ${errorDescription || '認可に失敗しました'}`)
        sessionStorage.removeItem('oauth_processed_code')
        return
      }

      if (!code) {
        setStatus('error')
        setMessage('認可コードがありません')
        return
      }

      // Verify state
      const storedState = sessionStorage.getItem('oauth_state')
      if (state !== storedState) {
        setStatus('error')
        setMessage('state が一致しません（CSRF攻撃の可能性）')
        sessionStorage.removeItem('oauth_processed_code')
        return
      }

      // Get stored verifier
      const verifier = sessionStorage.getItem('oauth_verifier')
      if (!verifier) {
        setStatus('error')
        setMessage('code_verifier が見つかりません')
        sessionStorage.removeItem('oauth_processed_code')
        return
      }

      // Get OAuth metadata
      try {
        const metadataRes = await fetch('/.well-known/oauth-authorization-server')
        if (!metadataRes.ok) {
          throw new Error('OAuth メタデータの取得に失敗しました')
        }
        const metadata = await metadataRes.json()

        // Exchange code for token
        setMessage('トークンを交換中...')
        const redirectUri = `${window.location.origin}/oauth/callback`

        const tokenRes = await fetch(metadata.token_endpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
          },
          // FIXME: RFC 8707 Resource Indicators — resource パラメータ未送信 (BL-060)
          body: new URLSearchParams({
            grant_type: 'authorization_code',
            code: code,
            redirect_uri: redirectUri,
            client_id: getOAuthClientId(),
            code_verifier: verifier,
          }),
        })

        if (!tokenRes.ok) {
          const errorText = await tokenRes.text()
          throw new Error(`トークン交換に失敗しました: ${errorText}`)
        }

        const tokenData = await tokenRes.json()
        setStatus('success')
        setMessage('')

        // Clean up sessionStorage (keep oauth_processed_code to prevent re-processing)
        sessionStorage.removeItem('oauth_state')
        sessionStorage.removeItem('oauth_verifier')

        // Store token data for MCP connection page
        // Calculate expiration time (current time + expires_in seconds)
        const expiresAt = Date.now() + (tokenData.expires_in || 3600) * 1000
        sessionStorage.setItem('mcp_access_token', tokenData.access_token)
        sessionStorage.setItem('mcp_refresh_token', tokenData.refresh_token || '')
        sessionStorage.setItem('mcp_token_expires_at', String(expiresAt))
        sessionStorage.setItem('mcp_token_scope', tokenData.scope || '')
      } catch (err) {
        setStatus('error')
        setMessage(String(err))
        // Remove processed code marker on error so user can retry with new code
        sessionStorage.removeItem('oauth_processed_code')
      }
    }

    processCallback()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const updateTestStep = useCallback((index: number, update: Partial<TestStep>) => {
    setTestSteps((prev) =>
      prev.map((step, i) => (i === index ? { ...step, ...update } : step))
    )
  }, [])

  // MCP Connection Test using the obtained OAuth token
  const testMcpConnection = async () => {
    const accessToken = sessionStorage.getItem('mcp_access_token')
    if (!accessToken) {
      return
    }

    setIsTesting(true)
    setTestComplete(false)
    setTestSteps([
      { name: 'MCP Server 接続', status: 'pending' },
      { name: 'initialize', status: 'pending' },
      { name: 'tools/list', status: 'pending' },
    ])

    const mcpServerUrl = process.env.NEXT_PUBLIC_MCP_SERVER_URL!
    const mcpEndpoint = `${mcpServerUrl}/v1/mcp`

    // Step 1: Connect to MCP Server
    updateTestStep(0, { status: 'running' })
    try {
      const response = await fetch(mcpEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${accessToken}`,
        },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'initialize',
          params: {
            protocolVersion: '2025-03-26',
            capabilities: {},
            clientInfo: { name: 'MCPist Console', version: '1.0.0' },
          },
        }),
      })

      if (response.status === 401) {
        const errorText = await response.text()
        let errorData: unknown = errorText
        try { errorData = JSON.parse(errorText) } catch { /* keep as text */ }
        updateTestStep(0, { status: 'error', message: '認証失敗 (401)', response: errorData })
        setIsTesting(false)
        setTestComplete(true)
        return
      }

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`HTTP ${response.status}: ${errorText}`)
      }

      updateTestStep(0, { status: 'success', message: '接続成功' })

      // Step 2: Check initialize response
      updateTestStep(1, { status: 'running' })
      const initData = await response.json()
      if (initData.result) {
        updateTestStep(0, { status: 'success', message: '接続成功', response: initData })
        updateTestStep(1, { status: 'success', message: `v${initData.result.protocolVersion}`, response: initData })
      } else if (initData.error) {
        updateTestStep(1, { status: 'error', message: initData.error.message, response: initData })
        setIsTesting(false)
        setTestComplete(true)
        return
      }

      // Step 3: Get tools/list
      updateTestStep(2, { status: 'running' })
      const toolsRes = await fetch(mcpEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${accessToken}`,
        },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 2,
          method: 'tools/list',
        }),
      })

      const toolsData = await toolsRes.json()
      if (toolsData.result) {
        const toolCount = toolsData.result.tools?.length || 0
        updateTestStep(2, { status: 'success', message: `${toolCount} tools`, response: toolsData })
      } else if (toolsData.error) {
        updateTestStep(2, { status: 'error', message: toolsData.error.message, response: toolsData })
      }
    } catch (err) {
      updateTestStep(0, { status: 'error', message: String(err) })
    }

    setIsTesting(false)
    setTestComplete(true)
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          {status === 'processing' && (
            <Loader2 className="h-12 w-12 animate-spin text-primary mx-auto mb-4" />
          )}
          {status === 'success' && (
            <CheckCircle2 className="h-12 w-12 text-green-500 mx-auto mb-4" />
          )}
          {status === 'error' && (
            <XCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
          )}
          <CardTitle>
            {status === 'processing' && '処理中...'}
            {status === 'success' && '認証成功'}
            {status === 'error' && '認証失敗'}
          </CardTitle>
          {message && <CardDescription>{message}</CardDescription>}
        </CardHeader>
        <CardContent className="space-y-4">
          {status === 'success' && (
            <>
              <p className="text-sm text-center text-muted-foreground">
                安全にMCPクライアントと接続できます
              </p>

              {/* MCP Connection Test */}
              {!testComplete && (
                <Button
                  onClick={testMcpConnection}
                  disabled={isTesting}
                  variant="outline"
                  className="w-full"
                >
                  {isTesting ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      テスト中...
                    </>
                  ) : (
                    <>
                      <Play className="h-4 w-4 mr-2" />
                      MCP接続テスト
                    </>
                  )}
                </Button>
              )}

              {/* Test Steps */}
              {testSteps.length > 0 && (
                <div className="space-y-2">
                  {testSteps.map((step, index) => (
                    <div key={index}>
                      <div
                        className={cn(
                          'flex items-center gap-2 p-2 rounded-lg text-sm',
                          step.status === 'success' && 'bg-green-500/10',
                          step.status === 'error' && 'bg-destructive/10',
                          step.status === 'running' && 'bg-primary/10'
                        )}
                      >
                        {step.status === 'pending' && (
                          <div className="h-4 w-4 rounded-full border-2 border-muted" />
                        )}
                        {step.status === 'running' && (
                          <Loader2 className="h-4 w-4 animate-spin text-primary" />
                        )}
                        {step.status === 'success' && (
                          <CheckCircle2 className="h-4 w-4 text-green-500" />
                        )}
                        {step.status === 'error' && (
                          <XCircle className="h-4 w-4 text-destructive" />
                        )}
                        <span className="flex-1">{step.name}</span>
                        {step.message && (
                          <span className="text-xs text-muted-foreground">
                            {step.message}
                          </span>
                        )}
                      </div>
                      {step.response !== undefined && (
                        <div className="ml-6">
                          <JsonResponseViewer data={step.response} label="Response JSON" />
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              <Button
                onClick={() => router.push('/dashboard')}
                className="w-full"
              >
                ダッシュボードに戻る
              </Button>
            </>
          )}
          {status === 'error' && (
            <Button
              variant="outline"
              onClick={() => router.push('/dashboard')}
              className="w-full"
            >
              戻る
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function CallbackPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    }>
      <CallbackContent />
    </Suspense>
  )
}
