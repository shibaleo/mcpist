import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Zap, ArrowLeft } from "lucide-react"

export default function TermsPage() {
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

          <h1 className="text-3xl font-bold text-foreground mb-8">利用規約</h1>

          <div className="prose prose-neutral dark:prose-invert max-w-none space-y-6">
            <p className="text-muted-foreground">最終更新日: 2026年1月30日</p>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第1条（適用）</h2>
              <p className="text-muted-foreground">
                本規約は、MCPist（以下「本サービス」）の利用条件を定めるものです。
                ユーザーは本規約に同意した上で本サービスを利用するものとします。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第2条（サービス内容）</h2>
              <p className="text-muted-foreground">
                本サービスは、MCPプロトコルを通じて各種外部サービス（Notion、GitHub、Jira、Google
                Calendarなど）への接続を提供するプラットフォームです。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第3条（アカウント）</h2>
              <p className="text-muted-foreground">
                ユーザーは、正確な情報を提供してアカウントを作成する必要があります。
                アカウント情報の管理はユーザーの責任とし、第三者への譲渡・貸与は禁止します。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第4条（クレジット）</h2>
              <p className="text-muted-foreground">
                本サービスの一部機能はクレジット制で提供されます。
                クレジットの有効期限、返金条件等は別途定めるものとします。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第5条（禁止事項）</h2>
              <p className="text-muted-foreground">以下の行為を禁止します：</p>
              <ul className="list-disc list-inside text-muted-foreground space-y-2">
                <li>法令または公序良俗に反する行為</li>
                <li>本サービスの運営を妨害する行為</li>
                <li>他のユーザーへの迷惑行為</li>
                <li>不正アクセスまたはそれを試みる行為</li>
                <li>本サービスを利用した営利目的の転売行為</li>
              </ul>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第6条（免責事項）</h2>
              <p className="text-muted-foreground">
                本サービスは現状有姿で提供され、特定目的への適合性を保証するものではありません。
                外部サービスの障害・変更等による影響について、当社は責任を負いません。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第7条（規約の変更）</h2>
              <p className="text-muted-foreground">
                当社は、必要に応じて本規約を変更できるものとします。
                変更後の規約は、本サービス上での公開をもって効力を生じます。
              </p>
            </section>

            <section className="space-y-4">
              <h2 className="text-xl font-semibold text-foreground">第8条（準拠法・管轄）</h2>
              <p className="text-muted-foreground">
                本規約は日本法に準拠し、本サービスに関する紛争は東京地方裁判所を第一審の専属的合意管轄裁判所とします。
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
