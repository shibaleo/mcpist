# DAY033 作業ログ

## 日付

2026-02-20

---

## コミット一覧 (8件)

| # | ハッシュ | 時刻 | メッセージ |
|---|---------|------|-----------|
| 1 | 99fabda | 01:41 | fix: correct module identification (UUID→name) across Console, harden credential/API key lifecycle |
| 2 | f1d1c7f | 01:58 | docs: worklog and backlog on day 32 |
| 3 | 9997295 | 11:36 | refactor: replace GATEWAY_SECRET with Ed25519 JWT gateway auth |
| 4 | a5e1bdd | 12:51 | fix: pass required query params to /v1/me/usage endpoint |
| 5 | 0aa3edc | 22:11 | fix: align Stripe customer linking with OpenAPI spec, harden Stripe config handling |
| 6 | 7b65748 | 23:16 | refactor: remove Supabase, migrate DB schema to database/, downgrade pnpm to 9.15.4 |
| 7 | a34d4e4 | 23:44 | chore: integrate local PostgreSQL via Docker, rewrite README, add missing turbo env vars |
| 8 | 66ec4ce | 00:40* | refactor: replace hand-written REST handlers with ogen-generated server |

> *66ec4ce は 02-21 00:40 だが day033 作業の延長として含む。

---

## 実施内容

### 1. Ed25519 JWT ゲートウェイ認証への移行

**課題:** Worker → Go Server 間の認証が共有シークレット (`GATEWAY_SECRET`) + カスタムヘッダ (`X-User-ID`, `X-User-Email`) で、ヘッダ偽装に脆弱。

**対応:**
- Worker: `gateway-token.ts` を新規作成。Ed25519 秘密鍵 (`GATEWAY_SIGNING_KEY`) で短命 JWT (30s TTL) を署名、`/.well-known/jwks.json` で公開鍵を配布
- Go Server: `auth/gateway.go` を新規作成。JWKS を 5 分キャッシュで取得し、`X-Gateway-Token` ヘッダの JWT を検証。user_id / email をクレームから抽出
- 全プロキシルート (me, admin, modules, MCP) を `X-Gateway-Token` ベースに更新
- Console: AuthProvider の無限フェッチループを修正 (ref guard + stable deps)
- Config: `GATEWAY_SECRET` → `GATEWAY_SIGNING_KEY` + `WORKER_JWKS_URL`

**変更規模:** 30 ファイル, +755/-432

### 2. OpenAPI spec 不整合の修正

**2a. usage query params (a5e1bdd)**
- Console の `GET /v1/me/usage` 呼び出しで `start`/`end` クエリパラメータが送信されていなかった
- `plan.ts` を修正し ISO 8601 日付を送信

**2b. Stripe customer_id フィールド名 (0aa3edc)**
- Go Server の JSON フィールドが `customer_id` だったが OpenAPI spec は `stripe_customer_id` を定義
- `PUT /v1/me/stripe` が 400 エラー → webhook でのユーザー解決が失敗
- Server: フィールド名を `stripe_customer_id` に修正
- Console: checkout で customer linking 失敗時にエラーを返すように変更 (503)
- sync-env: Stripe 環境変数を Console に配布

### 3. Supabase 完全除去 + ローカル PostgreSQL

**3a. Supabase 除去 (7b65748)**
- `supabase/migrations/` → `database/migrations/` に移動
- Supabase CLI config (`config.toml`), 型定義 (`types.ts`), 36 個の旧マイグレーションファイルを削除
- レガシー `schema.sql` (2,689 行) を削除
- pnpm 10 → 9.15.4 にダウングレード (Vercel ビルド互換性)

**変更規模:** 45 ファイル, -10,722 行 (大幅削減)

**3b. ローカル Docker PostgreSQL (a34d4e4)**
- `docker-compose.yml` に `database/migrations` をマウント → コンテナ起動時に自動マイグレーション
- `pnpm db:up` / `pnpm db:down` スクリプト追加、`pnpm dev` で PG コンテナ自動起動
- `.env.dev` をローカル Docker PG にデフォルト変更
- README を全面書き直し (使い方ファースト構成)
- turbo.json に `CLERK_SECRET_KEY`, `CLERK_JWKS_URL`, `STRIPE_*` を追加

### 4. ogen サーバーコード自動生成への移行

**課題:** Go Server の REST ハンドラが手書きのため、OpenAPI spec との乖離が AI コーディングで発生しやすい。

**対応:**
- `server-openapi.yaml` (990 行) を新規作成 — Go Server スコープの OpenAPI 3.0.3 spec、28 エンドポイント
- ogen v1.18.0 でサーバーコード自動生成 (16 ファイル、`internal/ogenserver/gen/`)
- `handler.go` (530 行): 全 28 エンドポイントの Handler interface 実装、DB 型 → ogen 型変換
- `security.go` (116 行): 3 層ミドルウェア (withGateway / withAuth / withAdmin) を単一の `HandleGatewayToken` に統合、operationName で分岐
- `context.go` (32 行): userID / email のコンテキストヘルパー
- `stripe.go`: Stripe webhook ハンドラを ogen スコープ外で維持 (raw body + HMAC 署名検証)
- `main.go`: `rest.NewHandler` → `gen.NewServer` に切り替え、ルート優先度を設定
- spec ギャップの修正: `POST /v1/me/register`, `GET /v1/oauth/apps/{provider}/credentials` を追加
- `GET /v1/me/usage` を日付範囲 + module 別集計に対応
- Worker spec (`openapi.yaml`) に register endpoint + `RegisterResult` schema を追加
- Console: `UsageData` 型対応、register の `as never` キャスト除去
- 旧 `internal/rest/` パッケージ (6 ファイル) を全削除

**変更規模:** 39 ファイル, +17,760/-859

---

## アーキテクチャ変更

### Before (day032)

```
Console → Worker → Go Server
                    ├── internal/rest/     (手書き REST ハンドラ)
                    ├── internal/middleware/ (3 層: withGateway/withAuth/withAdmin)
                    └── GATEWAY_SECRET      (共有シークレット認証)
```

### After (day033)

```
Console → Worker → Go Server
                    ├── internal/ogenserver/gen/  (自動生成: ルーター, デコーダー, バリデーター)
                    ├── internal/ogenserver/      (Handler + SecurityHandler 実装)
                    ├── auth/gateway.go            (Ed25519 JWT JWKS 検証)
                    └── X-Gateway-Token            (短命 JWT 認証)
```

### ルート優先度 (Go Server ServeMux)

| 優先度 | パターン | ハンドラ |
|--------|----------|----------|
| 1 | `GET /health` | インラインヘルスチェック |
| 2 | `/v1/mcp` | MCP JSON-RPC (ogen 範囲外) |
| 3 | `POST /v1/stripe/webhook` | Stripe HMAC 署名検証 (ogen 範囲外) |
| 4 | `/v1/` | ogen 自動生成サーバー (28 エンドポイント) |
| 5 | `GET /.well-known/jwks.json` | JWKS (API キー検証用) |

---

## DAY033 サマリ

| 項目 | 内容 |
|------|------|
| テーマ | セキュリティ強化 + Supabase 完全除去 + spec 駆動サーバー移行 |
| コミット数 | 8 |
| 主な成果 | Ed25519 JWT ゲートウェイ認証、Supabase 全削除 (-10k 行)、ogen サーバー自動生成 (28 EP) |
| 削除行数 | ~12,000 行 (Supabase + 旧 REST ハンドラ) |
| 追加行数 | ~18,500 行 (ogen 生成コード + spec + 実装) |
| 実質追加 | ~1,700 行 (手書き: handler.go, security.go, context.go, server-openapi.yaml, gateway.go) |

---

## 本番 (dev) 環境ステータス

- `b85f0f3` 時点で dev 環境ヘルスチェック OK 確認済み:
  ```
  $ curl -s https://mcp.dev.mcpist.app/health
  {"status":"ok","backend":{"healthy":true,"statusCode":200,"latencyMs":766}}
  ```
- ogen 移行後のデプロイはまだ (66ec4ce はコミットのみ)

---

## 未完了・次ステップ

- ogen 移行後の dev 環境デプロイ + E2E 動作確認
- Console から OAuth 認可フロー実行 → `GET /v1/oauth/apps/{provider}/credentials` の検証
- `go test ./...` のパス確認 (ogen 移行でテスト壊れている可能性)
- Worker spec と Server spec の型の一致検証 (UserProfile の `user_id` vs `id` 等)
