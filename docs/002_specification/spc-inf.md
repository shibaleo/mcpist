# MCPist インフラストラクチャ仕様書（spc-inf）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `approved` |
| Version | v2.0 (Sprint-003) |
| Note | Infrastructure Specification |

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

## コンポーネントとインフラのマッピング

### クライアント層（実装範囲外）

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| MCPクライアント (OAuth) | Claude.ai, ChatGPT等 | 実装範囲外 |
| MCPクライアント (APIキー) | Claude Code, Cursor等 | 実装範囲外 |

### ゲートウェイ層

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| APIゲートウェイ | 認証検証、プロキシ | **Cloudflare Worker** |
| APIキーキャッシュ | APIキー検証結果キャッシュ | **Cloudflare KV** |

### MCPサーバー層

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| Authミドルウェア | Gateway Secret検証 | **Render** (Primary) / **Koyeb** (Secondary) |
| MCPハンドラ | tools/list, tools/call | **Render** (Primary) / **Koyeb** (Secondary) |
| モジュールレジストリ | get_module_schema, call, batch | **Render** (Primary) / **Koyeb** (Secondary) |
| モジュール | Notion, GitHub, Jira等 | **Render** (Primary) / **Koyeb** (Secondary) |

### Auth + DB Backend層

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| Authサーバー | OAuth 2.1, JWT発行 | **Supabase Auth** |
| Sessionマネージャー | ユーザーID発行、ソーシャルログイン | **Supabase Auth** |
| Data Store | ユーザー情報、課金情報、設定 | **Supabase PostgreSQL** (mcpistスキーマ) |
| Token Vault | 外部サービストークン保存 | **Supabase PostgreSQL + Vault** |

### フロントエンド層

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| ユーザーコンソール | 管理画面、OAuth連携、設定 | **Vercel** |
| 静的アセット | CSS, JS, 画像 | **Vercel Edge Network** |

### 外部サービス（実装範囲外）

| コンポーネント | 説明 | インフラ配置 |
|--------------|------|-------------|
| 認証プロバイダ | ソーシャルログイン | Google, GitHub等 |
| 決済代行サービス | クレジットカード決済 | Stripe |
| 外部Authサーバー | 外部サービス認可 | Notion, Google等 |
| 外部サービスAPI | リソースアクセス | Notion API, Google Calendar API等 |

---

## インフラサービス一覧

| インフラサービス              | 用途                      | プラン            |
| --------------------- | ----------------------- | -------------- |
| **Cloudflare Worker** | APIゲートウェイ               | Free           |
| **Cloudflare KV**     | APIキーキャッシュ              | Free           |
| **Cloudflare DNS**    | ドメイン管理                  | Free           |
| **Render**            | MCPサーバー (Primary)       | Free (Starter) |
| **Koyeb**             | MCPサーバー (Secondary)     | Free (nano)    |
| **Supabase**          | Auth, PostgreSQL, Vault | Free           |
| **Vercel**            | Console, Edge Network   | Free (Hobby)   |
| **DockerHub**         | Dockerイメージホスト           | Free (Public)  |
| **GitHub**            | ソースコード、Actions          | Free           |
| **Grafana Cloud**     | ログ収集 (Loki)             | Free           |

---

## データストア配置

| データ種別 | テーブル/スキーマ | インフラ配置 |
|-----------|-----------------|-------------|
| ユーザー情報 | mcpist.users | Supabase PostgreSQL |
| サブスクリプション | mcpist.subscriptions | Supabase PostgreSQL |
| プラン | mcpist.plans | Supabase PostgreSQL |
| APIキー | mcpist.api_keys | Supabase PostgreSQL |
| OAuthトークン | mcpist.oauth_tokens | Supabase PostgreSQL |
| 暗号化シークレット | vault.secrets | Supabase Vault |
| 認証ユーザー | auth.users | Supabase Auth |

**スキーマ設計方針:**

| スキーマ | 用途 | 管理 |
|---------|------|------|
| auth | 共通認証基盤 | Supabase管理 |
| vault | 暗号化ストア | Supabase管理 |
| mcpist | MCPist関連テーブル | MCPist |
| public | RPCラッパー関数 | MCPist |

---

## アーキテクチャ図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCP Client                                      │
│                        (Claude.ai, ChatGPT, Claude Code, Cursor)             │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Cloudflare                                         │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                         Worker (APIゲートウェイ)                         ││
│  └───────────────────────────────────┬─────────────────────────────────────┘│
│                                      │                                       │
│                      ┌───────────────┴───────────────┐                      │
│                      │              KV               │                      │
│                      │     (APIキーキャッシュ)        │                      │
│                      └───────────────────────────────┘                      │
└──────────────────────────────────────┬──────────────────────────────────────┘
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
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                             Supabase                                         │
│                                                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │      Auth       │  │   PostgreSQL    │  │     Vault       │             │
│  │                 │  │                 │  │                 │             │
│  │  Session管理    │  │  mcpistスキーマ  │  │  暗号化保存     │             │
│  │  OAuth Server   │  │  publicスキーマ  │  │                 │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              ▼                        ▼                        ▼
┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐
│       Vercel        │   │       Stripe        │   │    External APIs    │
│                     │   │                     │   │                     │
│   User Console      │   │   決済代行          │   │  Notion, Google     │
│   (Next.js)         │   │   (実装範囲外)      │   │  Calendar, etc.     │
└─────────────────────┘   └─────────────────────┘   └─────────────────────┘
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
| DockerHub | Free | $0 |
| GitHub | Free | $0 |
| Grafana Cloud | Free | $0 |

**月額固定費: $0**

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書（コンポーネント定義） |
| [spc-dpl.md](./spc-dpl.md) | デプロイ仕様書（環境構成、CI/CD） |
| [spc-tbl.md](./spc-tbl.md) | テーブル仕様書 |
