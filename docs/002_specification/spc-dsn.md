# MCPist 設計仕様書（spc-dsn）

## ドキュメント管理情報

| 項目      | 値                    |
| ------- | -------------------- |
| Status  | `reviewed`           |
| Version | v2.0 (Sprint-006)    |
| Note    | Design Specification |

---

## 概要

本ドキュメントは、MCPistの各コンポーネントを実装する技術スタックを定義する。

---

## 技術スタック概要

| カテゴリ         | 技術                     | 用途              |
| ------------ | ---------------------- | --------------- |
| サーバー言語       | Go 1.23+               | MCP Server      |
| コンテナ         | Docker                 | デプロイ            |
| フロントエンド      | Next.js 15+ / React 19 | User Console    |
| Edge Runtime | Cloudflare Workers     | API Gateway     |
| データベース       | PostgreSQL 15+         | Supabase        |
| 認証           | Supabase Auth          | OAuth 2.1 / JWT |

---

## コンポーネント別技術スタック

### MCP Server（SRV）

| 項目 | 技術 | 備考 |
|------|------|------|
| 言語 | Go 1.23+ | 並行処理、メモリ効率 |
| フレームワーク | 標準ライブラリ (net/http) | 外部依存ゼロ |
| コンテナ | Docker (Alpine) | マルチステージビルド |
| デプロイ先 | Render (Primary) / Koyeb (Secondary) | |

**SRV内部コンポーネント:**

| コンポーネント | 実装方式 |
|---------------|----------|
| Authorization Middleware (AMW) | Go middleware (`middleware/authz.go`) |
| MCP Handler (HDL) | Go handler (`mcp/handler.go`)。プリミティブへのルーティングを担当 |
| Modules (MOD) | Go modules (`modules/`)。各モジュールが `Module` interface を実装 |
| Store (STR) | Go store (`store/`)。Supabase REST API + RPC 経由のデータアクセス |
| Observability (OBS) | Go observability (`observability/loki.go`)。Grafana Loki への構造化ログ送信 |

**依存ライブラリ:**

Go 標準ライブラリのみで実装。外部モジュール依存なし（`net/http`, `crypto`, `encoding/json` 等）。

全ての外部サービス（Supabase, Loki, 各モジュールAPI）は HTTP REST で通信する。

---

### API Gateway（Cloudflare Worker）

| 項目 | 技術 | 備考 |
|------|------|------|
| 言語 | TypeScript | Cloudflare Workers |
| ランタイム | Cloudflare Workers | Edge |
| ツール | Wrangler | デプロイ |
| ストレージ | Cloudflare KV | APIキー検証キャッシュ |

**責務:**
- JWT検証（Supabase userinfo / Auth API / JWKS の3段構え）
- APIキー検証（SHA-256 → KVキャッシュ → Supabase RPC fallback）
- X-User-ID / X-Auth-Type / X-Gateway-Secret ヘッダ付与
- X-Request-ID 発行（`crypto.randomUUID()`）
- Backend フェイルオーバー（Primary → Secondary）
- OAuth Discovery メタデータ提供（RFC 9728 / 8414）
- ヘルスチェック（`/health` + cron 5分間隔）
- CORS ハンドリング

---

### User Console（CON）

| 項目 | 技術 | 備考 |
|------|------|------|
| フレームワーク | Next.js 15+ | App Router |
| 言語 | TypeScript | |
| UI | React 19 + Tailwind CSS | |
| デプロイ先 | Vercel | Edge Functions対応 |
| 認証 | Supabase Auth | @supabase/ssr |

**主要ライブラリ:**

| ライブラリ | 用途 |
|-----------|------|
| @supabase/supabase-js | Supabase SDK |
| @supabase/ssr | SSR対応認証 |
| @radix-ui/* | UIコンポーネント（Dialog, Switch, Tabs 等） |
| jose | JWT検証 |
| lucide-react | アイコン |
| sonner | Toast通知 |
| next-themes | テーマ管理 |
| tailwindcss | スタイリング |

---

### Data Store（Supabase PostgreSQL）

| 項目 | 技術 | 備考 |
|------|------|------|
| データベース | PostgreSQL 15+ | Supabase |
| スキーマ | mcpist | 専用スキーマ |
| アクセス制御 | RLS (Row Level Security) | |
| 暗号化 | Supabase Vault | OAuthトークン・APIシークレット |
| マイグレーション | Supabase CLI | |

**アクセス方式:**

| クライアント | 接続方式 |
|-------------|----------|
| SRV (Go) | Supabase REST API + RPC |
| CON (Next.js) | @supabase/supabase-js |

ユーザーごとのアクセス権限（Authentication ）は各コンポーネントの責務として実装する。

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
- OAuth（Google, Apple, Microsoft, GitHub）
- MFA（TOTP）

---

## 開発環境

### ローカル開発

| ツール | 用途 |
|--------|------|
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
| mainマージ | Deploy (Render, Koyeb, Vercel, Cloudflare) |

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
│   │   │   ├── server/       # エントリーポイント
│   │   │   └── tools-export/ # tools.json 生成CLI
│   │   ├── internal/
│   │   │   ├── mcp/          # MCP Protocol Handler
│   │   │   ├── middleware/   # Authorization Middleware
│   │   │   ├── modules/     # Module interface + 各モジュール実装
│   │   │   ├── store/       # Data Store アクセス (user, token, module)
│   │   │   ├── observability/ # Loki ログ送信
│   │   │   └── httpclient/  # HTTP Client ユーティリティ
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── console/              # User Console (Next.js)
│   │   ├── src/
│   │   │   ├── app/          # App Router
│   │   │   ├── components/
│   │   │   └── lib/
│   │   └── package.json
│   │
│   └── worker/               # API Gateway (Cloudflare Worker)
│       ├── src/
│       │   └── index.ts
│       ├── wrangler.toml
│       └── package.json
│
├── supabase/                 # Supabase設定
│   ├── migrations/           # DBマイグレーション
│   ├── seed.sql
│   └── config.toml
│
├── docs/                     # VitePress ドキュメント
│
├── .github/
│   └── workflows/            # CI/CD
│
├── pnpm-workspace.yaml       # pnpm workspace設定
└── turbo.json                # Turborepo設定
```

**モノレポ管理:**

| ツール | 用途 |
|--------|------|
| pnpm workspace | パッケージ管理 |
| Turborepo | ビルドキャッシュ、タスク並列実行、CI時間短縮 |

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
