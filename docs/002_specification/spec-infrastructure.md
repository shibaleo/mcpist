# MCPist インフラストラクチャ仕様書（spc-inf）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v4.0 (Sprint-012) |
| Note | Infrastructure Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist の各コンポーネントのインフラ配置を定義する。

**設計原則:**
- 放置運用（平日昼間に対応不可）
- 無料枠活用
- 自動デプロイ

---

## インフラ配置

| コンポーネント      | インフラ                 | プラン            |
| ------------ | -------------------- | -------------- |
| **Worker**   | Cloudflare Workers   | Free           |
| **Server**   | Render (Go Runtime)  | Free           |
| **Console**  | Vercel               | Free (Hobby)   |
| **Database** | Neon PostgreSQL      | Free           |
| **認証**       | Clerk                | Free           |
| **決済**       | Stripe               | 手数料のみ, サンドボックス |
| **DNS**      | Cloudflare DNS       | ドメイン費用のみ       |
| **ログ**       | Grafana Cloud (Loki) | Free           |
| **CI/CD**    | GitHub Actions       | Free           |

**月額固定費: $0**

---

## レイヤー構成

```
┌─────────────────────────────────────────────────────────────┐
│ Consumer Layer (実装範囲外)                                  │
│  MCP Client: Claude.ai, ChatGPT, Claude Code, Cursor 等     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Edge Layer                                                   │
│  ┌───────────────────────────┐  ┌────────────────────────┐  │
│  │ Worker (Cloudflare Workers)│ │ Cloudflare DNS         │  │
│  │  認証・プロキシ・OAuth      │  │                        │  │
│  └─────────────┬─────────────┘  └────────────────────────┘  │
└────────────────┼────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Compute Layer                                               │
│  ┌───────────────────────────┐                              │
│  │ Server (Render)           │                              │
│  │  MCP 処理・REST API        │                              │
│  └─────────────┬─────────────┘                              │
└────────────────┼────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Data Layer                                                  │
│  ┌───────────────────────────┐                              │
│  │ Database (Neon PostgreSQL)│                              │
│  │  mcpist スキーマ           │                              │
│  └───────────────────────────┘                              │
└─────────────────────────────────────────────────────────────┘

┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐
│ UI Layer         │  │ Auth Layer       │  │ Observability  │
│ Console (Vercel) │  │ Clerk            │  │ Grafana (Loki) │
└──────────────────┘  └──────────────────┘  └────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ External Layer (実装範囲外)                                  │
│  Stripe, Identity Provider (Google/Apple/MS/GitHub)         │
│  External Service API (Notion, GitHub, Jira, Google 等)     │
└─────────────────────────────────────────────────────────────┘
```

---

## データストア

| スキーマ | ホスティング | 用途 |
|---|---|---|
| `mcpist` | Neon PostgreSQL | 全業務データ (ユーザー、資格情報、設定、使用量ログ) |

ローカル開発では Docker Compose で PostgreSQL 17 を起動する (port 57432)。

**詳細は [spc-tbl.md](spec-tables.md) を参照。**

---

## 環境変数

### Server

| 変数 | 用途 |
|---|---|
| `DATABASE_URL` | Neon 接続文字列 |
| `CREDENTIAL_ENCRYPTION_KEY` | AES-256-GCM 暗号化キー (base64) |
| `API_KEY_PRIVATE_KEY` | API キー署名用 Ed25519 秘密鍵 (base64) |
| `WORKER_JWKS_URL` | Worker JWKS エンドポイント (Gateway JWT 検証用) |
| `ADMIN_EMAILS` | 管理者メールアドレス (カンマ区切り) |
| `GRAFANA_LOKI_URL` | Loki エンドポイント |
| `GRAFANA_LOKI_USER` | Loki ユーザー名 |
| `GRAFANA_LOKI_API_KEY` | Loki API キー |
| `STRIPE_WEBHOOK_SECRET` | Stripe 署名検証キー |
| `APP_ENV` | 環境名 |
| `INSTANCE_ID` | インスタンス識別子 |
| `INSTANCE_REGION` | リージョン |

### Worker

| 変数 | 用途 |
|---|---|
| `SERVER_URL` | Server のベース URL |
| `GATEWAY_SIGNING_KEY` | Gateway JWT 署名用 Ed25519 秘密鍵 (base64) |
| `CLERK_JWKS_URL` | Clerk JWKS エンドポイント |
| `SERVER_JWKS_URL` | Server JWKS エンドポイント |
| `GRAFANA_LOKI_URL` | Loki エンドポイント |
| `GRAFANA_LOKI_USER` | Loki ユーザー名 |
| `GRAFANA_LOKI_API_KEY` | Loki API キー |

### Console

| 変数 | 用途 |
|---|---|
| `NEXT_PUBLIC_CLERK_*` | Clerk 公開設定 |
| `NEXT_PUBLIC_OAUTH_SERVER_URL` | OAuth フロー用 Server URL |
| `STRIPE_SECRET_KEY` | Stripe サーバーサイドキー |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | Stripe 公開キー |
| `STRIPE_PLUS_PRICE_ID` | Plus プランの Stripe Price ID |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|---|---|
| [spc-sys.md](spec-systems.md) | システム仕様書 (コンポーネント責務) |
| [spc-dsn.md](spec-design.md) | 設計仕様書 (技術スタック) |
| [spc-tbl.md](spec-tables.md) | テーブル仕様書 |
