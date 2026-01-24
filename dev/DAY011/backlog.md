# DAY011 バックログ

## Sprint目標

**mcpistリポジトリにTraefikを導入し、OAuth Serverを分離する。本番同様の開発環境を構築する。**

## タスク一覧

### Phase 1: Traefik統合 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-001 | `docker-compose.yml` (profiles対応) | ✅ 完了 | 1h |
| T-002 | `apps/console/Dockerfile.dev` 作成 | ✅ 完了 | 0.5h |
| T-003 | `apps/worker/Dockerfile.dev` 作成 | ✅ 完了 | 0.5h |
| T-004 | `apps/server/Dockerfile.dev` 作成 | ✅ 完了 | 0.5h |
| T-005 | `.env.local` 統合 | ✅ 完了 | 0.5h |
| T-006 | `package.json` スクリプト追加 | ✅ 完了 | 0.25h |
| T-007 | Traefik動作確認 | ✅ 完了 | 1h |

### Phase 2: OAuth Server分離 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-008 | `apps/oauth` ディレクトリ・package.json作成 | ✅ 完了 | 0.5h |
| T-009 | Hono + @hono/node-server セットアップ | ✅ 完了 | 0.5h |
| T-010 | lib/jwt.ts 実装（RS256, 自動鍵生成） | ✅ 完了 | 0.5h |
| T-011 | lib/codes.ts 実装（Supabase RPC） | ✅ 完了 | 0.5h |
| T-012 | lib/pkce.ts 実装（S256検証） | ✅ 完了 | 0.25h |
| T-013 | routes/authorize.ts 実装 | ✅ 完了 | 1h |
| T-014 | routes/token.ts 実装 | ✅ 完了 | 1h |
| T-015 | routes/authorization.ts 実装 (approve/deny) | ✅ 完了 | 0.5h |
| T-016 | routes/jwks.ts 実装 | ✅ 完了 | 0.25h |
| T-017 | routes/well-known.ts 実装 (RFC 8414) | ✅ 完了 | 0.25h |
| T-018 | OAuth Dockerfile.dev 作成 | ✅ 完了 | 0.5h |
| T-019 | docker-compose.yml に oauth 追加 | ✅ 完了 | 0.25h |
| T-020 | traefik routes.yml に oauth 追加 | ✅ 完了 | 0.25h |
| T-021 | DBマイグレーション追加 | ✅ 完了 | 0.5h |

### Phase 3: Console 変更 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-022 | `/api/auth/authorize` をプロキシ化 | ✅ 完了 | 0.5h |
| T-023 | `/api/auth/token` をプロキシ化 | ✅ 完了 | 0.5h |
| T-024 | `/api/auth/jwks` をプロキシ化 | ✅ 完了 | 0.25h |
| T-025 | `/api/auth/consent` 削除 | ✅ 完了 | 0.25h |
| T-026 | `/oauth/consent` をauthorization_id対応に変更 | ✅ 完了 | 0.5h |
| T-027 | `env.ts` に `getOAuthServerUrl()` 追加 | ✅ 完了 | 0.25h |
| T-028 | `next.config.ts` に環境変数追加 | ✅ 完了 | 0.25h |

### Phase 4: 動作確認 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-029 | OAuth Server ヘルスチェック | ✅ 完了 | 0.25h |
| T-030 | /.well-known/oauth-authorization-server 確認 | ✅ 完了 | 0.25h |
| T-031 | /jwks 確認 | ✅ 完了 | 0.25h |
| T-032 | /authorize フロー確認 | ✅ 完了 | 0.5h |
| T-033 | /authorization/:id 確認 | ✅ 完了 | 0.25h |
| T-034 | テストドキュメント作成 | ✅ 完了 | 0.25h |

## 完了条件

### Phase 1 ✅
- [x] `pnpm dev:docker` で全サービスが起動する
- [x] `console.localhost` でConsole UIにアクセスできる
- [x] `mcp.localhost` でWorkerにアクセスできる
- [x] `api.localhost` でGo Serverにアクセスできる
- [x] `localhost:8080` でTraefik Dashboardにアクセスできる

### Phase 2 ✅
- [x] `oauth.localhost` でOAuth Serverにアクセスできる
- [x] `oauth.localhost/health` でヘルスチェックが成功
- [x] `oauth.localhost/jwks` でJWKSが取得できる
- [x] `oauth.localhost/.well-known/oauth-authorization-server` でメタデータが取得できる

### Phase 3 ✅
- [x] `/authorize` が `oauth.localhost` へリダイレクト
- [x] 同意画面が `authorization_id` パラメータで動作
- [x] `/api/auth/consent` が削除されている

### Phase 4 ✅
- [x] OAuth認可フロー（authorize → consent）が動作
- [x] テストドキュメントが作成されている

## アクセスURL（完成）

| URL | サービス | 状態 |
|-----|---------|------|
| http://console.localhost | Console UI | ✅ 動作確認済 |
| http://oauth.localhost | OAuth Mock Server | ✅ 動作確認済 |
| http://mcp.localhost | MCP Gateway (Worker) | ✅ 動作確認済 |
| http://api.localhost | Go Server | ✅ 動作確認済 |
| http://localhost:8080 | Traefik Dashboard | ✅ 動作確認済 |
| http://localhost:54323 | Supabase Studio | ✅ 動作確認済 |

## 技術スタック

| 項目 | 選定 |
|------|------|
| リバースプロキシ | Traefik v3.3 |
| OAuth Server | Hono + @hono/node-server |
| JWT | jose (RS256) |
| コンテナ | Docker Compose (profiles対応) |
| DB | Supabase (ホストで起動) |

## 認証方針（設計メモ）

### MCPクライアントの2つの認証方式

| 方式 | 対象 | フロー |
|------|------|--------|
| OAuth認可 | LLMチャットアプリ（Claude.ai等） | MCPクライアント → Authサーバー → JWT発行 → APIゲートウェイ |
| APIキー認証 | デスクトップ/CLI（Claude Code, Cursor等） | ユーザーがコンソールでAPIキー発行 → 設定ファイルに記載 → APIゲートウェイ |

### APIキー検証のレイテンシ対策

**方針: 初回アクセス時キャッシュ（KV）**

```
初回: APIゲートウェイ --KVミス--> Token Vault (10-50ms) --> KVにキャッシュ
2回目以降: APIゲートウェイ --KVヒット--> (1-5ms)
```

| 項目 | 値 |
|------|-----|
| キャッシュ | Cloudflare KV |
| TTL | 1-7日 |
| キャッシュ内容 | APIキーハッシュ → ユーザーID |
| 無効化 | TTL切れで自動反映 |

**採用理由:**
- Webhook事前キャッシュは実装が複雑（全エッジロケーションへの伝播、リトライ処理等）
- 初回アクセス時の10-50msレイテンシは許容範囲
- 実装がシンプル、追加インフラ不要

### コンポーネントの役割

| コンポーネント | 役割 |
|---------------|------|
| Authサーバー (Supabase Auth) | OAuth 2.1認可フロー、JWT発行、セッション管理 |
| APIゲートウェイ | JWT検証（JWKS）、APIキー検証（KV→Token Vault）、ロードバランシング |
| Token Vault | APIキー保存、外部サービストークン保存 |

※ Authサーバーの実装はSupabase Authの機能範囲内で完結させる（自前実装を最小化）

## 成果物

### 新規作成ファイル

| ファイル | 説明 |
|---------|------|
| `apps/oauth/` | OAuth Mock Server (Hono) |
| `apps/oauth/package.json` | 依存関係定義 |
| `apps/oauth/tsconfig.json` | TypeScript設定 |
| `apps/oauth/Dockerfile.dev` | 開発用Dockerfile |
| `apps/oauth/src/index.ts` | エントリーポイント |
| `apps/oauth/src/routes/authorize.ts` | 認可エンドポイント |
| `apps/oauth/src/routes/authorization.ts` | 認可詳細/承認/拒否 |
| `apps/oauth/src/routes/token.ts` | トークン交換 |
| `apps/oauth/src/routes/jwks.ts` | JWKS |
| `apps/oauth/src/routes/well-known.ts` | OAuthメタデータ |
| `apps/oauth/src/lib/jwt.ts` | JWT署名・検証 |
| `apps/oauth/src/lib/pkce.ts` | PKCE検証 |
| `apps/oauth/src/lib/codes.ts` | 認可コード管理 |
| `supabase/migrations/00000000000005_oauth_authorization_requests.sql` | DBマイグレーション |
| `traefik/default/routes.yml` | Traefikルート定義 |
| `traefik/infra/routes.yml` | Traefikルート定義 (infra) |
| `docs/test/tst-oauth-mock-server.md` | テストドキュメント |

### 更新ファイル

| ファイル | 変更内容 |
|---------|---------|
| `docker-compose.yml` | oauth サービス追加、profiles対応 |
| `apps/console/src/lib/env.ts` | `getOAuthServerUrl()` 追加 |
| `apps/console/src/app/api/auth/authorize/route.ts` | OAuth Serverへプロキシ |
| `apps/console/src/app/api/auth/token/route.ts` | OAuth Serverへプロキシ |
| `apps/console/src/app/api/auth/jwks/route.ts` | OAuth Serverへプロキシ |
| `apps/console/src/app/oauth/consent/page.tsx` | authorization_id対応 |
| `apps/console/next.config.ts` | NEXT_PUBLIC_OAUTH_SERVER_URL追加 |

### 削除ファイル

| ファイル | 理由 |
|---------|------|
| `apps/console/src/app/api/auth/consent/` | OAuth Serverに移行 |
| `traefik/default/routes.yml` | nginx に移行 |
| `traefik/infra/routes.yml` | nginx に移行 |

---

## Phase 5-6: nginx 移行 & E2E動作確認 (2026-01-22)

### Phase 5: nginx 移行 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-035 | `nginx/nginx.conf` 作成 | ✅ 完了 | 0.5h |
| T-036 | docker-compose.yml から Traefik 削除、nginx 追加 | ✅ 完了 | 0.5h |
| T-037 | Docker network aliases 設定 | ✅ 完了 | 0.25h |
| T-038 | Traefik 関連ファイル削除 | ✅ 完了 | 0.25h |
| T-039 | `scripts/sync-env.js` 更新（console/.env.local 生成） | ✅ 完了 | 0.5h |
| T-040 | Worker Dockerfile.dev に OAUTH_JWKS_URL 追加 | ✅ 完了 | 0.25h |
| T-041 | package.json スクリプト整理 | ✅ 完了 | 0.25h |

### Phase 6: E2E動作確認 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-042 | Console UI アクセス確認 | ✅ 完了 | 0.25h |
| T-043 | Supabase ログイン確認 | ✅ 完了 | 0.25h |
| T-044 | OAuth 認可フロー確認 | ✅ 完了 | 0.5h |
| T-045 | MCP Server 接続確認 | ✅ 完了 | 0.5h |
| T-046 | DB マイグレーション適用確認 | ✅ 完了 | 0.25h |

---

## バックログ（将来対応）

| ID | タスク | 優先度 | 備考 |
|----|--------|--------|------|
| B-001 | Supabase OAuth Server 有効化 | 高 | ベータ機能、本番で使用 |
| B-002 | HTTPS 設定（本番） | 高 | TLS 証明書設定 |
| B-003 | next.config.ts のデバッグログ削除 | 低 | console.log for NEXT_PUBLIC_SUPABASE_URL |
| B-004 | refresh_token grant テスト | 中 | 実装済み、テスト未実施 |
| B-005 | CI/CD パイプライン構築 | 中 | GitHub Actions |

### CI/CD 環境構成

| 環境 | Supabase/Render/Koyeb | Cloudflare | ドメイン |
|------|----------------------|------------|---------|
| dev | shiba.dog.leo.private | shiba.dog.leo.private | dev.mcpist.app |
| stage | fukudamakoto.private | shiba.dog.leo.private | stage.mcpist.app |
| production | fukudamakoto.work | shiba.dog.leo.private | cloud.mcpist.app |

- Cloudflare は shiba.dog.leo.private で全環境を管理（Workers 環境で分離）
- DNS (`*.mcpist.app`) も shiba.dog.leo.private の Cloudflare で管理
- シークレットは GitHub Actions の environment secrets で環境ごとに分離

### 備考: OAuth Server について

| 環境 | OAuth Server | JWT 鍵管理 |
|------|-------------|-----------|
| 開発 | OAuth Mock Server (`apps/oauth`) | 自動生成・ファイル保存 |
| 本番 | Supabase OAuth Server (ベータ) | Supabase が管理 |

OAuth Mock Server は Supabase OAuth Server の動作を模擬。
本番移行時は `OAUTH_SERVER_URL` 環境変数で切り替え。
