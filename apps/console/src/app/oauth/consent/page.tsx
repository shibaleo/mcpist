"use client"

import { Suspense } from "react"
import { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { useUser } from "@clerk/nextjs"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { CheckCircle2, Shield, AlertCircle, Loader2 } from "lucide-react"

function ConsentContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const { user: clerkUser, isLoaded } = useUser()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const scopes = searchParams.get("scope")?.split(" ") || ["openid", "profile", "email"]
  const redirectUri = searchParams.get("redirect_uri") || ""
  const clientId = searchParams.get("client_id") || "MCPist"
  const state = searchParams.get("state") || null

  const loading = !isLoaded

  useEffect(() => {
    if (!isLoaded) return

    if (!clerkUser) {
      const currentUrl = window.location.href
      router.push(`/login?returnTo=${encodeURIComponent(currentUrl)}`)
    }
  }, [clerkUser, isLoaded, router])

  const handleApprove = async () => {
    if (!redirectUri) {
      setError("リダイレクト先が指定されていません")
      return
    }
    setSubmitting(true)
    window.location.href = redirectUri
  }

  const handleDeny = () => {
    if (!redirectUri) return
    const url = new URL(redirectUri)
    url.searchParams.set("error", "access_denied")
    url.searchParams.set("error_description", "User denied the request")
    if (state) url.searchParams.set("state", state)
    window.location.href = url.toString()
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error && !redirectUri) {
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
              <span className="font-medium text-foreground">{clientId.toLowerCase().includes("mcpist") ? "MCPist Console" : clientId}</span>
              {" "}があなたのアカウントへのアクセスを要求しています
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          <div className="bg-muted/50 rounded-lg p-3 text-sm">
            <div className="text-muted-foreground">ログイン中のアカウント</div>
            <div className="font-medium">{clerkUser?.emailAddresses[0]?.emailAddress}</div>
          </div>

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
