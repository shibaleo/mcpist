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

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v3.0 (DAY5) |
| Base | DAY4からコピー |
| Note | DAY5で詳細化予定 |

---

本ドキュメントは、MCPistのインフラ構成とデプロイ設計を定義する。

---

## 1. インフラ構成概要

### 1.1 推奨構成

| 論理コンポーネント | 物理サービス | リージョン | 備考 |
|-------------------|-------------|-----------|------|
| MCPサーバー | Koyeb | ワシントンDC | Free Tier |
| Authサーバー | Supabase Auth | 東京 | 無料枠 |
| Tool Sieve | Supabase PostgreSQL + Edge Functions | 東京 | RPC でロール権限フィルタ |
| Token Broker | Supabase Vault + Edge Functions | 東京 | トークン暗号化・リフレッシュ |
| 管理UI | Vercel | Edge | Hobby Plan |
| DNS | Cloudflare | Edge | 無料枠、DNS Only |
| 可観測性（ログ） | Grafana Cloud Loki | US | 無料枠 |
| 可観測性（メトリクス） | Grafana Cloud Mimir | US | 無料枠、Prometheus互換 |
| 可観測性（トレース） | Grafana Cloud Tempo | US | 無料枠、OTLP |

### 1.2 コスト

| サービス | プラン | 月額 | 制限 |
|---------|--------|------|------|
| Koyeb | Free Tier | $0 | 1 Web Service |
| Supabase | Free | $0 | 50K MAU, 500MB DB |
| Vercel | Hobby | $0 | 100GB帯域 |
| Grafana Cloud | Free | $0 | Loki 50GB、Mimir 10K series、Tempo 50GB |
| **合計** | | **$0** | |

---

## 2. Koyeb（MCPサーバー）

### 2.1 構成

| 項目 | 値 |
|------|-----|
| ランタイム | Docker |
| ベースイメージ | golang:1.22-alpine |
| ポート | 8088（起動オプション `--port` で変更可能） |
| ヘルスチェック | GET /health |
| 環境変数 | 下記参照 |

**ポート番号の選定理由:**
- 8080は多くのアプリケーション(Tomcat, Jenkins等)と衝突リスクが高い
- 8088は比較的空いており、ローカル開発時の衝突を回避

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
- MCPサーバーはSupabaseに直接アクセスしない（Edge Function経由のみ）
- トークンの暗号化保存はSupabase Vaultで実施
- トークン取得・リフレッシュ処理はEdge Functionで実施

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

## 3. Supabase（Auth + DB + Vault + Edge Functions）

### 3.1 使用サービス

| サービス | 論理コンポーネント | 用途 |
|---------|-------------------|------|
| Auth | Authサーバー | ユーザー認証、JWT発行 |
| PostgreSQL | Tool Sieve | ロール権限管理（RLS） |
| Vault | Token Broker | トークン暗号化保存 |
| Edge Functions | Tool Sieve, Token Broker | RPC エンドポイント |
| Dashboard | 管理者用 | SQLエディタ、許可リスト登録、DB管理 |

### 3.2 データベーススキーマ

データモデルの詳細は [spec-dsn.md §3](spec-dsn.md#3-データモデル) を参照。

**主要テーブル:**

| テーブル | 用途 | RLS |
|---------|------|-----|
| users | ユーザー情報 | ✅ |
| roles | ロール定義 | ✅ |
| user_roles | ユーザー↔ロール紐付け | ✅ |
| role_permissions | ロール↔ツール権限（Tool Sieve） | ✅ |
| oauth_tokens | OAuthトークン（Vault連携） | ✅ |
| audit_logs | 監査ログ | ✅ |

### 3.3 Edge Functions

| Function | 用途 | 呼び出し元 |
|----------|------|-----------|
| tool-sieve | ロール権限に基づくツールフィルタリング | MCPサーバー |
| token-exchanger | トークン取得・リフレッシュ | MCPサーバー |
| oauth-callback | OAuth認可コールバック | 外部OAuthプロバイダ |

---

## 4. Vercel（管理UI）

### 4.1 フレームワーク選定

**採用: Next.js 14**

| 観点 | Astro | Next.js | 判定 |
|------|-------|---------|------|
| SPA対応 | △ 可能だがMPA向き | ◎ App Router で完全SPA | Next.js |
| モーダル/パネル | △ React統合必要 | ◎ ネイティブサポート | Next.js |
| URL状態管理 | △ 追加実装必要 | ◎ `useSearchParams` | Next.js |
| Supabase連携 | ○ 公式対応 | ◎ @supabase/ssr | Next.js |
| OAuth callback | △ API Routes追加 | ◎ Route Handlers | Next.js |
| admin/user出し分け | △ middleware追加 | ◎ middleware.ts | Next.js |
| バンドルサイズ | ◎ 小さい | ○ 普通 | Astro |
| Vercel最適化 | ○ | ◎ 完全統合 | Next.js |

**選定理由:**
- 管理UIはSPA設計（モーダル/パネル多用、URL query状態管理）のため、Next.jsが最適
- Astroは静的サイト/ブログ向き
- Phase 2でデスクトップアプリ（Electron）にバンドルする際、React/Next.jsベースなら移行が容易

### 4.2 構成

| 項目 | 値 |
|------|-----|
| フレームワーク | Next.js 14 (App Router) |
| スタイル | Tailwind CSS |
| 認証 | Supabase Auth (@supabase/ssr) |
| デプロイ | Vercel |

### 4.3 環境変数

| 変数名 | 説明 |
|--------|------|
| `NEXT_PUBLIC_SUPABASE_URL` | Supabase URL（公開） |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Supabase Anon Key（公開） |
| `SUPABASE_SERVICE_ROLE_KEY` | DB操作用（サーバーサイドのみ） |

### 4.4 アーキテクチャ

```
管理UI (Next.js App Router)
    │
    ├─→ middleware.ts
    │   - 認証チェック（Supabase Auth）
    │   - system_role判定（admin / user）
    │   - ルート保護（/admin/* → admin限定）
    │
    ├─→ Route Handlers (/api/...)
    │   - SUPABASE_SERVICE_ROLE_KEY でDB操作
    │   - system_roleに応じたアクセス制御
    │
    └─→ Client Components
        - ANON_KEY でSupabase Auth
        - RLS適用のデータ取得
```

**ルート保護:**
| パス | 必要なsystem_role |
|------|-------------------|
| /admin/* | admin |
| /settings | admin, user |
| /oauth/connect | admin, user |

### 4.5 vercel.json

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

## 5. Grafana Cloud（可観測性）

### 5.1 ログ設計（Grafana Cloud Loki）

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

### 5.2 メトリクス（Grafana Cloud Mimir）

MCPサーバーから OpenTelemetry SDK でメトリクスを送信。

**収集メトリクス:**

| メトリクス                        | 説明               | タイプ       |
| ---------------------------- | ---------------- | --------- |
| `mcpist_tool_calls_total`    | モジュール/ツール別呼び出し回数 | Counter   |
| `mcpist_tool_duration_ms`    | ツール別レイテンシ        | Histogram |
| `mcpist_tool_errors_total`   | ツール別エラー回数        | Counter   |
| `mcpist_requests_total`      | リクエスト総数          | Counter   |
| `mcpist_token_refresh_total` | トークンリフレッシュ回数     | Counter   |

### 5.3 トレース（Grafana Cloud Tempo）

MCPサーバーから OpenTelemetry SDK でトレースを送信。

**トレース対象:**
- LLMクライアント → MCPサーバー（リクエスト全体）
- MCPサーバー → Tool Sieve（権限チェック）
- MCPサーバー → Token Broker（トークン取得）
- MCPサーバー → 外部API（ツール実行）

**実装:** Go の `go.opentelemetry.io/otel` パッケージを使用

### 5.4 アラート（Grafana Cloud IRM）

| アラート | 条件 | 重要度 | 通知 |
|---------|------|--------|------|
| 5xxエラー | 発生時 | Critical | メール |
| ヘルスチェック失敗 | 3回連続 | Critical | メール |
| トークンリフレッシュ失敗 | 発生時 | Critical | メール |
| Slow Response | P95 > 5秒 (5分間) | Warning | ログのみ |
| 4xxエラー率 | > 10% (5分間) | Warning | ログのみ |

---

## 6. Cloudflare（DNS）

### 6.1 用途

| サービス | 用途 |
|---------|------|
| DNS | mcpist.app ドメインの名前解決 |
| プロキシ | 無効（DNS Only） |

**注:** CDN/WAF機能は使用せず、DNS のみ利用。Koyeb/Vercel の SSL 証明書を使用。

---

## 7. ネットワーク構成

### 7.1 通信経路

```
LLMクライアント
    │
    │ DNS: mcpist.app → Cloudflare → Koyeb IP
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

### 7.2 CORS設定

**MCPサーバー:** CORS設定不要（MCPはstdio/SSEプロトコル、ブラウザからのアクセスなし）

**管理UI → Supabase:** Supabase側で自動設定（`@supabase/ssr`使用）

| オリジン | 許可 |
|---------|------|
| 管理UI (vercel.app) | ○ |
| localhost:3000 | ○ (開発時) |
| その他 | × |

---

## 8. セキュリティ

### 8.1 シークレット管理

| 保存場所                    | シークレット                        | 用途                    |
| ----------------------- | ----------------------------- | --------------------- |
| Koyeb Secrets           | TOKEN_BROKER_URL              | Edge Function エンドポイント |
| Koyeb Secrets           | TOKEN_BROKER_KEY              | Supabase ANON KEY     |
| Supabase Edge Functions | SUPABASE_URL                  | Supabase プロジェクトURL    |
| Supabase Edge Functions | SUPABASE_SERVICE_ROLE_KEY     | DB操作（RLS回避）           |
| Supabase Vault          | OAuthクライアントシークレット        | 管理者登録、プロバイダ別        |
| Supabase Vault          | OAuthトークン（access/refresh）    | ユーザー別、暗号化保存          |
| GitHub Actions Secrets  | KOYEB_SERVICE_ID              | デプロイ対象サービス            |
| GitHub Actions Secrets  | KOYEB_API_TOKEN               | Koyeb API認証           |
| Vercel環境変数              | NEXT_PUBLIC_SUPABASE_URL      | Supabase URL（公開）      |
| Vercel環境変数              | NEXT_PUBLIC_SUPABASE_ANON_KEY | Supabase Anon Key（公開） |
| Vercel環境変数              | SUPABASE_SERVICE_ROLE_KEY     | Route Handlers用DB操作   |

### 8.2 アクセス制御

| リソース | 制御方法 |
|---------|---------|
| MCPサーバー | JWT認証 |
| Supabase DB | RLS (Row Level Security) |
| Edge Functions | JWT検証 → SERVICE_ROLE_KEYでDB操作 |
| 管理UI | Supabase Auth |

**Edge Functionsの認証フロー:**
1. クライアントからJWT受信
2. Edge Function内でJWT検証（Supabase Auth JWKS）
3. 検証成功後、SERVICE_ROLE_KEYでDB操作（ユーザー代理）

**緊急時のDB操作:**
- Supabase Dashboard（SQLエディタ）で直接操作

---

## 9. GitHub（CI/CD）

### 9.1 ブランチ戦略

**ブランチ構成:**
```
main (production)
  ↑
  │ PR + Squash merge（週次 or 手動）
  │
dev (CI/CD対象、shadow main)
  │
  ├── feat/xxx
  ├── fix/xxx
  └── docs/xxx
```

| ブランチ | 役割 | マージ先 | CI |
|---------|------|---------|-----|
| main | 本番環境 | - | - |
| dev | 開発統合ブランチ（shadow main） | main | ✅ push時 |
| feat/* | 新機能開発 | dev | ✅ PR時 |
| fix/* | バグ修正 | dev | ✅ PR時 |
| docs/* | ドキュメント更新 | dev | ✅ PR時 |

**ブランチ保護ルール:**

| ルール | main | dev |
|--------|------|-----|
| Require pull request before merging | ✅ | ✅ |
| Require status checks to pass (ci.yml) | ✅ | ✅ |
| Require branches to be up to date | ✅ | - |
| Require linear history | ✅ | - |
| Do not allow bypassing | ✅ | - |

**マージ戦略:**
- **feat/fix/docs → dev:** 通常マージ（複数コミット可）
- **dev → main:** Squash merge のみ（1リリース = 1コミット）
- コンフリクト解消は子ブランチ側で実施

**開発フロー:**
```
1. dev から feat/xxx ブランチ作成
2. 開発・コミット（複数コミット可）
3. PR作成（feat → dev）→ ci.yml 自動実行
4. レビュー・承認
5. dev へマージ
6. リリース時: dev → main PR作成
7. Squash merge → main
8. 手動で deploy.yml 実行（本番反映）
```

### 9.2 GitHub Actions

| ワークフロー             | トリガー                  | 内容                    |
| ------------------ | --------------------- | --------------------- |
| ci.yml             | push/PR to main(随時)   | ビルド、テスト               |
| deploy.yml         | workflow_dispatch(手動) | Koyeb 再デプロイ + ヘルスチェック |
| ping.yml           | schedule (毎日)         | 定期ヘルスチェック（コールドスタート回避） |
| version-notify.yml | schedule (週1)         | 外部APIバージョン差分通知        |

**version-notify.yml の動作:**
1. 公開APIエンドポイントからバージョン情報取得（トークン不要）
2. module_registry テーブルの `api_version` と比較
3. 差分があればGitHub Issue自動作成（ラベル: `api-version-change`）
4. 管理者がIssue確認後、手動でmodule_registryを更新しIssueクローズ

**対応サービス:**
- 自動取得可能: Notion（レスポンスヘッダ）, GitHub（API応答）, Supabase
- 手動確認: Jira, Confluence（Changelog監視）

シークレットは §8.1 シークレット管理 を参照。

### 9.3 デプロイ後検証とロールバック

**deploy.yml の検証フロー:**
```yaml
steps:
  - name: Trigger Koyeb Redeploy
    run: curl -X POST ...

  - name: Wait for deployment
    run: sleep 60

  - name: Smoke Test
    run: |
      curl -f https://mcpist.app/health || exit 1
      curl -f https://mcpist.app/api/version || exit 1

  - name: Rollback on Failure
    if: failure()
    run: |
      curl -X POST "https://app.koyeb.com/v1/services/$KOYEB_SERVICE_ID/rollback" \
        -H "Authorization: Bearer $KOYEB_API_TOKEN"
```

**障害時の対応:**

| 状況 | 対応 | 所要時間 |
|------|------|---------|
| デプロイ後にSmoke Test失敗 | 自動ロールバック | 1-2分 |
| 本番稼働中に障害発覚 | Koyebダッシュボードから手動ロールバック | 1-2分 |
| 緊急修正が必要 | `fix/` ブランチ → 緊急PR → main → deploy | 10-30分 |

**ロールバック手順（手動）:**
1. Koyebダッシュボード → サービス → Deployments
2. 前バージョンを選択 → "Redeploy" クリック
3. ヘルスチェック確認

**注意:** DBマイグレーションを含むデプロイは、ロールバック時にデータ不整合のリスクあり。マイグレーションは後方互換性を保つこと。

---

## 10. バックアップ・DR

### 10.1 バックアップ

| 対象 | 方法 | 頻度 |
|------|------|------|
| Supabase DB | 自動バックアップ | 日次 |
| 設定ファイル | Git | 都度 |
| シークレット | 手動エクスポート | 変更時 |

### 10.2 災害復旧

| シナリオ | 復旧方法 | RTO |
|---------|---------|-----|
| Koyeb障害 | 再デプロイ | 10分 |
| Supabase障害 | 待機 or 別リージョン | 1時間 |
| トークン漏洩 | 全トークンローテーション | 1時間 |

---

## 11. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [設計仕様書](spec-dsn.md) | 詳細設計 |
| [運用仕様書](spec-ops.md) | 運用設計 |
