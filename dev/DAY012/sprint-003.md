# Sprint 003: APIキー認証 & 本番デプロイ準備

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-003 |
| 期間 | 2026-01-22 〜 2026-01-23 |
| マイルストーン | M3: 本番デプロイ & APIキー認証 |
| 目標 | APIキー認証機能完成、CI/CDパイプライン構築、本番環境へのデプロイ |
| 状態 | ✅ 完了 |
| 前提 | Sprint-002 完了、DAY011 完了（OAuth Mock Server + nginx統合） |

---

## スプリント目標

1. **APIキー認証機能を完成させ、Claude Code/Cursorからの接続を可能にする**
2. **CI/CDパイプラインを構築し、自動デプロイ環境を整備する**
3. **公開開発環境（dev.mcpist.app）へのデプロイを完了する**

### 成果物

| タスク | 成果物 | 優先度 | 状態 |
|--------|--------|--------|------|
| APIキー管理機能 | Console UI + DB + Worker連携 | 高 | ✅ 完了 |
| CI/CDパイプライン | GitHub Actions ワークフロー | 高 | ⬜ 未着手 |
| dev環境デプロイ | dev.mcpist.app 稼働 | 高 | ⬜ 未着手 |
| refresh_token テスト | OAuth フロー検証完了 | 中 | ✅ 完了 |
| ドキュメント整備 | 運用・デプロイ手順書 | 中 | 🔄 進行中 |

---

## 前提: DAY011の成果

### 完成した開発環境

| URL | サービス | 状態 |
|-----|---------|------|
| http://console.localhost | Console UI | 動作確認済 |
| http://oauth.localhost | OAuth Mock Server | 動作確認済 |
| http://mcp.localhost | MCP Gateway (Worker) | 動作確認済 |
| http://api.localhost | Go Server | 動作確認済 |
| http://localhost:54323 | Supabase Studio | 動作確認済 |

### 認証方式（設計済み）

| 方式 | 対象 | フロー | 状態 |
|------|------|--------|------|
| OAuth認可 | LLMチャットアプリ（Claude.ai等） | MCPクライアント → Authサーバー → JWT発行 | 実装済み |
| **APIキー認証** | デスクトップ/CLI（Claude Code, Cursor等） | ユーザーがコンソールでAPIキー発行 | ✅ 完了 |

---

## Phase 1: APIキー認証機能実装

### 概要

デスクトップアプリ（Claude Code, Cursor等）からMCPサーバーに接続するためのAPIキー認証機能を実装する。

### アーキテクチャ

```
┌─────────────────┐
│  Claude Code    │
│  / Cursor       │
└────────┬────────┘
         │ Authorization: Bearer mpt_xxx
         ▼
┌─────────────────────────────────────────┐
│         Cloudflare Worker               │
│  ┌─────────────────────────────────┐   │
│  │  1. APIキー検証                  │   │
│  │     KVキャッシュ → Token Vault   │   │
│  │                                  │   │
│  │  2. X-User-ID ヘッダー付与       │   │
│  └─────────────────────────────────┘   │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│            MCP Server                   │
└─────────────────────────────────────────┘
```

### APIキー検証フロー

```
初回: Worker --KVミス--> Token Vault (10-50ms) --> KVにキャッシュ
2回目以降: Worker --KVヒット--> (1-5ms)
```

| 項目 | 値 |
|------|-----|
| キャッシュ | Cloudflare KV |
| TTL | ソフト: 1時間, ハード: 1日 |
| キャッシュ内容 | APIキーハッシュ → ユーザーID |
| 無効化 | Console削除時に即時無効化（Server Action → Worker `/internal/invalidate-api-key`）|

### Cloudflare KV 無料枠について

| 項目 | 無料枠 | 有料 |
|------|--------|------|
| Reads | 100,000/day | $0.50/million |
| Writes | 1,000/day | $5.00/million |

**考慮事項**: キャッシュヒットも1 readとしてカウントされる。1,000ユーザー × 100リクエスト/日 = 100,000 reads/dayでギリギリ。スケール時は有料プラン移行が必要。

### タスク

| ID | タスク | 見積 | 状態 |
|----|--------|------|------|
| T-001 | APIキーテーブル設計・DBマイグレーション | 1h | ✅ 完了 |
| T-002 | APIキー生成・管理RPC関数作成 | 1h | ✅ 完了 |
| T-003 | Console: APIキー管理画面UI作成 | 2h | ✅ 完了 |
| T-004 | Console: APIキー発行・削除API作成 | 1h | ✅ 完了 |
| T-005 | Worker: APIキー検証ミドルウェア改善 | 1h | ✅ 完了 |
| T-006 | Worker: KVキャッシュ統合 | 1h | ✅ 完了 |
| T-007 | APIキー認証E2Eテスト | 1h | ✅ 完了 |

### DB設計

```sql
-- mcpist.api_keys テーブル
CREATE TABLE mcpist.api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,                    -- キーの名前（識別用）
  key_hash TEXT NOT NULL UNIQUE,         -- SHA-256ハッシュ
  key_prefix TEXT NOT NULL,              -- mpt_xxxx（表示用）
  service TEXT NOT NULL DEFAULT 'mcpist', -- サービス名
  scopes TEXT[] DEFAULT '{}',            -- 権限スコープ
  last_used_at TIMESTAMPTZ,              -- 最終使用日時
  expires_at TIMESTAMPTZ,                -- 有効期限（NULL = 無期限）
  created_at TIMESTAMPTZ DEFAULT NOW(),
  revoked_at TIMESTAMPTZ                 -- 削除日時（論理削除）
);

-- インデックス
CREATE INDEX idx_api_keys_user_id ON mcpist.api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON mcpist.api_keys(key_hash);
```

### Console UI

**APIキー管理画面 (`/my/api-keys`)**

```
┌─────────────────────────────────────────────────────┐
│  API Keys                                           │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌───────────────────────────────────────────────┐ │
│  │ 🔑 Claude Code                                 │ │
│  │    mpt_c8b7...e0fc                            │ │
│  │    Created: 2026-01-15  Last used: 2026-01-22 │ │
│  │    [Copy] [Delete]                            │ │
│  └───────────────────────────────────────────────┘ │
│                                                     │
│  ┌───────────────────────────────────────────────┐ │
│  │ 🔑 Cursor                                      │ │
│  │    mpt_a1b2...c3d4                            │ │
│  │    Created: 2026-01-20  Last used: Never      │ │
│  │    [Copy] [Delete]                            │ │
│  └───────────────────────────────────────────────┘ │
│                                                     │
│  [+ Create New API Key]                            │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**MCP接続情報画面 (`/my/mcp-connection`)** - 2026-01-22 リファクタリング

- APIキー生成機能を削除（`/my/api-keys`に統一）
- `/my/api-keys`へのリンクを追加
- 接続テスト機能は維持

---

## Phase 2: 本番デプロイ準備

### 概要

手動で各サービスにデプロイし、動作確認後にCI/CDを構築する。

### 環境構成

| 環境       | Supabase/Render/Koyeb/Vercel | Cloudflare            | ドメイン           |
|------------|------------------------------|----------------------|-------------------|
| dev        | shiba.dog.leo.private        | shiba.dog.leo.private | dev.mcpist.app    |
| stg        | fukudamakoto.private         | shiba.dog.leo.private | stg.mcpist.app    |
| production | fukudamakoto.work            | shiba.dog.leo.private | cloud.mcpist.app  |

**注意**: Cloudflare は shiba アカウントのみ。ローカル開発環境は廃止。

### デプロイ状況（2026-01-23 更新）

| サービス | プラットフォーム | 状態 | デプロイ方法 |
|---------|-----------------|------|-------------|
| Console | Vercel | ✅ デプロイ済み | GitHub連携 |
| Worker | Cloudflare Workers | ✅ デプロイ済み | Wrangler |
| DB | Supabase | ✅ デプロイ済み | Supabase CLI |
| Server (Primary) | Render | ✅ デプロイ済み | Docker イメージ |
| Server (Secondary) | Koyeb | ✅ デプロイ済み | Docker イメージ |

### 取得済みAPIトークン

| サービス | 環境変数名 | 状態 |
|---------|-----------|------|
| Vercel | `VERCEL_API_TOKEN` | ✅ 取得済み |
| Cloudflare | `CLOUDFLARE_API_TOKEN` | ✅ 取得済み |
| Render | `RENDER_API_TOKEN` | ✅ 取得済み |
| Koyeb | `KOYEB_API_TOKEN` | ✅ 取得済み |
| Supabase | 既存 | ✅ 設定済み |

### タスク（Sprint-003スコープ）

| ID | タスク | 見積 | 状態 |
|----|--------|------|------|
| T-008 | DockerHub リポジトリ作成 | 0.25h | ✅ 完了 |
| T-009 | Go Server Docker イメージビルド & push | 0.5h | ✅ 完了 |
| T-010 | Render プロジェクト作成 & デプロイ | 0.5h | ✅ 完了 |
| T-011 | Koyeb プロジェクト作成 & デプロイ | 0.5h | ✅ 完了 |
| T-012 | Cloudflare Worker シークレット設定 & デプロイ | 0.5h | ✅ 完了 |
| T-013 | 環境変数設定・接続テスト | 1h | ✅ 完了 |
| T-014 | GitHub Actions ワークフロー作成 | 2h | ⬜ |

### 手動デプロイ手順

#### 1. DockerHub にイメージ push

```bash
cd apps/server
docker build -t <dockerhub-user>/mcpist-api:latest .
docker push <dockerhub-user>/mcpist-api:latest
```

#### 2. Render でサービス作成

- Image: `docker.io/<dockerhub-user>/mcpist-api:latest`
- 環境変数: `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY`, `GATEWAY_SECRET`, `INTERNAL_SECRET`

#### 3. Koyeb でサービス作成

- Image: `docker.io/<dockerhub-user>/mcpist-api:latest`
- 環境変数: 同上

#### 4. Cloudflare Worker デプロイ

```bash
cd apps/worker
wrangler secret put GATEWAY_SECRET
wrangler secret put SUPABASE_PUBLISHABLE_KEY
wrangler deploy --env production
```

---

## Phase 3: テスト・クリーンアップ

### タスク

| ID | タスク | 見積 | 状態 |
|----|--------|------|------|
| T-011 | refresh_token grant テスト | 0.5h | ✅ 完了 |
| T-012 | next.config.ts デバッグログ削除 | 0.25h | ⬜ |
| T-013 | ドキュメント整備 | 1h | 🔄 進行中 |

### refresh_token テストケース

| テスト | 期待結果 | 結果 |
|--------|---------|------|
| 有効なrefresh_tokenでトークン更新 | 新しいaccess_token + refresh_token発行 | ✅ 成功 |
| 無効なrefresh_tokenでトークン更新 | 400 Bad Request | ✅ 成功 |
| 期限切れrefresh_tokenでトークン更新 | 400 Bad Request | 未テスト |
| 使用済みrefresh_tokenでトークン更新 | 400 Bad Request（リプレイ攻撃対策） | ✅ 成功 |

**テスト実行日時**: 2026-01-22

### Claude Code APIキー認証テスト結果

**実行日時**: 2026-01-22 13:40 (初回), 14:26 (即時無効化テスト), 19:50 (Hostヘッダー対応)

| テスト | 期待結果 | 結果 |
|--------|---------|------|
| Claude Code VSCode から mcpist-dev 接続 | connected 表示 | ✅ 成功 |
| APIキー認証（Worker） | KVキャッシュ HIT/MISS ログ | ✅ 成功 |
| `get_module_schema` ツール呼び出し | Notion モジュールスキーマ取得 | ✅ 成功 |
| APIキー削除後の接続拒否 | 401 Unauthorized | ✅ 即時拒否 |
| Hostヘッダーでのnginxルーティング | 正常接続 | ✅ 成功 |

**APIキー削除時の動作（即時無効化実装済み）:**
- Console UIでAPIキーを削除すると、Server ActionからWorkerの `/internal/invalidate-api-key` エンドポイントを呼び出し
- WorkerがKVキャッシュを即座に削除
- 次回のMCPリクエストでCache MISS → Supabase RPC検証 → `revoked` で401 Unauthorized

**実装ファイル:**
- `apps/console/src/app/(console)/my/api-keys/actions.ts` - Server Action（キャッシュ無効化呼び出し）
- `apps/worker/src/index.ts` - `/internal/invalidate-api-key` エンドポイント
- `supabase/migrations/00000000000007_api_keys.sql` - `revoke_api_key` RPC が `key_hash` を返す

**2026-01-22 修正: INTERNAL_SECRET対応**
- `apps/worker/Dockerfile.dev` に `INTERNAL_SECRET` を追加
- これにより Console → Worker の `/internal/invalidate-api-key` が正常に認証される

**注意点（Windows環境）**:
- Docker内のnginxは `*.localhost` を自身でルーティングできるが、Windowsホストからは解決できない
- Claude Code VSCode（Windowsネイティブプロセス）から接続する場合は `Host` ヘッダーで指定が必要

**.mcp.json 設定例（ローカル開発環境）**:
```json
{
  "mcpist-dev": {
    "url": "http://localhost/mcp",
    "type": "sse",
    "headers": {
      "Authorization": "Bearer mpt_xxx...",
      "Host": "mcp.localhost"
    }
  }
}
```

**テスト方法**:
```bash
# 1. Console UIでOAuth認可フロー実行 → refresh_token取得
# 2. 有効なrefresh_tokenでトークン更新
curl -s "http://oauth.localhost/token" -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=mrt_xxx..." \
  -d "client_id=mcpist-console"
# → 新しいaccess_token + refresh_token発行 ✅

# 3. 同じrefresh_tokenを再使用（リプレイ攻撃検知）
# → {"error":"invalid_grant","error_description":"Refresh token is invalid, expired, or already used"} ✅

# 4. 無効なrefresh_tokenでトークン更新
# → {"error":"invalid_grant"} ✅
```

---

## 2026-01-22 実施内容

### 完了したタスク

1. **APIキー管理UIの統一**
   - `/my/mcp-connection` から APIキー生成機能を削除
   - `/my/api-keys` へのリンクに置き換え
   - 接続テスト機能は維持

2. **Worker Dockerfile.dev 修正**
   - `INTERNAL_SECRET` を `.dev.vars` 生成に追加
   - APIキーinvalidateが正常動作するように

3. **README.md 更新**
   - `*.localhost` ドメインの説明追加
   - Hostヘッダーでのアクセス方法追記
   - スクリプト名を現在のpackage.jsonに合わせて修正

4. **不要コードの削除**
   - `/api/apikey` route を削除（古い`oauth_tokens`テーブルを使う実装）

### コミット内容
```
refactor: unify API key management to /my/api-keys

- Remove deprecated /api/apikey route (used oauth_tokens table)
- Remove API key generation from mcp-connection page
- Add link to /my/api-keys page for key management
- Fix worker Dockerfile to include INTERNAL_SECRET for cache invalidation
- Update README with Host header access method for Docker environment
```

---

## 2026-01-23 実施内容

### 目標
OAuth認可フローを本番環境（Supabase OAuth Server）で動作させる

### 完了したタスク

1. **OAuth Mock Server依存の完全撤廃**
   - `NEXT_PUBLIC_OAUTH_SERVER_URL` 環境変数を削除
   - `ENVIRONMENT` による分岐を廃止（常にSupabase OAuth Serverを使用）
   - OAuth Mock Serverのコードは残存するが、使用しない設計に変更

2. **OAuth consent ページのSupabase SDK対応**
   - URL直接アクセスからSupabase OAuth SDK (`supabase.auth.oauth.*`) に変更
   - `getAuthorizationDetails()` で認可詳細を取得
   - `approveAuthorization()` / `denyAuthorization()` で認可処理
   - プロパティ名の修正: `redirect_to` → `redirect_url`, `scopes` → `scope`

3. **削除したファイル**
   - `apps/console/src/app/api/auth/authorize/route.ts`
   - `apps/console/src/app/api/auth/token/route.ts`
   - `apps/console/src/app/api/auth/jwks/route.ts`
   - `apps/console/src/app/api/auth/lib/codes.ts`
   - `apps/console/src/app/api/auth/lib/jwt.ts`
   - `apps/console/src/app/api/auth/lib/pkce.ts`

4. **簡素化されたファイル**
   - `apps/console/src/lib/env.ts` - 不要な関数を削除

5. **`.well-known/oauth-authorization-server` の更新**
   - Supabase OAuth Server のエンドポイントを返すように変更

### 発生した課題

1. **Supabase 認証キー形式の変更**
   - 旧: `anon`, `service_role` (JWT形式)
   - 新: `publishable`, `secret` (`sb_publishable_*`, `sb_secret_*` 形式)
   - `.env.local` を更新して対応

2. **redirect_uri の登録**
   - `http://localhost:3000/my/mcp-connection/callback` をSupabase OAuth Client設定に追加

3. **OAuth認可エラー: "authorization request cannot be processed"**
   - consent画面は正常に表示される
   - 「許可する」クリック時にSupabase OAuth Serverからエラー
   - 原因調査中（Supabase側の問題の可能性あり）

### 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `apps/console/src/app/oauth/consent/page.tsx` | Supabase OAuth SDK使用に変更 |
| `apps/console/src/app/.well-known/oauth-authorization-server/route.ts` | Supabase URLを返すように変更 |
| `apps/console/src/lib/env.ts` | 不要な関数を削除 |
| `apps/console/.env.local` | publishable key形式に更新 |

### 2026-01-23 午後: OAuth認可フロー完成

**解決した課題:**

1. **「authorization request cannot be processed」エラー**
   - 原因: 既に処理済みの認可リクエストに対して再度 `approveAuthorization()` を呼び出していた
   - 解決: `getAuthorizationDetails()` で `redirect_url` が存在し `client` が存在しない場合は「認可済み」と判定
   - 認可済みの場合は既存の `redirect_url` を直接使用してリダイレクト

2. **UX改善: 認可済み画面の追加**
   - 初回認可: 従来の同意画面（権限確認 + 許可/拒否ボタン）
   - 認可済み: 「認可済み」画面（続行ボタンのみ）
   - 管理者のみ: 「セッションを破棄して再認可」ボタンを表示

3. **管理者判定のRPC使用**
   - 問題: `supabase.from('users')` でTypeScriptエラー（usersテーブルはmcpistスキーマ）
   - 解決: 既存の `public.get_my_role()` RPCを使用
   - コード: `const { data: role } = await supabase.rpc('get_my_role')`

**変更ファイル:**
- `apps/console/src/app/oauth/consent/page.tsx` - 認可済み判定 + 管理者ボタン + RPC使用

**テスト結果:**
- Vercel本番デプロイ: ✅ 成功（ビルドエラー解消）
- OAuth認可フロー: ✅ 正常動作
- 認可済み状態の検出: ✅ 正常動作
- 管理者判定（RPC）: ✅ 正常動作

### サブドメイン分離アーキテクチャへの移行

**背景:**
当初は `dev.mcpist.app` をエントリーポイントとして、パスベースルーティング（`/mcp` → Worker、それ以外 → Vercel）を検討していた。

**検討した選択肢:**

| 方式 | 構成 | MCP APIレイテンシ |
|------|------|------------------|
| パスベース（Vercel起点） | `dev.mcpist.app/*` → Vercel、`/mcp` → Worker | 3ホップ（Vercel → Worker → Render） |
| パスベース（Worker起点） | `dev.mcpist.app/*` → Worker、Console paths → Vercel | 2ホップ（Worker → Render） |
| **サブドメイン分離** | Console: `dev.mcpist.app`、API: `mcp.dev.mcpist.app` | **2ホップ（Worker → Render）** |

**決定理由:**
- **このプロダクトの価値はMCPサーバーであり、UIではない**
- MCP APIのレイテンシ最小化が最優先
- サブドメイン分離なら各サービスが直接応答し、プロキシ不要
- Cloudflare Worker → Render/Koyeb の最短経路を実現

**最終構成:**
```
Console (dev.mcpist.app)
    └── Vercel直接応答

MCP API (mcp.dev.mcpist.app)
    └── Cloudflare Worker → 認証 → Render/Koyeb
```

**変更内容:**
- Worker名を `mcpist-gateway-production` → `mcpist-gateway-dev` に変更
- `CONSOLE_URL` 環境変数と `proxyToConsole` 関数を削除
- DNS設定: `dev.mcpist.app` → Vercel、`mcp.dev.mcpist.app` → Worker
- 仕様書追加: `spc-dmn.md`（ドメイン仕様）、`spc-dpl.md`（デプロイ仕様）

**dwhbiプロジェクトの移行:**
- `mcpist.app` → `dwhbi.mcpist.app` に移行完了
- `mcpist.app` は MCPist本番用に解放済み（未使用状態）

### 本番デプロイ完了

| サービス | URL | 状態 |
|---------|-----|------|
| Console | https://dev.mcpist.app | ✅ 稼働中 |
| Worker | https://mcp.dev.mcpist.app | ✅ 稼働中 |
| Server (Primary) | Render | ✅ 稼働中 |
| Server (Secondary) | Koyeb | ✅ 稼働中 |

### 本番接続テスト結果 ✅ 完了

| テスト | 対象 | 状態 |
|--------|------|------|
| Claude.ai MCP接続 | mcp.dev.mcpist.app | ✅ 成功 |
| ChatGPT Desktop MCP接続 | mcp.dev.mcpist.app | ✅ 成功 |

**テスト日時**: 2026-01-23

**Claude.ai テスト結果:**
- OAuth認可フロー: ✅ 正常完了（consent画面表示 → 許可 → 接続）
- ツールスキーマ取得: ✅ 14 tools, 4 resources, 3 prompts
- ツール呼び出し: ✅ 正常動作

**ChatGPT Desktop テスト結果:**
- OAuth認可フロー: ✅ 正常完了
- ツール呼び出し: ✅ 正常動作（チャット開始時にアプリを有効化）
- メタツール方式: ✅ `get_module_schema` → `call` で各モジュール呼び出し可能

### 2026-01-23 夜: Claude.ai MCP接続エラー調査

**症状:**
- Claude.aiからMCPサーバー（api.dev.mcpist.app）への接続でエラー
- DevToolsに 429 (Too Many Requests) エラー
- 「the MCP serverへの接続でエラーが発生しました」メッセージ

**dwhbi（動作する実装）との比較調査結果:**

| 項目 | dwhbi | mcpist (修正前) | mcpist (修正後) |
|------|-------|----------------|----------------|
| 401レスポンス | `WWW-Authenticate` ヘッダー付き | ヘッダーなし | ✅ ヘッダー追加 |
| メタデータパス | `/api/mcp/.well-known/oauth-protected-resource` | なし | ✅ `/mcp/.well-known/` 追加 |
| OAuth発見フロー | MCPクライアントが認可サーバーを自動発見可能 | 不可能 | ✅ 可能 |

**原因:**
Claude.aiのMCPクライアントは、401レスポンスの `WWW-Authenticate` ヘッダーから `resource_metadata` URLを取得してOAuthフローを開始する。mcpistのWorkerはこのヘッダーを返していなかったため、OAuth認可フローが開始できなかった。

**修正内容 (`apps/worker/src/index.ts`):**

1. **401レスポンスに `WWW-Authenticate` ヘッダー追加**
   ```typescript
   // WWW-Authenticate ヘッダーでOAuthフローを開始させる (RFC 9728)
   const resourceMetadataUrl = `${url.protocol}//${url.host}/mcp/.well-known/oauth-protected-resource`;
   return new Response(JSON.stringify({ error: "Unauthorized" }), {
     status: 401,
     headers: {
       "Content-Type": "application/json",
       "WWW-Authenticate": `Bearer resource_metadata="${resourceMetadataUrl}"`,
       "Access-Control-Allow-Origin": "*",
     },
   });
   ```

2. **`/mcp/.well-known/oauth-protected-resource` エンドポイント追加 (RFC 9728)**
   ```typescript
   const metadata = {
     resource: `${baseUrl}/mcp`,
     authorization_servers: [`${env.SUPABASE_URL}/auth/v1`],
     scopes_supported: ["openid", "profile", "email"],
     bearer_methods_supported: ["header"],
   };
   ```

3. **`/mcp/.well-known/oauth-authorization-server` エンドポイント追加 (RFC 8414)**
   - Supabase Authの `/auth/v1/.well-known/openid-configuration` をプロキシ

**MCP OAuth認可フロー（正しい実装）:**
```
1. MCPクライアント → GET /mcp (認証なし)
2. Worker → 401 + WWW-Authenticate: Bearer resource_metadata="https://api.dev.mcpist.app/mcp/.well-known/oauth-protected-resource"
3. MCPクライアント → GET /mcp/.well-known/oauth-protected-resource
4. Worker → { authorization_servers: ["https://xxx.supabase.co/auth/v1"] }
5. MCPクライアント → Supabase OAuth Server で認可フロー開始
6. ユーザー → consent画面で許可
7. MCPクライアント → GET /mcp (Authorization: Bearer <JWT>)
8. Worker → JWT検証 → プロキシ → 正常レスポンス
```

**参考RFC:**
- RFC 9728: OAuth 2.0 Protected Resource Metadata
- RFC 8414: OAuth 2.0 Authorization Server Metadata

### 追加修正: Supabase OAuth Serverのトークン検証

**問題:**
- OAuth認可フローは成功するが、ツール呼び出し時に401エラー
- 原因: Supabase OAuth Serverが発行するトークンはオペークトークン（JWT形式ではない）

**修正内容 (`apps/worker/src/index.ts`):**
```typescript
async function verifyJWT(token: string, env: Env): Promise<AuthResult | null> {
  // 1. OAuth Server発行トークン: /auth/v1/oauth/userinfo で検証
  try {
    const response = await fetch(`${env.SUPABASE_URL}/auth/v1/oauth/userinfo`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    if (response.ok) {
      const userInfo = await response.json() as { sub?: string };
      if (userInfo.sub) {
        return { userId: userInfo.sub, type: "jwt" };
      }
    }
  } catch (error) { /* ... */ }

  // 2. 従来のSupabase Auth トークン: /auth/v1/user で検証
  // ...existing code...

  // 3. フォールバック: JWT署名検証
  // ...existing code...
}
```

**結果:**
- Claude.ai: ✅ OAuth認可 + ツール呼び出し成功
- ChatGPT: ✅ OAuth認可 + ツール呼び出し成功

### 追加修正: ルートパスのOAuth Metadataエンドポイント

**問題:**
- MCPクライアント（Claude.ai等）がOAuth metadataを取得できない
- 原因: Workerは `/mcp/.well-known/*` でのみmetadataを返していたが、MCPクライアントはルートパス `/.well-known/*` にアクセスする

**修正内容:**
```typescript
// ルートパスと/mcpパスの両方に対応
if (url.pathname === "/.well-known/oauth-protected-resource" ||
    url.pathname === "/mcp/.well-known/oauth-protected-resource") {
  return handleOAuthProtectedResourceMetadata(request, env);
}
```

**結果:**
- `https://mcp.dev.mcpist.app/.well-known/oauth-protected-resource` が正しくmetadataを返す

### 教訓

1. **Supabase OAuth Server はまだBETA**
   - SDK メソッドのプロパティ名がドキュメントと異なる場合がある
   - `rejectAuthorization` → `denyAuthorization` など
   - `getAuthorizationDetails()` の戻り値で認可状態を判定可能
     - `redirect_url` あり + `client` なし = 認可済み（auto-approved）
     - `redirect_url` あり + `client` あり = 初回認可（要同意）

2. **Next.js 環境変数**
   - クライアントサイドで使用する変数は `NEXT_PUBLIC_` プレフィックスが必須
   - `process.env.ENVIRONMENT` はサーバーサイドのみで動作

3. **Supabase 認証キーの移行**
   - 旧JWT形式のanonキーから新しい `sb_publishable_*` 形式への移行が必要

4. **スキーマを跨ぐDBアクセス**
   - `mcpist` スキーマのテーブルに直接アクセスするとTypeScriptエラー
   - 解決策: `public` スキーマにラッパーRPCを作成（例: `public.get_my_role()`）
   - これにより型安全性を保ちつつ、スキーマ分離を維持できる

5. **MCP OAuth認可フローには `WWW-Authenticate` ヘッダーが必須**
   - MCPクライアント（Claude.ai等）は401レスポンスの `WWW-Authenticate` ヘッダーから認可サーバーを発見する
   - 形式: `Bearer resource_metadata="https://api.example.com/mcp/.well-known/oauth-protected-resource"`
   - このヘッダーがないとOAuthフローが開始できない
   - RFC 9728 (OAuth Protected Resource Metadata) に準拠

6. **Supabase OAuth Serverはオペークトークンを発行する**
   - JWT署名検証だけでは不十分
   - OAuth Serverトークンは `/auth/v1/oauth/userinfo` で検証
   - 従来のSupabase Authトークンは `/auth/v1/user` で検証
   - 検証順序: OAuth userinfo → Supabase API → JWT署名検証

7. **サブドメイン分離 vs パスベースルーティング**
   - プロダクトの価値がどこにあるかでアーキテクチャが決まる
   - MCPサーバーが価値 → MCP APIレイテンシ最小化 → サブドメイン分離
   - 管理UIが価値 → 統一ドメイン → パスベースルーティング

---

## 完了条件

### Phase 1: APIキー認証 ✅
- [x] Console画面でAPIキーを発行できる
- [x] Claude CodeからAPIキーで接続できる
- [x] APIキーを削除すると接続が即座に拒否される（KVキャッシュ即時無効化実装済み）
- [x] KVキャッシュが機能している（2回目以降のレイテンシ低下）

### Phase 2: 本番デプロイ ✅
- [x] DockerHub にイメージ push 済み
- [x] Render でサービス稼働
- [x] Koyeb でサービス稼働
- [x] Cloudflare Worker 本番デプロイ済み
- [x] 各サービス間の接続テスト完了
- [x] Claude.ai / ChatGPT MCP接続テスト完了

### Phase 3: テスト・クリーンアップ
- [x] refresh_token grantが正常に動作
- [ ] デバッグログが削除されている
- [ ] 運用ドキュメントが整備されている

---

## リスクと対策

| リスク | 影響 | 対策 |
|--------|------|------|
| Supabase OAuth Server (BETA) の仕様変更 | 本番認証フロー | Mock Serverでの開発継続、移行時期を見極め |
| Cloudflare KVの無料枠制限 | スケール時のコスト | 有料プラン移行（$0.50/million reads） |
| 複数環境の設定ミス | デプロイ失敗 | 環境ごとのシークレット管理徹底、Dry-run実施 |

---

## 技術的負債（Sprint-004以降）

| 項目 | 優先度 | 備考 |
|------|--------|------|
| useSearchParams の Suspense 対応 | 低 | Next.js警告対応 |
| 未使用コード・変数の整理 | 低 | リファクタリング |
| 外見設定のDB永続化 | 低 | 現在localStorage |
| Supabase OAuth Server 移行 | 高 | BETA → GA 後 |

---

## 次にやるべきこと

### 優先度: 高
1. **GitHub Actions ワークフロー作成 (T-014)**
   - CI/CDパイプライン構築

### 優先度: 中
2. **next.config.ts デバッグログ削除**
3. **ドキュメント整備**

### 優先度: 低
4. **技術的負債の解消**
   - useSearchParams の Suspense 対応

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [DAY011 レビュー](../DAY011/review.md) | OAuth Mock Server + nginx統合 |
| [DAY011 バックログ](../DAY011/backlog.md) | 引き継ぎ課題 |
| [Sprint-002](../DAY010/sprint-002.md) | API Gateway & E2Eテスト |
| [Sprint-001](../DAY009/sprint-001.md) | 基盤構築 |
| [システムアーキテクチャ](../DAY011/mcpist-system-architecture.md) | 構成図 |
