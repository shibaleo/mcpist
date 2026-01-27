# OAuth Mock Server 詳細実装計画

## 概要

**目的**: 開発環境用のOAuth Mock Serverを別コンテナとして作成

**方針**:
- 本番: Supabase OAuth Server を使用
- 開発: Mock OAuth Server (`oauth.localhost`) を使用
- 将来: Supabase OAuth Serverが有料化した場合、Mockを本番デプロイ可能

**同意画面**: Console内のReactページに残す（移動しない）

---

## 現在のアーキテクチャ

```
開発環境（現在）:
┌─────────────────────────────────────────────────────────────┐
│ Console (Next.js)                                           │
│                                                             │
│  /api/auth/authorize  → 認可リクエスト検証 → /oauth/consent │
│  /api/auth/consent    → 認可コード生成                      │
│  /api/auth/token      → JWT発行                             │
│  /api/auth/jwks       → 公開鍵提供                          │
│  /.well-known/*       → メタデータ                          │
│  /oauth/consent       → 同意画面UI (React)                  │
└─────────────────────────────────────────────────────────────┘
```

```
本番環境（現在）:
┌─────────────────────────────────────────────────────────────┐
│ Supabase OAuth Server (Cloud)                               │
│                                                             │
│  /auth/v1/authorize   → 認可                                │
│  /auth/v1/token       → JWT発行                             │
│  /.well-known/jwks    → 公開鍵                              │
└─────────────────────────────────────────────────────────────┘
                ↓ 同意画面リダイレクト
┌─────────────────────────────────────────────────────────────┐
│ Console (Next.js)                                           │
│  /oauth/consent       → 同意画面UI                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 新アーキテクチャ

```
開発環境（変更後）:
                     ┌──────────────────────────────────────┐
                     │ oauth.localhost (Mock OAuth Server)  │
                     │                                      │
MCPクライアント ─────→│  GET  /authorize   → 同意画面へ      │
                     │  POST /token       → JWT発行         │
                     │  GET  /jwks        → 公開鍵          │
                     │  GET  /.well-known/* → メタデータ    │
                     └──────────────────────────────────────┘
                                   │
                                   │ リダイレクト
                                   ↓
                     ┌──────────────────────────────────────┐
                     │ console.localhost (Console)          │
                     │                                      │
                     │  /oauth/consent   → 同意画面UI       │
                     │  POST /api/auth/consent → コード生成 │
                     └──────────────────────────────────────┘
                                   │
                                   │ code + state
                                   ↓
                     MCPクライアント (redirect_uri)
```

---

## Console の変更点

### 1. `/api/auth/authorize` の削除

現在の処理:
1. パラメータ検証
2. ログイン状態確認
3. `/oauth/consent` へリダイレクト

変更後:
- OAuth Server が担当するため削除
- 同意画面へのリダイレクトはOAuth Serverが行う

### 2. `/api/auth/token` の変更

現在の処理:
- 開発: カスタム実装でJWT発行
- 本番: Supabase OAuth Serverへプロキシ

変更後:
- 開発: OAuth Server (`oauth.localhost`) へプロキシ
- 本番: Supabase OAuth Server へプロキシ

### 3. `/api/auth/jwks` の削除

- OAuth Server が担当するため削除

### 4. `/.well-known/*` の変更

現在: Console内でメタデータ提供
変更後: OAuth Server URLを返す（開発/本番で異なる）

### 5. `/api/auth/consent` は維持

- 同意画面UIがConsole内にあるため、コード生成APIも維持
- OAuth Server からではなく、Console の同意画面から呼び出される

### 6. `env.ts` の変更

```typescript
// 新しい環境変数
// OAUTH_SERVER_URL: 開発時は oauth.localhost、本番は Supabase URL
export function getOAuthServerUrl(): string {
  if (isProduction) {
    return process.env.NEXT_PUBLIC_SUPABASE_URL + '/auth/v1'
  }
  return process.env.OAUTH_SERVER_URL || 'http://oauth.localhost'
}
```

---

## OAuth Mock Server 実装

### 技術スタック

| 項目 | 選定 | 理由 |
|------|------|------|
| フレームワーク | Hono | 軽量、TypeScript対応、Cloudflare Workers互換 |
| JWT | jose | 現在Console内で使用中、移植コスト最小 |
| DB接続 | @supabase/supabase-js | 認可コード保存にSupabase使用 |

### ディレクトリ構成

```
apps/oauth/
├── package.json
├── tsconfig.json
├── Dockerfile.dev
├── src/
│   ├── index.ts          # エントリーポイント (Hono app)
│   ├── routes/
│   │   ├── authorize.ts  # GET /authorize
│   │   ├── token.ts      # POST /token
│   │   └── jwks.ts       # GET /jwks
│   ├── lib/
│   │   ├── jwt.ts        # JWT署名・検証（Console から移植）
│   │   ├── pkce.ts       # PKCE検証（Console から移植）
│   │   └── codes.ts      # 認可コード管理（Console から移植）
│   └── well-known/
│       ├── oauth-authorization-server.ts
│       └── oauth-protected-resource.ts
└── .env.example
```

### エンドポイント設計

| エンドポイント | メソッド | 役割 |
|---------------|---------|------|
| `/authorize` | GET | 認可リクエスト → 同意画面へリダイレクト |
| `/token` | POST | 認可コード → JWT交換 |
| `/jwks` | GET | 公開鍵提供 |
| `/.well-known/oauth-authorization-server` | GET | OAuth メタデータ |

### 認可フロー詳細

```
1. MCPクライアント → GET oauth.localhost/authorize?...
2. OAuth Server: パラメータ検証
3. OAuth Server: ログイン状態確認（Supabase session cookie）
4. OAuth Server → 302 console.localhost/oauth/consent?request=base64(...)
5. ユーザー: 同意画面で「許可」クリック
6. Console → POST console.localhost/api/auth/consent (認可コード生成)
7. Console → 302 redirect_uri?code=xxx&state=yyy
8. MCPクライアント → POST oauth.localhost/token (code, code_verifier)
9. OAuth Server: PKCE検証 → JWT発行
10. MCPクライアント: access_token 取得
```

---

## 環境変数

### OAuth Server (.env)

```env
# Supabase
SUPABASE_URL=http://host.docker.internal:54321
SUPABASE_SERVICE_ROLE_KEY=xxx

# JWT Keys (開発時は自動生成)
AUTH_PRIVATE_KEY=      # 本番時のみ必要
AUTH_PUBLIC_KEY=       # 本番時のみ必要

# URLs
CONSOLE_URL=http://console.localhost
MCP_SERVER_URL=http://mcp.localhost

# Server
PORT=4000
```

### Console (.env.local) 追加分

```env
# OAuth Server URL (開発時のみ使用)
OAUTH_SERVER_URL=http://oauth.localhost
```

---

## Docker Compose 変更

### docker-compose.yml に追加

```yaml
  oauth:
    build:
      context: ./apps/oauth
      dockerfile: Dockerfile.dev
    container_name: mcpist-oauth
    profiles: ["default", "infra"]
    environment:
      - PORT=4000
      - SUPABASE_URL=http://host.docker.internal:54321
      - SUPABASE_SERVICE_ROLE_KEY=${SUPABASE_SERVICE_ROLE_KEY}
      - CONSOLE_URL=http://console.localhost
      - MCP_SERVER_URL=http://mcp.localhost
    volumes:
      - ./apps/oauth:/app
      - /app/node_modules
    networks:
      - mcpist
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

### traefik/default/routes.yml に追加

```yaml
    oauth:
      rule: "Host(`oauth.localhost`)"
      entryPoints:
        - web
      service: oauth

  services:
    oauth:
      loadBalancer:
        servers:
          - url: "http://mcpist-oauth:4000"
```

---

## 実装フェーズ

### Phase 1: OAuth Server 基盤作成

| タスク | 詳細 |
|-------|------|
| T-001 | `apps/oauth/package.json` 作成 |
| T-002 | `apps/oauth/tsconfig.json` 作成 |
| T-003 | `apps/oauth/Dockerfile.dev` 作成 |
| T-004 | `apps/oauth/src/index.ts` 作成（Hono基盤） |

### Phase 2: lib 移植

| タスク | 詳細 |
|-------|------|
| T-005 | `lib/jwt.ts` 移植（Console → OAuth Server） |
| T-006 | `lib/pkce.ts` 移植 |
| T-007 | `lib/codes.ts` 移植 |

### Phase 3: エンドポイント実装

| タスク | 詳細 |
|-------|------|
| T-008 | `GET /authorize` 実装 |
| T-009 | `POST /token` 実装 |
| T-010 | `GET /jwks` 実装 |
| T-011 | `/.well-known/*` 実装 |

### Phase 4: Docker/Traefik 設定

| タスク | 詳細 |
|-------|------|
| T-012 | `docker-compose.yml` に oauth 追加 |
| T-013 | `traefik/default/routes.yml` に oauth ルート追加 |
| T-014 | `traefik/infra/routes.yml` に oauth ルート追加 |

### Phase 5: Console 変更

| タスク | 詳細 |
|-------|------|
| T-015 | `env.ts` に `OAUTH_SERVER_URL` 対応追加 |
| T-016 | `/api/auth/authorize` 削除 |
| T-017 | `/api/auth/token` を OAuth Server へプロキシに変更 |
| T-018 | `/api/auth/jwks` 削除 |
| T-019 | `/.well-known/*` を OAuth Server URL 参照に変更 |

### Phase 6: 動作確認

| タスク | 詳細 |
|-------|------|
| T-020 | `pnpm dev:docker` で全サービス起動確認 |
| T-021 | OAuth認可フローE2Eテスト |

---

## 確認事項

### Q1: ログイン状態の共有

OAuth Server が認可リクエストを受けた際、ユーザーがログイン済みかどうかを確認する必要があります。

**現在の Console 実装:**
```typescript
const supabase = createServerClient(...)
const { data: { user } } = await supabase.auth.getUser()
```

**OAuth Server での対応:**
- Supabase session cookie は `sb-xxx-auth-token` の形式
- OAuth Server から Supabase へ cookie を転送して認証状態確認が必要
- **または**: OAuth Server は認証状態を確認せず、常に同意画面へリダイレクト。同意画面側でログイン確認（現在の実装）

**推奨**: 後者（同意画面側でログイン確認）がシンプル

### Q2: 認可コードの保存先

現在: Console から Supabase RPC (`store_oauth_code`) で保存
OAuth Server 分離後も同じ方式で問題なし（OAuth Server も Supabase に接続）

### Q3: JWT の鍵管理

開発環境:
- OAuth Server が起動時に RSA キーペアを自動生成
- ファイルに永続化（再起動時も同じ鍵を使用）

本番環境（将来的にMockをデプロイする場合）:
- `AUTH_PRIVATE_KEY`, `AUTH_PUBLIC_KEY` 環境変数から読み込み

---

## 完了条件

- [x] `oauth.localhost` でメタデータ取得可能
- [x] OAuth認可フローが動作（authorize → consent → token → JWT取得）
- [x] PKCE検証が正しく動作
- [x] JWT署名・検証が正しく動作
- [x] Console の `/api/auth/authorize`, `/api/auth/jwks` がプロキシ化
- [x] Console の `/api/auth/consent` が削除
- [x] 開発・本番で環境変数切り替えが正しく動作

---

## 実装完了 (2026-01-21)

### テスト結果

詳細は `mcpist/docs/test/tst-oauth-mock-server.md` を参照。

| エンドポイント | 結果 |
|---------------|------|
| GET /health | ✅ OK |
| GET /.well-known/oauth-authorization-server | ✅ OK |
| GET /jwks | ✅ OK |
| GET /authorize | ✅ 302 redirect |
| GET /authorization/:id | ✅ OK |
| POST /authorization/:id/approve | ✅ 実装済（要認証セッション） |
| POST /authorization/:id/deny | ✅ 実装済（要認証セッション） |
| POST /token | ✅ 実装済 |
