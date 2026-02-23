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
  title: "利用規約",
  backLink: "トップへ戻る",
  lastUpdated: "最終更新日: 2026年2月22日",
  login: "ログイン",
  sections: [
    {
      heading: "第1条（適用）",
      body: "本規約は、MCPist（以下「本サービス」）の利用条件を定めるものです。ユーザーは本規約に同意した上で本サービスを利用するものとします。",
    },
    {
      heading: "第2条（サービス内容）",
      body: "本サービスは、MCPプロトコルを通じて各種外部サービス（Notion、GitHub、Jira、Google Calendarなど）への接続を提供するプラットフォームです。",
    },
    {
      heading: "第3条（アカウント）",
      body: "ユーザーは、正確な情報を提供してアカウントを作成する必要があります。アカウント情報の管理はユーザーの責任とし、第三者への譲渡・貸与は禁止します。",
    },
    {
      heading: "第4条（料金・利用制限）",
      body: "本サービスの利用料金はプランページに掲載されるものとし、ユーザーは適用される料金を支払う義務を負います。",
      items: [
        "有料プランは自動更新され、更新日に決済方法に対して課金されます",
        "解約は現在の利用期間の終了前に行う必要があります。解約後も期間満了まで利用可能です",
        "各プランには日次の利用上限が設定されており、上限を超えた場合はリクエストが制限されます",
        "支払い済みの料金は、法令で義務付けられている場合を除き、原則として返金されません",
      ],
    },
    {
      heading: "第5条（禁止事項）",
      body: "以下の行為を禁止します：",
      items: [
        "法令または公序良俗に反する行為",
        "本サービスの運営を妨害する行為",
        "他のユーザーへの迷惑行為",
        "不正アクセスまたはそれを試みる行為",
        "本サービスを利用した営利目的の転売行為",
      ],
    },
    {
      heading: "第6条（免責事項）",
      body: "本サービスは現状有姿で提供され、特定目的への適合性を保証するものではありません。外部サービスの障害・変更等による影響について、当社は責任を負いません。",
    },
    {
      heading: "第7条（規約の変更）",
      body: "当社は、必要に応じて本規約を変更できるものとします。変更後の規約は、本サービス上での公開をもって効力を生じます。",
    },
    {
      heading: "第8条（準拠法・管轄）",
      body: "本規約は日本法に準拠し、本サービスに関する紛争は東京地方裁判所を第一審の専属的合意管轄裁判所とします。",
    },
  ],
}

const en: Content = {
  title: "Terms of Service",
  backLink: "Back to Home",
  lastUpdated: "Last updated: February 22, 2026",
  login: "Login",
  sections: [
    {
      heading: "Article 1 (Application)",
      body: "These terms define the conditions for using MCPist (hereinafter \"the Service\"). Users shall use the Service upon agreeing to these terms.",
    },
    {
      heading: "Article 2 (Service Description)",
      body: "The Service is a platform that provides connectivity to various external services (Notion, GitHub, Jira, Google Calendar, etc.) through the MCP protocol.",
    },
    {
      heading: "Article 3 (Accounts)",
      body: "Users must provide accurate information when creating an account. Users are responsible for managing their account credentials. Transfer or lending of accounts to third parties is prohibited.",
    },
    {
      heading: "Article 4 (Fees & Usage Limits)",
      body: "Service fees are published on the Plans page. Users are responsible for paying all applicable fees.",
      items: [
        "Paid plans auto-renew and are charged to your payment method on each renewal date",
        "You must cancel before the end of your current billing period. Access continues until the period ends",
        "Each plan includes daily usage limits; requests exceeding the limit will be throttled",
        "All payments are non-refundable except where required by applicable law",
      ],
    },
    {
      heading: "Article 5 (Prohibited Activities)",
      body: "The following activities are prohibited:",
      items: [
        "Activities that violate laws or public order and morals",
        "Activities that interfere with the operation of the Service",
        "Activities that cause inconvenience to other users",
        "Unauthorized access or attempts thereof",
        "Commercial resale activities using the Service",
      ],
    },
    {
      heading: "Article 6 (Disclaimer)",
      body: "The Service is provided as-is without warranty of fitness for a particular purpose. We are not responsible for impacts caused by outages or changes in external services.",
    },
    {
      heading: "Article 7 (Changes to Terms)",
      body: "We may modify these terms as necessary. Modified terms shall become effective upon publication on the Service.",
    },
    {
      heading: "Article 8 (Governing Law & Jurisdiction)",
      body: "These terms are governed by the laws of Japan. The Tokyo District Court shall have exclusive jurisdiction of first instance for any disputes related to the Service.",
    },
  ],
}

export default async function TermsPage({
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
              href={isEn ? "/terms?lang=ja" : "/terms?lang=en"}
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
