# Sprint 003 レビュー

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-003 |
| 期間 | 2026-01-22 〜 2026-01-23 |
| マイルストーン | M3: 本番デプロイ & APIキー認証 |
| 結果 | ✅ 成功 |

---

## 達成した目標

### 1. APIキー認証機能 ✅

| 成果 | 詳細 |
|------|------|
| Console UI | APIキー管理画面 (`/my/api-keys`) |
| DB | `mcpist.api_keys` テーブル、RPC関数 |
| Worker | KVキャッシュ統合、即時無効化 |
| テスト | Claude Code から正常接続確認 |

### 2. 本番環境デプロイ ✅

| サービス | URL | プラットフォーム |
|---------|-----|-----------------|
| Console | https://dev.mcpist.app | Vercel |
| MCP API | https://mcp.dev.mcpist.app | Cloudflare Workers |
| Server (Primary) | - | Render |
| Server (Secondary) | - | Koyeb |

### 3. OAuth認可フロー完成 ✅

| クライアント | テスト結果 |
|-------------|-----------|
| Claude.ai | ✅ OAuth認可 + ツール呼び出し成功 |
| ChatGPT Desktop | ✅ OAuth認可 + ツール呼び出し成功 |
| Claude Code | ✅ APIキー認証 + ツール呼び出し成功 |

---

## アーキテクチャ決定

### サブドメイン分離方式の採用

**背景:**
パスベースルーティングとサブドメイン分離の2つの選択肢を検討。

**決定:**
サブドメイン分離を採用。

| 方式 | MCP APIレイテンシ |
|------|------------------|
| パスベース（Vercel起点） | 3ホップ |
| パスベース（Worker起点） | 2ホップ |
| **サブドメイン分離** | **2ホップ** |

**理由:**
- このプロダクトの価値はMCPサーバーであり、UIではない
- MCP APIのレイテンシ最小化が最優先
- 各サービスが直接応答、プロキシ不要

**最終構成:**
```
Console (dev.mcpist.app)
    └── Vercel直接応答

MCP API (mcp.dev.mcpist.app)
    └── Cloudflare Worker → 認証 → Render/Koyeb
```

---

## 技術的な学び

### 1. MCP OAuth認可フローには `WWW-Authenticate` ヘッダーが必須

MCPクライアント（Claude.ai等）は401レスポンスの `WWW-Authenticate` ヘッダーから認可サーバーを発見する。

```
WWW-Authenticate: Bearer resource_metadata="https://mcp.dev.mcpist.app/.well-known/oauth-protected-resource"
```

参考: RFC 9728 (OAuth Protected Resource Metadata)

### 2. Supabase OAuth Serverはオペークトークンを発行

- OAuth Serverトークンは `/auth/v1/oauth/userinfo` で検証
- 従来のSupabase Authトークンは `/auth/v1/user` で検証
- JWT署名検証は最後のフォールバック

### 3. OAuth Metadataはルートパスでアクセスされる

MCPクライアントは `/.well-known/oauth-protected-resource` にアクセスするため、`/mcp/.well-known/*` だけでなくルートパスも対応が必要。

### 4. Supabase OAuth Server (BETA) の注意点

- SDKメソッドのプロパティ名がドキュメントと異なる場合がある
- `getAuthorizationDetails()` の戻り値で認可状態を判定可能
  - `redirect_url` あり + `client` なし = 認可済み（auto-approved）
  - `redirect_url` あり + `client` あり = 初回認可（要同意）

---

## デプロイ作業手順（CI/CDたたき台）

### 今回実施したデプロイ作業フロー

以下の手順で本番デプロイを実施。将来のCI/CD自動化の参考となる。

### 1. Console (Vercel)

```bash
# Vercel CLIでデプロイ（GitHub連携でも可）
vercel --prod

# 環境変数はVercel Dashboardで設定済み
# - NEXT_PUBLIC_SUPABASE_URL
# - NEXT_PUBLIC_SUPABASE_ANON_KEY
# - SUPABASE_SERVICE_ROLE_KEY
# - INTERNAL_SECRET
```

**結果:** `https://dev.mcpist.app` で稼働

### 2. Worker (Cloudflare)

```bash
# 開発環境でテスト
wrangler dev

# dev環境にデプロイ
cd apps/worker
wrangler deploy --env dev

# シークレット設定（初回のみ）
wrangler secret put GATEWAY_SECRET --env dev
wrangler secret put SUPABASE_PUBLISHABLE_KEY --env dev
wrangler secret put INTERNAL_SECRET --env dev
```

**環境設定 (`wrangler.toml`):**
```toml
[env.dev]
name = "mcpist-gateway-dev"
vars = { ENVIRONMENT = "dev", ... }
routes = [{ pattern = "mcp.dev.mcpist.app", custom_domain = true }]

[[env.dev.kv_namespaces]]
binding = "API_KEY_CACHE"
id = "04a54e251d5b47f9bcc5d7a7143edbe7"
```

**結果:** `https://mcp.dev.mcpist.app` で稼働

### 3. Server (Render)

```bash
# Docker イメージビルド
docker build -t shibaleo/mcpist-server:latest -f apps/server/Dockerfile .

# Docker Hub にプッシュ
docker push shibaleo/mcpist-server:latest

# Render Dashboardで手動デプロイ
# Settings > Deploy Hook URL でWebhook呼び出しも可能
```

**環境変数（Render Dashboard）:**
- `SUPABASE_URL`
- `SUPABASE_PUBLISHABLE_KEY`
- `GATEWAY_SECRET`
- `ENVIRONMENT=dev`

**結果:** Render で稼働（Primary）

### 4. Server (Koyeb)

```bash
# 同じDocker イメージを使用
# Koyeb Dashboardで手動デプロイ
# Docker Hub連携で自動デプロイも可能
```

**結果:** Koyeb で稼働（Secondary/Failover）

### 5. DNS設定 (Cloudflare)

| レコード | タイプ | 値 | プロキシ |
|---------|--------|-----|---------|
| `dev` | CNAME | `cname.vercel-dns.com` | Off |
| `mcp.dev` | - | Cloudflare Workers Routes | - |

**注意:**
- `mcp.dev.mcpist.app` は Workers Routes の `custom_domain` で自動設定
- Vercel側で `dev.mcpist.app` のドメイン検証が必要

### 6. Supabase OAuth Server設定

**Authentication > OAuth Applications:**
- Application Name: `MCPist Dev`
- Redirect URIs: `https://dev.mcpist.app/oauth/consent`
- Scopes: `read`, `write`

**Authentication > URL Configuration:**
- Site URL: `https://dev.mcpist.app`
- Redirect URLs: `https://dev.mcpist.app/**`

### 作業順序のポイント

1. **Serverを先にデプロイ** - Worker がヘルスチェックできる状態にする
2. **Workerをデプロイ** - Server への疎通確認
3. **Consoleをデプロイ** - OAuth コールバック設定
4. **Supabase OAuth設定** - 最後に認可フローを有効化
5. **DNS設定** - 既存サービスへの影響を最小化

### CI/CD化の検討事項

| 項目 | 現状 | CI/CD化 |
|------|------|---------|
| Console | Vercel CLI | GitHub連携（自動） |
| Worker | wrangler CLI | GitHub Actions + wrangler |
| Server | 手動Docker push | GitHub Actions + Docker Hub |
| Secrets | 各Dashboard | GitHub environment secrets |
| DNS | 手動設定 | 初回のみ（変更なし） |

---

## 残課題（Sprint-004へ）

| 項目 | 優先度 | 備考 |
|------|--------|------|
| GitHub Actions CI/CD | 高 | 自動デプロイ環境構築 |
| デバッグログ削除 | 低 | next.config.ts |
| 運用ドキュメント整備 | 中 | デプロイ手順書 |

---

## 成果物

### 新規作成ファイル

| ファイル | 説明 |
|---------|------|
| `docs/specification/spc-dmn.md` | ドメイン仕様書 |
| `docs/specification/spc-dpl.md` | デプロイ仕様書 |

### 主要変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `apps/worker/src/index.ts` | OAuth metadata, WWW-Authenticate, トークン検証 |
| `apps/worker/wrangler.toml` | 環境別設定（dev/stg/prd） |
| `apps/console/src/app/oauth/consent/page.tsx` | Supabase OAuth SDK対応 |

### コミット

```
refactor: migrate to subdomain separation architecture
fix(worker): support root path OAuth metadata and OAuth Server token verification
```

---

## 総評

Sprint-003は2日間で以下を達成した：

1. **APIキー認証の完成** - Claude Code / Cursor からの接続が可能に
2. **本番環境デプロイ** - dev.mcpist.app / mcp.dev.mcpist.app 稼働
3. **OAuth認可フローの完成** - Claude.ai / ChatGPT Desktop から接続可能
4. **サブドメイン分離アーキテクチャへの移行** - MCP APIレイテンシ最適化

MCPistの開発環境が本番稼働し、主要なMCPクライアント（Claude.ai, ChatGPT, Claude Code）からの接続が可能になった。

次のスプリントではCI/CDパイプラインの構築と運用ドキュメントの整備を行う。
