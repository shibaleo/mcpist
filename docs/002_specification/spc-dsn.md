# MCPist 設計仕様書（spc-dsn）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Design Specification |

---

## 概要

本ドキュメントは、MCPistの各コンポーネントを実装する技術スタックを定義する。

---

## 技術スタック概要

| カテゴリ | 技術 | 用途 |
|---------|------|------|
| サーバー言語 | Go 1.21+ | MCP Server |
| コンテナ | Docker | デプロイ |
| フロントエンド | Next.js 14+ / React | User Console |
| Edge Runtime | Cloudflare Workers | API Gateway |
| データベース | PostgreSQL 15+ | Supabase |
| 認証 | Supabase Auth | OAuth 2.1 / JWT |

---

## コンポーネント別技術スタック

### MCP Server（SRV）

| 項目 | 技術 | 備考 |
|------|------|------|
| 言語 | Go 1.21+ | 並行処理、メモリ効率 |
| フレームワーク | 標準ライブラリ (net/http) | シンプル、依存最小化 |
| コンテナ | Docker (Alpine) | マルチステージビルド |
| デプロイ先 | Koyeb (Primary) / Fly.io (Standby) | |

**SRV内部コンポーネント:**

| コンポーネント | 実装方式 |
|---------------|----------|
| Auth Middleware (AMW) | Go middleware |
| MCP Handler (HDL) | Go handler |
| Module Registry (REG) | Go struct + interface |
| Modules (MOD) | Go modules (internal/modules/) |

**主要ライブラリ:**

| ライブラリ | 用途 |
|-----------|------|
| net/http (標準ライブラリ) | HTTPサーバー・ルーター |
| github.com/golang-jwt/jwt/v5 | JWT検証 |
| github.com/supabase-community/supabase-go | Supabase SDK |
| github.com/prometheus/client_golang | メトリクス |

---

### API Gateway（Cloudflare Worker）

| 項目 | 技術 | 備考 |
|------|------|------|
| 言語 | TypeScript | Cloudflare Workers |
| ランタイム | Cloudflare Workers | Edge |
| ツール | Wrangler | デプロイ |
| ストレージ | Cloudflare KV | Rate Limitカウンター |

**責務:**
- JWT署名検証
- グローバルRate Limit（IP単位）
- Burst制限（ユーザー単位）
- X-User-ID付与

---

### User Console（CON）

| 項目 | 技術 | 備考 |
|------|------|------|
| フレームワーク | Next.js 14+ | App Router |
| 言語 | TypeScript | |
| UI | React + Tailwind CSS | |
| デプロイ先 | Vercel | Edge Functions対応 |
| 認証 | Supabase Auth | @supabase/ssr |

**主要ライブラリ:**

| ライブラリ | 用途 |
|-----------|------|
| @supabase/supabase-js | Supabase SDK |
| @supabase/ssr | SSR対応認証 |
| @stripe/stripe-js | Stripe決済 |
| tailwindcss | スタイリング |

---

### Entitlement Store / Token Vault（ENT / TVL）

| 項目 | 技術 | 備考 |
|------|------|------|
| データベース | PostgreSQL 15+ | Supabase |
| スキーマ | mcpist | 専用スキーマ |
| アクセス制御 | RLS (Row Level Security) | |
| 暗号化 | Supabase Vault | OAuthトークン |
| マイグレーション | Supabase CLI | |

**アクセス方式:**

| クライアント | 接続方式 |
|-------------|----------|
| SRV (Go) | supabase-go SDK / 直接PostgreSQL |
| CON (Next.js) | @supabase/supabase-js |

---

### Auth Server（AUS）

| 項目 | 技術 | 備考 |
|------|------|------|
| サービス | Supabase Auth | マネージド |
| プロトコル | OAuth 2.1 | |
| トークン | JWT (RS256) | |
| JWKS | Supabase公開エンドポイント | |

**対応認証方式:**
- Email/Password
- Magic Link
- OAuth（Google, GitHub等）
- MFA（TOTP）

---

## 開発環境

### ローカル開発

| ツール | 用途 |
|--------|------|
| Docker Compose | ローカル環境構築 |
| Supabase CLI | ローカルDB、Auth |
| Air | Go Hot Reload |
| pnpm | Node.js パッケージ管理 |

### CI/CD

| ツール | 用途 |
|--------|------|
| GitHub Actions | CI/CD |
| golangci-lint | Go Linter |
| ESLint + Prettier | TypeScript Linter/Formatter |

### ブランチ戦略

| ブランチ | 用途 | 保護 |
|---------|------|------|
| main | 本番リリース | Protected（PRマージのみ） |
| dev | 開発統合 | Protected（PRマージのみ） |
| feature/* | 機能開発 | - |
| fix/* | バグ修正 | - |
| docs/* | ドキュメント | - |

**フロー:**
```
feature/xxx → PR → dev → PR → main → 本番デプロイ
```

**CIトリガー:**
| イベント | 実行内容 |
|---------|----------|
| PR作成/更新 | Lint, Test, Build |
| devマージ | 統合テスト（将来ステージング環境追加時に自動デプロイ） |
| mainマージ | Deploy (Koyeb, Fly.io, Vercel, Cloudflare) |

**備考:**
- 機能開発は feature/* → dev にマージ
- devで統合確認後、dev → main にPRで本番リリース
- リリースタグ（v1.0.0等）は mainから作成
- hotfixは fix/* → main で直接対応可

---

## リポジトリ構成（モノレポ）

```
mcpist/
├── apps/
│   ├── server/               # MCP Server (Go)
│   │   ├── cmd/
│   │   │   └── server/       # エントリーポイント
│   │   ├── internal/
│   │   │   ├── mcp/          # MCP Protocol Handler
│   │   │   ├── modules/      # Module Registry + Modules
│   │   │   │   ├── registry.go
│   │   │   │   ├── notion/
│   │   │   │   ├── google_calendar/
│   │   │   │   └── microsoft_todo/
│   │   │   ├── auth/         # Auth Middleware
│   │   │   ├── entitlement/  # ENT アクセス
│   │   │   └── vault/        # TVL アクセス
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── console/              # User Console (Next.js)
│   │   ├── app/              # App Router
│   │   ├── components/
│   │   ├── lib/
│   │   │   └── supabase/
│   │   └── package.json
│   │
│   └── worker/               # API Gateway (Cloudflare Worker)
│       ├── src/
│       │   └── index.ts
│       ├── wrangler.toml
│       └── package.json
│
├── packages/                 # 共有パッケージ（将来）
│   └── shared-types/         # 共通型定義等
│
├── supabase/                 # Supabase設定
│   ├── migrations/           # DBマイグレーション
│   ├── functions/            # Edge Functions（必要に応じて）
│   └── config.toml
│
├── .github/
│   └── workflows/            # CI/CD
│
├── docker-compose.yml        # ローカル開発環境
├── pnpm-workspace.yaml       # pnpm workspace設定
└── turbo.json                # Turborepo設定（オプション）
```

**モノレポ管理:**

| ツール | 用途 |
|--------|------|
| pnpm workspace | パッケージ管理 |
| Turborepo | ビルドキャッシュ、タスク並列実行、CI時間短縮 |
| Docker Compose | ローカル統合環境 |

---

## バージョン管理

| コンポーネント | バージョニング |
|---------------|---------------|
| MCP Server | Git tag (v1.0.0) |
| API Gateway | Wrangler environments |
| User Console | Vercel deployments |
| DB Schema | Supabase migrations |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-inf.md](spc-inf.md) | インフラストラクチャ仕様書 |
| [spc-tbl.md](./spc-tbl.md) | テーブル仕様書 |
