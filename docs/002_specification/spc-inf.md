# MCPist インフラストラクチャ仕様書（spc-inf）

## ドキュメント管理情報

| 項目      | 値                            |
| ------- | ---------------------------- |
| Status  | `reviewed`                   |
| Version | v3.0 (Sprint-006)            |
| Note    | Infrastructure Specification |

---

## 概要

本ドキュメントは、MCPistのコンポーネントとインフラストラクチャの対応関係を定義する。
各コンポーネントがどのインフラサービスに配置されるかを明確化する。

**設計原則:**
- 放置運用（平日昼間に対応不可）
- ベンダー分散（単一障害点の排除）
- 自動フェイルオーバー
- 無料枠活用

---

## レイヤー別インフラ構成

### Consumer Layer（実装範囲外）

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| MCPクライアント (OAuth) | Claude.ai, ChatGPT等 | 実装範囲外 |
| MCPクライアント (APIキー) | Claude Code, Cursor等 | 実装範囲外 |

### Edge Service Layer

| コンポーネント         | 説明                      | インフラ配置                                       |
| --------------- | ----------------------- | -------------------------------------------- |
| API Gateway     | 認証検証、フェイルオーバー、プロキシ      | **Cloudflare Worker**                        |
| API Key Cache   | APIキー検証結果キャッシュ          | **Cloudflare KV**                            |
| DNS             | ドメイン管理                  | **Cloudflare DNS**                           |
| OAuth Discovery | RFC 9728 / 8414 メタデータ提供 | **Cloudflare Worker**                        |
| Observability   | 構造化ログ送信                 | Cloudflare Worker → **Grafana Cloud (Loki)** |

### Compute Layer

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| MCP Server (Primary) | Go + Docker Container | **Render** |
| MCP Server (Secondary) | フェイルオーバー先、Primary と同一構成 | **Koyeb** |

### Backend Platform Layer

| コンポーネント         | 説明                     | インフラ配置                                |
| --------------- | ---------------------- | ------------------------------------- |
| OAuth Server    | OAuth 2.1, JWT発行       | **Supabase Auth**                     |
| Session Manager | ユーザーID発行、ソーシャルログイン     | **Supabase Auth**                     |
| Data Store      | ユーザー情報、クレジット、設定、トークン   | **Supabase PostgreSQL** (mcpist スキーマ) |
| Token Vault     | 外部サービストークン・APIシークレット保存 | **Supabase PostgreSQL + Vault**       |

### UI Layer

| コンポーネント      | 説明                 | インフラ配置                  |
| ------------ | ------------------ | ----------------------- |
| User Console | 管理画面、OAuth連携、ツール設定 | **Vercel**              |
| 静的アセット       | CSS, JS, 画像        | **Vercel Edge Network** |

### Delivery Layer

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| Source Code / CI/CD | ソースコード管理、Lint、Test、Build、Deploy | **GitHub** (Actions) |
| Container Registry | Docker イメージ配布 | **DockerHub** |

### Observability Layer

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| Log Store | 構造化ログ、セキュリティイベント、X-Request-ID トレース | **Grafana Cloud (Loki)** |

### External Integration Layer（実装範囲外）

| コンポーネント                  | 説明              | インフラ配置                           |
| ------------------------ | --------------- | -------------------------------- |
| Payment Service Provider | 決済代行、Checkout / Webhook | **Stripe** |
| Identity Provider        | ソーシャルログイン       | Google, Apple, Microsoft, GitHub  |
| External Auth Server     | 外部サービス認可（OAuth） | Notion, Google, Microsoft等       |
| External Service API     | リソースアクセス        | Notion API, Google Calendar API等 |

---

## インフラサービス一覧

| インフラサービス | 用途 | プラン |
|-----------------|------|--------|
| **Cloudflare Worker** | APIゲートウェイ、OAuth Discovery | Free |
| **Cloudflare KV** | APIキーキャッシュ | Free |
| **Cloudflare DNS** | ドメイン管理 | Free |
| **Render** | MCPサーバー (Primary) | Free (Starter) |
| **Koyeb** | MCPサーバー (Secondary) | Free (nano) |
| **Supabase** | Auth, PostgreSQL, Vault | Free |
| **Vercel** | Console, Edge Network | Free (Hobby) |
| **GitHub** | ソースコード、Actions | Free |
| **DockerHub** | Docker イメージ配布 | Free |
| **Grafana Cloud** | ログ収集 (Loki) | Free |
| **Stripe** | 決済代行 | 決済手数料のみ |

---

## データストア配置

| スキーマ | 用途 | インフラ配置 | 管理 |
|---------|------|-------------|------|
| auth | 共通認証基盤 | Supabase Auth | Supabase管理 |
| vault | 暗号化ストア | Supabase Vault | Supabase管理 |
| mcpist | MCPist業務データ（ユーザー、クレジット、トークン、設定等） | Supabase PostgreSQL | MCPist |
| public | RPCラッパー関数 | Supabase PostgreSQL | MCPist |
MCPistのビジネスデータはmcpistスキーマに保存し、スキーマ公開しない。データアクセスはpublicに配置されたRPC関数経由で行う。
個別テーブルの定義は [spc-tbl.md](./spc-tbl.md) を参照。

---

## インフラ構成図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Consumer Layer (Out of Scope)                       │
│                    (Claude.ai, ChatGPT, Claude Code, Cursor)               │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Edge Service Layer                                  │
│                                                                             │
│  ┌──────────────────────┐  ┌───────────────┐  ┌────────────────────────┐   │
│  │  Worker (API Gateway) │  │ Cloudflare KV │  │    Cloudflare DNS     │   │
│  │  JWT/APIキー検証       │  │ APIキーキャッシュ │  │    ドメイン管理        │   │
│  │  OAuth Discovery      │  │               │  │                        │   │
│  │  フェイルオーバー       │  │               │  │                        │   │
│  └──────────────────────┘  └───────────────┘  └────────────────────────┘   │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │
                   ┌───────────────┴───────────────┐
                   ▼                               ▼
          ┌─────────────────┐             ┌─────────────────┐
          │     Render      │             │     Koyeb       │
          │   (Primary)     │             │   (Secondary)   │
          │                 │             │                 │
          │   MCP Server    │             │   MCP Server    │
          │   (Docker)      │             │   (Docker)      │
          └────────┬────────┘             └────────┬────────┘
                   └───────────────┬───────────────┘
                                   │         Compute Layer
                   ┌───────────────┼───────────────┐
                   ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Backend Platform Layer                                │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
│  │  Supabase Auth  │  │   PostgreSQL    │  │  Supabase Vault │            │
│  │                 │  │                 │  │                 │            │
│  │  OAuth Server   │  │  mcpistスキーマ  │  │  暗号化保存     │            │
│  │  Session管理    │  │  publicスキーマ  │  │                 │            │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐                           ┌────────────────────────────┐
│    UI Layer      │                           │    Delivery Layer          │
│                  │                           │                            │
│  Vercel          │                           │  GitHub → DockerHub        │
│  (Next.js)       │                           │  (CI/CD)  (Docker Image)   │
└──────────────────┘                           └────────────────────────────┘

┌────────────────────────────────────────────────┐
│              Observability Layer                │
│                                                │
│  Grafana Cloud (Loki)                          │
│  ← Edge Service Layer / Compute Layer から送信  │
└────────────────────────────────────────────────┘
```

---

## コスト

| サービス | プラン | 月額コスト |
|----------|--------|-----------|
| Cloudflare | Free | $0 |
| Render | Free (Starter) | $0 |
| Koyeb | Free (nano) | $0 |
| Supabase | Free | $0 |
| Vercel | Free (Hobby) | $0 |
| GitHub | Free | $0 |
| DockerHub | Free | $0 |
| Grafana Cloud | Free | $0 |
| Stripe | 決済手数料のみ | 決済額の3.6% |

**月額固定費: $0**（Stripe は月額料金なし。決済発生時に手数料3.6%が差し引かれる）

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書（コンポーネント定義） |
| [spc-dpl.md](./spc-dpl.md) | デプロイ仕様書（環境構成、CI/CD） |
| [spc-tbl.md](./spc-tbl.md) | テーブル仕様書 |
