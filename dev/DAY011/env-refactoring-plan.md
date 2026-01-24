# 環境変数リファクタリング計画（確定版 v5）

## 目的

1. **ベンダー固有名を排除** - Render, Koyeb 等の名前をコードから除去（API Server のみ）
2. **Supabase 環境変数は維持** - Supabaseクライアントを使用する以上、コードベースに入り込むのは避けられない
3. **無料枠オーケストレーション** - 各サービスの無料枠を組み合わせてコストゼロ運用

---

## サービス構成

### 論理的役割と責務

| 論理役割 | 責務 | 現在の実体 | 将来の候補 |
|---------|------|-----------|-----------|
| **IdP** | 実際のユーザー認証 | Google, GitHub, Microsoft | - |
| **Session Manager** | IdP 連携 + Session 管理 | Supabase Auth | Clerk, Auth0, WorkOS |
| **Token Vault** | ユーザー資産の暗号化保存 | Supabase Vault (pgsodium) | HashiCorp Vault, 自前実装 |
| **Auth Server** | OAuth 2.0 認可サーバー | 自前実装（Console内） | Supabase OAuth Server (beta) |
| **Data Store** | ビジネスデータ（RLSで保護） | Supabase DB | Neon, Turso, PlanetScale (*) |
| **API Server** | ビジネスロジック実行 | Render/Koyeb | Fly.io, Railway, Heroku |

### 重要な設計判断

```
IdP = Google, GitHub, Microsoft（実際の認証）
Session Manager = IdP 連携 + Session 管理（Session JWT 発行）
Token Vault = ユーザー資産の暗号化保存
Auth Server = OAuth 2.0 認可サーバー（OAuth JWT 発行）
Data Store = ビジネスデータ（RLSで保護）
```

**コンポーネント間の関係:**

```
IdP (Google/GitHub/Microsoft)
       │
       │ ID連携（ユーザー情報を流し込む）
       ↓
┌───────────────┐     ┌──────────────┐
│Session Manager│     │ Token Vault  │
│(Session JWT)  │     │ (ユーザー資産) │
└───────────────┘     └──────────────┘
       │                    │
       │Session JWT検証      │トークン登録/取得
       ↓                    ↓
┌──────────────┐      Console / MCP Server
│ Auth Server  │
│ (自前実装)    │
│ OAuth 2.0    │
│ (OAuth JWT)  │
└──────────────┘
```

**コンポーネントの役割:**
- **IdP**: Google, GitHub, Microsoft が実際の認証を行い、ユーザー情報を Session Manager に流し込む
- **Session Manager**: IdP と連携し、Session JWT を発行（現在: Supabase Auth、将来: Clerk, Auth0, WorkOS）
- **Token Vault**: ユーザー資産の暗号化保存（現在: Supabase Vault / pgsodium、将来: HashiCorp Vault, 自前実装）
- **Auth Server**: OAuth 2.0 認可サーバー（現在: 自前実装、将来: Supabase OAuth Server beta）
  - Session JWT を検証してユーザーを特定
  - OAuth JWT（アクセストークン）を発行
  - Vault には関与しない
  - Session Manager の選択に制約なし（Clerk でも Auth0 でも移行可能）

**データ分類の定義:**

| 分類 | 定義 | 保護方法 | 例 |
|-----|------|---------|---|
| **ユーザー資産** | ユーザーが所有するクレデンシャル | **暗号化必須**（運営も見えない） | OAuth トークン、API Key |
| **ビジネスデータ** | サービス運営に必要なデータ | RLSで保護 | 残高、課金、設定、権限 |

**なぜユーザー資産を暗号化するか:**
- OAuth トークン = ユーザーの外部サービス（Google/GitHub等）へのアクセス権
- API Key = ユーザーのサービス利用権
- これらは「ユーザーの鍵」であり、運営でさえ平文で見てはいけない

**理由:**
- Clerk, Auth0, WorkOS 等の業界標準 Auth Provider は全て Vault 機能を内包
- ユーザー資産は Vault で暗号化保存（運営も平文アクセス不可）
- プロバイダー移行時はRPC実装を変更、アプリケーションコードは変更不要

---

## 環境変数マッピング

### 方針

**Supabase 環境変数はそのまま維持:**
- Supabaseクライアント（`@supabase/supabase-js`）を使用する以上、`SUPABASE_*` 環境変数はコードベースに入り込む
- 移行時はクライアントライブラリごと変わるため、環境変数名だけ抽象化しても意味がない
- API Server の環境変数のみベンダー固有名を排除

### 環境変数の変更（API Server のみ）

| 旧名 | 新名 | 役割 |
|-----|------|-----|
| `RENDER_URL` | `PRIMARY_API_URL` | Primary API Server |
| `KOYEB_URL` | `SECONDARY_API_URL` | Secondary API Server |

### 維持する環境変数（変更なし）

| 環境変数 | 用途 |
|---------|-----|
| `SUPABASE_URL` | Supabase インスタンス URL |
| `SUPABASE_ANON_KEY` | Public Key（クライアント用） |
| `SUPABASE_SERVICE_ROLE_KEY` | Service Key（サーバー用） |
| `SUPABASE_JWKS_URL` | JWT 検証用 JWKS エンドポイント |
| `NEXT_PUBLIC_SUPABASE_URL` | クライアント側 Supabase URL |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | クライアント側 Public Key |

### Worker 内部コード変更

| 旧名 | 新名 |
|-----|-----|
| `KoyebState` | `SecondaryState` |
| `metrics.renderHealthy` | `metrics.primaryHealthy` |
| `metrics.koyebHealthy` | `metrics.secondaryHealthy` |
| `metrics.koyebState` | `metrics.secondaryState` |
| `wakeKoyeb()` | `wakeSecondary()` |

---

## 開発環境 .env.local

```env
# =============================================================================
# API Servers (Worker → Server)
# =============================================================================
PRIMARY_API_URL=http://localhost:8081
SECONDARY_API_URL=http://localhost:8082

# =============================================================================
# Supabase
# =============================================================================
SUPABASE_URL=http://localhost:54321
SUPABASE_ANON_KEY=eyJ...（anon key）
SUPABASE_SERVICE_ROLE_KEY=eyJ...（service_role key）
SUPABASE_JWKS_URL=http://localhost:54321/auth/v1/.well-known/jwks.json

# Next.js Client（ブラウザで使用）
NEXT_PUBLIC_SUPABASE_URL=http://localhost:54321
NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJ...（anon key）
```

---

## データの保存場所

| データ種別 | 分類 | 保存先 | 保護方法 |
|-----------|-----|-------|---------|
| ユーザー認証情報 | ユーザー資産 | Auth Provider | 暗号化 |
| OAuth トークン | ユーザー資産 | Auth Provider (Vault) | 暗号化（運営も見えない） |
| API Key | ユーザー資産 | Auth Provider (Vault) | 暗号化（運営も見えない） |
| クレジット残高 | ビジネスデータ | Data Store | RLS |
| 課金情報 | ビジネスデータ | Data Store | RLS |
| ユーザー属性 | ビジネスデータ | Data Store | RLS |
| 権限・ロール情報 | ビジネスデータ | Data Store | RLS |
| 接続設定 | ビジネスデータ | Data Store | RLS |

---

## 変更対象ファイル

### Phase A: Worker リファクタリング

| ファイル | 変更内容 |
|---------|---------|
| `apps/worker/src/index.ts` | Env interface（API URL 変更）+ 内部変数名変更 |
| `apps/worker/wrangler.toml` | vars 更新（API URL のみ） |
| `apps/worker/Dockerfile.dev` | env 変換スクリプト更新（API URL のみ） |

```typescript
// Before
interface Env {
  RENDER_URL: string;
  KOYEB_URL: string;
  SUPABASE_URL: string;
  SUPABASE_JWKS_URL: string;
  SUPABASE_ANON_KEY: string;
}

// After
interface Env {
  PRIMARY_API_URL: string;      // ← RENDER_URL から変更
  SECONDARY_API_URL: string;    // ← KOYEB_URL から変更
  SUPABASE_URL: string;         // 維持
  SUPABASE_JWKS_URL: string;    // 維持
  SUPABASE_ANON_KEY: string;    // 維持
}
```

### Phase B: Docker Compose 更新

| ファイル | 変更内容 |
|---------|---------|
| `docker-compose.traefik.yml` | `RENDER_URL` → `PRIMARY_API_URL`, `KOYEB_URL` → `SECONDARY_API_URL` |
| `docker-compose.yml` | 同上 |
| `compose/api.yml` | 同上 |
| `.devcontainer/docker-compose.yml` | 同上 |

### Phase C: Server リファクタリング

| ファイル | 変更内容 |
|---------|---------|
| Server 内の環境変数参照箇所 | `SUPABASE_*` をそのまま維持 |

**注**: Server も `SUPABASE_*` 環境変数をそのまま使用。変更不要。

### Phase D: .env 更新

| ファイル | 変更内容 |
|---------|---------|
| `.env.local` | `RENDER_URL` → `PRIMARY_API_URL`, `KOYEB_URL` → `SECONDARY_API_URL` |
| `.env.example` | 同上 |

---

## 実装順序

```
Phase A: Worker リファクタリング
├── 1. apps/worker/src/index.ts
│   ├── Env interface 更新（RENDER_URL → PRIMARY_API_URL, KOYEB_URL → SECONDARY_API_URL）
│   ├── 内部変数名変更 (render→primary, koyeb→secondary)
│   └── 関数名変更 (wakeKoyeb→wakeSecondary)
├── 2. apps/worker/wrangler.toml（API URL 変数名のみ）
├── 3. apps/worker/Dockerfile.dev（API URL 変数名のみ）
└── 4. ローカルテスト

Phase B: Docker Compose 更新
├── 1. docker-compose.traefik.yml（RENDER_URL → PRIMARY_API_URL, KOYEB_URL → SECONDARY_API_URL）
├── 2. docker-compose.yml（同上）
├── 3. compose/api.yml（同上）
├── 4. .devcontainer/docker-compose.yml（同上）
└── 5. Traefik モードでテスト

Phase C: Server リファクタリング
└── 変更なし（SUPABASE_* をそのまま使用）

Phase D: .env 更新
├── 1. .env.local（RENDER_URL → PRIMARY_API_URL, KOYEB_URL → SECONDARY_API_URL）
└── 2. .env.example（同上）
```

---

## 本番環境移行

### Cloudflare Workers secrets

```bash
# 新しい secrets を追加（API Server のみ）
wrangler secret put PRIMARY_API_URL
wrangler secret put SECONDARY_API_URL

# デプロイ成功後、古い secrets を削除
wrangler secret delete RENDER_URL
wrangler secret delete KOYEB_URL

# SUPABASE_* は維持（変更なし）
# - SUPABASE_URL
# - SUPABASE_ANON_KEY
# - SUPABASE_JWKS_URL
```

---

## 無料枠分散戦略

| 役割 | 候補サービス | 無料枠 |
|------|-------------|-------|
| **Auth Provider** | Clerk | 10,000 MAU |
| | Auth0 | 7,000 MAU |
| | Supabase Auth | 50,000 MAU |
| **Data Store** | Neon | 512MB + 無制限ブランチ |
| | PlanetScale | 5GB reads/month |
| | Turso | 9GB storage |
| | Supabase | 500MB |
| **API Server** | Render | 750時間/月 |
| | Koyeb | 常時2インスタンス無料 |
| | Fly.io | 3 shared VMs |

---

## 技術的懸念事項

### Data Store 候補の技術互換性

| サービス | ベースDB | RPC (ストアドプロシージャ) | RLS |
|---------|---------|--------------------------|-----|
| **Neon** | PostgreSQL | ✅ 対応 | ✅ 対応 |
| **PlanetScale** | MySQL | ❌ 非対応 | ❌ 非対応 |
| **Turso** | SQLite (libSQL) | ❌ 非対応 | ❌ 非対応 |
| **Supabase** | PostgreSQL | ✅ 対応 | ✅ 対応 |

**現状の評価:**

- **RPC**: 現在の実装は Supabase RPC に依存（`upsert_oauth_token`, `get_my_oauth_connections` 等）
  - PlanetScale/Turso 移行時はアプリケーション層での実装が必要

- **RLS**: MCPist では必須技術ではない
  - ユーザーレベルのデータマスクはアプリケーションロジックで実装
  - RLS は多層防御の保険として機能
  - PlanetScale/Turso でも運用可能（アプリ層でのアクセス制御で代替）

**移行時の考慮点:**

1. **Neon**: PostgreSQL 互換のため移行コスト最小
2. **PlanetScale/Turso**: RPC をアプリケーション層に移植する必要あり

---

## 完了要件

**本計画の完了条件:**

1. **すべての環境変数が `.env.local` に登録されていること**
   - `PRIMARY_API_URL`
   - `SECONDARY_API_URL`
   - `SUPABASE_URL`
   - `SUPABASE_ANON_KEY`
   - `SUPABASE_SERVICE_ROLE_KEY`
   - `SUPABASE_JWKS_URL`
   - `NEXT_PUBLIC_SUPABASE_URL`
   - `NEXT_PUBLIC_SUPABASE_ANON_KEY`

2. **ベンダー固有名がコードから除去されていること**
   - `RENDER_URL` → `PRIMARY_API_URL`
   - `KOYEB_URL` → `SECONDARY_API_URL`
   - 内部変数名 (render → primary, koyeb → secondary)

3. **ローカル環境で動作確認できること**
   - `pnpm dev:traefik` で起動
   - ヘルスチェックが正常動作

---

## 確認チェックリスト

- [x] Phase A: Worker リファクタリング完了
- [x] Phase B: Docker Compose 更新完了
- [x] Phase C: Server リファクタリング（変更なし）
- [x] Phase D: .env 更新完了
- [x] **完了要件 1**: すべての環境変数が `.env.local` に登録
- [x] **完了要件 2**: ベンダー固有名がコードから除去
- [x] **完了要件 3**: ローカル動作確認（`pnpm dev:traefik:infra`）
- [ ] 本番 secrets 更新
- [ ] 本番デプロイ・動作確認

---

## 実施レビュー (2026-01-21)

### 完了した作業

1. **Phase A: Worker リファクタリング**
   - `apps/worker/src/index.ts`: Env interface、型名、変数名、関数名を更新
   - `apps/worker/wrangler.toml`: 本番 vars 更新
   - `apps/worker/Dockerfile.dev`: grep パターン更新

2. **Phase B: Docker Compose 更新**
   - `docker-compose.traefik.yml`: 環境変数名更新 + Traefik設定の大幅変更
   - `.devcontainer/docker-compose.yml`: 環境変数名更新

3. **Phase D: .env 更新**
   - `.env.local` には `RENDER_URL`/`KOYEB_URL` が元々存在せず（Docker Compose で注入）
   - 変更不要

### 解決に時間がかかった問題

#### 1. Traefik の Docker ソケット接続問題

**症状**: Traefik が Docker daemon に接続できない
```
Error response from daemon: ""
```

**原因**: Windows Docker Desktop (WSL2) 環境で、`/var/run/docker.sock` のマウントが正しく機能しない

**試行した解決策**:
1. `//var/run/docker.sock:/var/run/docker.sock:ro` → 失敗
2. `//./pipe/docker_engine://./pipe/docker_engine:ro` (Windows named pipe) → 失敗
3. `DOCKER_API_VERSION=1.44` 環境変数追加 → 失敗
4. Traefik v3.0 → v3.3 アップグレード → 失敗

**最終解決策**: Docker プロバイダーからファイルプロバイダーへ切り替え
- `traefik/default/routes.yml` と `traefik/infra/routes.yml` で静的ルーティング定義
- Docker ソケットのマウントが不要になり、Windows 環境でも安定動作

#### 2. Profile の設定漏れ

**症状**: `--profile default` で起動しても一部コンテナが起動しない

**原因**: `console` と `traefik` にプロファイルが設定されていなかった

**解決策**:
- `traefik` を `traefik` (default) と `traefik-infra` (infra) に分離
- `console` に `profiles: ["default", "infra"]` を追加
- プロファイルごとに異なる Traefik ルート設定を使用

### 動作確認結果

```bash
# infra モードでの起動
$ pnpm dev:traefik:infra

# ヘルスチェック
$ curl http://mcp.localhost/health
{"status":"ok","traffic":{"primary":100,"secondary":0},"secondaryServerState":"ready","backends":{"primary":{"healthy":true},"secondary":{"healthy":true}}}

$ curl http://api.localhost/primary/health
{"status":"ok","instance":"local-primary","region":"render"}

$ curl http://api.localhost/secondary/health
{"status":"ok","instance":"local-secondary","region":"koyeb"}
```

### 追加で変更されたファイル（計画外）

| ファイル | 変更内容 |
|---------|---------|
| `traefik/default/routes.yml` | 新規作成 - default プロファイル用ルート定義 |
| `traefik/infra/routes.yml` | 新規作成 - infra プロファイル用ルート定義 |
| `package.json` | `--env-file .env.traefik` → `--env-file .env.local` |
