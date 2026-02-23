"use client"

import { SignIn } from "@clerk/nextjs"

export default function LoginPage() {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4 lp-dot-grid">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center space-y-4">
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-xl bg-primary flex items-center justify-center">
              <span className="text-primary-foreground font-bold text-2xl">M</span>
            </div>
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">MCPist</h1>
            <p className="text-sm text-muted-foreground mt-1">MCP Server 管理プラットフォーム</p>
          </div>
        </div>
        <div className="flex justify-center">
          <SignIn
            routing="hash"
            afterSignInUrl="/dashboard"
            afterSignUpUrl="/onboarding"
          />
        </div>
      </div>
    </div>
  )
}
