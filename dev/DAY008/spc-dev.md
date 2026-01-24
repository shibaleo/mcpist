# MCPist 開発計画書（spc-dev）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Development Plan Specification |

---

## 概要

本ドキュメントは、MCPistの開発計画・リソース・体制を定義する。

---

## 開発体制

### チーム構成

| 役割 | 担当 | 備考 |
|------|------|------|
| 開発者 | 1名（個人） | 設計・実装・テスト・運用 |
| AIアシスタント | Claude | 設計レビュー・コード生成・ドキュメント |

### 開発方針

| 方針 | 説明 |
|------|------|
| AIファースト | Claude Codeを活用した高速開発 |
| ドキュメント駆動 | 仕様書を先に書き、実装の指針とする |
| 段階的リリース | Phase 1（MVP）から段階的に機能拡張 |
| 無料枠運用 | Phase 1は全サービス無料枠内で運用 |

---

## フェーズ計画

### Phase 1: MVP

**目標:** 最小限の機能でサービスを公開

| 項目      | 内容                                                 |
| ------- | -------------------------------------------------- |
| 対象ユーザー  | 5-10名（招待制）                                         |
| コア機能    | MCP Server、Module Registry、OAuth連携                 |
| 対応モジュール | Notion, GitHub, Google Calendar, Microsoft Todoなど． |
| 認証      | Supabase Auth（ソーシャルログイン）                           |
| 課金      | 無料のみ（Free Plan）                                    |

**成果物:**
- MCP Server（Go）デプロイ済み
- User Console（Next.js）デプロイ済み
- API Gateway（Cloudflare Worker）デプロイ済み
- 基本的な監視・アラート設定

### Phase 2: 有料プラン導入

**目標:** 有料プランを導入し、収益化

| 項目 | 内容 |
|------|------|
| 対象ユーザー | 100+ MAU |
| 追加機能 | Starter/Pro/Unlimited プラン |
| 課金連携 | Stripe |
| Rate Limit | プラン別制限 |
| Quota/Credit | 月間使用量制限、従量課金 |

**成果物:**
- プラン管理機能
- Stripe決済連携
- 使用量ダッシュボード

### Phase 3: スケーラビリティ

**目標:** 大規模ユーザーに対応

| 項目 | 内容 |
|------|------|
| 対象ユーザー | 1,000+ MAU |
| インフラ強化 | インスタンス増強、キャッシュ導入 |
| 監視強化 | 詳細メトリクス、アラート最適化 |
| ステージング | ステージング環境追加 |

---

## マイルストーン（Phase 1）

### M1: 基盤構築

| タスク | 成果物 |
|--------|--------|
| リポジトリ初期化 | モノレポ構成（apps/server, apps/console, apps/worker, apps/supabase） |
| CI/CD設定 | GitHub Actions（Lint, Test, Build） |
| Supabase設定 | プロジェクト作成、Auth設定、スキーマ作成 |
| ローカル開発環境 | Docker Compose + Supabase CLI |

### M2: MCP Server

| タスク | 成果物 |
|--------|--------|
| MCP Protocol Handler | JSON-RPC over SSE実装 |
| Module Registry | get_module_schema, call_module_tool, batch |
| Auth Middleware | JWT検証、X-User-ID抽出 |
| モジュール移行 | 既存モジュールを新インターフェースに移行 |

### M3: Entitlement Store

| タスク | 成果物 |
|--------|--------|
| DBスキーマ | mcpist.users, subscriptions, plans, usage, credits等 |
| Rate Limit実装 | プラン別制限（メモリ管理） |
| Quota/Credit実装 | 月間使用量、従量課金 |
| RLS設定 | Row Level Security |

### M4: Token Vault

| タスク | 成果物 |
|--------|--------|
| DBスキーマ | mcpist.oauth_tokens |
| Vault連携 | Supabase Vault暗号化 |
| トークンリフレッシュ | 自動更新ロジック |
| OAuth フロー | 各プロバイダ対応 |

### M5: API Gateway

| タスク | 成果物 |
|--------|--------|
| Worker実装 | JWT検証、Rate Limit、Burst制限 |
| Load Balancer | Cloudflare LB設定 |
| オリジン保護 | Gateway Secret検証 |

### M6: User Console

| タスク | 成果物 |
|--------|--------|
| 認証画面 | ログイン/ログアウト |
| ダッシュボード | 使用量表示 |
| OAuth連携画面 | 外部サービス連携 |
| モジュール設定 | 有効/無効切替 |

### M7: デプロイ・監視

| タスク | 成果物 |
|--------|--------|
| Koyeb/Fly.ioデプロイ | MCP Server本番環境 |
| Vercelデプロイ | User Console本番環境 |
| Cloudflareデプロイ | API Gateway本番環境 |
| Grafana Cloud | ダッシュボード、アラート |

### M8: テスト・リリース

| タスク | 成果物 |
|--------|--------|
| 結合テスト | 主要フロー検証 |
| E2Eテスト | Playwright |
| セキュリティテスト | 認証・認可検証 |
| 招待ユーザーリリース | Phase 1完了 |

---

## 技術負債管理

### 許容する技術負債（Phase 1）

| 項目 | 理由 | 解消時期 |
|------|------|----------|
| ステージング環境なし | コスト削減 | Phase 3 |
| Rate Limitメモリ管理 | Redis不要、シンプル | 1,000 MAU超過時 |
| 単一DB（Supabase） | コスト削減 | スケール必要時 |
| 手動バックアップ | 自動化コスト | Phase 2 |

### 解消すべき技術負債

| 項目 | 優先度 | 対応 |
|------|--------|------|
| テストカバレッジ不足 | 高 | CI必須化 |
| ドキュメント不整合 | 中 | リリース前レビュー |
| 未使用コード | 低 | 定期クリーンアップ |

---

## リスク管理

### 技術リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| 外部API仕様変更 | モジュール動作不良 | バージョン監視、アダプター層 |
| Supabase障害 | 全機能停止 | 復旧待ち、ステータスページ監視 |
| 無料枠超過 | コスト発生 | 使用量監視、アラート設定 |

### 運用リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| トークン漏洩 | セキュリティインシデント | Vault暗号化、アクセスログ監視 |
| 単一開発者 | バスファクター1 | ドキュメント整備、AIアシスト |

---

## 開発ツール

### 必須

| ツール | 用途 |
|--------|------|
| Go 1.21+ | MCP Server開発 |
| Node.js 20+ | User Console、Worker開発 |
| pnpm | パッケージ管理 |
| Docker | ローカル開発環境 |
| Git | バージョン管理 |

### 推奨

| ツール | 用途 |
|--------|------|
| VSCode | IDE |
| Claude Code | AIアシスト開発 |
| Supabase CLI | ローカルDB、マイグレーション |
| Air | Go Hot Reload |
| Turborepo | モノレポビルド |

---

## ローカル開発環境

### 構成概要

```
┌─────────────────────────────────────────────────────────────┐
│                    ローカルPC (Windows/Mac/Linux)            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Docker Compose                          │   │
│  │  ┌─────────────────┐  ┌─────────────────┐          │   │
│  │  │ Console (Next.js)│  │ Server (Go)     │          │   │
│  │  │ :3000            │  │ :8089           │          │   │
│  │  └─────────────────┘  └─────────────────┘          │   │
│  └─────────────────────────────────────────────────────┘   │
│                              │                              │
│                              ▼                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         Supabase CLI (supabase start)               │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐           │   │
│  │  │ API      │ │ Auth     │ │ Storage  │           │   │
│  │  │ :54321   │ │          │ │          │           │   │
│  │  └──────────┘ └──────────┘ └──────────┘           │   │
│  │  ┌──────────┐ ┌──────────┐                        │   │
│  │  │ DB       │ │ Studio   │                        │   │
│  │  │ :54322   │ │ :54323   │                        │   │
│  │  └──────────┘ └──────────┘                        │   │
│  │         (内部でDockerコンテナを起動)                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         Wrangler CLI (将来: Worker開発用)            │   │
│  │         (ローカルバイナリ、内部でDocker不使用)        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 起動方法

```bash
# 1. 依存関係インストール（初回のみ、Supabase CLIも自動インストール）
pnpm install

# 2. 開発サーバー起動（Supabase → Docker Compose）
pnpm dev

# 3. 停止
pnpm stop
```

### サービス一覧

| サービス | ポート | 実行環境 | 説明 |
|----------|--------|----------|------|
| Console (Next.js) | 3000 | Docker | ユーザー管理画面 |
| Server (Go) | 8089 | Docker | MCP Server |
| Supabase API | 54321 | Supabase CLI (Docker内部) | REST/GraphQL API |
| Supabase DB | 54322 | Supabase CLI (Docker内部) | PostgreSQL |
| Supabase Studio | 54323 | Supabase CLI (Docker内部) | DB管理UI |
| Worker | (未定) | Wrangler CLI | API Gateway（将来） |

### 環境変数

```bash
# .env（ルートディレクトリ）
SUPABASE_ANON_KEY=<supabase start出力から>
SUPABASE_SERVICE_ROLE_KEY=<supabase start出力から>
INTERNAL_SERVICE_KEY=dev-internal-key
```

### DBリセット

```bash
# ローカル: マイグレーション全適用し直し
cd apps/supabase && supabase db reset

# リモート: マイグレーション削除後、push
# （開発初期のみ。本番リリース後は差分マイグレーション方式へ移行）
```

---

## 環境設定・シークレット管理

### 環境別URL一覧

| 変数 | ローカル | 本番 |
|------|----------|------|
| `SUPABASE_URL` | `http://localhost:54321` | `https://xstfrjvgpqxvyuochtss.supabase.co` |
| `NEXT_PUBLIC_APP_URL` | `http://localhost:3000` | `https://console.mcpist.app` |
| `SERVER_URL` | `http://localhost:8089` | `https://mcp.mcpist.app` |
| `WORKER_URL` | - | `https://api.mcpist.app` |

### シークレット管理方針

- **ローカル**: `.env`ファイル（gitignore）
- **本番**: 各デプロイ先のCLIでシークレット登録
- **CI/CD**: GitHub Secretsに登録し、デプロイ時に各サービスへ同期

### デプロイ先CLI一覧

| サービス | デプロイ先 | CLI | シークレット登録 |
|----------|------------|-----|------------------|
| Console | Vercel | `vercel` | `vercel env add` |
| Server | Koyeb | `koyeb` | `koyeb secrets create` |
| Worker | Cloudflare | `wrangler` | `wrangler secret put` |
| DB | Supabase | `supabase` | `supabase secrets set` |

### GitHub Secrets（CI/CD用）

```
# 共通
SUPABASE_PROJECT_REF=xstfrjvgpqxvyuochtss
SUPABASE_ACCESS_TOKEN=<Supabase Access Token>
SUPABASE_DB_PASSWORD=<DB Password>
SUPABASE_SERVICE_ROLE_KEY=<Service Role Key>
INTERNAL_SERVICE_KEY=<Internal Key>

# Vercel
VERCEL_TOKEN=<Vercel Token>
VERCEL_ORG_ID=<Org ID>
VERCEL_PROJECT_ID=<Project ID>

# Koyeb
KOYEB_TOKEN=<Koyeb Token>

# Cloudflare
CLOUDFLARE_API_TOKEN=<Cloudflare Token>
CLOUDFLARE_ACCOUNT_ID=<Account ID>
```

### デプロイスクリプト

```bash
# scripts/deploy-secrets.sh
# 本番環境へのシークレット一括登録

# 使用方法:
#   export SUPABASE_SERVICE_ROLE_KEY=xxx
#   export INTERNAL_SERVICE_KEY=xxx
#   ./scripts/deploy-secrets.sh
```

---

## コミュニケーション

### ドキュメント管理

| 種別 | 保存場所 |
|------|----------|
| 仕様書 | `dev/DAY*/spc-*.md` |
| 設計書 | `dev/DAY*/dsn-*.md` |
| ADR | `docs/adr/` |
| ポストモーテム | `docs/postmortems/` |

### 意思決定

| 種別 | 方法 |
|------|------|
| 技術選定 | ADR（Architecture Decision Record） |
| 仕様変更 | 仕様書更新 + Git履歴 |
| 障害対応 | ポストモーテム |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-dsn.md](./spc-dsn.md) | 設計仕様書 |
| [spc-tst.md](./spc-tst.md) | テスト仕様書 |
| [spc-ops.md](./spc-ops.md) | 運用仕様書 |
