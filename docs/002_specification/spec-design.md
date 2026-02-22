# MCPist 設計仕様書（spc-dsn）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | Design Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist の各コンポーネントの技術スタックとリポジトリ構成を定義する。

---

## 技術スタック概要

| カテゴリ | 技術 | 用途 |
|---|---|---|
| サーバー言語 | Go 1.24 | Server |
| Edge Runtime | Cloudflare Workers (TypeScript) | Worker (API Gateway) |
| フロントエンド | Next.js 16 / React 19 | Console |
| データベース | PostgreSQL 17 (Neon) | Database |
| ORM | GORM | Server → Database アクセス |
| 認証 | Clerk | ユーザー認証、JWT 発行 |

---

## コンポーネント別技術スタック

### Server (`apps/server/`)

| 項目          | 技術                                   | 備考                       |
| ----------- | ------------------------------------ | ------------------------ |
| 言語          | Go 1.24                              |                          |
| HTTP ルーティング | 標準ライブラリ (net/http 1.22 method-aware) | 外部フレームワークなし              |
| ORM         | GORM                                 | PostgreSQL ドライバ (pgx)    |
| API コード生成   | ogen v1.18                           | OpenAPI 3.0 → Go サーバーコード |
| JWT         | golang-jwt/jwt/v5                    | Ed25519 署名               |
| 暗号化         | 標準ライブラリ (crypto/aes, crypto/cipher)  | AES-256-GCM              |
| デプロイ先       | Render                               | Go ランタイムビルド              |

### Worker (`apps/worker/`)

| 項目 | 技術 | 備考 |
|---|---|---|
| ランタイム | Cloudflare Workers | |
| フレームワーク | Hono v4 | 軽量 Web フレームワーク |
| JWT | jose v5 | 署名・検証 |
| 言語 | TypeScript 5 | |
| ツール | Wrangler v4 | デプロイ・開発 |
| デプロイ先 | Cloudflare | |

### Console (`apps/console/`)

| 項目 | 技術 | 備考 |
|---|---|---|
| フレームワーク | Next.js 16 (App Router) | |
| 言語 | TypeScript | |
| UI | React 19 + Tailwind CSS v4 + shadcn/ui | |
| 認証 | @clerk/nextjs v6 | |
| API クライアント | openapi-fetch | OpenAPI スキーマから型生成 |
| デプロイ先 | Vercel | |

### Database

| 項目 | 技術 | 備考 |
|---|---|---|
| DBMS | PostgreSQL 17 | |
| ホスティング | Neon (本番) / Docker (ローカル) | |
| スキーマ | `mcpist` | |
| マイグレーション | SQL ファイル (`database/migrations/`) | |
| 暗号化 | AES-256-GCM (アプリケーション層) | 資格情報・シークレット |

---

## 依存ライブラリ

### Server (Go)

| ライブラリ                        | 用途                         |
| ---------------------------- | -------------------------- |
| gorm.io/gorm                 | ORM                        |
| gorm.io/driver/postgres      | PostgreSQL ドライバ            |
| github.com/ogen-go/ogen      | OpenAPI コード生成 (jx, otel 等を transitive に含む) |
| github.com/golang-jwt/jwt/v5 | JWT                        |

### Worker (TypeScript)

| ライブラリ | 用途 |
|---|---|
| hono | Web フレームワーク |
| jose | JWT 署名・検証 |

### Console (TypeScript)

| ライブラリ | 用途 |
|---|---|
| @clerk/nextjs | 認証 |
| openapi-fetch | 型安全 API クライアント |
| @radix-ui/* | UI プリミティブ |
| tailwindcss | スタイリング |
| lucide-react | アイコン |
| sonner | Toast 通知 |
| next-themes | テーマ管理 |
| cmdk | コマンドパレット |
| class-variance-authority | コンポーネントバリアント |

---

## リポジトリ構成（モノレポ）

```
mcpist/
├── apps/
│   ├── server/               # Server (Go)
│   │   ├── cmd/server/       # エントリーポイント
│   │   ├── api/openapi/      # OpenAPI 仕様 (ogen 入力)
│   │   ├── internal/
│   │   │   ├── auth/         # Ed25519 鍵管理、Gateway JWT 検証
│   │   │   ├── broker/       # ユーザーコンテキスト、トークンリフレッシュ
│   │   │   ├── db/           # GORM モデル、リポジトリ、暗号化
│   │   │   ├── mcp/          # MCP プロトコルハンドラ
│   │   │   ├── middleware/   # 認可、レート制限、トランスポート、リカバリ
│   │   │   ├── modules/      # モジュール実装
│   │   │   ├── observability/ # Loki ログ送信
│   │   │   ├── ogenserver/   # ogen 生成 REST ハンドラ
│   │   │   └── jsonrpc/      # JSON-RPC 2.0 型定義
│   │   ├── pkg/              # ogen 生成 API クライアント (per module)
│   │   └── go.mod
│   │
│   ├── console/              # Console (Next.js)
│   │   ├── src/
│   │   │   ├── app/          # App Router (pages)
│   │   │   ├── components/   # React コンポーネント
│   │   │   ├── hooks/        # カスタムフック
│   │   │   └── lib/          # ユーティリティ (auth, billing, worker client)
│   │   └── package.json
│   │
│   └── worker/               # Worker (Cloudflare Workers)
│       ├── src/
│       │   ├── index.ts      # エントリーポイント (Hono app)
│       │   ├── auth.ts       # 認証 (Clerk JWT, API Key)
│       │   ├── gateway-token.ts # Gateway JWT 発行
│       │   ├── v1/           # v1 ルーティング
│       │   ├── health.ts     # ヘルスチェック
│       │   ├── logging.ts    # リクエストログ
│       │   └── observability.ts # Loki 送信
│       ├── wrangler.toml
│       └── package.json
│
├── database/
│   ├── migrations/           # SQL マイグレーション
│   └── seed.sql              # シードデータ (未使用、削除予定)
│
├── docs/                     # ドキュメント
│
├── .github/workflows/        # CI/CD
│
├── docker-compose.yml        # ローカル開発用 DB
├── pnpm-workspace.yaml       # pnpm workspace
└── turbo.json                # Turborepo
```

**モノレポ管理:**

| ツール | 用途 |
|---|---|
| pnpm workspace | パッケージ管理 (Console, Worker) |
| Turborepo | ビルドキャッシュ、タスク並列実行 |

---

## 開発環境

| ツール | 用途 |
|---|---|
| Docker Compose | ローカル PostgreSQL 17 (port 57432) |
| Air | Go ホットリロード |
| pnpm | Node.js パッケージ管理 |
| Wrangler | Worker ローカル開発 |

---

## CI/CD

| ツール | 用途 |
|---|---|
| GitHub Actions | CI パイプライン |
| golangci-lint | Go リンター |
| ESLint | TypeScript リンター |

**CI ジョブ:**

| ジョブ | 内容 |
|---|---|
| lint-server | Go lint (golangci-lint) |
| test-server | Go test (`go test -v -race ./...`) |
| build-server | Go build |
| lint-console | Next.js lint |
| build-console | Next.js build |
| lint-worker | TypeScript チェック |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|---|---|
| [spc-sys.md](spec-systems.md) | システム仕様書 (コンポーネント責務) |
| [spc-inf.md](spec-infrastructure.md) | インフラ仕様書 |
| [spc-tbl.md](spec-tables.md) | テーブル仕様書 |
