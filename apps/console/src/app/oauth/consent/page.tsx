"use client"

import { Suspense } from "react"
import { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { createClient } from "@/lib/supabase/client"
import { CheckCircle2, Shield, AlertCircle, Loader2 } from "lucide-react"

// Check if we're in production (use Supabase OAuth) or development (use OAuth Mock Server)
const isProduction = process.env.ENVIRONMENT === 'production'

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

      // Fetch authorization details
      try {
        if (isProduction) {
          // Production: Use Supabase SDK
          // @ts-expect-error - Supabase OAuth API is in beta
          const { data, error: oauthError } = await supabase.auth.oauth.getAuthorizationDetails(authorizationId)

          if (oauthError || !data) {
            console.error("Supabase OAuth error:", oauthError)
            setError(oauthError?.message || "認可リクエストの取得に失敗しました")
            setLoading(false)
            return
          }

          setAuthDetails({
            id: authorizationId,
            client_id: data.client?.name || data.client_id || 'Unknown App',
            redirect_uri: data.redirect_uri || '',
            scope: data.scopes?.join(' ') || '',
            scopes: data.scopes,
            state: data.state || null,
          })
        } else {
          // Development: Use OAuth Mock Server REST API
          const oauthServerUrl = process.env.NEXT_PUBLIC_OAUTH_SERVER_URL || 'http://oauth.localhost'
          const response = await fetch(`${oauthServerUrl}/authorization/${authorizationId}`)

          if (!response.ok) {
            const errorData = await response.json()
            setError(errorData.error_description || "認可リクエストの取得に失敗しました")
            setLoading(false)
            return
          }

          const details = await response.json()
          setAuthDetails({
            id: details.id,
            client_id: details.client_id,
            redirect_uri: details.redirect_uri,
            scope: details.scope,
            state: details.state,
          })
        }
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
      const authorizationId = searchParams.get("authorization_id")

      if (isProduction) {
        // Production: Use Supabase SDK
        const supabase = createClient()
        // @ts-expect-error - Supabase OAuth API is in beta
        const { data, error: oauthError } = await supabase.auth.oauth.approveAuthorization(authorizationId)

        if (oauthError) {
          throw new Error(oauthError.message || "認可の承認に失敗しました")
        }

        // Supabase handles the redirect automatically via data.redirect_to
        if (data?.redirect_to) {
          window.location.href = data.redirect_to
        } else {
          throw new Error("リダイレクト先が取得できませんでした")
        }
      } else {
        // Development: Use OAuth Mock Server REST API
        const oauthServerUrl = process.env.NEXT_PUBLIC_OAUTH_SERVER_URL || 'http://oauth.localhost'
        const response = await fetch(`${oauthServerUrl}/authorization/${authorizationId}/approve`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ user_id: user.id }),
        })

        if (!response.ok) {
          const data = await response.json()
          throw new Error(data.error_description || "認可の承認に失敗しました")
        }

        const { code, redirect_uri, state } = await response.json()

        // Redirect back to client with authorization code
        const redirectUrl = new URL(redirect_uri)
        redirectUrl.searchParams.set("code", code)
        if (state) {
          redirectUrl.searchParams.set("state", state)
        }

        window.location.href = redirectUrl.toString()
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
      if (isProduction) {
        // Production: Use Supabase SDK
        const supabase = createClient()
        // @ts-expect-error - Supabase OAuth API is in beta
        const { data, error: oauthError } = await supabase.auth.oauth.rejectAuthorization(authorizationId)

        if (!oauthError && data?.redirect_to) {
          window.location.href = data.redirect_to
          return
        }
      } else {
        // Development: Use OAuth Mock Server REST API
        const oauthServerUrl = process.env.NEXT_PUBLIC_OAUTH_SERVER_URL || 'http://oauth.localhost'
        const response = await fetch(`${oauthServerUrl}/authorization/${authorizationId}/deny`, {
          method: "POST",
        })

        if (response.ok) {
          const { redirect_uri, state } = await response.json()
          const redirectUrl = new URL(redirect_uri)
          redirectUrl.searchParams.set("error", "access_denied")
          redirectUrl.searchParams.set("error_description", "User denied the request")
          if (state) {
            redirectUrl.searchParams.set("state", state)
          }
          window.location.href = redirectUrl.toString()
          return
        }
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
