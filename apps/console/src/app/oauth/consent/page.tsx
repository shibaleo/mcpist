"use client"

import { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { createClient } from "@/lib/supabase/client"
import { CheckCircle2, Shield, AlertCircle, Loader2 } from "lucide-react"

interface AuthRequest {
  client_id: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: string
  scope: string
  state: string
}

export default function ConsentPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const [authRequest, setAuthRequest] = useState<AuthRequest | null>(null)
  const [user, setUser] = useState<{ id: string; email: string | null } | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const init = async () => {
      // Parse auth request from query params
      const requestParam = searchParams.get("request")
      if (!requestParam) {
        setError("認可リクエストが見つかりません")
        setLoading(false)
        return
      }

      try {
        const decoded = JSON.parse(atob(requestParam))
        setAuthRequest(decoded)
      } catch {
        setError("認可リクエストの解析に失敗しました")
        setLoading(false)
        return
      }

      // Check user session
      const supabase = createClient()
      const { data: { user } } = await supabase.auth.getUser()

      if (!user) {
        // Redirect to login
        const currentUrl = window.location.href
        router.push(`/login?returnTo=${encodeURIComponent(currentUrl)}`)
        return
      }

      setUser({ id: user.id, email: user.email ?? null })
      setLoading(false)
    }

    init()
  }, [searchParams, router])

  const handleApprove = async () => {
    if (!authRequest || !user) return

    setSubmitting(true)
    setError(null)

    try {
      // Call API to generate authorization code
      const response = await fetch("/api/auth/consent", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...authRequest,
          user_id: user.id,
        }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || "Failed to generate authorization code")
      }

      const { code } = await response.json()

      // Redirect back to client with authorization code
      const redirectUrl = new URL(authRequest.redirect_uri)
      redirectUrl.searchParams.set("code", code)
      redirectUrl.searchParams.set("state", authRequest.state)

      window.location.href = redirectUrl.toString()
    } catch (err) {
      setError(err instanceof Error ? err.message : "エラーが発生しました")
      setSubmitting(false)
    }
  }

  const handleDeny = () => {
    if (!authRequest) return

    // Redirect back with error
    const redirectUrl = new URL(authRequest.redirect_uri)
    redirectUrl.searchParams.set("error", "access_denied")
    redirectUrl.searchParams.set("error_description", "User denied the request")
    redirectUrl.searchParams.set("state", authRequest.state)

    window.location.href = redirectUrl.toString()
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error && !authRequest) {
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

  const scopes = authRequest?.scope.split(" ") || []

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
              <span className="font-medium text-foreground">{authRequest?.client_id === "mcpist-console" ? "MCPist" : authRequest?.client_id}</span>
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
