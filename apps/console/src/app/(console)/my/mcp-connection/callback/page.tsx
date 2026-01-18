'use client'

import { useEffect, useState } from 'react'
import { useSearchParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CheckCircle2, XCircle, Loader2 } from 'lucide-react'

/**
 * OAuth 2.1 Callback Page
 *
 * 認可コードを受け取り、トークン交換を行う
 */

export default function CallbackPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing')
  const [message, setMessage] = useState('認可コードを処理中...')

  const code = searchParams.get('code')
  const state = searchParams.get('state')
  const error = searchParams.get('error')
  const errorDescription = searchParams.get('error_description')

  useEffect(() => {
    const processCallback = async () => {
      if (error) {
        setStatus('error')
        setMessage(`${error}: ${errorDescription || '認可に失敗しました'}`)
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
        return
      }

      // Get stored verifier
      const verifier = sessionStorage.getItem('oauth_verifier')
      if (!verifier) {
        setStatus('error')
        setMessage('code_verifier が見つかりません')
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
        const redirectUri = `${window.location.origin}/my/mcp-connection/callback`

        const tokenRes = await fetch(metadata.token_endpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
          },
          body: new URLSearchParams({
            grant_type: 'authorization_code',
            code: code,
            redirect_uri: redirectUri,
            client_id: 'mcpist-console',
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

        // Clean up sessionStorage
        sessionStorage.removeItem('oauth_state')
        sessionStorage.removeItem('oauth_verifier')

        // Store token for MCP connection page
        sessionStorage.setItem('mcp_access_token', tokenData.access_token)
      } catch (err) {
        setStatus('error')
        setMessage(String(err))
      }
    }

    processCallback()
  }, [code, state, error, errorDescription])

  return (
    <div className="p-6 flex items-center justify-center min-h-[60vh]">
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
              <Button
                onClick={() => router.push('/my/mcp-connection')}
                className="w-full"
              >
                接続ページに戻る
              </Button>
            </>
          )}
          {status === 'error' && (
            <Button
              variant="outline"
              onClick={() => router.push('/my/mcp-connection')}
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
