# DAY032 バックログ

## 日付

2026-02-19

---

## 1. OAuth Client 設定

### 1A. Clerk Dynamic Client Registration 有効化

**優先度:** HIGH
**状態:** 未着手

Clerk ダッシュボードで Dynamic Client Registration を有効化する。
MCP クライアント (Claude Desktop 等) が MCPist に OAuth 2.1 で接続する際に必要。

- Clerk Dashboard → OAuth → Dynamic Client Registration ON
- `/.well-known/oauth-authorization-server` が Clerk の discovery URL を返すように設定済み
- `lib/oauth/client.ts` が `registration_endpoint` に POST するロジック実装済み
- **検証:** MCP クライアントから OAuth フローが通るか E2E テスト

### 1B. OAuth プロバイダの App Credentials 登録

**優先度:** HIGH
**状態:** 未着手

DB リセットで `oauth_apps` テーブルが空になっている。
各プロバイダの client_id / client_secret / redirect_uri を admin API 経由で再登録する必要がある。

対象プロバイダ (11):
- Google, Microsoft, GitHub, Notion, Asana, Atlassian, Todoist, Airtable, TickTick, Trello, Dropbox

**方法:** `PUT /v1/admin/oauth/apps/{provider}` (Console admin ページ or curl)

### 1C. OAuth コールバック URL の本番設定

**優先度:** HIGH
**状態:** 未着手

各プロバイダの Developer Console で、dev 環境の redirect_uri を登録する。
- 形式: `https://dev.mcpist.app/api/oauth/{provider}/callback`
- 現在はローカル (`localhost:3000`) のみ設定されている可能性あり

---

## 2. 環境変数の設定

### 2A. Render (Go Server) 環境変数

**優先度:** CRITICAL
**状態:** 一部完了

| 変数 | 状態 | 備考 |
|------|------|------|
| `DATABASE_URL` | 要設定 | Supabase pooler 接続文字列 + `?search_path=mcpist` |
| `API_KEY_PRIVATE_KEY` | 要生成・設定 | Ed25519 seed (32 bytes) を base64 エンコード |
| `ADMIN_EMAILS` | 要設定 | カンマ区切りメールアドレス |
| `GATEWAY_SECRET` | 設定済み | `dev_gateway_secret_for_local` |
| `GRAFANA_LOKI_*` | 設定済み | そのまま |
| ~~`POSTGREST_URL`~~ | 要削除 | 不要 |
| ~~`POSTGREST_API_KEY`~~ | 要削除 | 不要 |

**API_KEY_PRIVATE_KEY 生成手順:**
```bash
openssl rand -base64 32
```
この値を Render と `.env.local` の両方に設定。

### 2B. Cloudflare Workers (dev) シークレット

**優先度:** HIGH
**状態:** 基本設定完了

| シークレット | 状態 |
|-------------|------|
| `PRIMARY_API_URL` | 設定済み |
| `API_SERVER_JWKS_URL` | 設定済み |
| `CLERK_JWKS_URL` | 設定済み |
| `GATEWAY_SECRET` | 設定済み |
| `STRIPE_WEBHOOK_SECRET` | 未設定 |
| `GRAFANA_LOKI_URL` | 未設定 (optional) |
| `GRAFANA_LOKI_USER` | 未設定 (optional) |
| `GRAFANA_LOKI_API_KEY` | 未設定 (optional) |

### 2C. Console (Vercel / dev) 環境変数

**優先度:** HIGH
**状態:** 要確認

| 変数 | 備考 |
|------|------|
| `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` | Clerk publishable key |
| `CLERK_SECRET_KEY` | Clerk secret key |
| `NEXT_PUBLIC_MCP_SERVER_URL` | Worker dev URL (`https://mcpist-gateway-dev.shibaleo.workers.dev`) |
| `INTERNAL_SECRET` | Console → Worker 内部通信用 (現状使われている?) |

### 2D. Stripe Webhook 設定

**優先度:** MEDIUM
**状態:** 未着手

- Stripe Dashboard で dev 環境用の Webhook endpoint を設定
- endpoint: `https://mcpist-gateway-dev.shibaleo.workers.dev/v1/stripe/webhook`
- `STRIPE_WEBHOOK_SECRET` を Worker に設定

---

## 3. Console UI バグフィックス

### 3A. OAuth コールバックの expires_at 形式不統一

**優先度:** MEDIUM
**状態:** 未修正

OAuth コールバックルート間で `expires_at` の形式が混在:
- ISO 文字列 (`new Date(...).toISOString()`): Airtable, Dropbox, TickTick
- Unix タイムスタンプ (秒): Google, Microsoft, Notion, Atlassian

Go Server の `token.go` がどちらの形式を期待しているか確認し、統一する。

### 3B. Clerk セッション無限リダイレクト

**優先度:** HIGH
**状態:** 要調査

「Clerk: Refreshing the session token resulted in an infinite redirect loop」エラーが発生。
Clerk 側の一時的な 500 エラーが原因の可能性が高いが、再現する場合は以下を確認:
- `proxy.ts` の `clerkMiddleware()` 設定
- Clerk publishable key と secret key の一致
- ブラウザクッキーのクリア

### 3C. Console 本番 (dev) デプロイ設定

**優先度:** HIGH
**状態:** 未着手

Console を Vercel にデプロイする際の設定:
- `NEXT_PUBLIC_MCP_SERVER_URL` をローカルから dev Worker URL に変更
- Clerk のリダイレクト URL をデプロイ先ドメインに設定
- ビルドエラーがないか確認

---

## 4. 環境変数最小化

### 4A. INTERNAL_SECRET の廃止検討

**優先度:** MEDIUM
**状態:** 要調査

`INTERNAL_SECRET` が Console と Worker の両方に設定されているが、
Clerk JWT 認証に移行した今、Console → Worker 間の認証は Clerk JWT で行われるため不要の可能性。

- 使用箇所を調査し、完全に不要なら削除
- Worker の `Env` interface からも削除

### 4B. SECONDARY_API_URL の廃止検討

**優先度:** LOW
**状態:** 未使用

Worker の `Env` に `SECONDARY_API_URL` が定義されているが、実際に使われていない。
- `types.ts` から削除
- HA 構成が必要になるまで不要

### 4C. OAuth App Credentials の DB 一元管理

**優先度:** 設定済み (確認のみ)

各 OAuth プロバイダの client_id / client_secret は DB (`oauth_apps` テーブル) に格納。
Console の authorize ルートは Go Server 経由で取得するため、
Console / Worker に個別の OAuth 環境変数 (`GOOGLE_OAUTH2_CLIENT_ID` 等) は不要。

- `.env.dev` テンプレートから OAuth 個別変数を削除 → sync-env.js も更新
- **確認:** Go Server の `GetOAuthAppCredentials` が正しく動作するか

### 4D. Clerk JWKS URL の自動解決

**優先度:** LOW
**状態:** 検討中

`CLERK_JWKS_URL` は `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` から自動導出可能:
- publishable key をデコードすると Clerk インスタンス名が得られる
- `https://{instance}.clerk.accounts.dev/.well-known/jwks.json`

ただし Worker 側で publishable key を持っていないため、現状維持が無難。

### 4E. Go Server の環境変数整理

**優先度:** MEDIUM
**状態:** 未着手

`.env.dev` テンプレートに Go Server 固有の変数が分散している。整理:

| 必須 | 変数 |
|------|------|
| DB | `DATABASE_URL` |
| 認証 | `API_KEY_PRIVATE_KEY` |
| セキュリティ | `GATEWAY_SECRET` |
| 管理 | `ADMIN_EMAILS` |
| オプション | `PORT`, `INSTANCE_ID`, `INSTANCE_REGION` |
| Observability | `GRAFANA_LOKI_URL`, `GRAFANA_LOKI_USER`, `GRAFANA_LOKI_API_KEY` |

---

## 5. ユーザークレデンシャルの暗号化とシークレットローテーション

### 5A. アプリ層暗号化の導入

**優先度:** HIGH
**状態:** 未着手

現在 `user_credentials.credentials` は平文 JSON で DB に格納されている。
AES-256-GCM でアプリ層暗号化を導入する。

**設計:**
- 暗号化/復号は Go Server 側で実施 (DB に入る前に暗号化、読み出し時に復号)
- 暗号化キーは環境変数 `CREDENTIAL_ENCRYPTION_KEY` (base64 encoded 32 bytes)
- 暗号化フォーマット: `{v: <key_version>, iv: <base64>, ct: <base64>}`
- `user_credentials` テーブルにカラム追加は不要 (既存の `credentials` カラムに暗号化テキストを格納)

**実装箇所:**
- `apps/server/internal/crypto/` — 新規パッケージ: `Encrypt(plaintext, key)`, `Decrypt(ciphertext, keys)`
- `apps/server/internal/db/repo_credentials.go` — `UpsertCredential` で暗号化、`GetCredential` で復号
- `apps/server/internal/broker/token.go` — `fetchCredentials` で復号済みデータを受け取る

### 5B. OAuth App Credentials の暗号化

**優先度:** HIGH
**状態:** 未着手

`oauth_apps.credentials` (provider の client_secret 等) も同様に暗号化。
- admin API (`PUT /v1/admin/oauth/apps/{provider}`) で設定時に暗号化
- `GetOAuthAppCredentials` で読み出し時に復号

### 5C. 暗号化キーのローテーション

**優先度:** MEDIUM
**状態:** 設計のみ

**方式:** Key Versioning
- 環境変数で複数キーをサポート: `CREDENTIAL_ENCRYPTION_KEY` (現行), `CREDENTIAL_ENCRYPTION_KEY_PREV` (旧)
- 暗号化は常に現行キーで行い、`v` (version) を付与
- 復号時に `v` を見て適切なキーを選択
- ローテーション手順:
  1. 新キーを `CREDENTIAL_ENCRYPTION_KEY` に、旧キーを `CREDENTIAL_ENCRYPTION_KEY_PREV` に設定
  2. バックグラウンドで全レコードを再暗号化するスクリプト実行
  3. 完了後 `CREDENTIAL_ENCRYPTION_KEY_PREV` を削除

### 5D. 既存データの暗号化マイグレーション

**優先度:** HIGH (5A 完了後)
**状態:** 未着手

暗号化機能をデプロイした後、既存の平文クレデンシャルを暗号化するワンタイムスクリプト:
- `user_credentials` の全レコードを読み出し → 暗号化 → 更新
- `oauth_apps` の全レコードを読み出し → 暗号化 → 更新
- 実行後に平文が残っていないことを検証

**スクリプト:** `scripts/migrate-encryption.go` (Go で実装、直接 DB 接続)

---

## 6. その他

### 6A. Go Server の Render デプロイ確認

**優先度:** CRITICAL
**状態:** 未確認

環境変数設定後、Go Server が正常に起動するか確認:
- `/health` エンドポイントで DB 接続チェック
- ログで `Database connected`, `Ed25519 key pair loaded` を確認
- `SyncModules` が成功しているか確認

### 6B. Worker → Go Server 接続テスト

**優先度:** CRITICAL
**状態:** 未確認

Worker が Go Server に正しくプロキシできるか:
- `GET /v1/modules` (パブリック、認証不要)
- `GET /v1/me/profile` (Clerk JWT 認証経由)

### 6C. `.env.dev` テンプレートの更新

**優先度:** MEDIUM
**状態:** 未着手

`.env.dev` に古い変数 (OAuth 個別変数等) が残っている。
DB 一元管理に移行した変数をテンプレートから削除し、必要最小限に絞る。

### 6D. Stripe Webhook の Go Server 移行完了確認

**優先度:** MEDIUM
**状態:** 要確認

`POST /v1/stripe/webhook` が Worker → Go Server でプロキシされているが、
Go Server 側で Stripe 署名検証が実装されているか確認。
`STRIPE_WEBHOOK_SECRET` が Go Server 側にも必要。

### 6E. seed.sql の OAuth Apps 初期データ

**優先度:** LOW
**状態:** 検討

dev 環境の DB reset 後に毎回 admin API で OAuth Apps を手動登録するのは面倒。
`seed.sql` に dev 用の OAuth App credentials を入れるか検討。
(ただしシークレットを git に入れるのはセキュリティ上 NG → 別の方法を考える)

---

## 実施順序 (推奨)

```
1. Render 環境変数設定 (2A) → Go Server 起動確認 (6A)
2. Worker → Go Server 接続テスト (6B)
3. Console デプロイ設定 (3C) → Clerk リダイレクト確認 (3B)
4. OAuth App Credentials DB 登録 (1B) → コールバック URL 設定 (1C)
5. E2E: ログイン → サービス接続 → MCP ツール実行
6. expires_at 形式統一 (3A)
7. 暗号化導入 (5A, 5B) → マイグレーション (5D)
8. 環境変数最小化 (4A-4E) → .env.dev 更新 (6C)
9. Clerk DCR 有効化 (1A) → MCP OAuth フロー E2E
10. キーローテーション設計実装 (5C)
```
