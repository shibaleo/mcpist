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
  title: "プライバシーポリシー",
  backLink: "トップへ戻る",
  lastUpdated: "最終更新日: 2026年2月22日",
  login: "ログイン",
  sections: [
    {
      heading: "1. 収集する情報",
      body: "本サービスでは、以下の情報を収集します：",
      items: [
        "アカウント情報（メールアドレス、名前など）",
        "OAuth認証を通じて取得するアクセストークン",
        "サービス利用ログ（アクセス日時、利用機能など）",
        "決済に関する情報（Stripeを通じて処理）",
      ],
    },
    {
      heading: "2. 情報の利用目的",
      body: "収集した情報は以下の目的で利用します：",
      items: [
        "本サービスの提供・運営",
        "ユーザーサポートの提供",
        "サービス改善のための分析",
        "重要なお知らせの送信",
      ],
    },
    {
      heading: "3. 情報の保護",
      body: "ユーザーの情報は適切なセキュリティ対策を講じて保護します。アクセストークンは暗号化して保存し、不正アクセスの防止に努めます。",
    },
    {
      heading: "4. 第三者への提供",
      body: "以下の場合を除き、ユーザーの同意なく第三者に情報を提供することはありません：",
      items: [
        "法令に基づく場合",
        "人の生命・身体・財産の保護に必要な場合",
        "サービス提供に必要な業務委託先（決済代行等）",
      ],
    },
    {
      heading: "5. 外部サービスとの連携",
      body: "本サービスは、Notion、GitHub、Google等の外部サービスとOAuth認証で連携します。各サービスでの情報の取り扱いは、それぞれのプライバシーポリシーに従います。",
    },
    {
      heading: "6. Cookieの使用",
      body: "本サービスでは、セッション管理およびユーザー体験向上のためにCookieを使用します。ブラウザの設定でCookieを無効にすることも可能ですが、一部機能が利用できなくなる場合があります。",
    },
    {
      heading: "7. データの削除",
      body: "ユーザーはアカウント設定からデータの削除を要求できます。アカウント削除時には、関連するすべてのデータを削除します。",
    },
    {
      heading: "8. ポリシーの変更",
      body: "本ポリシーは、必要に応じて変更されることがあります。重要な変更がある場合は、サービス上でお知らせします。",
    },
    {
      heading: "9. お問い合わせ",
      body: "プライバシーに関するお問い合わせは、サービス内のお問い合わせフォームよりご連絡ください。",
    },
  ],
}

const en: Content = {
  title: "Privacy Policy",
  backLink: "Back to Home",
  lastUpdated: "Last updated: February 22, 2026",
  login: "Login",
  sections: [
    {
      heading: "1. Information We Collect",
      body: "We collect the following information through our service:",
      items: [
        "Account information (email address, name, etc.)",
        "Access tokens obtained through OAuth authentication",
        "Service usage logs (access times, features used, etc.)",
        "Payment information (processed through Stripe)",
      ],
    },
    {
      heading: "2. Purpose of Use",
      body: "We use collected information for the following purposes:",
      items: [
        "Provision and operation of the service",
        "Providing user support",
        "Analysis for service improvement",
        "Sending important notifications",
      ],
    },
    {
      heading: "3. Information Protection",
      body: "We protect user information with appropriate security measures. Access tokens are stored encrypted, and we work to prevent unauthorized access.",
    },
    {
      heading: "4. Disclosure to Third Parties",
      body: "We do not disclose user information to third parties without consent, except in the following cases:",
      items: [
        "When required by law",
        "When necessary to protect life, body, or property",
        "Service providers necessary for operation (payment processing, etc.)",
      ],
    },
    {
      heading: "5. External Service Integration",
      body: "This service integrates with external services such as Notion, GitHub, and Google via OAuth authentication. Information handling by each service is governed by their respective privacy policies.",
    },
    {
      heading: "6. Use of Cookies",
      body: "We use cookies for session management and to improve user experience. You may disable cookies in your browser settings, but some features may become unavailable.",
    },
    {
      heading: "7. Data Deletion",
      body: "Users can request data deletion through their account settings. When an account is deleted, all associated data will be removed.",
    },
    {
      heading: "8. Policy Changes",
      body: "This policy may be updated as needed. We will notify users of significant changes through the service.",
    },
    {
      heading: "9. Contact",
      body: "For privacy-related inquiries, please contact us through the inquiry form within the service.",
    },
  ],
}

export default async function PrivacyPage({
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
              href={isEn ? "/privacy?lang=ja" : "/privacy?lang=en"}
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
