# MCPist デプロイ仕様書（spc-dpl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `approved` |
| Version | v1.0 (Sprint-003) |
| Note | Deployment Specification |

---

## 概要

本ドキュメントは、MCPistの環境構成とデプロイ戦略を定義する。
実際のコマンドや手順は別途運用ドキュメントで管理する。

---

## 環境構成

### 3環境体制

| 環境 | Console | MCP API | 用途 |
|------|---------|---------|------|
| Dev | dev.mcpist.app | mcp.dev.mcpist.app | 開発・検証環境 |
| Stg | stg.mcpist.app | mcp.stg.mcpist.app | ステージング環境 |
| Prd | mcpist.app | mcp.mcpist.app | 本番環境 |

### 環境別アカウント

| 環境      | Supabase/Render/Koyeb/Vercel           | Cloudflare            |
| ------- | -------------------------------------- | --------------------- |
| Dev     | shiba.dog.leo.private                  | shiba.dog.leo.private |
| Stg/Prd | fukudamakoto.private/fukudamakoto.work | shiba.dog.leo.private |

**注意**: Cloudflareは全環境で同一アカウント（shiba）を使用。
**注意**: Stg と Prd はデプロイのたびに交換される（Blue-Green方式）。

### 環境別URL

| 環境 | Console | MCP API |
|------|---------|---------|
| Dev | https://dev.mcpist.app | https://mcp.dev.mcpist.app |
| Stg | https://stg.mcpist.app | https://mcp.stg.mcpist.app |
| Prd | https://mcpist.app | https://mcp.mcpist.app |

---

## ローカル開発環境

### 構成

| コンポーネント | 実行環境 |
|--------------|---------|
| MCP Server | Go Runtime |
| Console | Node.js |
| Worker | Node.js + Wrangler |
| Supabase | Supabase CLI（内部でDockerを使用） |

---

## CI/CD ツール

### 共通インフラ

| サービス | 用途 |
|---------|------|
| GitHub | ソースコード管理、ブランチ管理 |
| GitHub Actions | CI/CDパイプライン |
| DockerHub | Dockerイメージホスト |
| Cloudflare DNS | ドメイン管理（全環境共通） |

### 監視インフラ

| サービス | 対象環境 | 用途 |
|---------|---------|------|
| Grafana Loki (Dev) | Dev | ログ収集 |
| Grafana Loki (Prd) | Stg, Prd | ログ収集 |

---

## CIフロー

Dev, Stg, Prd 全環境で共通のフロー。

```
┌─────────────────┐
│     GitHub      │
│   (main push)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GitHub Actions  │
│      (CI)       │
└────────┬────────┘
         │
         ├─────────────────────────────────────┐
         │                                     │
         ▼                                     ▼
┌─────────────────┐                   ┌─────────────────┐
│   Build & Test  │                   │   Supabase      │
│                 │                   │   Migration     │
│  - Console      │                   │                 │
│  - Worker       │                   │                 │
│  - Server       │                   │                 │
└────────┬────────┘                   └─────────────────┘
         │
         ▼
┌─────────────────┐
│   DockerHub     │
│  (Server Image) │
└─────────────────┘
```

---

## デプロイフロー

### Console (Vercel)

```
GitHub (main) → Vercel (自動デプロイ)
```

### Worker (Cloudflare)

```
GitHub Actions → Cloudflare Workers
```

### Server (Render / Koyeb)

```
GitHub Actions → DockerHub → Render/Koyeb (イメージ pull)
```

### Database (Supabase)

```
GitHub Actions → Supabase (マイグレーション適用)
```

---

## サービス別デプロイトリガー

| コンポーネント | トリガー | デプロイ先 |
|--------------|---------|----------|
| Console | GitHub連携（自動） | Vercel |
| Worker | GitHub Actions | Cloudflare Workers |
| Server | GitHub Actions → DockerHub | Render, Koyeb |
| Database | GitHub Actions | Supabase |

---

## 環境変数

### Console (Vercel)

| 変数名 | 説明 |
|--------|------|
| `NEXT_PUBLIC_SUPABASE_URL` | Supabase URL |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Supabase Publishable Key |
| `SUPABASE_SERVICE_ROLE_KEY` | Supabase Service Role Key |

### Worker (Cloudflare)

| 変数名 | 説明 | 種別 |
|--------|------|------|
| `PRIMARY_API_URL` | Render URL | vars |
| `SECONDARY_API_URL` | Koyeb URL | vars |
| `SUPABASE_URL` | Supabase URL | vars |
| `SUPABASE_JWKS_URL` | JWKS URL | vars |
| `GATEWAY_SECRET` | Gateway認証シークレット | secret |
| `SUPABASE_PUBLISHABLE_KEY` | Supabase Publishable Key | secret |
| `INTERNAL_SECRET` | 内部API認証シークレット | secret |

### Server (Render / Koyeb)

| 変数名 | 説明 |
|--------|------|
| `SUPABASE_URL` | Supabase URL |
| `SUPABASE_SERVICE_ROLE_KEY` | Supabase Service Role Key |
| `GATEWAY_SECRET` | Gateway認証シークレット |
| `PORT` | サーバーポート |

---

## ブランチ戦略

| ブランチ     | 用途       | デプロイ先 |
| -------- | -------- | ----- |
| `main`   | 本番リリース   | Stg   |
| `dev`    | 本番リリース検証 | Stg   |
| `feat/*` | 機能開発など   | Dev   |
| `fix/*`  | バグ修正     | Dev   |

---

## デプロイ検証

### ヘルスチェック

| エンドポイント | 期待結果 |
|--------------|---------|
| `GET /health` | 200 OK |
| `GET /mcp` (認証なし) | 401 Unauthorized |
| `GET /mcp` (APIキー付き) | 200 OK |

### MCP接続テスト

| クライアント | 認証方式 |
|------------|---------|
| Claude.ai | OAuth 2.0 |
| ChatGPT | OAuth 2.0 |
| Claude Code | APIキー |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-inf.md](./spc-inf.md) | インフラ仕様書（コンポーネント配置） |
| [spc-sys.md](./spc-sys.md) | システム仕様書（コンポーネント定義） |
