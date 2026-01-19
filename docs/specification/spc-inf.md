# MCPist インフラストラクチャ仕様書（spc-infra）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Infrastructure Specification |

---

## 概要

本ドキュメントは、MCPistのインフラストラクチャ構成を定義する。アプリケーションコンポーネント（spc-sys）とインフラコンポーネントの対応関係を明確化する。

**設計原則:**
- 放置運用（平日昼間に対応不可）
- ベンダー分散（単一障害点の排除）
- 自動フェイルオーバー

---

## インフラコンポーネント

| インフラコンポーネント | サービス | 役割 |
|----------------------|----------|------|
| API Gateway | Cloudflare Worker | JWT検証、Burst制限、DDoS対策 |
| Load Balancer | Cloudflare LB | ヘルスチェック、フェイルオーバー |
| KVキャッシュ | Cloudflare KV | Rate Limitカウンター |
| Primary Server | Koyeb | MCPサーバー（Primary） |
| Standby Server | Fly.io | MCPサーバー（Standby） |
| Database | Supabase PostgreSQL | ENT, TVLのデータストア |
| Secret Store | Supabase Vault | OAuthトークン暗号化保存 |
| Auth Provider | Supabase Auth | OAuth 2.1認証 |
| Monitoring | Grafana Cloud | Prometheus + Loki |
| Frontend | Vercel | User Console（CON） |
| CDN | Vercel Edge Network | 静的アセット配信 |
| メール配信 | Supabase Auth | パスワードリセット、認証メール |

---

## コンポーネント検討（採用/除外）

Phase 1で必要なコンポーネントを検討し、採用/除外を決定した。

### 採用

| カテゴリ | 採用サービス | 理由 |
|---------|-------------|------|
| メール配信 | Supabase Auth | 認証関連メール（パスワードリセット等）はSupabase Auth標準機能で対応。課金通知等が必要になった段階で別途検討 |
| CDN | Vercel Edge Network | User Console（CON）の静的アセット配信。Vercel標準機能 |
| キャッシュ | Cloudflare KV | Rate Limitカウンター用。既存構成で対応済み |
| バックアップ | Supabase標準 | PostgreSQLの自動バックアップ。Supabase標準機能 |

### 除外

| カテゴリ | 除外理由 |
|---------|----------|
| Redis/Memcached | セッションはJWT方式で不要。Rate LimitカウンターはCloudflare KVで対応済み |
| メッセージキュー | 現状は同期処理で十分。非同期バッチ処理の需要が発生した段階で再検討 |
| ファイルストレージ（S3/R2） | ユーザーファイルアップロード機能なし。将来ファイル添付等が必要になれば検討 |
| Sentry（エラートラッキング） | Grafana Cloud（Loki）でログ収集・分析可能。専用ツールの利便性より運用シンプル化を優先 |
| ステージング環境 | Phase 1は本番のみ。ユーザー数増加に応じて検討 |

---

## アプリケーションとインフラの対応

### コンポーネントマッピング

| アプリコンポーネント | 略称 | インフラ配置 |
|---------------------|------|-------------|
| MCP Client | CLT | 実装範囲外（Claude Code等） |
| Auth Server | AUS | Supabase Auth |
| MCP Server | SRV | Koyeb (Primary) / Fly.io (Standby) |
| Auth Middleware | AMW | SRV内部 |
| MCP Handler | HDL | SRV内部 |
| Module Registry | REG | SRV内部 |
| Modules | MOD | SRV内部 |
| Entitlement Store | ENT | Supabase PostgreSQL (public スキーマ) |
| Token Vault | TVL | Supabase PostgreSQL + Vault |
| User Console | CON | Vercel |
| External API Server | EXT | 実装範囲外（Notion API等） |
| Payment Service Provider | PSP | 実装範囲外（Stripe） |

### データストア配置

| データ種別 | テーブル群 | Supabaseスキーマ |
|-----------|-----------|-----------------|
| ENT | users, subscriptions, plans, etc. | mcpist |
| TVL | oauth_tokens | mcpist |
| TVL（暗号化） | vault.secrets | vault |
| AUS | auth.users | auth |

**スキーマ設計方針:**

同一Supabaseプロジェクトで複数サービスを運用可能な設計とする。

| スキーマ | 用途 | 管理 |
|---------|------|------|
| auth | 共通認証基盤 | Supabase管理 |
| vault | 暗号化ストア | Supabase管理 |
| mcpist | MCPist関連テーブル（ENT + TVL） | MCPist |
| （将来）pkmist等 | 他サービス関連テーブル | 各サービス |

**メリット:**
- auth.usersを共通ユーザー基盤として共有（シングルサインオン）
- スキーマ分離で責務が明確、RLSポリシーも分離可能
- 将来的にスキーマ単位でのexport/分割が容易

---

## アーキテクチャ図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCP Client                                      │
│                        (Claude Code, Cursor等)                               │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │ MCP Protocol (JSON-RPC over SSE)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Cloudflare                                         │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                    Worker (API Gateway)                                 │ │
│  │                                                                         │ │
│  │  1. JWT署名検証（未登録ユーザー遮断）                                  │ │
│  │  2. グローバルRate Limit（IP単位、DDoS対策）                           │ │
│  │  3. Burst制限（ユーザー単位）                                          │ │
│  │  4. X-User-ID付与 → オリジンに転送                                     │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                         │
│                    ┌───────────────┴───────────────┐                        │
│                    │      Load Balancer            │                        │
│                    │   (ヘルスチェック + フェイルオーバー)                   │
│                    └───────────────┬───────────────┘                        │
│                          ┌─────────┴─────────┐                              │
│                          ▼                   ▼                              │
│                    ┌──────────┐        ┌──────────┐                         │
│                    │  Koyeb   │        │  Fly.io  │                         │
│                    │ (Primary)│        │(Standby) │                         │
│                    └────┬─────┘        └────┬─────┘                         │
│                         └─────────┬─────────┘                               │
└───────────────────────────────────┼─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Supabase                                          │
│                                                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │   Auth Server   │  │   PostgreSQL    │  │     Vault       │             │
│  │                 │  │                 │  │                 │             │
│  │  • OAuth 2.1    │  │  • ENT tables   │  │ • oauth_tokens  │             │
│  │  • JWT発行      │  │  • TVL tables   │  │ • vault.secrets │             │
│  │  • JWKS公開     │  │                 │  │ (暗号化保存)     │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              ▼                     ▼                     ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Vercel      │    │     Stripe      │    │  External APIs  │
│                 │    │                 │    │                 │
│  User Console   │    │  PSP (課金)     │    │ Notion, Google  │
│  (CON)          │    │                 │    │ Calendar, etc.  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

---

## 負荷対策（2層構成）

### Worker層（インフラ保護）

| 制御 | 対象 | 制限値 | 目的 |
|------|------|--------|------|
| グローバルRate Limit | IP単位 | 1000 req/min | DDoS対策 |
| Burst制限 | ユーザー単位 | 5 req/s | 瞬間スパイク防止 |

### オリジン層（ビジネスロジック）

| 制御 | 対象 | 保存場所 | 目的 |
|------|------|----------|------|
| Rate Limit | ユーザー×プラン | メモリ | 過負荷防止 |
| Quota | ユーザー×月 | DB (usage) | 月間使用量制限 |
| Credit | ユーザー | DB (credits) | 従量課金 |

**プラン別Rate Limit:**

| プラン | 制限 |
|--------|------|
| Free | 30 req/min |
| Starter | 60 req/min |
| Pro | 120 req/min |
| Unlimited | 無制限 |

---

## フェイルオーバー

### ヘルスチェック

| 対象 | 間隔 | 判定 |
|------|------|------|
| Koyeb /health | 30秒 | 3回連続失敗 → unhealthy |
| Fly.io /health | 30秒 | 3回連続失敗 → unhealthy |

### 切替ロジック

```
Koyeb: healthy,  Fly.io: healthy  → Primary (Koyeb) にルーティング
Koyeb: unhealthy, Fly.io: healthy  → 自動フェイルオーバー (Fly.io)
Koyeb: healthy,  Fly.io: unhealthy → Koyeb にルーティング（問題なし）
Koyeb: unhealthy, Fly.io: unhealthy → エラーページ + アラート
```

---

## 監視・アラート

### Grafana Cloud

| 機能 | 用途 |
|------|------|
| Prometheus | メトリクス（CPU, メモリ, レイテンシ） |
| Loki | ログ収集 |
| Alerting | アラート通知 |

### アラート条件

| アラート | 条件 | 重要度 |
|----------|------|--------|
| CPU高負荷 | CPU > 80% (5分間) | Warning |
| メモリ逼迫 | Memory > 200MB | Critical |
| レイテンシ悪化 | p95 > 2s (5分間) | Warning |
| エラー率上昇 | 5xx > 5% (5分間) | Critical |
| オリジン停止 | /health 失敗 (3回連続) | Critical |
| Rate Limit多発 | Rate Limit > 100/min | Warning |

---

## セキュリティ

### オリジン直接アクセス防止

Worker経由でない直接アクセスを防止する。

| 方法 | 実装 |
|------|------|
| Gateway Secret | X-Gateway-Secret ヘッダー検証 |
| IP制限 | CloudflareのIPのみ許可（オプション） |

### ヘルスチェックのバイパス

`/health` エンドポイントは Gateway Secret なしでアクセス可能（LB用）。

---

## CI/CD

### デプロイフロー

```
GitHub Push (main)
    │
    ├─→ Koyeb: 自動デプロイ（GitHub連携）
    ├─→ Fly.io: GitHub Actions経由
    └─→ Cloudflare Worker: wrangler deploy
```

### デプロイ後検証

| テスト | 期待結果 |
|--------|----------|
| Koyeb + SECRET | 200 |
| Fly.io + SECRET | 200 |
| Worker経由 | 200 |
| 直接（SECRETなし） | 403 |

---

## コスト（Phase 1）

| サービス | プラン | コスト |
|----------|--------|--------|
| Koyeb | nano（永年無料） | $0 |
| Fly.io | 無料枠 | $0 |
| Supabase | 無料枠 | $0 |
| Cloudflare | 無料枠 | $0 |
| Grafana Cloud | 無料枠 | $0 |
| Vercel | 無料枠 | $0 |
| Stripe | 従量課金 | 決済額の3.6% |

**月額固定費: $0**（Stripe手数料のみ）

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書（アプリコンポーネント） |
| [spc-itr.md](./spc-itr.md) | インタラクション仕様書 |
| [spc-tbl.md](./spc-tbl.md) | テーブル仕様書 |
| [dsn-infrastructure.md](../DAY7/dsn-infrastructure.md) | インフラ設計（詳細） |
| [dsn-deployment.md](../DAY7/dsn-deployment.md) | デプロイ戦略（詳細） |
| [dsn-load-management.md](../DAY7/dsn-load-management.md) | 負荷対策設計（詳細） |
