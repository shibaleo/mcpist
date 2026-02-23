"use client"

import { useState, useEffect, useMemo } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Link from "next/link"
import { Suspense } from "react"
import {
  ArrowRight,
  Zap,
  Shield,
  Link2,
  Settings,
  Terminal,
  Github,
  Globe,
} from "lucide-react"
import { ModuleIcon } from "@/components/module-icon"
import { ArchitectureDiagram } from "@/components/architecture-diagram"
import { getModules } from "@/lib/modules/module-data"

interface LandingContent {
  login: string
  heroTitle1: string
  heroTitle2: string
  heroSub: string
  getStarted: string
  viewOnGithub: string
  feat1Label: string
  feat1Title1: string
  feat1Title2: string
  feat1Desc: string
  feat1Card: string
  feat2Label: string
  feat2Title1: string
  feat2Title2: string
  feat2Desc: string
  feat2Card: string
  feat2Enc: string
  feat2EncSub: string
  feat2OAuth: string
  feat2OAuthSub: string
  feat2Perm: string
  feat2PermSub: string
  feat3Label: string
  feat3Title1: string
  feat3Title2: string
  feat3Desc: string
  feat3Card: string
  ecoLabel: string
  ecoTitle: string
  connect: string
  ctaTitle: string
  ctaDesc: string
  ctaButton: string
}

const ja: LandingContent = {
  login: "ログイン",
  heroTitle1: "MCP Servers,",
  heroTitle2: "Connected Effortlessly.",
  heroSub: "Notion、GitHub、Jira、Google Calendar など —\nすべてのサービスを1つの MCP エンドポイントで接続。",
  getStarted: "無料で始める",
  viewOnGithub: "GitHub で見る",
  feat1Label: "特徴",
  feat1Title1: "ひとつのエンドポイント、",
  feat1Title2: "すべてのサービス。",
  feat1Desc: "Notion、GitHub、Jira、Google Calendar など15以上のサービスを1つの MCP サーバー URL で接続。複数の設定を管理する必要はありません。",
  feat1Card: "統合エンドポイント",
  feat2Label: "特徴",
  feat2Title1: "トークンを",
  feat2Title2: "安全に管理。",
  feat2Desc: "すべての認証情報は暗号化して安全に保管。OAuth でワンクリック接続、または API トークンを手動で追加。きめ細かなツール権限で AI アシスタントがアクセスできる範囲を制御できます。",
  feat2Card: "セキュアトークン保管庫",
  feat2Enc: "暗号化ストレージ",
  feat2EncSub: "AES-256 暗号化で保管",
  feat2OAuth: "OAuth 2.0 対応",
  feat2OAuthSub: "トークンの手動管理不要",
  feat2Perm: "きめ細かな権限",
  feat2PermSub: "必要なツールのみ有効化",
  feat3Label: "特徴",
  feat3Title1: "あらゆる MCP 対応",
  feat3Title2: "クライアントで動作。",
  feat3Desc: "Claude Desktop、Cursor、Windsurf、VS Code など、あらゆる MCP 対応 AI アシスタントで利用可能。エンドポイント URL を貼り付けるだけですぐに使い始められます。",
  feat3Card: "対応クライアント",
  ecoLabel: "エコシステム",
  ecoTitle: "あなたのスタックと連携",
  connect: "接続",
  ctaTitle: "接続する準備はできましたか？",
  ctaDesc: "MCP ゲートウェイを数分でセットアップ。サービスを接続し、エンドポイント URL を取得して、すべてのツールで AI を活用しましょう。",
  ctaButton: "無料で始める",
}

const en: LandingContent = {
  login: "Login",
  heroTitle1: "MCP Servers,",
  heroTitle2: "Connected Effortlessly.",
  heroSub: "Notion, GitHub, Jira, Google Calendar and more —\nconnect all your services through a single MCP endpoint.",
  getStarted: "Get Started",
  viewOnGithub: "View on GitHub",
  feat1Label: "Key Features",
  feat1Title1: "One endpoint,",
  feat1Title2: "all your services.",
  feat1Desc: "Connect Notion, GitHub, Jira, Google Calendar and 15+ services through a single MCP server URL. No more juggling multiple configurations — one endpoint handles everything.",
  feat1Card: "Unified Endpoint",
  feat2Label: "Key Features",
  feat2Title1: "Your tokens,",
  feat2Title2: "securely managed.",
  feat2Desc: "All credentials are encrypted and stored securely. Connect via OAuth with a single click, or add API tokens manually. Fine-grained tool permissions let you control exactly what your AI assistant can access.",
  feat2Card: "Secure Token Vault",
  feat2Enc: "Encrypted Storage",
  feat2EncSub: "AES-256 encryption at rest",
  feat2OAuth: "OAuth 2.0 Support",
  feat2OAuthSub: "No manual token management",
  feat2Perm: "Granular Permissions",
  feat2PermSub: "Enable only the tools you need",
  feat3Label: "Key Features",
  feat3Title1: "Works with any",
  feat3Title2: "MCP-compatible client.",
  feat3Desc: "Claude Desktop, Cursor, Windsurf, VS Code, and any other MCP-compatible AI assistant. Just paste your endpoint URL and start using all connected services instantly.",
  feat3Card: "Compatible Clients",
  ecoLabel: "Ecosystem",
  ecoTitle: "Works with your stack",
  connect: "Connect",
  ctaTitle: "Ready to connect?",
  ctaDesc: "Set up your MCP gateway in minutes. Connect your services, grab your endpoint URL, and start using AI across all your tools.",
  ctaButton: "Get Started for Free",
}

function resolveClientLang(langParam: string | null): "ja" | "en" {
  if (langParam === "ja" || langParam === "en") return langParam
  if (typeof navigator !== "undefined") {
    const browserLang = navigator.language.toLowerCase()
    if (browserLang.startsWith("ja")) return "ja"
  }
  return "en"
}

function LandingPageContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [services, setServices] = useState<{ id: string; name: string }[]>([])

  const lang = resolveClientLang(searchParams.get("lang"))
  const t = useMemo(() => (lang === "ja" ? ja : en), [lang])

  useEffect(() => {
    getModules().then((mods) =>
      setServices(mods.map((m) => ({ id: m.id, name: m.name })))
    )
  }, [])

  useEffect(() => {
    const code = searchParams.get("code")
    if (code) {
      const params = new URLSearchParams(searchParams.toString())
      router.replace(`/auth/callback?${params.toString()}`)
      return
    }
  }, [searchParams, router])

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col lp-dot-grid">
      {/* Header */}
      <header className="fixed top-0 left-0 right-0 z-50 border-b border-border bg-background/80 backdrop-blur-md">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Zap className="h-4 w-4 text-primary-foreground" />
            </div>
            <span className="font-bold text-xl tracking-tight">MCPist</span>
          </div>
          <div className="flex items-center gap-6">
            <Link
              href={lang === "en" ? "/?lang=ja" : "/?lang=en"}
              className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <Globe className="h-4 w-4" />
              <span className="hidden sm:inline">{lang === "en" ? "日本語" : "English"}</span>
            </Link>
            <a
              href="https://github.com/shibaleo/mcpist"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <Github className="h-4 w-4" />
              <span className="hidden sm:inline">GitHub</span>
            </a>
            <Link
              href="/login"
              className="text-sm font-medium px-4 py-2 rounded-lg bg-secondary hover:bg-accent transition-colors"
            >
              {t.login}
            </Link>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative pt-32 pb-24 md:pt-44 md:pb-36 overflow-hidden">
        {/* Gradient glow */}
        <div className="absolute inset-0 flex items-start justify-center pointer-events-none">
          <div className="w-[800px] h-[500px] rounded-full blur-[150px] -translate-y-1/3 bg-[color:var(--accent-value)] opacity-25" />
        </div>
        <div className="absolute inset-0 flex items-start justify-center pointer-events-none">
          <div className="w-[400px] h-[300px] rounded-full blur-[80px] translate-y-8 bg-[color:var(--accent-value)] opacity-20" />
        </div>
        <div className="absolute inset-0 flex items-start justify-center pointer-events-none">
          <div className="w-[200px] h-[150px] rounded-full blur-[50px] translate-y-20 bg-white opacity-[0.07]" />
        </div>

        <div className="relative max-w-4xl mx-auto px-6 text-center">
          <h1 className="text-4xl sm:text-5xl md:text-7xl font-bold tracking-tight leading-[1.1] mb-6">
            <span className="text-transparent bg-clip-text bg-gradient-to-b from-foreground to-muted-foreground">
              {t.heroTitle1}
            </span>
            <br />
            <span className="text-primary">
              {t.heroTitle2}
            </span>
          </h1>
          <p className="text-base sm:text-lg md:text-xl text-muted-foreground max-w-2xl mx-auto mb-10 leading-relaxed whitespace-pre-line">
            {t.heroSub}
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/login"
              className="group flex items-center gap-2 px-7 py-3.5 rounded-xl bg-primary text-primary-foreground font-semibold text-sm hover:opacity-90 transition-all shadow-lg shadow-[color:var(--accent-value)]/25"
            >
              {t.getStarted}
              <ArrowRight className="h-4 w-4 group-hover:translate-x-0.5 transition-transform" />
            </Link>
            <a
              href="https://github.com/shibaleo/mcpist"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 px-7 py-3.5 rounded-xl border border-border text-muted-foreground font-medium text-sm hover:bg-secondary hover:text-foreground transition-all"
            >
              <Github className="h-4 w-4" />
              {t.viewOnGithub}
            </a>
          </div>
        </div>

        {/* Architecture diagram */}
        <div className="relative mt-16 md:mt-20 px-6">
          <ArchitectureDiagram />
        </div>
      </section>

      {/* Features Section */}
      <section className="relative py-24 md:py-32">

        <div className="max-w-6xl mx-auto px-6">
          {/* Feature 1: text left, visual right */}
          <div className="relative grid md:grid-cols-2 gap-12 md:gap-20 items-center mb-32">
            {/* Radial glow - centered near card left-top corner, nudged toward card center */}
            <div className="hidden md:block absolute left-[calc(50%+2.5rem)] top-0 -translate-x-1/2 -translate-y-1/2 pointer-events-none">
              <div className="w-[400px] h-[400px] translate-x-[1.5rem] translate-y-[1.5rem]" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.40 }} />
              <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[200px] h-[200px]" style={{ background: 'radial-gradient(circle at center, white 5%, transparent 70%)', opacity: 0.07 }} />
            </div>
            <div>
              <span className="text-xs font-semibold tracking-[0.2em] uppercase text-primary mb-4 block">
                {t.feat1Label}
              </span>
              <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold leading-tight mb-6">
                {t.feat1Title1}
                <br />
                {t.feat1Title2}
              </h2>
              <div className="w-16 h-px bg-gradient-to-r from-primary to-transparent mb-6" />
              <p className="text-muted-foreground leading-relaxed text-base md:text-lg">
                {t.feat1Desc}
              </p>
            </div>
            <div className="relative">
              {/* Mobile glow */}
              <div className="md:hidden absolute -top-16 -left-16 w-[250px] h-[250px] pointer-events-none" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.30 }} />
              <div className="relative rounded-xl border border-border/60 bg-card/50 backdrop-blur-xl p-8">
                <div className="flex items-center gap-3 mb-6">
                  <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                    <Link2 className="h-5 w-5 text-primary" />
                  </div>
                  <div className="text-sm font-semibold text-card-foreground">{t.feat1Card}</div>
                </div>
                <div className="space-y-3">
                  {["notion", "github", "jira", "google_calendar", "supabase"].map(
                    (id) => (
                      <div
                        key={id}
                        className="flex items-center gap-3 px-4 py-2.5 rounded-lg bg-secondary/30 border border-border/40"
                      >
                        <ModuleIcon moduleName={id} className="h-4 w-4" colored={false} />
                        <span className="text-sm text-foreground capitalize">
                          {id.replace(/_/g, " ")}
                        </span>
                        <div className="ml-auto w-2 h-2 rounded-full bg-success" />
                      </div>
                    )
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Feature 2: visual left, text right */}
          <div className="relative grid md:grid-cols-2 gap-12 md:gap-20 items-center mb-32">
            {/* Radial glow - centered near card right-top corner, nudged toward card center */}
            <div className="hidden md:block absolute right-[calc(50%+2.5rem)] top-0 translate-x-1/2 -translate-y-1/2 pointer-events-none">
              <div className="w-[400px] h-[400px] -translate-x-[1.5rem] translate-y-[1.5rem]" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.40 }} />
              <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[200px] h-[200px]" style={{ background: 'radial-gradient(circle at center, white 5%, transparent 70%)', opacity: 0.07 }} />
            </div>
            <div className="order-2 md:order-1 relative">
              {/* Mobile glow */}
              <div className="md:hidden absolute -top-16 -right-16 w-[250px] h-[250px] pointer-events-none" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.30 }} />
              <div className="relative rounded-xl border border-border/60 bg-card/50 backdrop-blur-xl p-8">
                <div className="flex items-center gap-3 mb-6">
                  <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                    <Shield className="h-5 w-5 text-primary" />
                  </div>
                  <div className="text-sm font-semibold text-card-foreground">{t.feat2Card}</div>
                </div>
                <div className="space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center">
                      <Shield className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-card-foreground">{t.feat2Enc}</div>
                      <div className="text-xs text-muted-foreground">{t.feat2EncSub}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center">
                      <Zap className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-card-foreground">{t.feat2OAuth}</div>
                      <div className="text-xs text-muted-foreground">{t.feat2OAuthSub}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center">
                      <Settings className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-card-foreground">{t.feat2Perm}</div>
                      <div className="text-xs text-muted-foreground">{t.feat2PermSub}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div className="order-1 md:order-2">
              <span className="text-xs font-semibold tracking-[0.2em] uppercase text-primary mb-4 block">
                {t.feat2Label}
              </span>
              <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold leading-tight mb-6">
                {t.feat2Title1}
                <br />
                {t.feat2Title2}
              </h2>
              <div className="w-16 h-px bg-gradient-to-r from-primary to-transparent mb-6" />
              <p className="text-muted-foreground leading-relaxed text-base md:text-lg">
                {t.feat2Desc}
              </p>
            </div>
          </div>

          {/* Feature 3: text left, visual right */}
          <div className="relative grid md:grid-cols-2 gap-12 md:gap-20 items-center">
            {/* Radial glow - centered near card left-top corner, nudged toward card center */}
            <div className="hidden md:block absolute left-[calc(50%+2.5rem)] top-0 -translate-x-1/2 -translate-y-1/2 pointer-events-none">
              <div className="w-[400px] h-[400px] translate-x-[1.5rem] translate-y-[1.5rem]" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.40 }} />
              <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[200px] h-[200px]" style={{ background: 'radial-gradient(circle at center, white 5%, transparent 70%)', opacity: 0.07 }} />
            </div>
            <div>
              <span className="text-xs font-semibold tracking-[0.2em] uppercase text-primary mb-4 block">
                {t.feat3Label}
              </span>
              <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold leading-tight mb-6">
                {t.feat3Title1}
                <br />
                {t.feat3Title2}
              </h2>
              <div className="w-16 h-px bg-gradient-to-r from-primary to-transparent mb-6" />
              <p className="text-muted-foreground leading-relaxed text-base md:text-lg">
                {t.feat3Desc}
              </p>
            </div>
            <div className="relative">
              {/* Mobile glow */}
              <div className="md:hidden absolute -top-16 -left-16 w-[250px] h-[250px] pointer-events-none" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.30 }} />
              <div className="relative rounded-xl border border-border/60 bg-card/50 backdrop-blur-xl p-8">
                <div className="flex items-center gap-3 mb-6">
                  <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                    <Terminal className="h-5 w-5 text-primary" />
                  </div>
                  <div className="text-sm font-semibold text-card-foreground">{t.feat3Card}</div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  {[
                    "Claude Desktop",
                    "Cursor",
                    "Windsurf",
                    "VS Code",
                    "Cline",
                    "& more...",
                  ].map((client) => (
                    <div
                      key={client}
                      className="px-4 py-3 rounded-lg bg-secondary/30 border border-border/40 text-sm text-foreground text-center"
                    >
                      {client}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Ecosystem Section */}
      <section className="py-24 md:py-32 ">
        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center mb-16">
            <span className="text-xs font-semibold tracking-[0.2em] uppercase text-primary mb-4 block">
              {t.ecoLabel}
            </span>
            <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold">
              {t.ecoTitle}
            </h2>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
            {services.map((service) => (
              <div
                key={service.id}
                className="group flex flex-col items-center gap-4 p-6 rounded-xl border border-border/60 bg-card/50 backdrop-blur-lg hover:border-primary/30 hover:bg-primary/[0.06] transition-all"
              >
                <span className="text-sm font-medium text-foreground">
                  {service.name}
                </span>
                <ModuleIcon
                  moduleName={service.id}
                  className="h-8 w-8"
                  colored
                />
                <span className="text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                  {t.connect} {">"}
                </span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-24 md:py-32  relative overflow-hidden">
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="w-[500px] h-[500px]" style={{ background: 'radial-gradient(circle at center, var(--accent-value) 20%, transparent 70%)', opacity: 0.18 }} />
        </div>
        <div className="relative max-w-3xl mx-auto px-6 text-center">
          <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold mb-6">
            {t.ctaTitle}
          </h2>
          <p className="text-muted-foreground text-base md:text-lg mb-10 max-w-xl mx-auto">
            {t.ctaDesc}
          </p>
          <Link
            href="/login"
            className="group inline-flex items-center gap-2 px-8 py-4 rounded-xl bg-primary text-primary-foreground font-semibold hover:opacity-90 transition-all shadow-lg shadow-[color:var(--accent-value)]/25"
          >
            {t.ctaButton}
            <ArrowRight className="h-4 w-4 group-hover:translate-x-0.5 transition-transform" />
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className=" py-12">
        <div className="max-w-6xl mx-auto px-6">
          <div className="flex flex-col md:flex-row items-center justify-between gap-6">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
                <Zap className="h-3.5 w-3.5 text-primary-foreground" />
              </div>
              <span className="font-bold text-lg">MCPist</span>
            </div>
            <div className="flex items-center gap-6 text-sm text-muted-foreground">
              <Link href="/terms" className="hover:text-foreground transition-colors">
                Terms
              </Link>
              <Link href="/privacy" className="hover:text-foreground transition-colors">
                Privacy
              </Link>
              <Link href="/security" className="hover:text-foreground transition-colors">
                Security
              </Link>
              <a
                href="https://github.com/shibaleo/mcpist"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-foreground transition-colors"
              >
                GitHub
              </a>
            </div>
          </div>
          <div className="mt-8 pt-6 border-t border-border text-center text-xs text-muted-foreground">
            &copy; 2026 MCPist. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  )
}

export default function LandingPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-background" />}>
      <LandingPageContent />
    </Suspense>
  )
}
