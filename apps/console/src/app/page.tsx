import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowRight, Zap, Shield, Link2 } from "lucide-react";

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-background flex flex-col">
      {/* Header */}
      <header className="border-b border-border">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Zap className="h-4 w-4 text-primary-foreground" />
            </div>
            <span className="font-bold text-xl text-foreground">MCPist</span>
          </div>
          <Link href="/login">
            <Button variant="outline">ログイン</Button>
          </Link>
        </div>
      </header>

      {/* Hero Section */}
      <main className="flex-1 flex items-center justify-center">
        <div className="container mx-auto px-4 py-16 text-center">
          <h1 className="text-4xl md:text-6xl font-bold text-foreground mb-6">
            MCPサーバーを
            <br />
            <span className="text-primary">簡単に接続</span>
          </h1>
          <p className="text-lg md:text-xl text-muted-foreground mb-8 max-w-2xl mx-auto">
            Notion、GitHub、Jira、Google Calendar など、
            様々なサービスをMCPプロトコルで統合管理
          </p>
          <Link href="/login">
            <Button size="lg" className="gap-2">
              今すぐ始める
              <ArrowRight className="h-4 w-4" />
            </Button>
          </Link>
        </div>
      </main>

      {/* Features Section */}
      <section className="border-t border-border bg-secondary/30 py-16">
        <div className="container mx-auto px-4">
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center p-6">
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <Link2 className="h-6 w-6 text-primary" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">
                簡単接続
              </h3>
              <p className="text-muted-foreground">
                OAuthまたはAPIトークンで各サービスに簡単接続
              </p>
            </div>
            <div className="text-center p-6">
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <Shield className="h-6 w-6 text-primary" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">
                セキュア
              </h3>
              <p className="text-muted-foreground">
                トークンは暗号化して安全に保管
              </p>
            </div>
            <div className="text-center p-6">
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <Zap className="h-6 w-6 text-primary" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">
                統合MCP
              </h3>
              <p className="text-muted-foreground">
                複数サービスを1つのMCPエンドポイントで利用
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border py-6">
        <div className="container mx-auto px-4 text-center text-sm text-muted-foreground">
          &copy; 2026 MCPist. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
