"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ServiceIcon } from "@/components/service-icon"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Check, ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"

export const dynamic = "force-dynamic"

// カテゴリ定義（背景色付き）
const categories = [
  { id: "productivity", name: "生産性", bgClass: "bg-blue-500/10" },
  { id: "development", name: "開発", bgClass: "bg-purple-500/10" },
  { id: "communication", name: "コミュニケーション", bgClass: "bg-green-500/10" },
  { id: "storage", name: "ストレージ", bgClass: "bg-amber-500/10" },
  { id: "analytics", name: "分析", bgClass: "bg-cyan-500/10" },
  { id: "marketing", name: "マーケティング", bgClass: "bg-pink-500/10" },
] as const

type CategoryId = (typeof categories)[number]["id"]

// サービス定義（カテゴリ付き）
interface ServiceItem {
  id: string
  name: string
  description: string
  icon: string
  category: CategoryId
  price: number
  features: string[]
}

const allServices: ServiceItem[] = [
  // 生産性
  { id: "google-calendar", name: "Google Calendar", description: "カレンダーイベントの管理と同期", icon: "calendar", category: "productivity", price: 0, features: ["イベントの取得・作成・削除", "複数カレンダー対応"] },
  { id: "notion", name: "Notion", description: "ドキュメントとデータベースの連携", icon: "file-text", category: "productivity", price: 0, features: ["ページ検索・取得", "データベース操作"] },
  { id: "microsoft-todo", name: "Microsoft To Do", description: "タスク・リストの管理", icon: "check-square", category: "productivity", price: 0, features: ["タスク一覧・作成", "リスト管理"] },
  { id: "todoist", name: "Todoist", description: "タスク管理とプロジェクト整理", icon: "check-circle", category: "productivity", price: 0, features: ["タスク作成・完了", "プロジェクト管理"] },
  { id: "asana", name: "Asana", description: "チームプロジェクト管理", icon: "layout", category: "productivity", price: 300, features: ["タスク管理", "チームコラボレーション"] },
  { id: "trello", name: "Trello", description: "カンバンボード形式のタスク管理", icon: "trello", category: "productivity", price: 0, features: ["ボード・カード操作", "チェックリスト"] },
  { id: "airtable", name: "Airtable", description: "スプレッドシート型データベース", icon: "table", category: "productivity", price: 500, features: ["レコード操作", "ビュー管理"] },
  { id: "clickup", name: "ClickUp", description: "オールインワンプロジェクト管理", icon: "zap", category: "productivity", price: 400, features: ["タスク・ドキュメント", "時間追跡"] },
  { id: "monday", name: "Monday.com", description: "ワークマネジメントプラットフォーム", icon: "calendar-days", category: "productivity", price: 400, features: ["プロジェクト管理", "自動化ワークフロー"] },
  { id: "basecamp", name: "Basecamp", description: "チームコラボレーション", icon: "tent", category: "productivity", price: 300, features: ["メッセージボード", "スケジュール管理"] },
  { id: "evernote", name: "Evernote", description: "ノートとメモの管理", icon: "sticky-note", category: "productivity", price: 0, features: ["ノート作成", "タグ管理"] },
  { id: "onenote", name: "OneNote", description: "デジタルノートブック", icon: "notebook", category: "productivity", price: 0, features: ["ノート整理", "共同編集"] },
  { id: "coda", name: "Coda", description: "ドキュメントとアプリの融合", icon: "file-code", category: "productivity", price: 300, features: ["ドキュメント作成", "自動化"] },
  { id: "smartsheet", name: "Smartsheet", description: "エンタープライズワーク管理", icon: "sheet", category: "productivity", price: 500, features: ["シート管理", "レポート作成"] },
  { id: "wrike", name: "Wrike", description: "プロジェクト管理ソフトウェア", icon: "gantt-chart", category: "productivity", price: 400, features: ["タスク追跡", "ガントチャート"] },
  { id: "teamwork", name: "Teamwork", description: "プロジェクト管理ツール", icon: "users-2", category: "productivity", price: 300, features: ["タスク管理", "時間追跡"] },

  // 開発
  { id: "github", name: "GitHub", description: "リポジトリとイシューの管理", icon: "github", category: "development", price: 0, features: ["リポジトリ一覧", "Issue作成・管理"] },
  { id: "jira", name: "Jira", description: "プロジェクト管理連携", icon: "layout-grid", category: "development", price: 500, features: ["Issue検索・作成", "プロジェクト管理", "ワークフロー連携"] },
  { id: "confluence", name: "Confluence", description: "Wiki・ドキュメント管理", icon: "book-open", category: "development", price: 500, features: ["ページ検索・取得", "スペース管理"] },
  { id: "gitlab", name: "GitLab", description: "DevOpsプラットフォーム", icon: "gitlab", category: "development", price: 0, features: ["リポジトリ管理", "CI/CD連携"] },
  { id: "bitbucket", name: "Bitbucket", description: "Gitリポジトリホスティング", icon: "git-branch", category: "development", price: 0, features: ["リポジトリ管理", "プルリクエスト"] },
  { id: "linear", name: "Linear", description: "モダンなイシュートラッキング", icon: "circle", category: "development", price: 300, features: ["イシュー管理", "サイクル計画"] },
  { id: "sentry", name: "Sentry", description: "エラー監視とパフォーマンス", icon: "alert-triangle", category: "development", price: 500, features: ["エラー追跡", "パフォーマンス監視"] },
  { id: "vercel", name: "Vercel", description: "フロントエンドデプロイ", icon: "triangle", category: "development", price: 0, features: ["デプロイ管理", "環境変数"] },
  { id: "netlify", name: "Netlify", description: "Webサイトホスティング", icon: "globe", category: "development", price: 0, features: ["デプロイ自動化", "フォーム処理"] },
  { id: "railway", name: "Railway", description: "インフラストラクチャプラットフォーム", icon: "train", category: "development", price: 0, features: ["デプロイ", "データベース"] },
  { id: "render", name: "Render", description: "クラウドアプリケーション", icon: "server", category: "development", price: 0, features: ["自動デプロイ", "スケーリング"] },
  { id: "datadog", name: "Datadog", description: "監視とセキュリティ", icon: "dog", category: "development", price: 500, features: ["メトリクス", "ログ管理"] },
  { id: "pagerduty", name: "PagerDuty", description: "インシデント管理", icon: "bell-ring", category: "development", price: 400, features: ["オンコール", "アラート"] },
  { id: "circleci", name: "CircleCI", description: "CI/CDプラットフォーム", icon: "circle-dot", category: "development", price: 0, features: ["ビルド自動化", "テスト実行"] },
  { id: "travis", name: "Travis CI", description: "継続的インテグレーション", icon: "wrench", category: "development", price: 0, features: ["ビルド", "テスト"] },
  { id: "sonarqube", name: "SonarQube", description: "コード品質管理", icon: "scan", category: "development", price: 400, features: ["静的解析", "品質ゲート"] },

  // コミュニケーション
  { id: "slack", name: "Slack", description: "チームコミュニケーション", icon: "message-square", category: "communication", price: 0, features: ["メッセージ送信", "チャンネル管理"] },
  { id: "discord", name: "Discord", description: "コミュニティチャット", icon: "message-circle", category: "communication", price: 0, features: ["メッセージ送信", "サーバー管理"] },
  { id: "teams", name: "Microsoft Teams", description: "ビジネスコラボレーション", icon: "users", category: "communication", price: 0, features: ["チャット", "会議スケジュール"] },
  { id: "zoom", name: "Zoom", description: "ビデオ会議", icon: "video", category: "communication", price: 300, features: ["会議作成", "参加者管理"] },
  { id: "gmail", name: "Gmail", description: "メール管理", icon: "mail", category: "communication", price: 0, features: ["メール送受信", "ラベル管理"] },
  { id: "outlook", name: "Outlook", description: "メールとカレンダー", icon: "inbox", category: "communication", price: 0, features: ["メール管理", "予定調整"] },
  { id: "webex", name: "Webex", description: "ビデオ会議ソリューション", icon: "video-off", category: "communication", price: 300, features: ["会議", "ウェビナー"] },
  { id: "google-meet", name: "Google Meet", description: "ビデオ通話", icon: "webcam", category: "communication", price: 0, features: ["ビデオ会議", "画面共有"] },
  { id: "loom", name: "Loom", description: "動画メッセージング", icon: "play-circle", category: "communication", price: 0, features: ["録画", "共有"] },
  { id: "calendly", name: "Calendly", description: "スケジュール調整", icon: "calendar-check", category: "communication", price: 0, features: ["予約管理", "カレンダー連携"] },
  { id: "twilio", name: "Twilio", description: "コミュニケーションAPI", icon: "phone", category: "communication", price: 500, features: ["SMS送信", "音声通話"] },
  { id: "sendgrid", name: "SendGrid", description: "メール配信サービス", icon: "mail-plus", category: "communication", price: 300, features: ["メール送信", "分析"] },

  // ストレージ
  { id: "google-drive", name: "Google Drive", description: "クラウドストレージ", icon: "hard-drive", category: "storage", price: 0, features: ["ファイル管理", "共有設定"] },
  { id: "dropbox", name: "Dropbox", description: "ファイル同期と共有", icon: "box", category: "storage", price: 0, features: ["ファイルアップロード", "フォルダ管理"] },
  { id: "onedrive", name: "OneDrive", description: "Microsoft クラウドストレージ", icon: "cloud", category: "storage", price: 0, features: ["ファイル同期", "共同編集"] },
  { id: "box", name: "Box", description: "エンタープライズストレージ", icon: "archive", category: "storage", price: 500, features: ["セキュアな共有", "ワークフロー"] },
  { id: "aws-s3", name: "AWS S3", description: "オブジェクトストレージ", icon: "database", category: "storage", price: 300, features: ["ファイル保存", "バージョニング"] },
  { id: "cloudflare-r2", name: "Cloudflare R2", description: "S3互換ストレージ", icon: "cloud-cog", category: "storage", price: 0, features: ["オブジェクト保存", "グローバル配信"] },
  { id: "backblaze", name: "Backblaze B2", description: "クラウドバックアップ", icon: "upload-cloud", category: "storage", price: 0, features: ["バックアップ", "復元"] },
  { id: "wasabi", name: "Wasabi", description: "ホットクラウドストレージ", icon: "flame", category: "storage", price: 300, features: ["高速アクセス", "大容量"] },

  // 分析
  { id: "google-analytics", name: "Google Analytics", description: "ウェブ解析", icon: "bar-chart-2", category: "analytics", price: 0, features: ["トラフィック分析", "レポート取得"] },
  { id: "mixpanel", name: "Mixpanel", description: "プロダクト分析", icon: "pie-chart", category: "analytics", price: 500, features: ["イベント追跡", "ファネル分析"] },
  { id: "amplitude", name: "Amplitude", description: "行動分析プラットフォーム", icon: "activity", category: "analytics", price: 500, features: ["ユーザー行動", "コホート分析"] },
  { id: "hotjar", name: "Hotjar", description: "ヒートマップと録画", icon: "map", category: "analytics", price: 300, features: ["ヒートマップ", "セッション録画"] },
  { id: "plausible", name: "Plausible", description: "プライバシー重視の分析", icon: "eye-off", category: "analytics", price: 0, features: ["軽量", "GDPR対応"] },
  { id: "posthog", name: "PostHog", description: "オープンソース分析", icon: "bar-chart", category: "analytics", price: 0, features: ["イベント追跡", "機能フラグ"] },
  { id: "segment", name: "Segment", description: "顧客データプラットフォーム", icon: "git-merge", category: "analytics", price: 500, features: ["データ統合", "ルーティング"] },
  { id: "heap", name: "Heap", description: "デジタルインサイト", icon: "layers", category: "analytics", price: 500, features: ["自動キャプチャ", "分析"] },

  // マーケティング
  { id: "mailchimp", name: "Mailchimp", description: "メールマーケティング", icon: "send", category: "marketing", price: 0, features: ["キャンペーン作成", "リスト管理"] },
  { id: "hubspot", name: "HubSpot", description: "CRM・マーケティング", icon: "target", category: "marketing", price: 500, features: ["コンタクト管理", "メール自動化"] },
  { id: "intercom", name: "Intercom", description: "カスタマーメッセージング", icon: "message-square", category: "marketing", price: 500, features: ["チャットサポート", "自動メッセージ"] },
  { id: "zendesk", name: "Zendesk", description: "カスタマーサポート", icon: "headphones", category: "marketing", price: 400, features: ["チケット管理", "ナレッジベース"] },
  { id: "salesforce", name: "Salesforce", description: "CRMプラットフォーム", icon: "cloud-lightning", category: "marketing", price: 500, features: ["顧客管理", "営業自動化"] },
  { id: "pipedrive", name: "Pipedrive", description: "セールスCRM", icon: "filter", category: "marketing", price: 300, features: ["パイプライン", "取引管理"] },
  { id: "activecampaign", name: "ActiveCampaign", description: "マーケティング自動化", icon: "zap", category: "marketing", price: 300, features: ["メール自動化", "CRM"] },
  { id: "klaviyo", name: "Klaviyo", description: "Eコマースマーケティング", icon: "shopping-bag", category: "marketing", price: 400, features: ["メール/SMS", "セグメント"] },
  { id: "drift", name: "Drift", description: "会話型マーケティング", icon: "message-circle", category: "marketing", price: 500, features: ["チャットボット", "リード獲得"] },
  { id: "freshdesk", name: "Freshdesk", description: "ヘルプデスクソフトウェア", icon: "life-buoy", category: "marketing", price: 0, features: ["チケット管理", "自動化"] },
]

// ユーザーが購入済みのサービス（モック）
const purchasedServices = ["google-calendar", "notion", "github", "microsoft-todo", "slack", "google-drive"]

export default function MarketplacePage() {
  const [purchased, setPurchased] = useState<string[]>(purchasedServices)
  const [expanded, setExpanded] = useState<string | null>(null)

  const handlePurchase = (serviceId: string) => {
    setPurchased((prev) => [...prev, serviceId])
  }

  const formatPrice = (price: number) => {
    if (price === 0) return "無料"
    return `¥${price}/月`
  }

  // カテゴリごとにサービスをグループ化し、利用中を先頭にソート
  const getServicesByCategory = (categoryId: CategoryId) => {
    return allServices
      .filter((s) => s.category === categoryId)
      .sort((a, b) => {
        const aPurchased = purchased.includes(a.id)
        const bPurchased = purchased.includes(b.id)
        if (aPurchased !== bPurchased) return aPurchased ? -1 : 1
        return a.price - b.price
      })
  }

  const renderServiceCard = (service: ServiceItem) => {
    const isPurchased = purchased.includes(service.id)
    const isFree = service.price === 0
    const isExpanded = expanded === service.id

    return (
      <Collapsible
        key={service.id}
        open={isExpanded}
        onOpenChange={(open) => setExpanded(open ? service.id : null)}
      >
        <div
          className={cn(
            "rounded-lg border bg-background/50 transition-colors",
            isPurchased ? "border-success/50" : "border-border/50 hover:border-border",
          )}
        >
          <CollapsibleTrigger className="w-full p-3 text-left">
            <div className="flex items-center gap-3">
              {/* アイコン */}
              <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                <ServiceIcon icon={service.icon} className="h-4 w-4 text-foreground" />
              </div>

              {/* サービス情報 */}
              <div className="flex-1 min-w-0">
                <span className="font-medium text-sm text-foreground">{service.name}</span>
                <p className="text-xs text-muted-foreground line-clamp-1">{service.description}</p>
              </div>

              {/* 価格・利用中バッジ・展開アイコン */}
              <div className="flex items-center gap-2 shrink-0">
                {isPurchased ? (
                  <Badge variant="outline" className="text-success border-success/50 text-[10px] py-0 px-1.5">
                    <Check className="h-3 w-3 mr-0.5" />
                    利用中
                  </Badge>
                ) : (
                  <span className={cn("text-xs font-medium", isFree ? "text-success" : "text-muted-foreground")}>
                    {formatPrice(service.price)}
                  </span>
                )}
                <ChevronDown
                  className={cn(
                    "h-4 w-4 text-muted-foreground transition-transform duration-200",
                    isExpanded && "rotate-180"
                  )}
                />
              </div>
            </div>
          </CollapsibleTrigger>

          <CollapsibleContent>
            <div className="px-3 pb-3 pt-0">
              <div className="border-t border-border/30 pt-3">
                {/* 機能一覧 */}
                <div className="mb-3">
                  <p className="text-xs text-muted-foreground mb-1.5">機能</p>
                  <ul className="space-y-1">
                    {service.features.map((feature, i) => (
                      <li key={i} className="text-xs text-foreground flex items-center gap-1.5">
                        <Check className="h-3 w-3 text-success shrink-0" />
                        {feature}
                      </li>
                    ))}
                  </ul>
                </div>

                {/* アクション */}
                {!isPurchased && (
                  <Button
                    size="sm"
                    variant={isFree ? "outline" : "default"}
                    className="w-full h-7 text-xs"
                    onClick={(e) => {
                      e.stopPropagation()
                      handlePurchase(service.id)
                    }}
                  >
                    {isFree ? "追加する" : "購入する"}
                  </Button>
                )}
              </div>
            </div>
          </CollapsibleContent>
        </div>
      </Collapsible>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="p-6 pb-4">
        <h1 className="text-2xl font-bold text-foreground">マーケットプレイス</h1>
        <p className="text-muted-foreground mt-1">利用したいサービスを選んで追加できます</p>
      </div>

      {/* カテゴリ別横スクロール - 残り全体を使用 */}
      <div className="flex-1 overflow-x-auto px-6 pb-6 min-h-0">
        <div className="flex gap-4 h-full" style={{ width: "max-content" }}>
          {categories.map((category) => {
            const services = getServicesByCategory(category.id)
            return (
              <div
                key={category.id}
                className={cn("w-80 shrink-0 flex flex-col rounded-xl p-4 h-full overflow-hidden", category.bgClass)}
              >
                {/* カテゴリヘッダー */}
                <div className="flex items-center gap-2 mb-3 shrink-0">
                  <h2 className="text-sm font-semibold text-foreground">{category.name}</h2>
                  <Badge variant="secondary" className="text-xs">
                    {services.length}
                  </Badge>
                </div>

                {/* サービス一覧 - 列ごとに縦スクロール（スクロールバー非表示） */}
                <div className="space-y-2 flex-1 overflow-y-auto min-h-0 [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]">
                  {services.map(renderServiceCard)}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
