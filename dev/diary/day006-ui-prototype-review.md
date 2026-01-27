# MCPist UI プロトタイプレビュー

## 概要

**日付**: 2026-01-16
**成果物**: `mcpist-ui-v1`（Next.js 16 + shadcn/ui）
**ページ数**: 19ページ

---

## 作成した画面一覧

### メイン
| ページ | パス | 説明 |
|--------|------|------|
| Dashboard | `/dashboard` | 統計カード、最近のアクティビティ |
| Tools | `/tools` | サービス一覧（プラン別表示） |
| Tool詳細 | `/tools/[module]` | サービス内の機能一覧 |
| API Tokens | `/tokens` | APIトークン管理 |

### My（ユーザー向け）
| ページ | パス | 説明 |
|--------|------|------|
| マイ接続 | `/my/connections` | サービス接続管理（OAuth/トークン選択） |
| MCP接続情報 | `/my/mcp-connection` | エンドポイント＋APIトークン発行 |
| 機能設定 | `/my/preferences` | 利用する機能の選択 |

### Admin（管理者向け）
| ページ | パス | 説明 |
|--------|------|------|
| Users | `/users` | ユーザー管理 |
| Roles | `/roles` | ロール管理 |
| Profiles | `/profiles` | 権限マトリクス設定 |
| サービス認証設定 | `/service-auth` | OAuthクライアント設定 |
| プランと請求 | `/billing` | Stripe風課金UI |
| Requests | `/requests` | 利用申請承認 |
| Logs | `/logs` | 操作ログ |

### 認証フロー
| ページ | パス | 説明 |
|--------|------|------|
| Login | `/login` | OAuth認証（Google/GitHub/Microsoft） |
| Consent | `/consent` | 権限同意画面 |
| Onboarding | `/onboarding` | 初回セットアップ |

---

## プロトタイプで得た知見

### 1. 認証の分離

```
┌─────────────────────────────────────────────────────────────┐
│ 管理者（組織レベル）                                         │
│ → OAuthクライアント設定（Client ID/Secret）                  │
│ → サービスごとに利用可能な認証方法を定義                     │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ ユーザー（個人レベル）                                       │
│ → 自分のクレデンシャル（OAuth認可 or トークン入力）          │
│ → 認証方法を選択可能（管理者が許可した範囲内）               │
└─────────────────────────────────────────────────────────────┘
```

**例**:
- Aさん: JiraはOAuth、NotionはIntegration Token
- Bさん: JiraはAPIキー、NotionはOAuth

### 2. 権限の階層構造

```
プラン（課金）
  └── サービス（Google Calendar, Notion, etc.）
        └── 機能/ツール（list_events, create_event, delete_event）
              ├── 管理者が有効化（組織で使える範囲）
              └── ユーザーが自分で選択（自分が使いたいもの）
```

### 3. 課金モデル

#### プロトタイプ時点（プラン課金）

| プラン | 価格 | サービス | 機能 | ユーザー |
|--------|------|----------|------|----------|
| Free | ¥0 | 4つ | 読み取り系のみ | 5名 |
| Pro | ¥2,980/月 | 8つ | 読み書き | 20名 |
| Max | ¥9,800/月 | 全て | 全機能（削除含む） | 無制限 |

#### 検討後（サービス単位課金に変更予定）

**問題点**: プラン課金だと使わないサービス/ツールにも払うことになる

**新モデル**:
```
基本料金: ¥0（プラットフォーム利用料）
+ サービスごとに選んで追加
```

| サービス | 月額 | 含まれるツール |
|----------|------|----------------|
| Google Calendar | ¥300 | list/create/update/delete等 全20個 |
| Notion | ¥500 | search/create/update等 全25個 |
| GitHub | ¥500 | repo/issue/PR操作等 全30個 |
| Jira | ¥800 | issue/project/sprint等 全35個 |
| Slack | ¥300 | message/channel等 全15個 |

**メリット**:
- ユーザー: 使うサービスだけ払う、無駄がない
- 運営: サービス追加のたびに収益機会
- シンプル: ツール単位より選びやすい

**価格設定の考え方**:
- API複雑度（認証が複雑なほど高い）
- ツール数（多いほど高い）
- 需要（人気サービスは高めでも売れる）

**UI変更**: Billingページをカート方式に変更が必要

**課金導線**:
- 未契約サービスをグレーアウトで表示
- クリックで追加促進ダイアログ

### 4. ユーザー自己設定

管理者が許可した範囲内で、ユーザー自身が：
- 使いたいサービスに接続
- 使いたい機能を有効化/無効化
- MCPトークンを自己発行（再発行も可能）

---

## データモデル（主要なもの）

### サービス認証設定
```typescript
interface ServiceAuthConfig {
  serviceId: string
  availableMethods: {
    type: "oauth2" | "apikey" | "personal_token" | "integration_token"
    enabled: boolean
    label: string
    oauth?: { clientId, clientSecret, scopes }
    helpText?: string
  }[]
}
```

### ユーザークレデンシャル
```typescript
interface UserServiceCredential {
  userId: string
  serviceId: string
  authMethod: "oauth2" | "apikey" | "personal_token" | "integration_token"
  status: "active" | "expired" | "error"
  connectedAt: string
}
```

### MCP接続情報
```typescript
interface UserMcpConnection {
  userId: string
  endpoint: string           // "https://mcp.example.com/u/{userId}"
  apiToken: string | null    // 発行済みはマスク表示
  status: "not_generated" | "active" | "revoked"
}
```

### プラン要件
```typescript
interface ServicePlanRequirement {
  serviceId: string
  requiredPlan: "free" | "pro" | "max"
}

interface ToolPlanRequirement {
  serviceId: string
  toolId: string
  requiredPlan: "free" | "pro" | "max"
}
```

---

## 残タスク（UI改善）

| タスク | 優先度 | 説明 |
|--------|--------|------|
| マイ接続の整理 | 中 | 課金無効サービスを下部にまとめる |
| プロファイル一覧 | 中 | 一覧タブを追加（現状マトリクスのみ） |
| 戻るボタン | 低 | 全ページにナビゲーション追加 |
| サイドバーデザイン | 低 | 折りたたみスライダーの改善 |
| Tools表示切替 | 中 | 管理者/ユーザーで表示を分ける |
| ダッシュボードリンク | 低 | カードをクリック可能に |

---

## ユーザーストーリーへの反映ポイント

### 追加・変更が必要なストーリー

1. **US-AUTH-01**: 認証方法の選択
   - 管理者がサービスごとに認証方法を有効化
   - ユーザーが自分の認証方法を選択

2. **US-BILLING-01**: プラン管理（新規）
   - 組織のプラン選択
   - サービス/機能の利用制限
   - アップグレード/ダウングレード

3. **US-PERM-01**: 権限の階層化
   - プラン → サービス → 機能の3層構造
   - ユーザー自身の機能選択

4. **US-TOKEN-01**: MCPトークン管理
   - ユーザー自身で発行・再発行
   - Claude Desktop設定例の表示

---

## 技術的な決定事項

| 項目 | 決定 |
|------|------|
| フレームワーク | Next.js 16 (App Router) |
| UIライブラリ | shadcn/ui + Tailwind CSS |
| 状態管理 | React useState（プロトタイプ） |
| 認証 | OAuth 2.0 / OIDC / APIキー / 長期トークン |
| ホスティング | Vercel（Hobbyプラン可） |

---

---

## プロジェクトの本質（再定義）

### ターゲット

```
開発者 = 管理者 = ユーザー = 自分（+ 友人数人）
```

### ゴール: 放置運用

```
Phase 1: 開発者として構築
    ↓
Phase 2: 管理者としてGUI運用
    ↓
Phase 3: ユーザーとして利用のみ（開発・管理を忘れる）
```

**将来の自分が何も覚えていなくても運用できる**システムが必要。

### スケールしないが必須な機能

| 機能 | 理由 |
|------|------|
| 課金 | 友人との費用割り勘、利用量に応じた負担 |
| ユーザー管理 | 友人の招待・削除をGUIで |
| ロール/権限 | 誰が何を使えるかをGUIで管理 |
| サービス認証設定 | OAuthクライアントをGUIで管理（忘れても再設定可能） |
| 監査ログ | 問題発生時の振り返り |
| セキュリティ | Vault、Auth、暗号化は必須 |

### 課金の目的

- コスト回収（クラウド費用を友人と割り勘）
- 利用量に応じた公平な負担
- → **サービス単位課金**が妥当

---

## TODO: 追加で検討が必要な仕様

### 1. ステークホルダー分析

| フェーズ | ターゲット | 規模 | 特徴 |
|----------|-----------|------|------|
| Phase 0 | 自分のみ | 1人 | 開発・検証 |
| Phase 1 | 自分 + 友人 | 5-10人 | 小規模運用、費用割り勘 |
| Phase 2 | 小規模チーム | 10-50人 | スタンドアローン提供 or SaaS |
| Phase 3 | 情シス向けSaaS | 100人+ | マルチテナント、エンタープライズ |

**現在の設計**: Phase 1（自分 + 友人）をターゲットに開発
**UIの設計**: Phase 3（情シス向けSaaS）まで見据えて大げさに作る

### 2. 運用仕様（TODO）

- [ ] デプロイ方式（Vercel? Cloud Run? 自前VPS?）
- [ ] バックアップ戦略
- [ ] 監視・アラート
- [ ] 障害対応フロー（放置運用でも気づける仕組み）
- [ ] シークレットローテーション
- [ ] ユーザーサポート導線（友人向け）

### 3. 課金仕様（TODO）

- [ ] 決済プロバイダ（Stripe? PAY.JP?）
- [ ] 課金サイクル（月次? 年次?）
- [ ] 無料枠の設計
- [ ] 請求書発行
- [ ] 未払い時の対応
- [ ] 友人向けの特別対応（割引? 無料?）

### 4. スケール計画

```
Phase 1: 自分 + 友人（現在）
├── シングルテナント
├── OAuthクライアントは自分が管理
├── 課金は友人との割り勘（Stripe or 手動）
└── UI: 管理者機能は全て実装、でも自分しか使わない

Phase 2: 小規模チーム向けスタンドアローン
├── Docker/Helm で配布
├── 各チームが自分のOAuthクライアントを設定
├── 課金なし（OSS or ライセンス販売）
└── UI: そのまま使える

Phase 3: 情シス向けSaaS
├── マルチテナント化
├── MCPist側でOAuthクライアント提供（オプション）
├── Stripe課金
└── UI: そのまま使える
```

### 5. UI設計方針

**方針**: 小さく始めてスケールするために、UIは最初から大げさに作る

| 機能 | Phase 1で使う? | 実装する? | 理由 |
|------|---------------|-----------|------|
| マルチテナント | No | Yes | Phase 3で必要 |
| ロール/権限 | 簡易的に | Yes | Phase 2-3で必要 |
| 課金/Billing | 割り勘用 | Yes | Phase 3で必要 |
| 監査ログ | デバッグ用 | Yes | セキュリティ要件 |
| ユーザー招待 | 友人用 | Yes | 全Phase共通 |
| プロファイル | 簡易的に | Yes | Phase 2-3で必要 |
| サービス認証設定 | 自分で設定 | Yes | Phase 3でSaaS化時に必要 |

---

## バックエンド実装状況（Go MCP Server）

### 概要

MCPサーバーはすでにDockerで稼働中（`gocp` MCPツール）

### 実装済みコンポーネント

| コンポーネント | ファイル | 状況 |
|---------------|----------|------|
| MCPハンドラ | `internal/mcp/handler.go` | ✅ 実装済み |
| MCP型定義 | `internal/mcp/types.go` | ✅ 実装済み |
| 認証ミドルウェア | `internal/auth/middleware.go` | ✅ Bearer Token認証 |
| モジュールレジストリ | `internal/modules/registry.go` | ✅ 遅延読み込み対応 |
| HTTPクライアント | `internal/httpclient/client.go` | ✅ 共通HTTP処理 |
| Loki連携 | `internal/observability/loki.go` | ✅ ログ送信 |

### 実装済みモジュール

| モジュール | ファイル | ツール数 | テスト日 |
|-----------|----------|----------|----------|
| Notion | `internal/modules/notion/module.go` | 14ツール | 2026-01-10 |
| Jira | `internal/modules/jira/module.go` | 13ツール | 2026-01-10 |
| Confluence | `internal/modules/confluence/module.go` | - | 2026-01-10 |
| GitHub | `internal/modules/github/module.go` | - | - |
| Supabase | `internal/modules/supabase/module.go` | - | - |
| Airtable | `internal/modules/airtable/module.go` | - | - |

### アーキテクチャ

```
Claude Desktop / AI Client
    ↓ MCP Protocol (SSE + JSON-RPC)
    ↓
┌─────────────────────────────────────┐
│  go-mcp-dev Server (:8080)          │
│  ├── /health (ヘルスチェック)        │
│  └── /mcp (MCPエンドポイント)        │
│       └── auth.NewMiddleware        │
│            └── mcp.Handler          │
│                 ├── initialize      │
│                 ├── tools/list      │
│                 └── tools/call      │
│                      ├── get_module_schema  │
│                      └── call_module_tool   │
│                           └── modules.Registry │
└─────────────────────────────────────┘
    ↓ HTTP API
    ↓
┌─────────────────┐
│ External APIs   │
│ - Notion        │
│ - Jira          │
│ - Confluence    │
│ - GitHub        │
│ - Supabase      │
│ - Airtable      │
└─────────────────┘
```

### 特徴

1. **遅延読み込み（Lazy Loading）**
   - `get_module_schema` で必要なモジュールのみスキーマ取得
   - トークン消費を最小化

2. **SSE対応**
   - MCP 2024-11-05 プロトコル準拠
   - リアルタイム双方向通信

3. **オブザーバビリティ**
   - Grafana Loki への構造化ログ送信
   - ツール呼び出しの計測（duration_ms）

4. **シンプルな認証**
   - 現状: Bearer Token（`INTERNAL_SECRET`）
   - 将来: OAuth 2.0 / OIDC 対応予定

### Dockerデプロイ

```dockerfile
# ビルドステージ
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# 本番ステージ
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]
```

---

## ポートフォリオ評価（更新）

### 完成度: 50-60%

| カテゴリ | 項目 | 状況 | 備考 |
|---------|------|------|------|
| **バックエンド** | MCPサーバー | ✅ 稼働中 | Go実装、Docker化済み |
| | モジュール実装 | ✅ 6モジュール | Notion, Jira, Confluence, GitHub, Supabase, Airtable |
| | 認証 | ⚠️ 簡易実装 | Bearer Token → OAuth 2.0 移行予定 |
| | オブザーバビリティ | ✅ 実装済み | Grafana Loki連携 |
| | テスト | ⚠️ 一部のみ | `*_test.go` 存在 |
| **フロントエンド** | UIプロトタイプ | ✅ 19画面 | Next.js 16 + shadcn/ui |
| | API接続 | ❌ 未実装 | モックデータのみ |
| **インフラ** | Docker | ✅ 完成 | マルチステージビルド |
| | CI/CD | ❌ 未実装 | |
| | 本番デプロイ | ⚠️ 部分的 | MCPサーバーは稼働中 |
| **ドキュメント** | README | ⚠️ 要更新 | |
| | API仕様 | ❌ 未作成 | OpenAPI等 |

### 次のステップ（優先順）

1. **UI-バックエンド接続**
   - 管理UI（mcpist-ui-v1）のAPI接続
   - ユーザー管理、サービス設定のCRUD

2. **認証強化**
   - OAuth 2.0 / OIDC 実装
   - マルチユーザー対応

3. **テスト充実**
   - ユニットテスト追加
   - E2Eテスト（Playwright等）

4. **CI/CD構築**
   - GitHub Actions
   - 自動テスト・デプロイ

5. **ドキュメント整備**
   - README更新
   - API仕様書作成

---

## 次のステップ

1. **ユーザーストーリーのレビュー・更新**
2. **運用仕様の検討**
3. **課金仕様の検討**
4. **UI-バックエンド接続の設計**
5. **データベーススキーマ設計**（Supabase）
6. **認証フロー実装**（OAuth 2.0）
7. **フロントエンド・バックエンド接続**
