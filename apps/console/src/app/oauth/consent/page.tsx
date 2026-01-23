"use client"

import { Suspense } from "react"
import { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { createClient } from "@/lib/supabase/client"
import { CheckCircle2, Shield, AlertCircle, Loader2, RotateCcw } from "lucide-react"

interface AuthorizationDetails {
  id: string
  client_id: string
  redirect_uri: string
  scope: string
  scopes?: string[]
  state: string | null
}

function ConsentContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const [authDetails, setAuthDetails] = useState<AuthorizationDetails | null>(null)
  const [user, setUser] = useState<{ id: string; email: string | null } | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isAlreadyAuthorized, setIsAlreadyAuthorized] = useState(false)
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    const init = async () => {
      // Get authorization_id from query params (Supabase OAuth Server compatible)
      const authorizationId = searchParams.get("authorization_id")

      if (!authorizationId) {
        setError("認可リクエストが見つかりません")
        setLoading(false)
        return
      }

      // Check user session first
      const supabase = createClient()
      const { data: { user } } = await supabase.auth.getUser()

      if (!user) {
        // Redirect to login with return URL
        const currentUrl = window.location.href
        router.push(`/login?returnTo=${encodeURIComponent(currentUrl)}`)
        return
      }

      setUser({ id: user.id, email: user.email ?? null })

      // Check if user is admin from mcpist.users table
      const { data: userData } = await supabase
        .from('users')
        .select('role')
        .eq('id', user.id)
        .single()
      setIsAdmin(userData?.role === 'admin')

      // Fetch authorization details from Supabase OAuth Server
      try {
        const { data, error: oauthError } = await supabase.auth.oauth.getAuthorizationDetails(authorizationId)

        // Debug: log the response structure
        console.log("[OAuth Consent] Authorization details:", JSON.stringify(data, null, 2))

        if (oauthError || !data) {
          console.error("Supabase OAuth error:", oauthError)
          // ユーザーフレンドリーなエラーメッセージ
          if (oauthError?.message?.includes("cannot be processed") ||
              oauthError?.message?.includes("not found") ||
              oauthError?.message?.includes("expired")) {
            setError("この認可リクエストは無効または期限切れです。接続元のアプリに戻って再度お試しください。")
          } else {
            setError(oauthError?.message || "認可リクエストの取得に失敗しました")
          }
          setLoading(false)
          return
        }

        // Check if already authorized (redirect_url present but no client info requiring consent)
        const alreadyAuthorized = !!(data.redirect_url && !data.client)
        setIsAlreadyAuthorized(alreadyAuthorized)

        // Parse scope string into array
        const scopeArray = data.scope ? data.scope.split(' ') : []

        setAuthDetails({
          id: authorizationId,
          client_id: data.client?.name || 'MCPist',
          redirect_uri: data.redirect_url || '',
          scope: data.scope || '',
          scopes: scopeArray,
          state: null,
        })
      } catch (err) {
        console.error("Failed to fetch authorization details:", err)
        setError("認可サーバーとの通信に失敗しました")
        setLoading(false)
        return
      }

      setLoading(false)
    }

    init()
  }, [searchParams, router])

  const handleApprove = async () => {
    if (!authDetails || !user) return

    setSubmitting(true)
    setError(null)

    try {
      // If we already have a redirect_url from getAuthorizationDetails, use it directly
      // This happens when authorization was auto-approved by Supabase
      if (authDetails.redirect_uri) {
        console.log("[OAuth Consent] Using existing redirect_url:", authDetails.redirect_uri)
        window.location.href = authDetails.redirect_uri
        return
      }

      const authorizationId = searchParams.get("authorization_id")
      const supabase = createClient()

      const { data, error: oauthError } = await supabase.auth.oauth.approveAuthorization(authorizationId!)

      if (oauthError) {
        // 既に処理済みの場合、redirect_uriがあればそれを使用
        if (oauthError.message?.includes("cannot be processed") && authDetails.redirect_uri) {
          console.log("[OAuth Consent] Authorization already processed, using stored redirect_uri")
          window.location.href = authDetails.redirect_uri
          return
        }
        throw new Error(oauthError.message || "認可の承認に失敗しました")
      }

      // Supabase handles the redirect automatically via data.redirect_url
      if (data?.redirect_url) {
        window.location.href = data.redirect_url
      } else if (authDetails.redirect_uri) {
        // Fallback to stored redirect_uri
        window.location.href = authDetails.redirect_uri
      } else {
        throw new Error("リダイレクト先が取得できませんでした")
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "エラーが発生しました")
      setSubmitting(false)
    }
  }

  const handleDeny = async () => {
    if (!authDetails) return

    const authorizationId = searchParams.get("authorization_id")

    try {
      const supabase = createClient()
      const { data, error: oauthError } = await supabase.auth.oauth.denyAuthorization(authorizationId!)

      if (!oauthError && data?.redirect_url) {
        window.location.href = data.redirect_url
        return
      }
    } catch (err) {
      console.error("Failed to deny authorization:", err)
    }

    // Fallback: redirect with error using local data
    const redirectUrl = new URL(authDetails.redirect_uri)
    redirectUrl.searchParams.set("error", "access_denied")
    redirectUrl.searchParams.set("error_description", "User denied the request")
    if (authDetails.state) {
      redirectUrl.searchParams.set("state", authDetails.state)
    }

    window.location.href = redirectUrl.toString()
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error && !authDetails) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <AlertCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
            <CardTitle>エラー</CardTitle>
            <CardDescription>{error}</CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  const scopes = authDetails?.scopes || authDetails?.scope.split(" ") || []

  // Already authorized - show confirmation screen
  if (isAlreadyAuthorized && authDetails) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center space-y-4">
            <div className="flex justify-center">
              <div className="w-16 h-16 rounded-xl bg-green-500 flex items-center justify-center">
                <CheckCircle2 className="h-8 w-8 text-white" />
              </div>
            </div>
            <div>
              <CardTitle className="text-xl">認可済み</CardTitle>
              <CardDescription className="mt-2">
                このアプリケーションは既に認可されています。
                続行してアプリに戻ります。
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="space-y-4">
            <div className="bg-muted/50 rounded-lg p-3 text-sm">
              <div className="text-muted-foreground">ログイン中のアカウント</div>
              <div className="font-medium">{user?.email}</div>
            </div>

            {error && (
              <div className="bg-destructive/10 text-destructive text-sm p-3 rounded-lg">
                {error}
              </div>
            )}
          </CardContent>

          <CardFooter className="flex flex-col gap-3">
            <Button
              className="w-full"
              onClick={handleApprove}
              disabled={submitting}
            >
              {submitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  リダイレクト中...
                </>
              ) : (
                "続行"
              )}
            </Button>

            {/* Admin only: Force re-authorization */}
            {isAdmin && (
              <Button
                variant="outline"
                className="w-full text-muted-foreground"
                onClick={() => {
                  // Clear the authorized state and show consent screen
                  setIsAlreadyAuthorized(false)
                }}
                disabled={submitting}
              >
                <RotateCcw className="h-4 w-4 mr-2" />
                セッションを破棄して再認可（管理者）
              </Button>
            )}
          </CardFooter>
        </Card>
      </div>
    )
  }

  // First time authorization - show consent screen
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center space-y-4">
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-xl bg-primary flex items-center justify-center">
              <Shield className="h-8 w-8 text-primary-foreground" />
            </div>
          </div>
          <div>
            <CardTitle className="text-xl">アクセス許可の確認</CardTitle>
            <CardDescription className="mt-2">
              <span className="font-medium text-foreground">{authDetails?.client_id?.toLowerCase().includes("mcpist") ? "MCPist Console" : authDetails?.client_id}</span>
              {" "}があなたのアカウントへのアクセスを要求しています
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* User info */}
          <div className="bg-muted/50 rounded-lg p-3 text-sm">
            <div className="text-muted-foreground">ログイン中のアカウント</div>
            <div className="font-medium">{user?.email}</div>
          </div>

          {/* Requested permissions */}
          <div className="space-y-2">
            <div className="text-sm font-medium">要求されている権限:</div>
            <ul className="space-y-2">
              {scopes.includes("openid") && (
                <li className="flex items-center gap-2 text-sm">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>ユーザー識別子へのアクセス</span>
                </li>
              )}
              {scopes.includes("profile") && (
                <li className="flex items-center gap-2 text-sm">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>プロフィール情報の読み取り</span>
                </li>
              )}
              {scopes.includes("email") && (
                <li className="flex items-center gap-2 text-sm">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>メールアドレスの読み取り</span>
                </li>
              )}
            </ul>
          </div>

          {error && (
            <div className="bg-destructive/10 text-destructive text-sm p-3 rounded-lg">
              {error}
            </div>
          )}
        </CardContent>

        <CardFooter className="flex gap-3">
          <Button
            variant="outline"
            className="flex-1"
            onClick={handleDeny}
            disabled={submitting}
          >
            拒否
          </Button>
          <Button
            className="flex-1"
            onClick={handleApprove}
            disabled={submitting}
          >
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                処理中...
              </>
            ) : (
              "許可する"
            )}
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}

export default function ConsentPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    }>
      <ConsentContent />
    </Suspense>
  )
}
