import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Zap, ArrowLeft, Globe } from "lucide-react"
import { resolveLang } from "@/lib/i18n"

interface Section {
  heading: string
  body?: string
  items?: string[]
}

interface Content {
  title: string
  backLink: string
  lastUpdated: string
  login: string
  sections: Section[]
}

const ja: Content = {
  title: "セキュリティポリシー",
  backLink: "トップへ戻る",
  lastUpdated: "最終更新日: 2026年2月22日",
  login: "ログイン",
  sections: [
    {
      heading: "1. データの暗号化",
      body: "MCPist では、ユーザーの機密情報を以下の方法で保護しています：",
      items: [
        "外部サービスのアクセストークン・OAuthクレデンシャルは AES-256-GCM で暗号化して保存",
        "すべての通信は TLS 1.2 以上で暗号化",
        "データベース接続は SSL/TLS で保護",
      ],
    },
    {
      heading: "2. 認証・認可",
      body: "多層的な認証・認可メカニズムを採用しています：",
      items: [
        "ユーザー認証は Clerk によるセキュアなセッション管理",
        "API キーは Ed25519 署名付き JWT で発行（mpt_ プレフィックス）",
        "Worker ↔ Server 間は短寿命（30秒）の Gateway JWT で認証",
        "モジュール・ツールごとのアクセス制御と日次利用上限",
      ],
    },
    {
      heading: "3. インフラストラクチャ",
      body: "本サービスは以下のインフラ上で運用されています：",
      items: [
        "API Gateway: Cloudflare Workers（DDoS 保護、WAF 標準装備）",
        "バックエンド: Render.com のマネージドサービス",
        "データベース: Supabase PostgreSQL（RLS、自動バックアップ）",
        "DNS/CDN: Cloudflare（DNSSEC 対応）",
      ],
    },
    {
      heading: "4. 脆弱性の報告",
      body: "セキュリティに関する脆弱性を発見された場合は、以下の方法でご報告ください。責任ある開示にご協力いただいた方には適切な対応を行います。",
      items: [
        "GitHub リポジトリの Security Advisory 機能をご利用ください",
        "公開前に修正する機会を確保するため、脆弱性の詳細は非公開でお送りください",
      ],
    },
    {
      heading: "5. ログと監視",
      body: "サービスの安全性を維持するため、以下の監視を実施しています：",
      items: [
        "API リクエストのアクセスログ記録",
        "異常なアクセスパターンの検出",
        "定期的なヘルスチェックによるサービス稼働監視",
      ],
    },
    {
      heading: "6. お問い合わせ",
      body: "セキュリティに関するお問い合わせは、GitHub リポジトリの Issue または Security Advisory よりご連絡ください。",
    },
  ],
}

const en: Content = {
  title: "Security Policy",
  backLink: "Back to Home",
  lastUpdated: "Last updated: February 22, 2026",
  login: "Login",
  sections: [
    {
      heading: "1. Data Encryption",
      body: "MCPist protects your sensitive information through the following measures:",
      items: [
        "External service access tokens and OAuth credentials are encrypted with AES-256-GCM at rest",
        "All communications are encrypted via TLS 1.2 or higher",
        "Database connections are secured with SSL/TLS",
      ],
    },
    {
      heading: "2. Authentication & Authorization",
      body: "We employ multi-layered authentication and authorization mechanisms:",
      items: [
        "User authentication via Clerk with secure session management",
        "API keys are issued as Ed25519-signed JWTs (mpt_ prefix)",
        "Worker-to-Server communication uses short-lived (30s) Gateway JWTs",
        "Per-module and per-tool access control with daily usage limits",
      ],
    },
    {
      heading: "3. Infrastructure",
      body: "The service operates on the following infrastructure:",
      items: [
        "API Gateway: Cloudflare Workers (built-in DDoS protection and WAF)",
        "Backend: Render.com managed services",
        "Database: Supabase PostgreSQL (RLS, automatic backups)",
        "DNS/CDN: Cloudflare (DNSSEC enabled)",
      ],
    },
    {
      heading: "4. Vulnerability Reporting",
      body: "If you discover a security vulnerability, please report it through the following channels. We appreciate responsible disclosure and will respond appropriately.",
      items: [
        "Please use the Security Advisory feature on our GitHub repository",
        "To allow us time to address the issue before public disclosure, please submit vulnerability details privately",
      ],
    },
    {
      heading: "5. Logging & Monitoring",
      body: "We maintain the following monitoring practices to ensure service security:",
      items: [
        "API request access logging",
        "Anomalous access pattern detection",
        "Periodic health checks for service availability monitoring",
      ],
    },
    {
      heading: "6. Contact",
      body: "For security-related inquiries, please contact us via GitHub Issues or the Security Advisory feature on our repository.",
    },
  ],
}

export default async function SecurityPage({
  searchParams,
}: {
  searchParams: Promise<{ lang?: string }>
}) {
  const { lang } = await searchParams
  const resolved = await resolveLang(lang)
  const isEn = resolved === "en"
  const t = isEn ? en : ja

  return (
    <div className="min-h-screen bg-background flex flex-col">
      {/* Header */}
      <header className="border-b border-border">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Zap className="h-4 w-4 text-primary-foreground" />
            </div>
            <span className="font-bold text-xl text-foreground">MCPist</span>
          </Link>
          <div className="flex items-center gap-3">
            <Link
              href={isEn ? "/security?lang=ja" : "/security?lang=en"}
              className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <Globe className="h-4 w-4" />
              {isEn ? "日本語" : "English"}
            </Link>
            <Link href="/login">
              <Button variant="outline">{t.login}</Button>
            </Link>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="flex-1 py-12">
        <div className="container mx-auto px-4 max-w-3xl">
          <Link
            href="/"
            className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-6"
          >
            <ArrowLeft className="h-4 w-4" />
            {t.backLink}
          </Link>

          <h1 className="text-3xl font-bold text-foreground mb-8">{t.title}</h1>

          <div className="prose prose-neutral dark:prose-invert max-w-none space-y-6">
            <p className="text-muted-foreground">{t.lastUpdated}</p>

            {t.sections.map((section) => (
              <section key={section.heading} className="space-y-4">
                <h2 className="text-xl font-semibold text-foreground">{section.heading}</h2>
                {section.body && <p className="text-muted-foreground">{section.body}</p>}
                {section.items && (
                  <ul className="list-disc list-inside text-muted-foreground space-y-2">
                    {section.items.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                )}
              </section>
            ))}
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-border py-6">
        <div className="container mx-auto px-4 text-center text-sm text-muted-foreground">
          <div className="flex justify-center gap-4 mb-2">
            <Link href="/terms" className="hover:text-foreground">Terms</Link>
            <Link href="/privacy" className="hover:text-foreground">Privacy</Link>
            <Link href="/security" className="hover:text-foreground">Security</Link>
          </div>
          &copy; 2026 MCPist. All rights reserved.
        </div>
      </footer>
    </div>
  )
}
