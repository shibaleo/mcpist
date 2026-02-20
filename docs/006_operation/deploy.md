# MCPist デプロイ手順書

## 前提条件

### CLI ツール

```bash
# Cloudflare Workers
npm i -g wrangler
wrangler login

# Render (Go Server)
# CLI なし — Render Dashboard で操作
# https://dashboard.render.com
```

### 環境変数

ルートの `.env.local` にすべてのシークレットが設定済みであること。

---

## 1. Worker (Cloudflare Workers)

### 1-1. シークレット登録

```bash
# .env.local を読み込み
source .env.local

# dev 環境
./scripts/deploy-secrets.sh dev
```

手動で個別に設定する場合:

```bash
# dev 環境 (--env dev)
echo "$SERVER_URL"           | wrangler secret put SERVER_URL --env dev
echo "$GATEWAY_SIGNING_KEY"  | wrangler secret put GATEWAY_SIGNING_KEY --env dev
echo "$CLERK_JWKS_URL"       | wrangler secret put CLERK_JWKS_URL --env dev
echo "$SERVER_JWKS_URL"      | wrangler secret put SERVER_JWKS_URL --env dev
echo "$STRIPE_WEBHOOK_SECRET"| wrangler secret put STRIPE_WEBHOOK_SECRET --env dev

# Grafana (任意)
echo "$GRAFANA_LOKI_URL"     | wrangler secret put GRAFANA_LOKI_URL --env dev
echo "$GRAFANA_LOKI_USER"    | wrangler secret put GRAFANA_LOKI_USER --env dev
echo "$GRAFANA_LOKI_API_KEY" | wrangler secret put GRAFANA_LOKI_API_KEY --env dev

# prd 環境 (--env なし)
echo "$SERVER_URL"           | wrangler secret put SERVER_URL
# ... 同様
```

### 1-2. デプロイ

```bash
# dev 環境
cd apps/worker
npm run generate:openapi && wrangler deploy --env dev

# prd 環境
npm run generate:openapi && wrangler deploy
```

### 1-3. 検証

```bash
# dev
curl -s https://mcp.dev.mcpist.app/health
curl -s https://mcp.dev.mcpist.app/.well-known/jwks.json
curl -s https://mcp.dev.mcpist.app/v1/modules | head -c 200

# prd
curl -s https://mcp.mcpist.app/health
```

期待する応答:

```json
{"status":"ok","backend":{"healthy":true,"statusCode":200,"latencyMs":...}}
```

---

## 2. Go Server (Render)

### 2-1. Render サービス作成 (初回のみ)

1. [Render Dashboard](https://dashboard.render.com) → New → Web Service
2. 設定:
   - **Environment**: Docker
   - **Docker Build Context**: `apps/server`
   - **Dockerfile Path**: `apps/server/Dockerfile`
   - **Region**: Oregon (US West)
   - **Instance Type**: Free (または Starter)

### 2-2. 環境変数設定

Render Dashboard → Service → Environment で以下を設定:

| 変数名 | 値 | 備考 |
|--------|-----|------|
| `DATABASE_URL` | `postgresql://...` | Neon 接続文字列 (`search_path=mcpist` 付き) |
| `API_KEY_PRIVATE_KEY` | `(base64)` | Ed25519 seed (API Key 署名用) |
| `WORKER_JWKS_URL` | `https://mcp.dev.mcpist.app/.well-known/jwks.json` | Worker の JWKS URL |
| `ADMIN_EMAILS` | `admin@example.com` | カンマ区切り |
| `STRIPE_WEBHOOK_SECRET` | `whsec_...` | Stripe Webhook 検証用 |
| `GRAFANA_LOKI_URL` | `https://logs-prod-xxx.grafana.net` | 任意 |
| `GRAFANA_LOKI_USER` | `(user id)` | 任意 |
| `GRAFANA_LOKI_API_KEY` | `glc_...` | 任意 |

以下はデフォルト値があるため省略可能（変更が必要な場合のみ設定）:

| 変数名 | デフォルト値 | 備考 |
|--------|------------|------|
| `PORT` | `8089` | Dockerfile では `EXPOSE 8080` なので Render では `8080` 推奨 |
| `INSTANCE_ID` | `local` | `render-dev` 等に設定推奨 |
| `INSTANCE_REGION` | `local` | `oregon` 等に設定推奨 |

### 2-3. デプロイ

- **自動デプロイ**: Render が GitHub リポジトリと連携し、push 時に自動ビルド
- **手動デプロイ**: Render Dashboard → Manual Deploy → Deploy latest commit

### 2-4. 検証

```bash
# Render のサービス URL を使用
RENDER_URL="https://your-service.onrender.com"

curl -s $RENDER_URL/health
curl -s $RENDER_URL/.well-known/jwks.json
```

期待する応答:

```json
{"status":"ok","instance":"render-dev","region":"oregon","db":"ok"}
```

---

## 3. Worker ↔ Server 接続確認

Worker と Server の相互 JWKS 検証が正しく動作するか確認する。

```bash
# Worker → Server のヘルスチェックプロキシ
curl -s https://mcp.dev.mcpist.app/health
# backend.healthy が true であること

# Server → Worker の JWKS フェッチ
# (Server ログで WORKER_JWKS_URL からの公開鍵取得を確認)

# API Key を使った E2E テスト
curl -s https://mcp.dev.mcpist.app/v1/me/profile \
  -H "Authorization: Bearer mpt_YOUR_API_KEY"
```

---

## 4. Cloudflare DNS 設定 (初回のみ)

Worker にカスタムドメインを割り当てる。

1. Cloudflare Dashboard → Workers & Pages → mcpist-gateway
2. Settings → Triggers → Custom Domains
3. 追加:
   - dev: `mcp.dev.mcpist.app`
   - prd: `mcp.mcpist.app`

または `wrangler.toml` にルートを追加:

```toml
# prd
routes = [
  { pattern = "mcp.mcpist.app/*", zone_name = "mcpist.app" }
]

# dev
[env.dev]
routes = [
  { pattern = "mcp.dev.mcpist.app/*", zone_name = "mcpist.app" }
]
```

---

## 5. チェックリスト

### デプロイ前

- [ ] `.env.local` のシークレットが最新
- [ ] `pnpm env:sync` で Worker/Console のローカル env が同期済み
- [ ] ローカルで Server と Worker が正常動作

### Worker デプロイ後

- [ ] `/health` が 200 を返す
- [ ] `/.well-known/jwks.json` が Ed25519 公開鍵を返す
- [ ] `/v1/modules` がモジュール一覧を返す
- [ ] Cron トリガーが動作（Workers ログで確認）

### Server デプロイ後

- [ ] `/health` で `db: "ok"` を返す
- [ ] `/.well-known/jwks.json` が Ed25519 公開鍵を返す

### 接続テスト

- [ ] Worker `/health` の `backend.healthy` が `true`
- [ ] API Key 認証で `/v1/me/profile` が 200 を返す
- [ ] MCP クライアント（Claude Code 等）から接続できる

---

## 環境変数一覧

### Worker (Cloudflare Secrets)

| 変数名 | 説明 |
|--------|------|
| `SERVER_URL` | Go Server の URL |
| `SERVER_JWKS_URL` | Go Server の JWKS URL (API Key 公開鍵) |
| `GATEWAY_SIGNING_KEY` | Ed25519 seed — Worker → Server JWT 署名 |
| `CLERK_JWKS_URL` | Clerk JWT 検証用 JWKS URL |
| `STRIPE_WEBHOOK_SECRET` | Stripe Webhook 署名検証 |
| `GRAFANA_LOKI_URL` | Loki Push API URL (任意) |
| `GRAFANA_LOKI_USER` | Loki Basic Auth ユーザー (任意) |
| `GRAFANA_LOKI_API_KEY` | Loki Basic Auth API キー (任意) |

### Server (Render Environment)

| 変数名 | 説明 |
|--------|------|
| `DATABASE_URL` | PostgreSQL 接続文字列 |
| `API_KEY_PRIVATE_KEY` | Ed25519 seed — API Key JWT 署名 |
| `WORKER_JWKS_URL` | Worker の JWKS URL (Gateway JWT 公開鍵) |
| `ADMIN_EMAILS` | 管理者メール (カンマ区切り) |
| `STRIPE_WEBHOOK_SECRET` | Stripe Webhook 署名検証 |
| `GRAFANA_LOKI_URL` | Loki Push API URL (任意) |
| `GRAFANA_LOKI_USER` | Loki Basic Auth ユーザー (任意) |
| `GRAFANA_LOKI_API_KEY` | Loki Basic Auth API キー (任意) |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-dpl.md](../002_specification/spc-dpl.md) | デプロイ仕様書 |
| [dsn-infrastructure.md](../003_design/system/dsn-infrastructure.md) | インフラ設計 |
