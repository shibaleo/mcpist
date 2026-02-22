# MCPist 運用仕様書（spc-ops）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | Operations Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist の運用に関する仕様を定義する。

---

## CI/CD

### GitHub Actions

| ジョブ | 内容 |
|---|---|
| lint-server | golangci-lint (apps/server) |
| test-server | `go test -v -race ./...` (apps/server) |
| build-server | `go build -v ./...` (apps/server) |
| lint-console | ESLint (apps/console) |
| build-console | `pnpm build` (apps/console) |
| lint-worker | `tsc --noEmit` (apps/worker) |

トリガー: workflow_dispatch (手動実行)。

### デプロイ

各コンポーネントは個別にデプロイする。

| コンポーネント | 方法 | コマンド |
|---|---|---|
| Worker | Wrangler CLI | `wrangler deploy --env <env>` |
| Server | Render ダッシュボード | Git 連携による自動デプロイ |
| Console | Vercel | Git 連携による自動デプロイ |
| Database | 手動 SQL | マイグレーションファイルを Neon Console で実行 |

### シークレット配布

`scripts/deploy-secrets.sh <dev|prd>` で Worker (wrangler) と Console (vercel) にシークレットを配布する。Server (Render) は CLI がないためダッシュボードで手動設定。

---

## 監視

### ログ

Worker・Server の両方から Grafana Cloud (Loki) にログを送信する。

| ラベル | 説明 |
|---|---|
| app | アプリ名 (mcpist-server / mcpist-worker) |
| instance | インスタンス ID |
| region | リージョン |
| type | request / security / error |
| level | info / warn / error |

**Server 側ログ関数:**

| 関数 | 用途 |
|---|---|
| LogToolCall | モジュール実行ログ (duration_ms, status) |
| LogRequest | HTTP リクエストログ |
| LogError | エラーコンテキスト |
| LogSecurityEvent | セキュリティイベント |

**Worker 側:** `ctx.waitUntil()` でノンブロッキング送信。

### Loki クエリ例

```
# エラーログ
{app="mcpist-server"} | json | level="error"

# セキュリティイベント
{type="security"} | json

# 特定モジュールのツール実行
{app="mcpist-server", type="request"} | json | module="notion"
```

---

## ヘルスチェック

### Worker → Server ヘルスチェック

| 項目 | 値 |
|---|---|
| 間隔 | 5 分 (wrangler.toml cron) |
| タイムアウト | 5 秒 |
| エンドポイント | `{SERVER_URL}/health` |

**Worker レスポンス (`GET /health`):**

```json
{
  "status": "ok",
  "backend": {
    "healthy": true,
    "statusCode": 200,
    "latencyMs": 123
  }
}
```

### Server ヘルスチェック

**Server レスポンス (`GET /health`):**

```json
{
  "status": "ok",
  "instance": "render-dev",
  "region": "oregon",
  "db": "ok"
}
```

DB 接続失敗時は HTTP 503 + `"status": "degraded"`, `"db": "unavailable"` を返す。

レスポンスヘッダーに `X-Instance-ID`, `X-Instance-Region` を付与。

---

## ローカル開発

### 起動

```bash
pnpm dev
```

以下を並列実行する:
1. Docker Compose で PostgreSQL 起動 (port 57432)
2. `scripts/sync-env.js` で `.env.local` → 各アプリの環境変数ファイルに同期
3. Server (Go)、Worker (Wrangler)、Console (Next.js) を起動

### 環境変数管理

ルートの `.env.local` (または `.env.dev`) が SSoT。`pnpm env:sync` (scripts/sync-env.js) で以下に同期:
- `apps/worker/.dev.vars`
- `apps/console/.env.local`

### Turborepo タスク

| コマンド | 内容 |
|---|---|
| `pnpm dev` | DB 起動 + env 同期 + 全アプリ起動 |
| `pnpm build` | 全アプリビルド |
| `pnpm lint` | 全アプリ lint |
| `pnpm test` | 全アプリテスト |
| `pnpm db:up` | Docker Compose 起動 |
| `pnpm db:down` | Docker Compose 停止 |

---

## デプロイ後検証

| チェック | 期待結果 |
|---|---|
| Worker `/health` | `backend.healthy: true` |
| Server `/health` | `db: "ok"` |
| Worker `/.well-known/jwks.json` | Ed25519 公開鍵 |
| Server `/.well-known/jwks.json` | Ed25519 公開鍵 |
| `/v1/modules` | モジュール一覧 |
| API Key 認証 `/v1/me/profile` | 200 |
| MCP クライアント接続 | 正常応答 |

---

## 障害対応

### 障害分類

| 分類 | 症状 | 対応 |
|---|---|---|
| 認証エラー | 401/403 | JWT / API Key の有効性確認 |
| 外部 API 障害 | ToolCallResult isError | 外部サービスのステータス確認 |
| レート制限 | 429 | 自動回復を待つ |
| 内部エラー | 500 | Loki ログで原因調査 |
| Server 停止 | /health 失敗 | Render ダッシュボードで確認、ロールバック |

### ロールバック

| コンポーネント | 方法 |
|---|---|
| Server | Render ダッシュボードで前デプロイメントを選択 |
| Worker | `wrangler rollback` |
| Console | Vercel ダッシュボードで前デプロイメントを選択 |

---

## セキュリティ運用

### 定期タスク

| タスク | 頻度 |
|---|---|
| 依存パッケージ更新 | 月次 (Dependabot) |
| シークレットローテーション | 年次 or 漏洩時 |
| アクセスログ監査 | 月次 |

### シークレットローテーション

**Ed25519 鍵 (GATEWAY_SIGNING_KEY / API_KEY_PRIVATE_KEY):**

1. 新しい鍵を生成
2. Worker と Server に同時配布
3. 旧 API キーは無効化される (ユーザーに再発行を案内)

**CREDENTIAL_ENCRYPTION_KEY:**

1. 新しい鍵を生成
2. Server に設定
3. key_version をインクリメントして既存データを再暗号化

---

## バックアップ

| 対象 | 方法 |
|---|---|
| Database | Neon 自動バックアップ |
| ソースコード | Git (GitHub) |
| 環境変数 | `.env.local` + `deploy-secrets.sh` |

---

## ポストモーテム

障害発生時は `docs/postmortems/YYYY-MM-DD-title.md` に記録する。

---

## 関連ドキュメント

| ドキュメント                                             | 内容        |
| -------------------------------------------------- | --------- |
| [spec-systems.md](./spec-systems.md)               | システム仕様書   |
| [spec-infrastructure.md](./spec-infrastructure.md) | インフラ仕様書   |
| [spec-security.md](./spec-security.md)             | セキュリティ仕様書 |
| [spec-test.md](spec-test.md)                       | テスト仕様書    |
