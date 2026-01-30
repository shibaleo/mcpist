import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Zap, ArrowLeft } from "lucide-react"

export default function PrivacyPage() {
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
          <Link href="/login">
            <Button variant="outline">ログイン</Button>
          </Link>
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
            トップへ戻る
          </Link>

          <h1 className="text-3xl font-bold text-foreground mb-8">プライバシーポリシー</h1>

          <div className="prose prose-neutral dark:prose-invert max-w-none space-y-6">
            <p className="text-muted-foreground">最終更新日: 2026年1月30日</p>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">1. 収集する情報</h2>
              <p className="text-muted-foreground">本サービスでは、以下の情報を収集します：</p>
              <ul className="list-disc list-inside text-muted-foreground space-y-2">
                <li>アカウント情報（メールアドレス、名前など）</li>
                <li>OAuth認証を通じて取得するアクセストークン</li>
                <li>サービス利用ログ（アクセス日時、利用機能など）</li>
                <li>決済に関する情報（Stripeを通じて処理）</li>
              </ul>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">2. 情報の利用目的</h2>
              <p className="text-muted-foreground">収集した情報は以下の目的で利用します：</p>
              <ul className="list-disc list-inside text-muted-foreground space-y-2">
                <li>本サービスの提供・運営</li>
                <li>ユーザーサポートの提供</li>
                <li>サービス改善のための分析</li>
                <li>重要なお知らせの送信</li>
              </ul>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">3. 情報の保護</h2>
              <p className="text-muted-foreground">
                ユーザーの情報は適切なセキュリティ対策を講じて保護します。
                アクセストークンは暗号化して保存し、不正アクセスの防止に努めます。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">4. 第三者への提供</h2>
              <p className="text-muted-foreground">
                以下の場合を除き、ユーザーの同意なく第三者に情報を提供することはありません：
              </p>
              <ul className="list-disc list-inside text-muted-foreground space-y-2">
                <li>法令に基づく場合</li>
                <li>人の生命・身体・財産の保護に必要な場合</li>
                <li>サービス提供に必要な業務委託先（決済代行等）</li>
              </ul>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">5. 外部サービスとの連携</h2>
              <p className="text-muted-foreground">
                本サービスは、Notion、GitHub、Google等の外部サービスとOAuth認証で連携します。
                各サービスでの情報の取り扱いは、それぞれのプライバシーポリシーに従います。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">6. Cookieの使用</h2>
              <p className="text-muted-foreground">
                本サービスでは、セッション管理およびユーザー体験向上のためにCookieを使用します。
                ブラウザの設定でCookieを無効にすることも可能ですが、一部機能が利用できなくなる場合があります。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">7. データの削除</h2>
              <p className="text-muted-foreground">
                ユーザーはアカウント設定からデータの削除を要求できます。
                アカウント削除時には、関連するすべてのデータを削除します。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">8. ポリシーの変更</h2>
              <p className="text-muted-foreground">
                本ポリシーは、必要に応じて変更されることがあります。
                重要な変更がある場合は、サービス上でお知らせします。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">9. お問い合わせ</h2>
              <p className="text-muted-foreground">
                プライバシーに関するお問い合わせは、サービス内のお問い合わせフォームよりご連絡ください。
              </p>
            </section>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-border py-6">
        <div className="container mx-auto px-4 text-center text-sm text-muted-foreground">
          <div className="flex justify-center gap-4 mb-2">
            <Link href="/terms" className="hover:text-foreground">
              利用規約
            </Link>
            <Link href="/privacy" className="hover:text-foreground">
              プライバシーポリシー
            </Link>
          </div>
          &copy; 2026 MCPist. All rights reserved.
        </div>
      </footer>
    </div>
  )
}
