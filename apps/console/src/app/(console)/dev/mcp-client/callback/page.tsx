'use client'

import { Suspense } from 'react'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'

function CallbackContent() {
  const searchParams = useSearchParams()
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing')
  const [message, setMessage] = useState('')

  const code = searchParams.get('code')
  const state = searchParams.get('state')
  const error = searchParams.get('error')
  const errorDescription = searchParams.get('error_description')

  useEffect(() => {
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

    setStatus('success')
    setMessage('認可コードを取得しました')

    // Clear stored state
    sessionStorage.removeItem('oauth_state')
  }, [code, state, error, errorDescription])

  const copyCode = () => {
    if (code) {
      navigator.clipboard.writeText(code)
    }
  }

  return (
    <div className="container mx-auto py-8 flex items-center justify-center min-h-[60vh]">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>
            {status === 'processing' && '処理中...'}
            {status === 'success' && '認可成功'}
            {status === 'error' && '認可失敗'}
          </CardTitle>
          <CardDescription>{message}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {status === 'success' && code && (
            <>
              <div>
                <p className="text-sm text-muted-foreground mb-2">認可コード:</p>
                <pre className="bg-muted p-3 rounded text-xs break-all">
                  {code}
                </pre>
              </div>
              <div className="flex gap-2">
                <Button onClick={copyCode} className="flex-1">
                  コードをコピー
                </Button>
                <Button
                  variant="outline"
                  onClick={() => window.close()}
                  className="flex-1"
                >
                  閉じる
                </Button>
              </div>
              <p className="text-sm text-muted-foreground">
                このコードを MCP Client Mock ページに貼り付けてトークン交換を行ってください。
              </p>
            </>
          )}
          {status === 'error' && (
            <Button variant="outline" onClick={() => window.close()}>
              閉じる
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
      <div className="container mx-auto py-8 flex items-center justify-center min-h-[60vh]">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    }>
      <CallbackContent />
    </Suspense>
  )
}
