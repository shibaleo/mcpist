# Sprint-002 レビュー（最終）

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-002 |
| 期間 | 2026-01-19 〜 2026-01-22 |
| マイルストーン | M2: 本番環境構築 & E2E検証 |
| 作業日数 | DAY010 〜 DAY011（2日間） |
| 状態 | **完了** |

---

## スプリント目標

**本番環境でMCPクライアントからツールを呼び出せる状態にする。開発環境で本番同等のLB検証が可能な構成を構築する。**

---

## 達成状況

### 成果物一覧

| タスク | 成果物 | 計画時優先度 | 状態 |
|--------|--------|-------------|------|
| Devcontainer + DinD構築 | 本番同等の開発環境 | 高 | ✅ 完了 |
| Cloudflare Worker 設定 | API Gateway（認証・Rate Limit・LB） | 高 | ✅ 完了 |
| E2Eテスト環境構築 | MCPクライアント→ツール呼び出し確認 | 高 | ✅ 完了 |
| 認証フローE2Eテスト | API Key + OAuth両方の検証 | 高 | ✅ 完了（OAuth Mock Server実装） |
| 本番デプロイ検証 | Render + Koyeb + Cloudflare統合 | 高 | 🔄 部分完了（ローカル検証完了、本番未着手） |
| OpenTofuモジュール作成 | インフラコード化 | 中 | ✅ 完了 |
| **[追加]** OAuth Mock Server | 開発用OAuth Server分離 | - | ✅ 完了 |
| **[追加]** nginx統合 | リバースプロキシ | - | ✅ 完了（Traefikから変更） |
| **[追加]** 環境変数リファクタリング | ベンダー固有名排除 | - | ✅ 完了 |
| **[追加]** システムアーキテクチャ図更新 | Canvas + 仕様書更新 | - | ✅ 完了 |

### 完了率

- **計画タスク**: 5/6（83%）
- **追加タスク**: 4/4（100%）
- **全体**: 9/10（90%）

---

## DAY010 サマリー

### 実施内容

1. **Devcontainer + DinD構築** ✅
   - `.devcontainer/` 作成
   - DinD有効化、Docker Compose統合
   - Go, Node.js, Supabase CLI, Wrangler, OpenTofu インストール

2. **Cloudflare Worker API Gateway実装** ✅
   - JWT署名検証（Supabase JWKS）
   - API Key検証（Supabase RPC）
   - グローバルRate Limit + Burst制限
   - 重み付けロードバランシング
   - X-Gateway-Secret検証

3. **E2Eテスト環境構築** ✅
   - Claude Code（CLI）から接続確認
   - curl でMCPプロトコル直接確認
   - API Key認証での接続成功

4. **OpenTofuモジュール作成** ✅
   - Cloudflare, Vercel, Render, Koyeb, Supabase モジュール

5. **環境変数統合 & OAuth Server切り替え** ✅
   - `.env.local` をルートに統一
   - `dotenv-cli` で各アプリに配布
   - `ENVIRONMENT` 変数で開発/本番切り替え

### 技術的決定

| 項目 | 決定 | 理由 |
|------|------|------|
| MCP Transport | `type: "sse"` | MCP公式仕様、将来のプッシュ通知対応 |
| ローカルテスト | Claude Code（CLI） | Claude DesktopはHTTPS必須 |
| 環境切り替え | `ENVIRONMENT` 変数 | 開発: カスタムOAuth、本番: Supabase OAuth Server |

### 躓いたポイント

1. **Windows で `sh -c` が動作しない** → Go の `-C` フラグで解決
2. **Next.js と Go Server のポート競合** → `dotenv-cli -v PORT=3000` で解決
3. **Supabase スキーマが存在しない** → `supabase stop --no-backup` で解決
4. **Supabase OAuth Server がローカルで使えない** → 環境変数で切り替え

---

## DAY011 サマリー

### 実施内容

1. **Traefik統合 → nginx移行** ✅
   - 当初Traefik v3を導入
   - Windows Docker環境でソケット接続問題発生
   - nginx に移行（シンプルで設定が分かりやすい）

2. **OAuth Mock Server実装** ✅
   - Hono + @hono/node-server
   - RS256 JWT署名（自動鍵生成・ファイル永続化）
   - authorization_id方式（Supabase OAuth Server互換）
   - 全エンドポイント実装（authorize, token, jwks, well-known）

3. **Console変更** ✅
   - `/api/auth/*` をOAuth Serverへプロキシ化
   - `/api/auth/consent` 削除
   - `/oauth/consent` を authorization_id 対応

4. **環境変数リファクタリング** ✅
   - `RENDER_URL` → `PRIMARY_API_URL`
   - `KOYEB_URL` → `SECONDARY_API_URL`
   - ベンダー固有名をコードから排除

5. **システムアーキテクチャ図更新** ✅
   - Canvas ファイル更新
   - spc-sys.md, spc-itr.md 更新
   - 新コンポーネント追加（API Gateway, Session Manager）
   - 名称変更（Entitlement Store → Data Store）

6. **E2E動作確認** ✅
   - Console UI アクセス確認
   - Supabase ログイン確認
   - OAuth 認可フロー確認
   - MCP Server 接続確認
   - initialize / tools/list 成功

### 技術的決定

| 項目 | 決定 | 理由 |
|------|------|------|
| リバースプロキシ | nginx | Traefikの動的設定機能未使用、シンプルさ優先 |
| OAuth Server | Hono + @hono/node-server | 軽量、TypeScript対応、Cloudflare Workers互換 |
| JWT署名 | RS256 | 業界標準、JWKS対応 |
| 認可フロー | authorization_id方式 | Supabase OAuth Server互換 |

### 躓いたポイント

1. **Traefik Docker ソケット接続問題（Windows）** → nginx に移行
2. **環境変数管理の複雑さ** → `scripts/sync-env.js` で一元管理
3. **502 Bad Gateway** → コンテナ起動タイミング、リロードで解消
4. **OAuth Server 関数が見つからない** → `supabase db reset` で解決

---

## 本番構成（確定）

```
                              ┌───────────────┐
                              │    Vercel     │
                              │   (Console)   │
                              └───────┬───────┘
                                      │
┌─────────────────────────────────────┼─────────────────────────────────┐
│                      Cloudflare     │                                  │
│  ┌─────────────────┐  ┌────────────┴┐  ┌─────────────────┐            │
│  │     Worker      │  │      KV     │  │    DNS/Proxy    │            │
│  │  - Routing      │  │  - Health   │  │                 │            │
│  │  - Load Balance │  │    状態     │  │                 │            │
│  │  - Health Check │  │  - Config   │  │                 │            │
│  └────────┬────────┘  └──────┬──────┘  └─────────────────┘            │
└───────────┼──────────────────┼────────────────────────────────────────┘
            │                  │
            ▼                  │
     ┌──────┴──────┐           │
     ▼             ▼           │
  ┌──────┐     ┌──────┐        │
  │Render│     │Koyeb │        │
  │(Pri) │     │(Fail)│        │
  └──┬───┘     └──┬───┘        │
     └─────┬──────┘            │
           ▼                   │
     ┌───────────┐             │
     │ Supabase  │◄────────────┘
     └───────────┘
```

---

## 開発環境（確定）

```
┌─────────────────────────────────────────────────────────────┐
│ Host Machine                                                │
│                                                             │
│   supabase start → localhost:54321                          │
│                         ↑                                   │
│                         │ host.docker.internal              │
│   ┌─────────────────────┼─────────────────────────────────┐ │
│   │ Docker Network      │                                 │ │
│   │                     │                                 │ │
│   │  nginx (:80) ───────┼─── *.localhost routing          │ │
│   │    ├── console.localhost → console:3000               │ │
│   │    ├── oauth.localhost → oauth:4000                   │ │
│   │    ├── mcp.localhost → worker:8787                    │ │
│   │    └── api.localhost → server:8089                    │ │
│   │                     │                                 │ │
│   │  console ───────────┤                                 │ │
│   │  oauth ─────────────┤                                 │ │
│   │  worker ────────────┤                                 │ │
│   │  server ────────────┘                                 │ │
│   └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### アクセスURL

| URL | サービス |
|-----|---------|
| http://console.localhost | Console UI |
| http://oauth.localhost | OAuth Mock Server |
| http://mcp.localhost | MCP Gateway (Worker) |
| http://api.localhost | Go Server |
| http://localhost:54323 | Supabase Studio |

---

## 成果物一覧

### DAY010 成果物

| カテゴリ | ファイル |
|---------|---------|
| Devcontainer | `.devcontainer/devcontainer.json`, `Dockerfile`, `docker-compose.yml`, `post-create.sh` |
| Worker | `apps/worker/` (API Gateway実装) |
| OpenTofu | `infra/modules/` (cloudflare, vercel, render, koyeb, supabase) |
| 環境設定 | `.env.local`, `.env.example` |
| ドキュメント | `sprint-002.md`, `worker-lb-design.md`, `auth-migration-plan.md` |

### DAY011 成果物

| カテゴリ | ファイル |
|---------|---------|
| OAuth Server | `apps/oauth/` (Hono + @hono/node-server) |
| nginx | `nginx/nginx.conf` |
| DB | `supabase/migrations/00000000000005_oauth_authorization_requests.sql` |
| DB | `supabase/migrations/00000000000006_oauth_refresh_tokens.sql` |
| Console | プロキシ化、authorization_id対応 |
| 環境設定 | `scripts/sync-env.js` |
| ドキュメント | `mcpist-system-architecture.canvas`, `env-refactoring-plan.md`, `oauth-mock-server-plan.md` |

### 削除されたファイル

| ファイル | 理由 |
|---------|------|
| `traefik/default/routes.yml` | nginx に移行 |
| `traefik/infra/routes.yml` | nginx に移行 |
| `apps/console/src/app/api/auth/consent/route.ts` | OAuth Server に移行 |

---

## 反省点・学び

### 技術選定

1. **Traefik採用の判断ミス**
   - 動的設定を使わないなら nginx で十分
   - 「本当に必要か？」をもっと吟味すべき

2. **Windows Docker環境の考慮不足**
   - Docker ソケットマウントの問題は事前に把握すべき
   - クロスプラットフォーム対応を考慮した技術選定が必要

### 環境変数管理

1. **monorepoでの環境変数配布は想定以上に複雑**
   - 各ツール（Next.js, wrangler）の環境変数読み込み仕様を事前に把握すべき
   - `sync-env.js` スクリプトで一元管理する方針は正解

2. **ベンダー固有名の排除**
   - `RENDER_URL` / `KOYEB_URL` → `PRIMARY_API_URL` / `SECONDARY_API_URL`
   - 将来のインフラ変更に備えた抽象化は有効

### ドキュメント

1. **アーキテクチャ図の重要性**
   - Canvas ファイルでの視覚的な設計は理解を助ける
   - 仕様書との整合性を保つことが重要

---

## 残課題（Sprint-003 以降）

### 高優先度

| ID | タスク | 備考 |
|----|--------|------|
| B-001 | 本番デプロイ | Render + Koyeb + Cloudflare + Vercel |
| B-002 | Supabase OAuth Server 有効化 | ベータ機能、本番で使用 |
| B-003 | HTTPS 設定（本番） | TLS 証明書設定 |
| B-004 | 本番 secrets 更新 | `PRIMARY_API_URL`, `SECONDARY_API_URL` |

### 中優先度

| ID | タスク | 備考 |
|----|--------|------|
| B-005 | refresh_token grant テスト | 実装済み、テスト未実施 |
| B-006 | CI/CD パイプライン構築 | GitHub Actions |
| B-007 | JWT 認証テスト（OAuth トークン） | E2E テスト |
| B-008 | Worker デバッグログ削除 | 本番デプロイ前 |

### 低優先度

| ID | タスク | 備考 |
|----|--------|------|
| B-009 | useSearchParams の Suspense 対応 | Next.js 15 対応 |
| B-010 | 外見設定の DB 永続化 | 現在 localStorage |
| B-011 | API 仕様書作成 | OpenAPI/Swagger |

---

## 次回スプリント候補

### Sprint-003: 本番デプロイ & 課金基盤

| タスク | 成果物 |
|--------|--------|
| 本番デプロイ | Render + Koyeb + Cloudflare + Vercel |
| Supabase OAuth Server 設定 | Dashboard 設定 |
| 課金基盤設計 | Stripe 連携設計 |

### Sprint-004: モジュール拡充

| タスク | 成果物 |
|--------|--------|
| Google Calendar モジュール | OAuth 連携、予定 CRUD |
| Microsoft Todo モジュール | タスク CRUD |

---

## 関連ドキュメント

### DAY010

| ドキュメント | 内容 |
|-------------|------|
| [sprint-002.md](./sprint-002.md) | Sprint-002 計画書 |
| [review.md](./review.md) | DAY010 振り返り |
| [backlog.md](./backlog.md) | バックログ |
| [worker-lb-design.md](./worker-lb-design.md) | Worker LB設計 |
| [auth-migration-plan.md](./auth-migration-plan.md) | 認証移行計画 |

### DAY011

| ドキュメント | 内容 |
|-------------|------|
| [review.md](../DAY011/review.md) | DAY011 レビュー |
| [backlog.md](../DAY011/backlog.md) | タスク一覧 |
| [work-log.md](../DAY011/work-log.md) | 作業ログ |
| [oauth-mock-server-plan.md](../DAY011/oauth-mock-server-plan.md) | OAuth Mock Server 実装計画 |
| [env-refactoring-plan.md](../DAY011/env-refactoring-plan.md) | 環境変数リファクタリング計画 |
| [mcpist-system-architecture.canvas](mcpist-system-architecture.canvas) | システム構成図 |

### 仕様書

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../../mcpist/docs/specification/spc-sys.md) | システム仕様書 |
| [spc-itr.md](../../mcpist/docs/specification/spc-itr.md) | インタラクション仕様書 |
| [spc-dsn.md](../../mcpist/docs/specification/spc-dsn.md) | 設計仕様書 |
| [spc-inf.md](../../mcpist/docs/specification/spc-inf.md) | インフラ仕様書 |
