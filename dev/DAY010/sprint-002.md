# Sprint 002: API Gateway & E2Eテスト

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-002 |
| 期間 | 2026-01-19 〜 |
| マイルストーン | M2: 本番環境構築 & E2E検証 |
| 目標 | Cloudflare API Gateway設定、Devcontainer + DinD構築、MCPクライアントからのツール呼び出しE2Eテスト |
| 状態 | 進行中 |
| 前提 | Sprint-001 完了 |

---

## スプリント目標

本番環境でMCPクライアントからツールを呼び出せる状態にする。
開発環境で本番同等のLB検証が可能な構成を構築する。

### 成果物

| タスク | 成果物 | 優先度 | 状態 |
|--------|--------|--------|------|
| Devcontainer + DinD構築 | 本番同等の開発環境 | 高 | ✅ 完了 |
| Cloudflare Worker 設定 | API Gateway（認証・Rate Limit・LB） | 高 | ✅ 完了 |
| E2Eテスト環境構築 | MCPクライアント→ツール呼び出し確認 | 高 | ✅ 完了 |
| 認証フローE2Eテスト | API Key + OAuth両方の検証 | 高 | 🔄 API Key完了、JWT未着手 |
| 本番デプロイ検証 | Render + Koyeb + Cloudflare統合 | 高 | 未着手 |
| OpenTofuモジュール作成 | インフラコード化 | 中 | ✅ 完了 |

---

## 本番構成

**構成:** Render (Primary) + Koyeb (Failover)

```
                              ┌───────────┐
                              │  Vercel   │
                              │ (Console) │
                              └─────┬─────┘
                                    │
┌───────────────────────────────────┼─────────────────────────┐
│                    Cloudflare     │                          │
│  ┌─────────────────┐  ┌──────────┴┐  ┌─────────────────┐    │
│  │     Worker      │  │     KV    │  │   DNS/Proxy     │    │
│  │  - Routing      │  │  - Health │  │                 │    │
│  │  - Load Balance │  │    状態   │  │                 │    │
│  │  - Health Check │  │  - Config │  │                 │    │
│  └────────┬────────┘  └─────┬─────┘  └─────────────────┘    │
└───────────┼─────────────────┼───────────────────────────────┘
            │                 │
            ▼                 │
     ┌──────┴──────┐          │
     ▼             ▼          │
  ┌──────┐     ┌──────┐       │
  │Render│     │Koyeb │       │
  │(Pri) │     │(Fail)│       │
  └──┬───┘     └──┬───┘       │
     └─────┬──────┘           │
           ▼                  │
     ┌───────────┐            │
     │ Supabase  │◄───────────┘
     └───────────┘
```

---

## 開発環境（Devcontainer + DinD）

```
ホストOS
└── Docker Desktop
    └── Devcontainer (DinD有効)
        │
        ├── VS Code Server + 拡張機能
        │
        ├── ツール:
        │   ├── Node.js (wrangler, next)
        │   ├── Go
        │   ├── Supabase CLI
        │   └── OpenTofu
        │
        ├── プロセス:
        │   ├── wrangler dev       (:8787)  ← Worker + KVエミュレート
        │   └── npm run dev        (:3000)  ← Next.js
        │
        └── DinDデーモン
            │
            ├── compose: api
            │   ├── api-render     (:8081)  ← Primary
            │   └── api-koyeb      (:8082)  ← Failover
            │
            └── compose: supabase (supabase start)
                ├── postgres       (:54321)
                ├── auth
                ├── storage
                └── studio         (:54323)
```

### リクエストフロー（開発環境）

```
Browser
   │
   ▼ :3000
┌─────────┐
│ Next.js │
└────┬────┘
     │
     ▼ :8787
┌─────────────────────┐
│ Worker (wrangler)   │
│                     │
│  KV: ヘルスチェック状態  │
│                     │
│  LB: 重み付け振り分け   │
│    ├─→ :8081 (50%)  │
│    └─→ :8082 (50%)  │
└──────────┬──────────┘
           │
     ┌─────┴─────┐
     ▼           ▼
┌──────────┐ ┌─────────┐
│api-render│ │api-koyeb│
│  :8081   │ │  :8082  │
└────┬─────┘ └────┬────┘
     │           │
     └─────┬─────┘
           ▼ :54321
     ┌───────────┐
     │ Supabase  │
     └───────────┘
```

---

## タスク詳細

### T-001: Devcontainer + DinD構築 ✅ 完了

**目的:** 本番環境のCloudflare Worker + KV + ロードバランシングを開発環境で完全に再現

**成果物:**
- [x] `.devcontainer/devcontainer.json` 作成
- [x] `.devcontainer/Dockerfile` 作成（DinD有効）
- [x] `.devcontainer/docker-compose.yml` 作成
- [x] `.devcontainer/post-create.sh` 作成
- [x] `compose/api.yml` 作成（APIコンテナ2台定義）

**インストール済みツール:**
- Go 1.24 + Air（ホットリロード）
- Node.js 22 + pnpm
- Docker CLI + Docker Compose Plugin
- Supabase CLI v2.72.7
- Wrangler（Cloudflare Workers CLI）
- OpenTofu（Terraform代替）

**ディレクトリ構成:**

```
mcpist/
├── .devcontainer/
│   ├── devcontainer.json
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── post-create.sh
│
├── compose/
│   └── api.yml                 # APIコンテナ2台の定義
│
├── apps/
│   ├── console/                # Next.js (UI)
│   ├── server/                 # Go API
│   └── worker/                 # Cloudflare Worker
│       ├── wrangler.toml
│       └── src/
│
├── infra/                      # OpenTofu
│   ├── modules/
│   │   ├── cloudflare/
│   │   ├── vercel/
│   │   ├── koyeb/
│   │   ├── flyio/
│   │   └── supabase/
│   └── environments/
│
└── supabase/
    └── migrations/
```

**開発環境で検証できること:**
- LBの重み付け: 複数リクエストで分散確認
- ヘルスチェック: `docker stop api-render` → KV更新 → koyebのみに流れる
- フェイルオーバー: 片方落として復旧シナリオ
- KVの状態管理: Miniflareがローカルエミュレート

---

### T-002: Cloudflare Worker API Gateway ✅ 完了

**目的:** MCP Serverの前段にAPI Gatewayを配置し、認証・Rate Limit・LBを実現

**機能:**
- [x] Cloudflare Workerプロジェクトセットアップ（`apps/worker/`）
- [x] JWT署名検証（Supabase JWKS）- `jose` ライブラリ使用
- [x] API Key検証（`mpt_*`形式）- Supabase RPC経由
- [x] グローバルRate Limit（IP単位: 1000 req/min）
- [x] Burst制限（ユーザー単位: 5 req/sec）
- [x] X-User-ID ヘッダー付与
- [x] ヘルスチェック実装
- [x] 重み付けロードバランシング（Primary/Secondary）
- [x] MCP Serverへのプロキシ
- [x] X-Gateway-Secret検証（本番環境用セキュリティ）

**構成ファイル:**

```
apps/worker/
├── package.json          # wrangler, jose依存
├── tsconfig.json
├── wrangler.toml         # 環境変数、KV設定
└── src/
    └── index.ts          # Gateway実装（438行）
```

**主要機能:**

| 機能 | 実装詳細 |
|------|---------|
| 認証 | JWT (JWKS検証) + API Key (Supabase RPC) |
| Rate Limit | KV使用、グローバル(IP/分) + バースト(ユーザー/秒) |
| ロードバランシング | 重み付けランダム選択、ヘルスチェック連動 |
| プロキシ | X-User-ID, X-Auth-Type, X-Gateway-Secret付与 |

**参考:**
- [spc-dsn.md](../../mcpist/docs/specification/spc-dsn.md) - API Gateway仕様
- [spc-inf.md](../../mcpist/docs/specification/spc-inf.md) - インフラ構成

---

### T-003: E2Eテスト環境構築 ✅ 完了

**目的:** MCPクライアントから実際にツールが呼び出せることを確認

**テストシナリオ:**
- [x] MCPクライアント → API Gateway → MCP Server → Notion API
- [x] API Key認証での接続確認
- [x] `tools/list` レスポンス確認
- [x] `tools/call` でNotionツール実行確認

**テスト方法:**
1. **手動テスト（優先）** ✅ 実施済み
   - Claude Code（CLI）から接続（Claude DesktopはHTTPS必須のため不可）
   - curl でMCPプロトコル直接確認

2. **自動E2Eテスト（任意）**
   - Go: `testing` + `httptest`
   - MCPプロトコルシミュレーション

**成功基準:** ✅ 達成
- MCPクライアントからの `initialize` が成功
- `tools/list` で get_module_schema, call, batch の3ツールが返却
- `tools/call` でモジュールツール実行可能

---

### T-004: 認証フローE2Eテスト 🔄 進行中

**目的:** 本番環境での認証ミドルウェア統合検証

**テスト対象:**
- [x] API Key認証でMCPリクエスト（正常系）
- [x] 無効なAPI Keyでの拒否（異常系）
- [ ] OAuth 2.0トークン認証でMCPリクエスト（正常系）
- [ ] 無効なトークンでの拒否（異常系）
- [ ] トークン有効期限切れ時の動作

**テスト環境:**
| 環境 | URL | 用途 |
|------|-----|------|
| ローカル | http://localhost:8787 | 開発・デバッグ（Worker経由） |
| 本番 | https://api.mcpist.app（仮） | 統合テスト |

---

### T-005: 本番デプロイ検証

**目的:** 本番環境でのシステム統合確認

**構成:** Render (Primary) + Koyeb (Failover)

**確認項目:**
- [ ] Render MCP Server デプロイ（Primary）
- [ ] Koyeb MCP Server デプロイ（Failover）
- [ ] Cloudflare Worker デプロイ
- [ ] DNS設定（api.mcpist.app等）
- [ ] SSL/TLS証明書
- [ ] ヘルスチェックエンドポイント動作
- [ ] LB動作確認（p95レイテンシベース振り分け）
- [ ] ログ・メトリクス確認

---

### T-006: OpenTofuモジュール作成 ✅ 完了

**目的:** インフラをコードで管理

**インフラ管理の分離:**
- **OpenTofu**: 何を作るか（KV namespace、DNS、プロジェクト設定等）
- **Wrangler**: 何を動かすか（Workerコード、デプロイ）

**OpenTofu管理対象:**
- [x] Cloudflare: KV namespace, DNS records, Worker設定
- [x] Vercel: Project, 環境変数, カスタムドメイン
- [x] Render: Web Service, 環境変数
- [x] Koyeb: App, Service, 環境変数
- [x] Supabase: Project設定（スキーマはCLI管理）

**作成済みファイル:**

```
infra/
├── modules/
│   ├── cloudflare/main.tf
│   ├── vercel/main.tf
│   ├── render/main.tf
│   ├── koyeb/main.tf
│   └── supabase/main.tf
└── environments/
    ├── dev/main.tf
    └── prod/main.tf
```

---

## LB戦略（Render Primary / Koyeb Failover）

p95レイテンシベースの動的ロードバランシング。

```
                        p95 レイテンシ
                             │
    ┌────────────────────────┼────────────────────────┐
    │                        │                        │
  p95 < 300ms           300ms ≤ p95 < 600ms       p95 ≥ 600ms
    │                        │                        │
    ▼                        ▼                        ▼
┌─────────┐            ┌──────────┐            ┌──────────┐
│ NORMAL  │            │ WARMUP   │            │ BALANCE  │
│         │            │          │            │          │
│ Render  │            │ Render   │            │ Render   │
│  100%   │            │  100%    │            │  50%     │
│         │            │          │            │          │
│ Koyeb   │            │ Koyeb    │            │ Koyeb    │
│ (sleep) │            │ (起動中)  │            │  50%     │
└─────────┘            └──────────┘            └──────────┘
```

**状態遷移:**

| 現状態 | 条件 | 次状態 |
|--------|------|--------|
| NORMAL | p95 ≥ 600ms | BALANCE |
| NORMAL | p95 ≥ 300ms | WARMUP |
| WARMUP | p95 ≥ 600ms | BALANCE |
| WARMUP | p95 < 300ms | NORMAL |
| BALANCE | p95 < 500ms | WARMUP |

**致命指標（即座にFAILOVER）:**
- タイムアウト（3秒）
- ヘルスチェック失敗
- fetch例外

---

## 技術的課題

### Cloudflare Worker JWKS検証 ✅ 解決済み

**課題:** Supabase JWKSからの公開鍵取得とJWT検証

**解決:**
1. `jose` ライブラリ使用（jose npm package）
2. JWKS URL: `https://<project>.supabase.co/.well-known/jwks.json`
3. KVキャッシュでJWKSをキャッシュ（TTL: 1時間）

### Rate Limit実装 ✅ 解決済み

**課題:** 分散環境でのRate Limit

**解決:**
1. Cloudflare KVでカウンター管理
2. IP単位: 1000 req/min
3. ユーザー単位: バースト 5 req/sec

### DinD ネットワーク

**課題:** Devcontainer内のDinDコンテナとホストプロセス間の通信

**解決案:**
1. Docker networkを共有
2. コンテナ名でのDNS解決
3. ポートマッピングの適切な設定

---

## 作業ログ

### 2026-01-19

#### E2Eテスト実施（ローカル環境）

**環境構築:**
- [x] `.env.local` を `apps/console/` に配置（Next.jsが読み込むため）
- [x] `apps/server/.env` 作成（Go serverの環境変数）
- [x] `main.go` の `.env` 読み込みパスを複数対応に修正

**ローカル環境での接続テスト:**

1. **curlでのMCPプロトコル確認:**
```bash
# initialize
curl -X POST http://localhost:8089/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mpt_xxx" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}'
# => {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26",...}}

# tools/list
curl -X POST http://localhost:8089/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mpt_xxx" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
# => get_module_schema, call, batch の3ツールが返却

# SSE接続確認
curl -N -H "Authorization: Bearer mpt_xxx" http://localhost:8089/mcp
# => event: endpoint / data: /mcp?sessionId=xxx
```

2. **Claude Code接続:**
- [x] `type: "sse"` で接続成功
- [x] ツール一覧が表示される
- [x] MCPクライアントからツール呼び出し可能

**注意: Claude DesktopはHTTPS必須**
- Claude Desktopのカスタムコネクタは`https://`必須
- ローカル開発（`http://localhost`）では接続不可
- **Claude Code（CLI）を使用してローカルE2Eテストを実施**

**Claude Code設定の注意点:**

| フィールド | 誤 | 正 |
|-----------|----|----|
| transport指定 | `"transport": "http"` | `"type": "sse"` |

`transport`は無効なフィールド。`type`を使用すること。

**MCP Transport種類:**

| Type | 方向 | 用途 |
|------|------|------|
| `stdio` | 双方向 | ローカルプロセス（stdin/stdout） |
| `sse` | Server→Client: SSE, Client→Server: HTTP POST | リモートサーバー接続 |
| `http` | 単発リクエスト/レスポンス | シンプルな接続 |

**推奨設定: `type: "sse"`**
- MCP公式仕様でリモートサーバー接続の標準
- 将来のプッシュ通知・進捗報告に対応可能
- MCPサーバー実装は既にSSE対応済み

**コンソールの設定例修正:**
- `apps/console/src/app/(console)/my/mcp-connection/page.tsx`
- `"transport": "http"` → `"type": "sse"` に修正済み

**本番環境でのClaude Desktop接続:**
- HTTPS必須のため、Cloudflare Worker経由でのみ接続可能
- `https://api.mcpist.app/mcp` などのHTTPS URLが必要

---

### 2026-01-20（午後）

#### 環境変数統合 & OAuth Server 切り替え実装

**環境変数の統合:**
- `.env.local` をルートに統一配置
- `dotenv-cli` で各アプリに読み込み
- `pnpm start` で全サービス一括起動（Supabase含む）

**Windows互換性:**
- `sh -c` → Go の `-C` フラグに変更
- クロスプラットフォーム対応

**ポート競合解決:**
- `dotenv-cli -v PORT=3000` で Next.js 用に上書き
- Go Server は `PORT=8089` を使用

**OAuth Server 切り替え:**
- `apps/console/src/lib/env.ts` 作成
- `ENVIRONMENT` 変数で判定（`development` | `production`）
- 開発: カスタム OAuth 実装
- 本番: Supabase OAuth Server

**変更ファイル:**
```
apps/console/src/lib/env.ts                    # 新規
apps/console/src/app/api/auth/authorize/route.ts
apps/console/src/app/api/auth/token/route.ts
apps/console/src/app/oauth/consent/            # /auth/consent から移動
.env.local                                     # ENVIRONMENT追加
.env.example                                   # ENVIRONMENT追加
package.json                                   # スクリプト修正
```

**Supabase OAuth Server について:**
- クラウド版のみ（BETA）、ローカル OSS では未サポート
- Supabase Vault は OSS でも利用可能

---

### 2026-01-20（午前）

#### Cloudflare Worker API Gateway実装完了

**T-002: Cloudflare Worker API Gateway** - ✅ 完了

- [x] Cloudflare Workerプロジェクトセットアップ（`apps/worker/`）
- [x] JWT署名検証（Supabase JWKS）- `jose` ライブラリ使用
- [x] API Key検証（`mpt_*`形式）- Supabase RPC経由
- [x] グローバルRate Limit（IP単位: 1000 req/min）
- [x] Burst制限（ユーザー単位: 5 req/sec）
- [x] X-User-ID ヘッダー付与
- [x] ヘルスチェック実装
- [x] 重み付けロードバランシング（Primary/Secondary）
- [x] MCP Serverへのプロキシ
- [x] X-Gateway-Secret検証（本番環境用セキュリティ）

**構成ファイル:**

```
apps/worker/
├── package.json          # wrangler, jose依存
├── tsconfig.json
├── wrangler.toml         # 環境変数、KV設定
└── src/
    └── index.ts          # Gateway実装（438行）
```

**主要機能:**

| 機能 | 実装詳細 |
|------|---------|
| 認証 | JWT (JWKS検証) + API Key (Supabase RPC) |
| Rate Limit | KV使用、グローバル(IP/分) + バースト(ユーザー/秒) |
| ロードバランシング | 重み付けランダム選択、ヘルスチェック連動 |
| プロキシ | X-User-ID, X-Auth-Type, X-Gateway-Secret付与 |

**Go Server Auth更新:**

- `apps/server/internal/auth/middleware.go`
- X-User-ID ヘッダー受け入れ（Gateway経由リクエスト用）
- X-Gateway-Secret 検証追加（本番環境セキュリティ）

**E2Eテスト成功:**

```bash
# Worker経由でMCPリクエスト
curl -X POST http://localhost:8787/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mpt_c8b7eae0fcb26873dcd96f8e661717e0" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'

# レスポンス
{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"mcpist","version":"0.1.0"}}}
```

**リクエストフロー確認済み:**

```
curl → Worker (8787)
       ├─ API Key検証 (Supabase RPC)
       ├─ Rate Limit (KV)
       ├─ X-User-ID付与
       └─ プロキシ → Go Server (8089)
                     ├─ X-User-ID受信
                     └─ MCP レスポンス返却
```

**本番デプロイ準備:**

1. Worker: `wrangler secret put GATEWAY_SECRET`
2. Go Server: 環境変数 `GATEWAY_SECRET` 設定
3. 本番URL設定: `wrangler.toml` の `[env.production]` セクション

**デバッグ時の学び:**

| 問題 | 原因 | 解決 |
|------|------|------|
| RPC 404 | `p_service` パラメータ欠落 | パラメータ追加 |
| API Key検証失敗 | サービス名 "mcp" vs "mcpist" | "mcpist" に修正 |
| 結果パース失敗 | RPC戻り値が配列 `[{user_id}]` | 配列として処理 |
| KV PUT失敗 | TTL 2秒 < 最小60秒 | TTL 60秒以上に |
| Go Server 401 | X-User-ID未対応 | ヘッダー処理追加 |

---

#### 残タスク

- [x] Sprint-002計画書作成
- [x] .devcontainer/ 作成
- [x] compose/api.yml 作成
- [x] infra/ OpenTofuモジュール作成
- [x] 環境変数統合（.env.local）
- [x] Windows互換スクリプト
- [x] OAuth Server 切り替え実装
- [ ] ネットワーク接続検証（DinD内）
- [ ] LB/ヘルスチェック動作検証（複数バックエンド）
- [ ] JWT認証テスト（API Keyは完了）
- [ ] Workerのデバッグログ削除（本番前）
- [ ] Render本番デプロイ（Primary）
- [ ] Koyeb本番デプロイ（Failover）
- [ ] Cloudflare Worker本番デプロイ

---

## Sprint-001からの引き継ぎ事項

### 完了済み

- [x] モノレポ構成（Turborepo + pnpm）
- [x] CI/CD設定（GitHub Actions）
- [x] Supabaseマイグレーション（ENT/TVLテーブル）
- [x] ローカル開発環境（Docker Compose）
- [x] OAuth 2.1 + PKCE認証フロー
- [x] API Key認証機能
- [x] トークン履歴/監査ログ
- [x] Notionモジュール実装

### 技術的負債（Sprint-003以降）

- [ ] useSearchParams の Suspense 対応
- [ ] 未使用コード・変数の整理
- [ ] 外見設定のDB永続化（現在localStorage）

---

## 今後のスプリント予定

| スプリント | 内容 |
|-----------|------|
| Sprint-003 | Google Calendar モジュール実装 |
| Sprint-004 | Microsoft Todo モジュール実装 |
| Sprint-005 | 課金・サブスクリプション機能 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [sprint-001.md](../DAY009/sprint-001.md) | Sprint-001計画書 |
| [backlog.md](../DAY009/backlog.md) | バックログ |
| [review.md](../DAY009/review.md) | Sprint-001レビュー |
| [spc-dsn.md](../../mcpist/docs/specification/spc-dsn.md) | 設計仕様書 |
| [spc-inf.md](../../mcpist/docs/specification/spc-inf.md) | インフラ仕様書 |
| [Notion: MCPist Devcontainer](https://www.notion.so/shibaleo/MCPist-Devcontainer-2ec2cd76e35b815eba78fad8940e1cac) | Devcontainer構成詳細 |
