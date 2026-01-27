# デプロイ計画

## 概要

mcpist の各環境を以下のアカウントでデプロイする。

---

## インフラ構成（全環境共通）

### アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Cloudflare (DNS + Workers)                          │
│                         アカウント: shiba.dog.leo.private                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  DNS (mcpist.app)                                                           │
│  ├── dev.mcpist.app   → Worker (mcpist-gateway-dev)                        │
│  ├── stg.mcpist.app   → Worker (mcpist-gateway-stg)                        │
│  └── cloud.mcpist.app → Worker (mcpist-gateway-prod)                       │
│                                                                             │
│  各Workerがロードバランサとして機能                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Worker (環境ごと)                                                    │   │
│  │ ├── PRIMARY_API_URL   → Render (環境ごと)                           │   │
│  │ ├── SECONDARY_API_URL → Koyeb (環境ごと)                            │   │
│  │ └── KV Namespace      → 環境ごとに分離                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Cloudflare Workers の利点

| 項目 | 説明 |
|------|------|
| Worker数 | **無制限**（無料プランでも） |
| DNS管理 | 1つのアカウントで一元管理 |
| ロードバランサ | 各環境で独立設定 |
| KVキャッシュ | 環境ごとに分離 |
| デプロイ | `wrangler deploy --env dev/stg/production` |

### 環境別リソース

| 環境 | Supabase | Render | Koyeb | Cloudflare Worker | Vercel | ドメイン |
|------|----------|--------|-------|-------------------|--------|---------|
| dev | shiba.dog.leo.private | shiba | shiba | mcpist-gateway-dev | dev branch | dev.mcpist.app |
| stg | fukudamakoto.private | fukuda | fukuda | mcpist-gateway-stg | stg branch | stg.mcpist.app |
| prod | fukudamakoto.work | fukuda | fukuda | mcpist-gateway-prod | main branch | cloud.mcpist.app |

---

## dev環境 デプロイ計画

## 依存関係図

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          依存関係                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  [1] Supabase ─────────────────────────────────────────────────────┐    │
│        │                                                           │    │
│        │ URL, Publishable Key, Service Role Key                    │    │
│        │                                                           │    │
│        ▼                                                           │    │
│  [2] DockerHub ────────────────────────────────────┐               │    │
│        │                                           │               │    │
│        │ Image URL                                 │               │    │
│        ▼                                           │               │    │
│  [3] Render ──────────────────┐                    │               │    │
│        │                      │                    │               │    │
│        │ Primary API URL      │ Secondary API URL  │               │    │
│        ▼                      ▼                    │               │    │
│  [4] Koyeb ───────────────────┘                    │               │    │
│        │                                           │               │    │
│        │ Render URL, Koyeb URL                     │               │    │
│        ▼                                           │               │    │
│  [5] Cloudflare Worker ────────────────────────────┘               │    │
│        │                                                           │    │
│        │ Worker URL                                                │    │
│        ▼                                                           │    │
│  [6] Vercel ◄──────────────────────────────────────────────────────┘    │
│        │                                                                │
│        │ Console URL                                                    │
│        ▼                                                                │
│  [7] DNS (dev.mcpist.app)                                               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Step 1: Supabase プロジェクト確認

**状態**: ✅ 既存プロジェクトあり

### 必要な情報の取得

| 項目 | 環境変数名 | 取得場所 |
|------|-----------|---------|
| Project URL | `SUPABASE_URL` | Settings → API |
| Anon/Publishable Key | `SUPABASE_PUBLISHABLE_KEY` | Settings → API |
| Service Role Key | `SUPABASE_SERVICE_ROLE_KEY` | Settings → API |
| JWKS URL | `SUPABASE_JWKS_URL` | `{SUPABASE_URL}/auth/v1/.well-known/jwks.json` |

### 確認事項

- [ ] マイグレーションが適用されている
- [ ] RLS ポリシーが設定されている
- [ ] RPC 関数が作成されている (`validate_api_key`, `revoke_api_key` など)

---

## Step 2: DockerHub イメージ push

**状態**: ⬜ 未実施

### 手順

1. DockerHub でリポジトリ作成
   - URL: https://hub.docker.com/
   - Repository: `shibaleo/mcpist-api`
   - Visibility: Public

2. ローカルでビルド & push
   ```bash
   cd c:\Users\shiba\HOBBY\mcpist\apps\server
   docker build -t shibaleo/mcpist-api:latest .
   docker login
   docker push shibaleo/mcpist-api:latest
   ```

### 出力

| 項目 | 値 |
|------|-----|
| Image URL | `docker.io/shibaleo/mcpist-api:latest` |

---

## Step 3: Render サービス作成

**状態**: ⬜ 未作成

### 手順

1. https://dashboard.render.com/ にアクセス
2. 「New +」→「Web Service」
3. 「Deploy an existing image from a registry」を選択

### 設定

| 項目 | 値 |
|------|-----|
| Name | `mcpist-api-dev` |
| Region | Oregon (US West) |
| Image URL | `docker.io/shibaleo/mcpist-api:latest` |
| Instance Type | Free |

### 環境変数

| Key | Value | 備考 |
|-----|-------|------|
| `SUPABASE_URL` | (Step 1 で取得) | |
| `SUPABASE_SERVICE_ROLE_KEY` | (Step 1 で取得) | Secret |
| `GATEWAY_SECRET` | (生成) | Secret, Worker と共有 |
| `INTERNAL_SECRET` | (生成) | Secret, Console と共有 |
| `CONSOLE_URL` | `https://mcpist-console-dev.vercel.app` | 後で確定 |

### 出力

| 項目 | 値 |
|------|-----|
| Service URL | `https://mcpist-api-dev.onrender.com` |

---

## Step 4: Koyeb サービス作成

**状態**: ⬜ 未作成

### 手順

1. https://app.koyeb.com/ にアクセス
2. 「Create App」
3. 「Docker」を選択

### 設定

| 項目 | 値 |
|------|-----|
| App name | `mcpist-api-dev` |
| Region | Frankfurt (fra) |
| Image | `docker.io/shibaleo/mcpist-api:latest` |
| Instance type | Free / Nano |

### 環境変数

| Key | Value | 備考 |
|-----|-------|------|
| `SUPABASE_URL` | (Step 1 で取得) | |
| `SUPABASE_SERVICE_ROLE_KEY` | (Step 1 で取得) | Secret |
| `GATEWAY_SECRET` | (Step 3 と同じ) | Secret |
| `INTERNAL_SECRET` | (Step 3 と同じ) | Secret |
| `CONSOLE_URL` | `https://mcpist-console-dev.vercel.app` | 後で確定 |

### 出力

| 項目 | 値 |
|------|-----|
| Service URL | `https://mcpist-api-dev.koyeb.app` |

---

## Step 5: Cloudflare Worker デプロイ

**状態**: ✅ プロジェクト存在、設定更新が必要

### 手順

1. wrangler.toml の production 環境を更新
   ```toml
   [env.production]
   vars = {
     PRIMARY_API_URL = "https://mcpist-api-dev.onrender.com",
     SECONDARY_API_URL = "https://mcpist-api-dev.koyeb.app",
     SUPABASE_URL = "https://xxx.supabase.co",
     SUPABASE_JWKS_URL = "https://xxx.supabase.co/auth/v1/.well-known/jwks.json"
   }
   ```

2. シークレット設定
   ```bash
   cd c:\Users\shiba\HOBBY\mcpist\apps\worker
   npx wrangler secret put GATEWAY_SECRET --env production
   npx wrangler secret put SUPABASE_PUBLISHABLE_KEY --env production
   ```

3. デプロイ
   ```bash
   npx wrangler deploy --env production
   ```

### 出力

| 項目 | 値 |
|------|-----|
| Worker URL | `https://mcpist-worker.{account}.workers.dev` |

---

## Step 6: Vercel 環境変数更新

**状態**: ✅ プロジェクト存在、環境変数更新が必要

### 手順

1. https://vercel.com/ → プロジェクト → Settings → Environment Variables

### 環境変数

| Key | Value | 備考 |
|-----|-------|------|
| `NEXT_PUBLIC_SUPABASE_URL` | (Step 1 で取得) | |
| `NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY` | (Step 1 で取得) | |
| `NEXT_PUBLIC_MCP_SERVER_URL` | (Step 5 の Worker URL) | |
| `INTERNAL_SECRET` | (Step 3 と同じ) | Secret |
| `WORKER_URL` | (Step 5 の Worker URL) | 内部通信用 |

### 再デプロイ

環境変数更新後、再デプロイが必要:
- Deployments → 最新のデプロイ → Redeploy

### 出力

| 項目 | 値 |
|------|-----|
| Console URL | `https://mcpist-console-dev.vercel.app` |

---

## Step 7: DNS 設定

**状態**: ⬜ 未設定

### 手順

1. Cloudflare Dashboard → DNS → Records

### レコード設定

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| CNAME | `dev` | `mcpist-worker.{account}.workers.dev` | Proxied |

### 確認

```bash
curl https://dev.mcpist.app/health
```

---

## Step 8: 接続テスト

### テスト項目

| テスト | コマンド/手順 | 期待結果 |
|--------|--------------|---------|
| Worker Health | `curl https://dev.mcpist.app/health` | 200 OK |
| Render Health | `curl https://mcpist-api-dev.onrender.com/health` | 200 OK |
| Koyeb Health | `curl https://mcpist-api-dev.koyeb.app/health` | 200 OK |
| Console Access | ブラウザで `https://mcpist-console-dev.vercel.app` | ログイン画面表示 |
| OAuth Login | Console でログイン | Supabase 認証成功 |
| API Key 発行 | Console で API Key 作成 | キー表示 |
| MCP 接続 | Claude Code から接続 | connected |

---

## 環境変数まとめ

### 生成が必要なシークレット

| 変数名 | 用途 | 共有先 |
|--------|------|--------|
| `GATEWAY_SECRET` | Worker ↔ API Server 認証 | Worker, Render, Koyeb |
| `INTERNAL_SECRET` | Console ↔ Worker 認証 | Console, Worker |

生成コマンド:
```bash
# GATEWAY_SECRET
openssl rand -hex 32

# INTERNAL_SECRET
openssl rand -hex 32
```

### サービス別環境変数

#### Render / Koyeb (API Server)

| 変数 | 値 |
|------|-----|
| `SUPABASE_URL` | `https://xxx.supabase.co` |
| `SUPABASE_SERVICE_ROLE_KEY` | (Supabase から取得) |
| `GATEWAY_SECRET` | (生成) |
| `INTERNAL_SECRET` | (生成) |

#### Cloudflare Worker

| 変数 | 種別 | 値 |
|------|------|-----|
| `PRIMARY_API_URL` | var | `https://mcpist-api-dev.onrender.com` |
| `SECONDARY_API_URL` | var | `https://mcpist-api-dev.koyeb.app` |
| `SUPABASE_URL` | var | `https://xxx.supabase.co` |
| `SUPABASE_JWKS_URL` | var | `https://xxx.supabase.co/auth/v1/.well-known/jwks.json` |
| `GATEWAY_SECRET` | secret | (生成) |
| `SUPABASE_PUBLISHABLE_KEY` | secret | (Supabase から取得) |

#### Vercel (Console)

| 変数 | 値 |
|------|-----|
| `NEXT_PUBLIC_SUPABASE_URL` | `https://xxx.supabase.co` |
| `NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY` | (Supabase から取得) |
| `NEXT_PUBLIC_MCP_SERVER_URL` | `https://dev.mcpist.app` |
| `INTERNAL_SECRET` | (生成) |
| `WORKER_URL` | `https://mcpist-worker.xxx.workers.dev` |

---

## チェックリスト

- [ ] Step 1: Supabase 情報取得
- [ ] Step 2: DockerHub イメージ push
- [ ] Step 3: Render サービス作成
- [ ] Step 4: Koyeb サービス作成
- [ ] Step 5: Cloudflare Worker デプロイ
- [ ] Step 6: Vercel 環境変数更新
- [ ] Step 7: DNS 設定
- [ ] Step 8: 接続テスト完了

---

## トラブルシューティング

### Worker → API Server 接続エラー

1. `GATEWAY_SECRET` が一致しているか確認
2. API Server の `/health` エンドポイントにアクセスできるか確認
3. Render/Koyeb のログを確認

### Console → Worker 接続エラー

1. `INTERNAL_SECRET` が一致しているか確認
2. `WORKER_URL` が正しいか確認
3. CORS 設定を確認

### MCP 接続エラー

1. API Key が有効か確認
2. Worker のログで認証エラーを確認
3. KV キャッシュの状態を確認

---

## wrangler.toml 環境設定

各環境用のWorker設定を `apps/worker/wrangler.toml` に追加:

```toml
name = "mcpist-gateway"
main = "src/index.ts"
compatibility_date = "2024-12-30"

# ローカル開発用 KV (preview_id)
[[kv_namespaces]]
binding = "RATE_LIMIT"
id = "xxx"
preview_id = "dev-rate-limit"

[[kv_namespaces]]
binding = "HEALTH_STATE"
id = "xxx"
preview_id = "dev-health-state"

[[kv_namespaces]]
binding = "API_KEY_CACHE"
id = "xxx"
preview_id = "dev-api-key-cache"

# =============================================================================
# dev 環境
# =============================================================================
[env.dev]
name = "mcpist-gateway-dev"
vars = {
  PRIMARY_API_URL = "https://mcpist-api-dev.onrender.com",
  SECONDARY_API_URL = "https://mcpist-api-dev.koyeb.app",
  SUPABASE_URL = "https://xxx.supabase.co",
  SUPABASE_JWKS_URL = "https://xxx.supabase.co/auth/v1/.well-known/jwks.json"
}

[[env.dev.kv_namespaces]]
binding = "RATE_LIMIT"
id = "dev-rate-limit-id"

[[env.dev.kv_namespaces]]
binding = "HEALTH_STATE"
id = "dev-health-state-id"

[[env.dev.kv_namespaces]]
binding = "API_KEY_CACHE"
id = "dev-api-key-cache-id"

# =============================================================================
# stg 環境
# =============================================================================
[env.stg]
name = "mcpist-gateway-stg"
vars = {
  PRIMARY_API_URL = "https://mcpist-api-stg.onrender.com",
  SECONDARY_API_URL = "https://mcpist-api-stg.koyeb.app",
  SUPABASE_URL = "https://yyy.supabase.co",
  SUPABASE_JWKS_URL = "https://yyy.supabase.co/auth/v1/.well-known/jwks.json"
}

[[env.stg.kv_namespaces]]
binding = "RATE_LIMIT"
id = "stg-rate-limit-id"

[[env.stg.kv_namespaces]]
binding = "HEALTH_STATE"
id = "stg-health-state-id"

[[env.stg.kv_namespaces]]
binding = "API_KEY_CACHE"
id = "stg-api-key-cache-id"

# =============================================================================
# production 環境
# =============================================================================
[env.production]
name = "mcpist-gateway-prod"
vars = {
  PRIMARY_API_URL = "https://mcpist-api.onrender.com",
  SECONDARY_API_URL = "https://mcpist-api.koyeb.app",
  SUPABASE_URL = "https://zzz.supabase.co",
  SUPABASE_JWKS_URL = "https://zzz.supabase.co/auth/v1/.well-known/jwks.json"
}

[[env.production.kv_namespaces]]
binding = "RATE_LIMIT"
id = "prod-rate-limit-id"

[[env.production.kv_namespaces]]
binding = "HEALTH_STATE"
id = "prod-health-state-id"

[[env.production.kv_namespaces]]
binding = "API_KEY_CACHE"
id = "prod-api-key-cache-id"
```

### デプロイコマンド

```bash
# dev環境
wrangler deploy --env dev
wrangler secret put GATEWAY_SECRET --env dev
wrangler secret put SUPABASE_PUBLISHABLE_KEY --env dev

# stg環境
wrangler deploy --env stg
wrangler secret put GATEWAY_SECRET --env stg
wrangler secret put SUPABASE_PUBLISHABLE_KEY --env stg

# production環境
wrangler deploy --env production
wrangler secret put GATEWAY_SECRET --env production
wrangler secret put SUPABASE_PUBLISHABLE_KEY --env production
```

---

## DNS設定 (Cloudflare)

mcpist.app ドメインに以下のCNAMEレコードを追加:

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| CNAME | `dev` | `mcpist-gateway-dev.shiba-dog-leo-private.workers.dev` | Proxied |
| CNAME | `stg` | `mcpist-gateway-stg.shiba-dog-leo-private.workers.dev` | Proxied |
| CNAME | `cloud` | `mcpist-gateway-prod.shiba-dog-leo-private.workers.dev` | Proxied |
