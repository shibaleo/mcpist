"use client"

import { useState, useEffect } from "react"
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
} from "lucide-react"
import { ModuleIcon } from "@/components/module-icon"
import { ArchitectureDiagram } from "@/components/architecture-diagram"
import { getModules } from "@/lib/module-data"

function LandingPageContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [services, setServices] = useState<{ id: string; name: string }[]>([])

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
    <div className="dark min-h-screen bg-background text-foreground flex flex-col lp-dot-grid" data-accent-color="orange">
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
            <a
              href="https://github.com/shibaisdog/mcpist"
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
              ログイン
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
              MCP Servers,
            </span>
            <br />
            <span className="text-primary">
              Connected Effortlessly.
            </span>
          </h1>
          <p className="text-base sm:text-lg md:text-xl text-muted-foreground max-w-2xl mx-auto mb-10 leading-relaxed">
            Notion, GitHub, Jira, Google Calendar and more —
            <br className="hidden sm:block" />
            connect all your services through a single MCP endpoint.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/login"
              className="group flex items-center gap-2 px-7 py-3.5 rounded-xl bg-primary text-primary-foreground font-semibold text-sm hover:opacity-90 transition-all shadow-lg shadow-[color:var(--accent-value)]/25"
            >
              Get Started
              <ArrowRight className="h-4 w-4 group-hover:translate-x-0.5 transition-transform" />
            </Link>
            <a
              href="https://github.com/shibaisdog/mcpist"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 px-7 py-3.5 rounded-xl border border-border text-muted-foreground font-medium text-sm hover:bg-secondary hover:text-foreground transition-all"
            >
              <Github className="h-4 w-4" />
              View on GitHub
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
                Key Features
              </span>
              <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold leading-tight mb-6">
                One endpoint,
                <br />
                all your services.
              </h2>
              <div className="w-16 h-px bg-gradient-to-r from-primary to-transparent mb-6" />
              <p className="text-muted-foreground leading-relaxed text-base md:text-lg">
                Connect Notion, GitHub, Jira, Google Calendar and 15+ services
                through a single MCP server URL. No more juggling multiple
                configurations — one endpoint handles everything.
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
                  <div className="text-sm font-semibold text-card-foreground">Unified Endpoint</div>
                </div>
                <div className="space-y-3">
                  {["notion", "github", "jira", "google_calendar", "supabase"].map(
                    (id) => (
                      <div
                        key={id}
                        className="flex items-center gap-3 px-4 py-2.5 rounded-lg bg-secondary/30 border border-border/40"
                      >
                        <ModuleIcon moduleId={id} className="h-4 w-4" colored={false} />
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
                  <div className="text-sm font-semibold text-card-foreground">Secure Token Vault</div>
                </div>
                <div className="space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center">
                      <Shield className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-card-foreground">Encrypted Storage</div>
                      <div className="text-xs text-muted-foreground">AES-256 encryption at rest</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center">
                      <Zap className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-card-foreground">OAuth 2.0 Support</div>
                      <div className="text-xs text-muted-foreground">No manual token management</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center">
                      <Settings className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <div className="text-sm font-medium text-card-foreground">Granular Permissions</div>
                      <div className="text-xs text-muted-foreground">Enable only the tools you need</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div className="order-1 md:order-2">
              <span className="text-xs font-semibold tracking-[0.2em] uppercase text-primary mb-4 block">
                Key Features
              </span>
              <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold leading-tight mb-6">
                Your tokens,
                <br />
                securely managed.
              </h2>
              <div className="w-16 h-px bg-gradient-to-r from-primary to-transparent mb-6" />
              <p className="text-muted-foreground leading-relaxed text-base md:text-lg">
                All credentials are encrypted and stored securely. Connect
                via OAuth with a single click, or add API tokens manually.
                Fine-grained tool permissions let you control exactly what your
                AI assistant can access.
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
                Key Features
              </span>
              <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold leading-tight mb-6">
                Works with any
                <br />
                MCP-compatible client.
              </h2>
              <div className="w-16 h-px bg-gradient-to-r from-primary to-transparent mb-6" />
              <p className="text-muted-foreground leading-relaxed text-base md:text-lg">
                Claude Desktop, Cursor, Windsurf, VS Code, and any other
                MCP-compatible AI assistant. Just paste your endpoint URL
                and start using all connected services instantly.
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
                  <div className="text-sm font-semibold text-card-foreground">Compatible Clients</div>
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
              Ecosystem
            </span>
            <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold">
              Works with your stack
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
                  moduleId={service.id}
                  className="h-8 w-8"
                  colored
                />
                <span className="text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                  Connect {">"}
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
            Ready to connect?
          </h2>
          <p className="text-muted-foreground text-base md:text-lg mb-10 max-w-xl mx-auto">
            Set up your MCP gateway in minutes. Connect your services, grab your
            endpoint URL, and start using AI across all your tools.
          </p>
          <Link
            href="/login"
            className="group inline-flex items-center gap-2 px-8 py-4 rounded-xl bg-primary text-primary-foreground font-semibold hover:opacity-90 transition-all shadow-lg shadow-[color:var(--accent-value)]/25"
          >
            Get Started for Free
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
              <a
                href="https://github.com/shibaisdog/mcpist"
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
