# DAY011 作業ログ

## 2025-01-21

### 計画策定

#### 当初計画
- OAuth Server コンテナ化 + Traefik統合を一括で実施予定

#### 計画分割
Traefik導入はmcpistリポジトリ全体に影響するため、以下の2段階に分割：

| Sprint | 内容 |
|--------|------|
| **DAY011** | mcpistにTraefik統合（リポジトリ本体への変更） |
| **DAY012** | OAuth Server分離 |

### DAY011 スコープ

mcpistリポジトリにTraefikリバースプロキシを導入し、`*.localhost` ドメインで各サービスにアクセスできるようにする。

#### ドメインマッピング

| 開発環境 | 本番環境 |
|---------|---------|
| `console.localhost` | `console.mcpist.app` |
| `mcp.localhost` | `mcp.mcpist.app` |
| `api.localhost` | `api.mcpist.app` |

#### 成果物

- `docker-compose.traefik.yml`
- `apps/console/Dockerfile.dev`
- `apps/worker/Dockerfile.dev`
- `apps/server/Dockerfile.dev`
- `.env.traefik`
- 更新された `package.json`

### 技術的決定事項

| 項目 | 決定 | 理由 |
|------|------|------|
| リバースプロキシ | Traefik v3 | 自動サービスディスカバリ、Docker統合 |
| Supabase連携 | host.docker.internal | ホストで起動、Docker内からアクセス |
| 開発モード | ボリュームマウント + ホットリロード | 開発体験向上 |

### DAY011 フェーズ構成

| Phase | 内容 | 工数 |
|-------|------|------|
| 1 | Traefik統合 | 0.5日 |
| 2 | OAuth Server分離 | 1日 |
| 3 | Console OAuth削除 + E2Eテスト | 0.5日 |
| **合計** | | **2日** |

### 次のアクション

**Phase 1: Traefik統合**
1. `docker-compose.traefik.yml` 作成
2. 各サービスのDockerfile.dev作成
3. `.env.traefik` 作成
4. `package.json` スクリプト追加
5. 動作確認

**Phase 2: OAuth Server分離**
1. `apps/oauth` ディレクトリ作成
2. 既存コードをHono形式に変換
3. docker-composeに追加

**Phase 3: Console OAuth削除 + E2Eテスト**
1. Console `/api/auth/*` 削除
2. E2Eテスト実行
3. ドキュメント更新

---

## 実施結果

### Phase 1: Traefik統合 ✅ 完了

#### 作成・更新したファイル

| ファイル | 内容 |
|---------|------|
| `docker-compose.traefik.yml` | Traefik v3 + Worker + Server の構成 |
| `docker-compose.profiles.yml` | Docker Compose profiles対応版（default/infra） |
| `apps/worker/Dockerfile.dev` | Worker開発用Dockerfile |
| `apps/server/Dockerfile.dev` | Server開発用Dockerfile（Air使用） |
| `.env.traefik` | Traefik環境変数（gitignore対象） |
| `package.json` | `dev:traefik`, `dev:traefik:infra` スクリプト追加 |

#### 技術的成果

1. **Docker Compose Profiles対応**
   - `default`: Traefik + Worker + Server（通常開発）
   - `infra`: Traefik + Worker のみ（Serverをホストで起動する場合）

2. **Worker ヘルスチェック強化**
   - エラー分類機能追加（timeout, dns_failure, connection_refused, ssl_error, http_error, unknown）
   - レスポンスにステータスコード・レイテンシ情報を追加
   - workerd内部エラー対応（DNS解決失敗時のエラー判定修正）
   - 並列fetch問題対応（Promise.allSettledでも解決せず、順次実行で対応）

3. **環境変数フロー確立**
   ```
   .env.local → docker-compose → .dev.vars（Worker）
                              → 環境変数（Server）
   ```

4. **その他の改善**
   - `.gitignore` に `apps/server/tmp/` 追加（Air成果物除外）
   - `.env.development` の内容を `.env.local` に統合、`.env.development` 削除

#### 動作確認結果

```bash
# Traefikモード起動
pnpm dev:traefik

# ヘルスチェック確認（両方稼働時）
curl http://mcp.localhost/health
# → {"primary":{"healthy":true,"statusCode":200,"latencyMs":5},"secondary":{"healthy":true,"statusCode":200,"latencyMs":3}}

# ヘルスチェック確認（Secondary停止時）
curl http://mcp.localhost/health
# → {"primary":{"healthy":true,"statusCode":200,"latencyMs":4},"secondary":{"healthy":false,"error":"dns_failure","latencyMs":3003}}
```

#### 未完了事項

- **devcontainer対応**: 断念（node_modules削除でハング、パフォーマンス問題）
  - Windows側で作成されたnode_modulesの削除がコンテナ内で極端に遅い
  - post-create.shの改善を試みたが根本解決には至らず

---

## 2026-01-21 (続き) - OAuth Mock Server 実装

### Phase 2: OAuth Server分離 ✅ 完了

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

#### 技術的決定事項

| 項目 | 決定 | 理由 |
|------|------|------|
| フレームワーク | Hono + @hono/node-server | Bun非使用、Node.js環境で動作 |
| JWT署名 | RS256 | 業界標準、JWKS対応 |
| 鍵管理（開発） | 自動生成・ファイル永続化 | 再起動時も同じ鍵を使用 |
| 鍵管理（本番） | 環境変数 | AUTH_PRIVATE_KEY, AUTH_PUBLIC_KEY |
| 認可フロー | authorization_id方式 | Supabase OAuth Server互換 |

#### DBマイグレーション

新規テーブル: `mcpist.oauth_authorization_requests`

| カラム | 型 | 説明 |
|--------|---|------|
| id | TEXT | authorization_id (PK) |
| client_id | TEXT | クライアントID |
| redirect_uri | TEXT | リダイレクトURI |
| code_challenge | TEXT | PKCE code_challenge |
| code_challenge_method | TEXT | S256 |
| scope | TEXT | スコープ |
| state | TEXT | state パラメータ |
| status | TEXT | pending/approved/denied/expired |
| user_id | UUID | 承認時に設定 |
| expires_at | TIMESTAMPTZ | 有効期限 |

新規RPC関数:
- `store_oauth_authorization_request`
- `get_oauth_authorization_request`
- `approve_oauth_authorization`
- `deny_oauth_authorization`

### Phase 3: Console変更 ✅ 完了

#### 変更内容

1. **`/api/auth/authorize`**: OAuth Serverへリダイレクト（プロキシ化）
2. **`/api/auth/token`**: OAuth Serverへプロキシ
3. **`/api/auth/jwks`**: OAuth Serverへプロキシ
4. **`/api/auth/consent`**: 削除（OAuth Serverに移行）
5. **`/oauth/consent`**: `authorization_id`パラメータ対応

#### env.ts の変更

```typescript
export function getOAuthServerUrl(): string {
  if (useSupabaseOAuthServer) {
    return `${process.env.NEXT_PUBLIC_SUPABASE_URL}/auth/v1/oauth`
  }
  return process.env.OAUTH_SERVER_URL || 'http://oauth.localhost'
}
```

### Phase 4: 動作確認 ✅ 完了

#### テスト結果

```bash
# ヘルスチェック
$ curl http://oauth.localhost/health
{"status":"ok","service":"oauth-mock-server"}

# OAuth メタデータ
$ curl http://oauth.localhost/.well-known/oauth-authorization-server
{"issuer":"http://oauth.localhost","authorization_endpoint":"http://oauth.localhost/authorize",...}

# JWKS
$ curl http://oauth.localhost/jwks
{"keys":[{"kty":"RSA","n":"...","e":"AQAB","kid":"mcpist-auth-key-1","use":"sig","alg":"RS256"}]}

# 認可リクエスト
$ curl -v 'http://oauth.localhost/authorize?response_type=code&client_id=test-client&...'
< HTTP/1.1 302 Found
< Location: http://console.localhost/oauth/consent?authorization_id=18823742ae93013a41de226043febe97

# 認可リクエスト詳細
$ curl 'http://oauth.localhost/authorization/18823742ae93013a41de226043febe97'
{"id":"18823742ae93013a41de226043febe97","client_id":"test-client","redirect_uri":"http://localhost:8080/callback",...}
```

#### コンテナ起動状態

```
CONTAINER ID   IMAGE           STATUS          NAMES
0967e89dbfcf   mcpist-oauth    Up              mcpist-oauth
21f8fb0d8d49   mcpist-console  Up              mcpist-console
2cefd1b4f7d2   traefik:v3.3    Up              mcpist-traefik
48b448ddd873   mcpist-server   Up              mcpist-server
bc0456e4918b   mcpist-worker   Up              mcpist-worker
```

---

---

## 2026-01-22 - Traefik → nginx 移行 & E2E動作確認

### Phase 5: nginx への移行 ✅ 完了

#### 背景・理由

- Traefik の動的設定機能は本プロジェクトでは使用していない
- nginx の方がシンプルで設定が分かりやすい
- デバッグも容易

#### 実施内容

1. **nginx 設定ファイル作成** (`nginx/nginx.conf`)
   - domain-based routing（`*.localhost`）
   - upstream 定義で各サービスへルーティング

2. **Docker Compose 更新**
   - Traefik を nginx に置き換え
   - profiles を削除（常に全サービス起動）
   - network aliases 追加（サービス間通信で `*.localhost` を解決可能に）

3. **環境変数フロー改善**
   - `scripts/sync-env.js` で `apps/console/.env.local` を生成
   - Next.js がプロジェクトディレクトリの `.env.local` を読み込む問題を解決
   - Worker の Dockerfile.dev に `OAUTH_JWKS_URL` 追加

4. **Traefik 関連ファイル削除**
   - `traefik/default/routes.yml`
   - `traefik/infra/routes.yml`

#### 技術的決定

| 項目 | Traefik | nginx |
|------|---------|-------|
| 設定方式 | ラベルベース動的設定 | 静的設定ファイル |
| ユースケース | サービスディスカバリ、K8s | シンプルなリバースプロキシ |
| 本プロジェクト | オーバースペック | 適切 |

#### サービス間通信

```
本番環境: サービス → パブリックDNS → 他サービス
Docker環境: サービス → nginx (network alias) → 他サービス
```

nginx に network aliases を設定することで、コンテナ内から `*.localhost` ドメインを nginx 経由で解決可能にした。これにより本番環境のDNSベース通信を再現。

### Phase 6: E2E動作確認 ✅ 完了

#### テスト結果

| テスト項目 | 結果 |
|-----------|------|
| Console UI アクセス (`console.localhost`) | ✅ 成功 |
| Supabase ログイン | ✅ 成功 |
| OAuth 認可フロー | ✅ 成功 |
| JWT 検証 (Worker) | ✅ 成功 |
| MCP Server 接続 | ✅ 成功 |
| initialize / tools/list | ✅ 成功 |

#### 発生した問題と解決

1. **502 Bad Gateway**
   - 原因: コンテナ起動タイミング
   - 解決: リロードで解消

2. **OAuth Server 関数が見つからない**
   - 原因: `store_oauth_refresh_token` 関数が未適用
   - 解決: `supabase db reset` でマイグレーション再適用

3. **Worker で OAUTH_JWKS_URL が未設定**
   - 原因: Dockerfile.dev の環境変数生成コマンドに含まれていなかった
   - 解決: grep パターンに `OAUTH_JWKS_URL` 追加

### 反省点

1. **環境変数管理の複雑さ**
   - monorepo での環境変数配布は想定以上に複雑
   - sync-env.js スクリプトで一元管理する方針は正解だった
   - 各ツール（Next.js, wrangler）の環境変数読み込み仕様を事前に把握すべきだった

2. **Traefik 採用の判断ミス**
   - 動的設定を使わないなら nginx で十分だった
   - 技術選定時に「本当に必要か？」をもっと吟味すべき

3. **Docker 環境のデバッグ**
   - ログ確認コマンドを pnpm script に追加したのは良かった (`pnpm logs:docker`)
   - コンテナ間通信の確認方法（exec + curl/wget）を最初から用意すべきだった

---

## DAY011 完了

すべてのPhaseが完了。OAuth Mock Server + nginx 統合が完了し、E2Eテストも成功した。

### 最終成果物

| カテゴリ | ファイル |
|---------|---------|
| OAuth Server | `apps/oauth/` (Hono + @hono/node-server) |
| DB | `supabase/migrations/00000000000005_oauth_authorization_requests.sql` |
| DB | `supabase/migrations/00000000000006_oauth_refresh_tokens.sql` |
| nginx | `nginx/nginx.conf` |
| Console | プロキシ化、authorization_id対応 |
| テスト | `docs/test/tst-oauth-mock-server.md` |

### 削除されたファイル

| ファイル | 理由 |
|---------|------|
| `traefik/default/routes.yml` | nginx に移行 |
| `traefik/infra/routes.yml` | nginx に移行 |
| `apps/console/src/app/api/auth/consent/route.ts` | OAuth Server に移行 |

### 残課題（将来対応）

- 本番デプロイ時の鍵管理
- OAuth プロバイダー切り替え（本番では実際のプロバイダーを使用）
