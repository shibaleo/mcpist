'use client'

import { Suspense, useMemo, useEffect } from 'react'
import { useSearchParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'

function CallbackContent() {
  const searchParams = useSearchParams()

  const code = searchParams.get('code')
  const state = searchParams.get('state')
  const error = searchParams.get('error')
  const errorDescription = searchParams.get('error_description')

  const { status, message } = useMemo(() => {
    if (error) {
      return { status: 'error' as const, message: `${error}: ${errorDescription || '認可に失敗しました'}` }
    }
    if (!code) {
      return { status: 'error' as const, message: '認可コードがありません' }
    }
    if (typeof window !== 'undefined') {
      const storedState = sessionStorage.getItem('oauth_state')
      if (state !== storedState) {
        return { status: 'error' as const, message: 'state が一致しません（CSRF攻撃の可能性）' }
      }
    }
    return { status: 'success' as const, message: '認可コードを取得しました' }
  }, [code, state, error, errorDescription])

  // Clear stored state on success
  useEffect(() => {
    if (status === 'success') {
      sessionStorage.removeItem('oauth_state')
    }
  }, [status])

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
            {status === 'success' ? '認可成功' : '認可失敗'}
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
