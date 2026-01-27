# DAY011 レビュー

## Sprint概要

**目標**: mcpistリポジトリにTraefikを導入し、OAuth Serverを分離する。本番同様の開発環境を構築する。

**期間**: 2025-01-21 〜 2026-01-22

**結果**: 完了（Traefik → nginx に途中変更）

---

## 実施内容

### Phase 1: Traefik統合 → nginx移行

#### 当初計画
- Traefik v3 による `*.localhost` ドメインルーティング

#### 実際の結果
- Traefik導入後、nginx に移行
- **理由**: Traefik の動的設定機能を使用しておらず、nginx の方がシンプル

#### 成果物
| ファイル | 説明 |
|---------|------|
| `nginx/nginx.conf` | nginx 設定（domain-based routing） |
| `docker-compose.yml` | profiles対応、nginx統合 |
| `apps/*/Dockerfile.dev` | 各サービス開発用Dockerfile |
| `scripts/sync-env.js` | 環境変数同期スクリプト |

### Phase 2: OAuth Mock Server 実装

#### 技術スタック
| 項目 | 選定 | 理由 |
|------|------|------|
| フレームワーク | Hono + @hono/node-server | 軽量、TypeScript対応 |
| JWT | jose (RS256) | 業界標準、JWKS対応 |
| 鍵管理（開発） | 自動生成・ファイル永続化 | 再起動時も同じ鍵を使用 |

#### 実装したエンドポイント
| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/health` | GET | ヘルスチェック |
| `/authorize` | GET | 認可リクエスト → 同意画面へリダイレクト |
| `/authorization/:id` | GET | 認可リクエスト詳細取得 |
| `/authorization/:id/approve` | POST | 認可承認（コード発行） |
| `/authorization/:id/deny` | POST | 認可拒否 |
| `/token` | POST | 認可コード → JWT交換 |
| `/jwks` | GET | 公開鍵 (JWKS) |
| `/.well-known/oauth-authorization-server` | GET | OAuth メタデータ (RFC 8414) |

#### DBマイグレーション
- `mcpist.oauth_authorization_requests` テーブル
- `mcpist.oauth_refresh_tokens` テーブル
- 関連RPC関数

### Phase 3: Console変更

| 変更内容 | 詳細 |
|---------|------|
| `/api/auth/authorize` | OAuth Serverへリダイレクト（プロキシ化） |
| `/api/auth/token` | OAuth Serverへプロキシ |
| `/api/auth/jwks` | OAuth Serverへプロキシ |
| `/api/auth/consent` | **削除**（OAuth Serverに移行） |
| `/oauth/consent` | `authorization_id`パラメータ対応 |

### Phase 4-6: 動作確認 & E2E テスト

| テスト項目 | 結果 |
|-----------|------|
| Console UI アクセス (`console.localhost`) | 成功 |
| Supabase ログイン | 成功 |
| OAuth 認可フロー | 成功 |
| JWT 検証 (Worker) | 成功 |
| MCP Server 接続 | 成功 |
| initialize / tools/list | 成功 |

---

## 環境変数リファクタリング

### 変更内容
| 旧名 | 新名 | 役割 |
|-----|------|-----|
| `RENDER_URL` | `PRIMARY_API_URL` | Primary API Server |
| `KOYEB_URL` | `SECONDARY_API_URL` | Secondary API Server |

### 維持（変更なし）
- `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY` 等

---

## 技術的決定事項

### 1. Traefik → nginx 移行

| 項目 | Traefik | nginx |
|------|---------|-------|
| 設定方式 | ラベルベース動的設定 | 静的設定ファイル |
| ユースケース | サービスディスカバリ、K8s | シンプルなリバースプロキシ |
| 本プロジェクト | オーバースペック | 適切 |

### 2. 認証方式の整理

| 方式 | 対象 | フロー |
|------|------|--------|
| OAuth認可 | LLMチャットアプリ（Claude.ai等） | MCPクライアント → Authサーバー → JWT発行 → APIゲートウェイ |
| APIキー認証 | デスクトップ/CLI（Claude Code, Cursor等） | ユーザーがコンソールでAPIキー発行 → 設定ファイルに記載 → APIゲートウェイ |

### 3. サービス間通信

```
本番環境: サービス → パブリックDNS → 他サービス
Docker環境: サービス → nginx (network alias) → 他サービス
```

nginx に network aliases を設定することで、コンテナ内から `*.localhost` ドメインを nginx 経由で解決可能にした。

---

## 解決に時間がかかった問題

### 1. Traefik の Docker ソケット接続問題（Windows）

**症状**: Traefik が Docker daemon に接続できない
```
Error response from daemon: ""
```

**原因**: Windows Docker Desktop (WSL2) 環境で、`/var/run/docker.sock` のマウントが正しく機能しない

**試行した解決策**:
1. `//var/run/docker.sock:/var/run/docker.sock:ro` → 失敗
2. Windows named pipe → 失敗
3. `DOCKER_API_VERSION=1.44` → 失敗
4. Traefik v3.0 → v3.3 → 失敗

**最終解決**: nginx に移行

### 2. 環境変数管理の複雑さ

**問題**: monorepo での環境変数配布
- Next.js がプロジェクトディレクトリの `.env.local` を読み込む
- Worker (wrangler) が `.dev.vars` を読み込む
- 各ツールの仕様が異なる

**解決**: `scripts/sync-env.js` で一元管理

---

## アーキテクチャ図

### システム構成

canvasファイル (`mcpist-system-architecture.canvas`) に基づく構成:

**実装コンポーネント**:
- API Gateway (nginx) - ロードバランシング、ユーザー認証
- Auth Server - OAuth 2.1準拠、JWT発行
- Session Manager - ユーザーID発行、ソーシャルログイン連携
- Data Store - ユーザー情報、課金情報、ツール設定
- Token Vault - OAuthトークン、APIキー保管
- MCP Server - Auth Middleware, MCP Handler, Module Registry, Modules
- User Console - Web UI

**外部依存**:
- MCP Client (OAuth2.0 / API KEY)
- Identity Provider (Google, GitHub等)
- External Auth Server / External Service API
- Payment Service Provider (Stripe)

---

## 成果物一覧

### 新規作成
| ファイル | 説明 |
|---------|------|
| `apps/oauth/` | OAuth Mock Server (Hono) |
| `nginx/nginx.conf` | nginx 設定 |
| `scripts/sync-env.js` | 環境変数同期 |
| `supabase/migrations/00000000000005_*.sql` | oauth_authorization_requests |
| `supabase/migrations/00000000000006_*.sql` | oauth_refresh_tokens |
| `docs/test/tst-oauth-mock-server.md` | テストドキュメント |

### 削除
| ファイル | 理由 |
|---------|------|
| `traefik/default/routes.yml` | nginx に移行 |
| `traefik/infra/routes.yml` | nginx に移行 |
| `apps/console/src/app/api/auth/consent/` | OAuth Server に移行 |

---

## アクセスURL（完成）

| URL | サービス |
|-----|---------|
| http://console.localhost | Console UI |
| http://oauth.localhost | OAuth Mock Server |
| http://mcp.localhost | MCP Gateway (Worker) |
| http://api.localhost | Go Server |
| http://localhost:54323 | Supabase Studio |

---

## 反省点

1. **Traefik 採用の判断ミス**
   - 動的設定を使わないなら nginx で十分だった
   - 技術選定時に「本当に必要か？」をもっと吟味すべき

2. **環境変数管理の複雑さ**
   - 各ツール（Next.js, wrangler）の環境変数読み込み仕様を事前に把握すべきだった
   - sync-env.js スクリプトで一元管理する方針は正解だった

3. **Windows Docker環境の問題**
   - Docker ソケットマウントの問題は事前に把握すべきだった
   - クロスプラットフォーム対応を考慮した技術選定が必要

4. **`*.localhost` ドメイン解決の違い**
   - Docker 内の nginx は `*.localhost` を自身でルーティングできるが、Windows ホストからは解決できない
   - Claude Code VSCode（Windows ネイティブプロセス）から `mcp.localhost` に接続できず、`127.0.0.1` + `Host` ヘッダーで指定する必要があった
   - **教訓**: Docker 内部のネットワーク設定と、ホストOSからのアクセス方法は別問題として考慮すべき
   - `.mcp.json` 設定例:
     ```json
     {
       "url": "http://127.0.0.1:80/mcp",
       "headers": {
         "Authorization": "Bearer mpt_xxx",
         "Host": "mcp.localhost"
       }
     }
     ```

---

## 残課題（将来対応）

| ID | タスク | 優先度 | 備考 |
|----|--------|--------|------|
| B-001 | Supabase OAuth Server 有効化 | 高 | ベータ機能、本番で使用 |
| B-002 | HTTPS 設定（本番） | 高 | TLS 証明書設定 |
| B-003 | refresh_token grant テスト | 中 | 実装済み、テスト未実施 |
| B-004 | CI/CD パイプライン構築 | 中 | GitHub Actions |
| B-005 | 本番デプロイ時の鍵管理 | 高 | 環境変数で管理 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [work-log.md](./work-log.md) | 作業ログ |
| [backlog.md](./backlog.md) | タスク一覧 |
| [oauth-mock-server-plan.md](./oauth-mock-server-plan.md) | OAuth Mock Server 実装計画 |
| [oauth-server-container-plan.md](./oauth-server-container-plan.md) | Traefik統合計画 |
| [env-refactoring-plan.md](./env-refactoring-plan.md) | 環境変数リファクタリング計画 |
| [mcpist-system-architecture.md](./mcpist-system-architecture.md) | システム構成図（Mermaid） |
| [mcpist-system-architecture.canvas](mcpist-system-architecture.canvas) | システム構成図（Canvas） |
