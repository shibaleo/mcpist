---
title: MCPist インフラ仕様書（spec-inf）
aliases:
  - spec-inf
  - MCPist-infrastructure-specification
tags:
  - MCPist
  - specification
  - infrastructure
document-type:
  - specification
document-class: specification
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist インフラ仕様書（spec-inf）

本ドキュメントは、MCPistのインフラ構成とデプロイ設計を定義する。

---

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `archived` |
| Version | v1.0 (MEGA版) |
| Superseded by | [go-mcp-dev/dev/DAY3/specification/spec-inf.md](../../../go-mcp-dev/dev/DAY3/specification/spec-inf.md) |
| 差分概要 | go-mcp-dev版はTool Sieve (Supabase PostgreSQL + Edge Functions)、DNS (Cloudflare)、メトリクス (Grafana Mimir)、トレース (Grafana Tempo)、GitHub Actions CI/CD詳細、デプロイ後検証・ロールバック手順を追加 |

---

## 1. インフラ構成概要

### 1.1 推奨構成

| コンポーネント | サービス | リージョン | 備考 |
|---------------|----------|-----------|------|
| MCPサーバー | Koyeb | ワシントンDC | Free Tier |
| Authサーバー | Supabase Auth | 東京 | 無料枠 |
| Vault | Supabase Vault | 東京 | Auth連携 |
| Edge Function | Supabase Edge Functions | 東京 | トークン管理 |
| 管理UI | Vercel | Edge | Hobby Plan |
| 可観測性 | Grafana Cloud | US | Loki無料枠 |

### 1.2 コスト

| サービス | プラン | 月額 | 制限 |
|---------|--------|------|------|
| Koyeb | Free Tier | $0 | 1 Web Service |
| Supabase | Free | $0 | 50K MAU, 500MB DB |
| Vercel | Hobby | $0 | 100GB帯域 |
| Grafana Cloud | Free | $0 | Loki 50GB/月 |
| **合計** | | **$0** | |

---

## 2. MCPサーバー（Koyeb）

### 2.1 構成

| 項目 | 値 |
|------|-----|
| ランタイム | Docker |
| ベースイメージ | golang:1.22-alpine |
| ポート | 8088 (デフォルト、PORT環境変数で変更可能) |
| ヘルスチェック | GET /health |
| 環境変数 | 下記参照 |

**ポート番号の選定理由:**
- 8080は多くのアプリケーション(Tomcat, Jenkins, Spring Boot, webpack-dev-server等)と衝突リスクが高い
- 8088は比較的空いており、ローカル開発時の衝突を回避
- MCPサーバーに標準化されたポート番号は存在しない（HTTP/SSE, WebSocket, stdioなど複数のトランスポートをサポート）
- 環境変数`PORT`で柔軟に変更可能

### 2.2 環境変数

| 変数名 | 説明 | 必須 |
|--------|------|------|
| `TOKEN_BROKER_URL` | Token Broker (Supabase Edge Function) のエンドポイントURL | ○ |
| `TOKEN_BROKER_KEY` | Token Broker認証用の共有秘密鍵（Supabase ANON KEY） | ○ |

**Token Brokerの実装:**
- Token Brokerは論理的なコンポーネント名であり、実装は **Supabase Vault + Edge Function**
- `TOKEN_BROKER_URL`: Edge FunctionのエンドポイントURL（例: `https://<project>.supabase.co/functions/v1/token-exchanger`）
- `TOKEN_BROKER_KEY`: Supabase ANON KEYを使用（MCPサーバー↔Edge Function間の認証）
- Edge Function側の環境変数: `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`

**JWT検証について:**
- LLMクライアントから受信したJWTはSupabase AuthのJWKS（JSON Web Key Set）エンドポイントを使用して検証
- 公開鍵暗号方式（RS256等）のため、MCPサーバー側に秘密鍵（JWT_SECRET）は不要
- JWKSエンドポイント: `https://<project>.supabase.co/auth/v1/jwks`

**注意:**
- MCPサーバーはSupabaseに直接アクセスしない（Token Broker経由のみ）
- トークンの暗号化保存はSupabase Vaultで実施
- トークン取得・リフレッシュ処理はEdge Functionで実施
- ポート番号は8088固定（ローカル開発時のDockerfile EXPOSEで指定）
- ログレベルはコード内でデフォルト値（info）を使用

### 2.3 Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mcpist ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/mcpist .
EXPOSE 8088
CMD ["./mcpist"]
```

### 2.4 koyeb.yaml

```yaml
name: mcpist
type: web
instance_types:
  - nano
regions:
  - was
env:
  - key: TOKEN_BROKER_URL
    secret: token-broker-url
  - key: TOKEN_BROKER_KEY
    secret: token-broker-key
ports:
  - port: 8088
    protocol: http
health_checks:
  - type: http
    path: /health
    port: 8088
    interval_seconds: 30
    timeout_seconds: 5
```

---

## 3. Supabase

### 3.1 プロジェクト構成

| 機能 | 使用サービス |
|------|-------------|
| 認証 | Auth |
| トークン保存 | Vault + PostgreSQL |
| APIゲートウェイ | Edge Functions |
| リアルタイム | 不使用 |

### 3.2 データベーススキーマ

※ 詳細は実装時に決定

### 3.3 Edge Functions

※ 詳細は実装時に決定

---

## 4. 管理UI（Vercel）

### 4.1 構成

| 項目 | 値 |
|------|-----|
| フレームワーク | Next.js 14 |
| スタイル | Tailwind CSS |
| 認証 | Supabase Auth |
| デプロイ | Vercel |

### 4.2 環境変数

| 変数名 | 説明 |
|--------|------|
| `NEXT_PUBLIC_SUPABASE_URL` | Supabase URL |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Supabase Anon Key |
| `SUPABASE_SERVICE_ROLE_KEY` | Service Role Key（サーバーのみ） |

### 4.3 vercel.json

```json
{
  "framework": "nextjs",
  "buildCommand": "npm run build",
  "installCommand": "npm install",
  "devCommand": "npm run dev",
  "regions": ["hnd1"]
}
```

---

## 5. 可観測性（Grafana Cloud）

### 5.1 ログ設計

#### ログ出力元と転送

| コンポーネント | 転送方法 | 出力先 |
|---------------|---------|--------|
| MCPサーバー（Koyeb） | stdout → Koyeb Log Drain | Grafana Cloud Loki |
| 管理UI（Vercel） | Vercel Log Drain | Grafana Cloud Loki |
| Edge Functions（Supabase） | HTTP API送信 | Grafana Cloud Loki |

※ ローカル開発時は stdout（JSON）のみ

#### ログフォーマット

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "Tool executed",
  "module": "notion",
  "tool": "search",
  "user_id": "user-123",
  "account_id": "account-456",
  "duration_ms": 245,
  "success": true
}
```

### 5.2 メトリクス

| メトリクス | 説明 | タイプ |
|-----------|------|--------|
| `mcpist_requests_total` | リクエスト総数 | Counter |
| `mcpist_request_duration_ms` | リクエスト処理時間 | Histogram |
| `mcpist_tool_calls_total` | ツール呼び出し回数 | Counter |
| `mcpist_errors_total` | エラー総数 | Counter |
| `mcpist_token_refresh_total` | トークンリフレッシュ回数 | Counter |

### 5.3 アラート（Grafana Cloud IRM）

| アラート | 条件 | 重要度 | 通知 |
|---------|------|--------|------|
| 5xxエラー | 発生時 | Critical | メール |
| ヘルスチェック失敗 | 3回連続 | Critical | メール |
| トークンリフレッシュ失敗 | 発生時 | Critical | メール |
| Slow Response | P95 > 5秒 (5分間) | Warning | ログのみ |
| 4xxエラー率 | > 10% (5分間) | Warning | ログのみ |

---

## 6. ネットワーク構成

### 6.1 通信経路

```
LLMクライアント
    │
    │ HTTPS (TLS 1.3)
    ▼
MCPサーバー (Koyeb)
    │
    ├──▶ Supabase Auth (JWT検証)
    │    HTTPS
    │
    ├──▶ Supabase Edge Functions (トークン取得)
    │    HTTPS
    │
    └──▶ 外部API (Notion, GitHub等)
         HTTPS

管理UI (Vercel)
    │
    │ HTTPS
    ▼
Supabase (Auth + DB)
```

### 6.2 CORS設定

| オリジン | 許可 |
|---------|------|
| 管理UI (vercel.app) | ○ |
| localhost:3000 | ○ (開発時) |
| その他 | × |

---

## 7. セキュリティ

### 7.1 シークレット管理

| シークレット | 保存場所 | 用途 |
|-------------|---------|------|
| JWT_SECRET | Koyeb Secrets | JWT署名検証 |
| SUPABASE_SERVICE_ROLE_KEY | Koyeb Secrets | Supabase管理操作 |
| 暗号化キー | Supabase Vault | トークン暗号化 |
| OAuthクライアントシークレット | Supabase DB | OAuth認可 |

### 7.2 アクセス制御

| リソース | 制御方法 |
|---------|---------|
| MCPサーバー | JWT認証 |
| Supabase DB | RLS (Row Level Security) |
| Edge Functions | Supabase Auth |
| 管理UI | Supabase Auth |

---

## 8. バックアップ・DR

### 8.1 バックアップ

| 対象 | 方法 | 頻度 |
|------|------|------|
| Supabase DB | 自動バックアップ | 日次 |
| 設定ファイル | Git | 都度 |
| シークレット | 手動エクスポート | 変更時 |

### 8.2 災害復旧

| シナリオ | 復旧方法 | RTO |
|---------|---------|-----|
| Koyeb障害 | 再デプロイ | 10分 |
| Supabase障害 | 待機 or 別リージョン | 1時間 |
| トークン漏洩 | 全トークンローテーション | 1時間 |

---

## 9. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [設計仕様書](spec-dsn.md) | 詳細設計 |
| [運用仕様書](spec-ops.md) | 運用設計 |
