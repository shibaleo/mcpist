# Auth移行計画: 自作OAuth認可サーバー → Supabase Auth Server

> **⚠️ OBSOLETE（廃止）**
>
> この計画書は **廃止** されました。
>
> **採用した方針:** 完全移行ではなく、**環境変数による切り替え**を実装。
>
> | 環境 | OAuth Server | 実装 |
> |------|-------------|------|
> | 開発 (`ENVIRONMENT=development`) | カスタム実装 | `/api/auth/*` |
> | 本番 (`ENVIRONMENT=production`) | Supabase OAuth Server | `/auth/v1/*` |
>
> **理由:**
> - ローカル OSS Supabase は OAuth Server 未サポート
> - 開発環境でのテストにカスタム実装が必要
> - 同一コードベースで両環境に対応可能
>
> **実装済みファイル:**
> - `apps/console/src/lib/env.ts` - 環境判定ユーティリティ
> - `apps/console/src/app/api/auth/authorize/route.ts` - 切り替え対応
> - `apps/console/src/app/api/auth/token/route.ts` - 切り替え対応
>
> **参照:** [work-log.md](./work-log.md) - 2026-01-20 の成果

---

## （以下は参考として残す）

## 概要

現在のMCPistは、Console（Next.js）で独自のOAuth 2.1認可サーバーを実装している。
本計画では、OAuth認可サーバー機能をSupabase Auth Serverに移行する。

### 移行の目的

1. **保守コスト削減**: OAuth認可サーバーの独自実装（約1000行）を削除
2. **セキュリティ強化**: Supabaseの専門チームが保守するOAuth実装を利用
3. **機能追加**: DCR（Dynamic Client Registration）、Token Revocation等の標準機能を自動取得

### 継続するもの

- **API Key認証（`mpt_xxx`形式）**: Supabase OAuth Serverの範囲外のため独自維持
- **Cloudflare Worker**: API Gateway（認証・Rate Limit・LB）はTypeScriptで継続
- **Token Vault統合**: 外部OAuthトークンの暗号化保存

### 認証の役割分担

| レイヤー | 責務 | 実装言語 |
|----------|------|----------|
| **Cloudflare Worker** | 認証（Authentication）: 存在するユーザーか？ | TypeScript |
| **Go Middleware** | 認可（Authorization）: このリクエストを実行する権限があるか？ | Go |

---

## 現状のアーキテクチャ

```
MCP Client
    │
    ├─ OAuth 2.1 + PKCE ─────────────────────────────────────────────┐
    │                                                                 │
    │                         ┌───────────────────────────────────────▼──────────┐
    │                         │               Console (Next.js)                   │
    │                         │  ┌──────────────────────────────────────────────┐ │
    │                         │  │         自作 OAuth 2.1 認可サーバー           │ │
    │                         │  │  /api/auth/authorize   → 認可エンドポイント   │ │
    │                         │  │  /api/auth/token       → トークン発行         │ │
    │                         │  │  /api/auth/jwks        → 公開鍵提供          │ │
    │                         │  │  /api/auth/consent     → 同意確認            │ │
    │                         │  │  /api/auth/lib/jwt.ts  → JWT生成(独自キー)   │ │
    │                         │  └──────────────────────────────────────────────┘ │
    │                         │                      │                            │
    │                         │                      ▼                            │
    │                         │              Supabase Auth                        │
    │                         │              (ログインのみ)                        │
    │                         └───────────────────────────────────────────────────┘
    │
    │  JWT (Console独自キー)
    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           Cloudflare Worker (TypeScript)                        │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  【認証 = Authentication】存在するユーザーか？                              │ │
│  │  - JWT検証 (jose + Supabase JWKS) ← 現在ConsoleのJWKSを参照               │ │
│  │  - API Key検証 (Supabase RPC)                                              │ │
│  │  - Rate Limit (KV)                                                         │ │
│  │  - Load Balancing                                                          │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
    │
    │  X-User-ID, X-Gateway-Secret
    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Go MCP Server                                      │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  【認可 = Authorization】このリクエストを実行する権限があるか？             │ │
│  │  internal/auth/middleware.go                                               │ │
│  │  - X-User-ID ヘッダー検証 (Worker経由)                                     │ │
│  │  - JWT検証 (JWKS) ← 直接アクセス用（開発環境）                             │ │
│  │  - API Key検証 (Supabase RPC) ← 直接アクセス用（開発環境）                 │ │
│  │  - スコープ/権限チェック ← TODO                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 現状の問題点

| 問題 | 詳細 |
|------|------|
| **独自OAuth** | Console内のOAuth認可サーバーの保守負担（約1000行） |
| **JWKS不整合** | Console独自キー vs Supabase JWKSの混在 |
| **JWT二重検証** | WorkerとGo Server両方でJWT検証している（不要なオーバーヘッド） |
| **認可ロジック未実装** | Go Server側のスコープ/権限チェックがない |

---

## 移行後のアーキテクチャ

```
MCP Client
    │
    ├─ OAuth 2.1 + PKCE ─────────────────────────────────────────────┐
    │                                                                 │
    │                         ┌───────────────────────────────────────▼──────────┐
    │                         │               Supabase Auth Server                │
    │                         │  ┌──────────────────────────────────────────────┐ │
    │                         │  │  /auth/v1/authorize     → 認可エンドポイント  │ │
    │                         │  │  /auth/v1/token         → トークン発行        │ │
    │                         │  │  /.well-known/jwks.json → 公開鍵提供         │ │
    │                         │  │  DCR, Revocation 等     → 標準機能           │ │
    │                         │  └──────────────────────────────────────────────┘ │
    │                         │                                                   │
    │                         │  Authorization Path (同意画面)                    │
    │                         │          ↓                                        │
    │                         │  ┌──────────────────────────────────────────────┐ │
    │                         │  │     Console (Next.js) - 同意画面のみ          │ │
    │                         │  │  /oauth/authorize → 同意UI                   │ │
    │                         │  │  (OAuth認可ロジックは削除)                    │ │
    │                         │  └──────────────────────────────────────────────┘ │
    │                         └───────────────────────────────────────────────────┘
    │
    │  JWT (Supabase発行)
    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        Cloudflare Worker (TypeScript)                           │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  【認証 = Authentication】存在するユーザーか？                              │ │
│  │  - JWT検証 (jose + Supabase JWKS) ← 統一                                  │ │
│  │  - API Key検証 (Supabase RPC) ← 継続                                      │ │
│  │  - MCPトークン検証 (64 hex) ← 追加                                        │ │
│  │  - Rate Limit (KV)                                                         │ │
│  │  - Load Balancing                                                          │ │
│  │  - CORS                                                                    │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
    │
    │  X-User-ID, X-Auth-Type, X-Scope, X-Gateway-Secret
    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Go MCP Server                                      │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  【認可 = Authorization】このリクエストを実行する権限があるか？             │ │
│  │  internal/auth/middleware.go                                               │ │
│  │  - Gateway Secret検証（JWT検証は不要 - Workerで完了済み）                   │ │
│  │  - X-User-ID 受け取り（Workerが検証済み）                                  │ │
│  │  - Entitlement Storeで権限チェック（ツール単位）                           │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 移行後の特徴

| 項目 | 移行前 | 移行後 |
|------|--------|--------|
| OAuth認可サーバー | Console（自作） | Supabase Auth Server |
| JWT発行 | Console（独自キー） | Supabase |
| JWKS | Console + Supabase混在 | Supabase統一 |
| API Gateway | Cloudflare Worker（TS） | **Cloudflare Worker（TS）継続** |
| 認証（Authentication） | Worker | **Worker（変更なし）** |
| 認可（Authorization） | なし | **Go Server + Entitlement Store** |
| API Key認証 | Worker | Worker継続 |

### 認証 vs 認可の責務分離

| 責務 | Worker（認証） | Go Server（認可） |
|------|----------------|-------------------|
| **質問** | 「あなたは誰？」 | 「あなたは何ができる？」 |
| **検証対象** | トークンの有効性、ユーザーの存在 | Entitlement Store（権限表） |
| **処理内容** | JWT署名検証、API Key検証、有効期限 | ユーザー状態、モジュール有効化、ツールアクセス権 |
| **失敗時** | 401 Unauthorized | 403 Forbidden |
| **データソース** | Supabase JWKS / mcp_tokens | Entitlement Store（users, subscriptions, user_module_preferences） |

---

## 参考実装: dwhbi console

`C:\Users\m_fukuda\Documents\dwhbi\packages\console` の認証実装を参考にする。

### 認証方式（2段階）- Worker側で実装

```typescript
// apps/worker/src/index.ts - 認証部分

export async function authenticate(request: Request, env: Env): Promise<AuthResult | null> {
  const token = authHeader.substring(7);

  // 1. API Key (mpt_xxx) - Vault経由で検証
  //    mcpist.mcp_tokensテーブルでtoken_hashを照合
  if (token.startsWith("mpt_")) {
    return await verifyApiKey(token, env);
  }

  // 2. Supabase JWT - JWKS検証
  return await verifyJwt(token, env);
}
```

### トークンの種類

| トークン | 形式 | 用途 | 保存場所 |
|----------|------|------|----------|
| **API Key** | `mpt_<32 hex>` | ユーザーがConsoleで作成する長期トークン | `mcpist.mcp_tokens`（hash） |
| **JWT** | Supabase発行 | OAuth 2.1フローで取得する短期トークン | なし（検証のみ） |

### Worker実装の変更点

現在のWorker（`apps/worker/src/index.ts`）の変更:

| 現在の実装 | 変更 |
|-----------|------|
| JWT検証（Supabase JWKS） | ✅ 継続 |
| API Key検証（`mpt_xxx`） | ✅ 継続（`validate_api_key` RPC） |
| Service Role Key検証 | ❌ **削除** |
| スコープ抽出 → X-Scope | ❌ **不要**（Entitlement Storeで管理） |

---

## 移行手順

### Phase 1: Worker認証の整理（TypeScript）

**期間目安**: 0.5日

#### 1.1 現在のWorker構成

```
apps/worker/
├── package.json
├── tsconfig.json
├── wrangler.toml
└── src/
    └── index.ts      # 既存実装（認証・Rate Limit・LB）
```

#### 1.2 Worker認証（変更なし、整理のみ）

```typescript
// apps/worker/src/index.ts - 認証部分（現状維持）

interface AuthResult {
  userId: string;
  authType: "jwt" | "api_key";
}

async function authenticate(
  request: Request,
  env: Env
): Promise<AuthResult | null> {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader?.startsWith("Bearer ")) {
    return null;
  }

  const token = authHeader.slice(7);

  // 1. API Key (mpt_xxx) - mcpist.mcp_tokens で検証
  if (token.startsWith("mpt_")) {
    return await verifyApiKey(token, env);
  }

  // 2. Supabase JWT - JWKS検証
  return await verifyJwt(token, env);
}
```

#### 1.3 プロキシ時のヘッダー付与（シンプル化）

```typescript
// プロキシ時のヘッダー付与
function addAuthHeaders(request: Request, auth: AuthResult, env: Env): Headers {
  const headers = new Headers(request.headers);
  headers.set("X-User-ID", auth.userId);
  headers.set("X-Auth-Type", auth.authType);
  headers.set("X-Gateway-Secret", env.GATEWAY_SECRET);
  // X-Scopeは不要（Go Server側でEntitlement Storeから取得）
  return headers;
}
```

---

### Phase 2: Go Server認可ミドルウェア実装

**期間目安**: 2日

**重要な設計方針**: Go ServerはJWT検証を行わない。WorkerでJWT検証が完了しており、Gateway Secretによって信頼性が担保されるため。

#### 2.1 認可ミドルウェアの責務（JWT検証は含まない）

```go
// apps/server/internal/auth/authorization.go
package auth

// 認可ミドルウェア: このリクエストを実行する権限があるか？
// 注意: JWT検証はWorker側で完了しているため、ここでは行わない
type Authorizer struct {
    gatewaySecret string                 // Workerからの通信であることを確認
    entitlementStore *entitlement.Store  // Entitlement Store接続
}

type AuthContext struct {
    UserID   string
    AuthType string   // "jwt", "api_key"
    // Entitlement Storeから取得する情報
    UserStatus      string   // "active", "suspended", "deleted"
    IsAdmin         bool     // 管理者フラグ（将来用）
    EnabledModules  []string // 有効なモジュール一覧
    Plan            *Plan    // 課金プラン情報
}

func (a *Authorizer) Authorize(r *http.Request) (*AuthContext, error) {
    // 1. Gateway Secret検証（Workerからの通信であることを確認）
    //    これが通れば、X-User-IDはWorkerによってJWT/API Key検証済み
    if r.Header.Get("X-Gateway-Secret") != a.gatewaySecret {
        return nil, ErrInvalidGatewaySecret
    }

    // 2. X-User-ID取得（Workerが検証済みなので信頼できる）
    userID := r.Header.Get("X-User-ID")
    if userID == "" {
        return nil, ErrMissingUserID
    }

    // 3. Entitlement Storeからユーザー情報取得
    entitlement, err := a.entitlementStore.GetUserEntitlement(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get entitlement: %w", err)
    }

    // 4. AuthContext構築
    ctx := &AuthContext{
        UserID:         userID,
        AuthType:       r.Header.Get("X-Auth-Type"),
        UserStatus:     entitlement.Status,
        IsAdmin:        entitlement.IsAdmin,
        EnabledModules: entitlement.EnabledModules,
        Plan:           entitlement.Plan,
    }

    // 5. アカウント状態チェック
    if ctx.UserStatus != "active" {
        return nil, ErrAccountNotActive
    }

    return ctx, nil
}
```

#### 2.2 Entitlement Storeとの連携

```go
// apps/server/internal/entitlement/store.go
package entitlement

type Store struct {
    supabaseURL string
    serviceKey  string
}

type UserEntitlement struct {
    Status         string   // mcpist.users.status
    IsAdmin        bool     // 将来: mcpist.users.role = 'admin'
    EnabledModules []string // mcpist.user_module_preferences (is_enabled=true)
    Plan           *Plan    // mcpist.subscriptions → mcpist.plans
}

type Plan struct {
    Name           string
    RateLimitRPM   int
    RateLimitBurst int
    QuotaMonthly   *int  // NULL = unlimited
    CreditEnabled  bool
}

// ユーザー権限情報を取得（Supabase RPC経由）
func (s *Store) GetUserEntitlement(userID string) (*UserEntitlement, error) {
    // RPC: get_user_entitlement(p_user_id)
    // → users, subscriptions, plans, user_module_preferences を結合して返す
}
```

#### 2.3 ツール単位のアクセス制御

```go
// apps/server/internal/auth/tool_access.go
package auth

// ツールアクセス権チェック
func (ctx *AuthContext) CanAccessTool(moduleName, toolName string) error {
    // 1. モジュールが有効か確認
    moduleEnabled := false
    for _, m := range ctx.EnabledModules {
        if m == moduleName {
            moduleEnabled = true
            break
        }
    }
    if !moduleEnabled {
        return &ForbiddenError{
            Code:    "MODULE_NOT_ENABLED",
            Message: fmt.Sprintf("Module '%s' is not enabled for this user", moduleName),
        }
    }

    // 2. ツール固有の制限チェック（将来: tool_costsでクレジット確認等）
    // 現時点ではモジュールが有効なら全ツール利用可能

    return nil
}
```

#### 2.4 MCPハンドラーへの統合

```go
// apps/server/internal/mcp/handler.go
func (h *Handler) HandleToolCall(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
    authCtx := auth.GetAuthContext(ctx)

    // ツールアクセス権チェック
    moduleName := req.Params.Arguments["module"].(string)
    toolName := req.Params.Arguments["tool_name"].(string)

    if err := authCtx.CanAccessTool(moduleName, toolName); err != nil {
        return nil, &MCPError{
            Code:    -32600,  // Invalid Request
            Message: err.Error(),
        }
    }

    // 実際のツール呼び出し
    return h.callModule(ctx, req)
}
```

#### 2.5 Supabase RPC関数（新規作成）

```sql
-- supabase/migrations/YYYYMMDD_get_user_entitlement.sql

CREATE OR REPLACE FUNCTION mcpist.get_user_entitlement(p_user_id UUID)
RETURNS TABLE (
    user_status TEXT,
    is_admin BOOLEAN,
    plan_name TEXT,
    rate_limit_rpm INTEGER,
    rate_limit_burst INTEGER,
    quota_monthly INTEGER,
    credit_enabled BOOLEAN,
    enabled_modules TEXT[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        u.status AS user_status,
        FALSE AS is_admin,  -- 将来: u.role = 'admin'
        p.name AS plan_name,
        p.rate_limit_rpm,
        p.rate_limit_burst,
        p.quota_monthly,
        p.credit_enabled,
        ARRAY(
            SELECT m.name
            FROM mcpist.user_module_preferences ump
            JOIN mcpist.modules m ON m.id = ump.module_id
            WHERE ump.user_id = p_user_id AND ump.is_enabled = true
        ) AS enabled_modules
    FROM mcpist.users u
    LEFT JOIN mcpist.subscriptions s ON s.user_id = u.id AND s.status = 'active'
    LEFT JOIN mcpist.plans p ON p.id = s.plan_id
    WHERE u.id = p_user_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

---

### Phase 3: Supabase OAuth Server設定

**期間目安**: 1日

#### 3.1 Supabase Dashboard設定

1. **Authentication → OAuth Server**
   - Enable OAuth 2.1 Server: ON
   - Authorization Path: `https://console.mcpist.app/oauth/authorize`

2. **Authentication → OAuth Server → Clients**
   - クライアント登録（事前登録方式）
   - redirect_uris設定

3. **（オプション）Dynamic Client Registration**
   - Enable DCR: 必要に応じてON

#### 3.2 Console同意画面実装

```typescript
// apps/console/src/app/oauth/authorize/page.tsx
// Authorization Pathとして設定されるページ

export default function ConsentPage() {
  // 1. Supabase Authでログイン確認
  // 2. 未ログイン → ログインページへリダイレクト
  // 3. ログイン済み → 同意画面表示
  //    - client_id（どのクライアントか）
  //    - 要求scope
  //    - 許可/拒否ボタン
  // 4. 許可 → Supabaseに認可を返す
}
```

---

### Phase 4: 既存コード削除・移行

**期間目安**: 1日

#### 4.1 削除対象（Console）

```
apps/console/src/app/api/auth/
├── authorize/route.ts    # 削除
├── token/route.ts        # 削除
├── consent/route.ts      # 削除（新UIに置き換え）
├── jwks/route.ts         # 削除
└── lib/
    ├── jwt.ts            # 削除
    ├── codes.ts          # 削除
    └── pkce.ts           # 削除
```

#### 4.2 更新対象

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/.well-known/oauth-authorization-server/route.ts` | Supabase URLを返すように修正 |
| `apps/console/src/app/.well-known/oauth-protected-resource/route.ts` | authorization_serversをSupabaseに変更 |
| `supabase/migrations/` | `oauth_authorization_codes`テーブルは不要（Supabase管理） |

#### 4.3 Go Server更新（JWT検証の削除）

**重要**: WorkerでJWT検証を行うため、Go ServerではJWT/JWKS検証を行わない。

```go
// apps/server/internal/auth/middleware.go
// 認証(Authentication)は削除、認可(Authorization)に特化
// JWT検証はWorkerで完了しているため不要

func (m *Middleware) ValidateRequest(r *http.Request) (*AuthContext, error) {
    // 1. Gateway Secret検証（Workerからの通信であることを確認）
    //    これにより、X-User-IDがWorkerによって検証済みであることを保証
    secret := r.Header.Get("X-Gateway-Secret")
    if secret != m.gatewaySecret {
        return nil, fmt.Errorf("invalid gateway secret")
    }

    // 2. X-User-ID受け取り（Workerが検証済み）
    userID := r.Header.Get("X-User-ID")
    if userID == "" {
        return nil, fmt.Errorf("missing user ID")
    }

    // 3. Entitlement Storeから権限情報取得
    entitlement, err := m.entitlementStore.GetUserEntitlement(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get entitlement: %w", err)
    }

    return &AuthContext{
        UserID:         userID,
        AuthType:       r.Header.Get("X-Auth-Type"),
        UserStatus:     entitlement.Status,
        EnabledModules: entitlement.EnabledModules,
        Plan:           entitlement.Plan,
    }, nil
}
```

#### 4.4 削除対象（Go Server）

JWT二重検証を排除するため、以下のファイル/機能を削除:

| 対象 | 理由 |
|------|------|
| `internal/auth/jwks.go` | JWT検証はWorkerで実施 |
| `internal/auth/crypto.go` | JWT署名検証はWorkerで実施 |
| `internal/auth/jwt.go`（検証部分） | Workerで完了済み |
| JWKSキャッシュ機構 | Go Server側では不要 |
| `jose`/`jwt-go`依存 | Go Serverから削除可能 |

**開発環境での直接アクセス**: Workerを経由せずGo Serverに直接アクセスする場合は、Gateway Secretを環境変数で設定して開発用ヘッダーを付与するローカルプロキシを使用する。

---

### Phase 5: デプロイ・検証

**期間目安**: 1日

#### 5.1 デプロイ順序

1. **Supabase OAuth Server有効化**
2. **Worker更新デプロイ**（Cloudflare）
3. **Go Server更新デプロイ**（Render/Koyeb）
4. **Console更新デプロイ**（Vercel）

#### 5.2 検証項目

| テスト | 内容 | 責務 |
|--------|------|------|
| OAuth 2.1フロー | MCPクライアントからの認可 → トークン取得 | Supabase |
| JWT認証 | Supabase発行JWTでの接続 | Worker（認証） |
| API Key認証 | 既存API Keyでの接続 | Worker（認証） |
| MCPトークン認証 | 64文字hexトークンでの接続 | Worker（認証） |
| スコープ検証 | モジュールアクセス権チェック | Go Server（認可） |
| Rate Limit | IP/ユーザー単位の制限 | Worker |
| Load Balancing | Primary/Secondary振り分け | Worker |
| Failover | Primary障害時の切り替え | Worker |

---

## 移行スケジュール

| Phase | 内容 | 期間 | 依存 |
|-------|------|------|------|
| 1 | Worker認証の整理（Service Role Key削除） | 0.5日 | なし |
| 2 | Go Server認可ミドルウェア + Entitlement Store連携 | 2日 | Phase 1 |
| 3 | Supabase OAuth Server設定 + 同意画面 | 1日 | なし（並行可） |
| 4 | 既存コード削除・移行 | 1日 | Phase 2, 3 |
| 5 | デプロイ・検証 | 1日 | Phase 4 |

**合計: 約5.5日**

---

## リスクと対策

| リスク | 影響 | 対策 |
|--------|------|------|
| 既存トークン互換性 | 旧JWTが使えなくなる | JIT移行（旧トークン期限切れまで許容） |
| Supabase OAuth Server障害 | 認証不可 | API Key/MCPトークンはSupabase RPC経由で継続可能 |
| 認可ロジックの複雑化 | Go Server側の実装負担 | 段階的にスコープを追加 |
| DCRスパム | 不正クライアント大量登録 | 初期は事前登録のみ、DCRは後から有効化 |

---

## 移行前後の比較

### コード量

| 対象 | 移行前 | 移行後 | 備考 |
|------|--------|--------|------|
| Console OAuth | 約1000行 | 0行 | 同意UIは別途100行程度 |
| Worker認証 | 約200行 | 約180行 | Service Role削除 |
| Go Server認証（JWT/JWKS） | 約300行 | **0行** | Worker側で完結、削除 |
| Go Server認可 | 0行 | 約200行 | Entitlement Store連携 |
| Supabase RPC | - | 約50行 | get_user_entitlement |
| **合計** | **約1500行** | **約430行** | **-71%削減** |

### JWT検証の一元化による効果

| 項目 | 移行前 | 移行後 |
|------|--------|--------|
| JWT検証箇所 | Worker + Go Server（2箇所） | Worker（1箇所） |
| JWKSキャッシュ | Worker + Go Server（2箇所） | Worker（1箇所） |
| jose/jwt-go依存 | TypeScript + Go | TypeScript のみ |
| 検証遅延 | 二重で発生 | 一度のみ |

### 依存関係

| 対象 | 移行前 | 移行後 |
|------|--------|--------|
| JWT署名キー | 独自管理 | Supabase管理 |
| JWKS | Console + Supabase | Supabase統一 |
| 認可コードDB | 自作テーブル | Supabase管理 |
| クライアント登録 | 未実装 | Supabase Dashboard |
| 認可ロジック | なし | **Entitlement Store（DB）** |
| 権限データソース | なし | **mcpist.users, subscriptions, user_module_preferences** |

---

## 認証・認可フロー図（移行後）

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              MCP Client                                         │
└─────────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ Authorization: Bearer <token>
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        Cloudflare Worker (TypeScript)                           │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  【認証 = Authentication】                                                  │ │
│  │                                                                            │ │
│  │  Q: 存在するユーザーか？                                                    │ │
│  │                                                                            │ │
│  │         ┌─────────────────┐              ┌─────────────────┐              │ │
│  │         │    API Key      │              │      JWT        │              │ │
│  │         │   (mpt_xxx)     │              │   (Supabase)    │              │ │
│  │         └────────┬────────┘              └────────┬────────┘              │ │
│  │                  │                                │                       │ │
│  │                  ▼                                ▼                       │ │
│  │           Supabase RPC                      JWKS検証                      │ │
│  │          validate_api_key                    (jose)                       │ │
│  │                  │                                │                       │ │
│  │                  └────────────┬───────────────────┘                       │ │
│  │                               ▼                                           │ │
│  │                         user_id 取得                                      │ │
│  │                                                                            │ │
│  │  失敗時: 401 Unauthorized                                                  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  成功時: X-User-ID, X-Auth-Type, X-Gateway-Secret を付与                        │
└─────────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Go MCP Server                                      │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  【認可 = Authorization】（JWT検証は行わない - Workerで完了済み）           │ │
│  │                                                                            │ │
│  │  Q: このリクエストを実行する権限があるか？                                  │ │
│  │                                                                            │ │
│  │  1. Gateway Secret検証 → Workerからの通信を確認（JWT再検証は不要）         │ │
│  │                                                                            │ │
│  │  2. Entitlement Store問い合わせ（Supabase RPC）                            │ │
│  │     ┌─────────────────────────────────────────────────────────────────┐   │ │
│  │     │  get_user_entitlement(user_id)                                  │   │ │
│  │     │  → users.status (active/suspended/deleted)                      │   │ │
│  │     │  → subscriptions → plans (rate_limit, quota)                    │   │ │
│  │     │  → user_module_preferences (enabled modules)                    │   │ │
│  │     └─────────────────────────────────────────────────────────────────┘   │ │
│  │                                                                            │ │
│  │  3. アカウント状態チェック                                                 │ │
│  │     - status != 'active' → 403                                            │ │
│  │                                                                            │ │
│  │  4. ツールアクセス権チェック（リクエストごと）                             │ │
│  │     - モジュールが有効化されているか？                                     │ │
│  │     - 例: notion → user_module_preferences で is_enabled=true か？        │ │
│  │                                                                            │ │
│  │  失敗時: 403 Forbidden                                                     │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  成功時: MCPリクエスト処理                                                      │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

---

## 通信パターンとGateway経由の判断基準

### MCPistの構成要素

| コンポーネント | 役割 | 実装 |
|---------------|------|------|
| **Auth Server** | ユーザー認証・JWT発行 | Supabase Auth |
| **Entitlement Store** | 権限・課金情報 | Supabase PostgreSQL |
| **Token Vault** | OAuthトークン暗号化保存 | Supabase Vault |
| **Console** | 管理UI | Next.js (Vercel) |
| **MCP Server** | MCPプロトコル処理 | Go (Render/Koyeb) |
| **API Gateway** | 認証・Rate Limit・LB | Cloudflare Worker |

### 通信パターン一覧

| 通信元 | 通信先 | Worker経由 | 理由 |
|--------|--------|-----------|------|
| **MCP Client → MCP Server** | Go Server | ✅ **経由** | 外部からのリクエスト、認証必須 |
| **Console → MCP Server** | Go Server | ✅ **経由** | 同上（ConsoleもMCPクライアントとして振る舞う場合） |
| **Console → Supabase Auth** | Supabase | ❌ 直接 | Supabase SDKが直接通信 |
| **Console → Supabase DB** | Supabase | ❌ 直接 | Supabase SDK + RLS |
| **Go Server → Supabase DB** | Supabase | ❌ 直接 | 内部通信（Service Role Key） |
| **Go Server → Token Vault** | Supabase | ❌ 直接 | 内部通信（Service Role Key） |
| **Worker → Supabase** | Supabase | ❌ 直接 | 認証検証のためのRPC呼び出し |

### 通信フロー図

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│                           ┌─────────────────┐                                   │
│                           │   Supabase      │                                   │
│                           │  ┌───────────┐  │                                   │
│                           │  │   Auth    │  │                                   │
│                           │  ├───────────┤  │                                   │
│                           │  │    DB     │  │                                   │
│                           │  │(Entitle-  │  │                                   │
│                           │  │  ment)    │  │                                   │
│                           │  ├───────────┤  │                                   │
│                           │  │  Vault    │  │                                   │
│                           │  └───────────┘  │                                   │
│                           └────────┬────────┘                                   │
│                                    │                                            │
│            ┌───────────────────────┼───────────────────────┐                    │
│            │                       │                       │                    │
│            ▼                       ▼                       ▼                    │
│     ┌────────────┐          ┌────────────┐          ┌────────────┐             │
│     │  Console   │          │   Worker   │          │ Go Server  │             │
│     │            │          │            │          │            │             │
│     │ 直接通信   │          │ 認証RPC    │          │ 直接通信   │             │
│     │ (SDK+RLS)  │          │ のみ直接   │          │(Service    │             │
│     │            │          │            │          │  Role)     │             │
│     └─────┬──────┘          └──────┬─────┘          └──────┬─────┘             │
│           │                        │                       │                    │
│           │                  ┌─────┴─────┐                 │                    │
│           │                  │  Gateway  │                 │                    │
│           │                  │  経由     │                 │                    │
│           │                  └─────┬─────┘                 │                    │
│           │                        │                       │                    │
│           ▼                        ▼                       │                    │
│     ┌────────────┐          ┌────────────┐                 │                    │
│     │  Browser   │          │ MCP Client │                 │                    │
│     │  (User)    │          │ (Claude等) │                 │                    │
│     └────────────┘          └────────────┘                 │                    │
│                                                            │                    │
│     【直接通信OK】          【Gateway必須】          【直接通信OK】              │
│     - 信頼されたSDK         - 外部クライアント      - 内部通信                  │
│     - RLSで保護             - 認証必須              - Service Role              │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 原則: Gateway経由の判断基準

#### 1. 外部 → 内部: Gateway必須

```
外部クライアント（MCP Client、ブラウザからのAPI呼び出し）
    ↓ Gateway経由
内部サーバー（Go Server）
```

**理由:**
- 認証（誰？）
- Rate Limit（濫用防止）
- DDoS対策
- 地理的負荷分散

#### 2. 内部 → 内部: 直接通信

```
Go Server → Supabase DB
          → Token Vault
          → Entitlement Store
```

**理由:**
- 内部ネットワーク / VPC内
- Service Role Key（信頼された認証）
- Gateway経由は不要なオーバーヘッド

#### 3. フロントエンド → BaaS: 直接通信

```
Console (Browser) → Supabase Auth
                  → Supabase DB (RLS)
```

**理由:**
- Supabase SDKが直接通信を前提に設計
- RLS（Row Level Security）でDBレベルのアクセス制御
- JWT検証はSupabase側で実施

### Consoleの二重性

Consoleには2つの通信パターンがある:

| パターン | 経路 | 認証 |
|----------|------|------|
| **管理UI** | Console → Supabase直接 | Supabase Auth Session（Cookie） |
| **MCP接続テスト** | Console → Worker → Go Server | Bearer Token（API Key / JWT） |

```typescript
// パターン1: 管理UI（Supabase直接）
// apps/console/src/app/(console)/settings/page.tsx
const supabase = createClient();
const { data } = await supabase.from('user_module_preferences').select('*');

// パターン2: MCP接続テスト（Worker経由）
// apps/console/src/app/(console)/playground/page.tsx
const response = await fetch('https://api.mcpist.app/mcp', {
  headers: { 'Authorization': `Bearer ${apiKey}` }
});
```

### 一般的なWeb/API構成との比較

| パターン | 一般的な構成 | MCPist |
|----------|-------------|--------|
| Browser → API | Gateway経由（認証・Rate Limit） | ✅ 同様（MCP Client → Worker → Go） |
| API → DB | 直接（内部ネットワーク） | ✅ 同様（Go → Supabase） |
| Browser → Auth | 直接（Auth0/Firebase等） | ✅ 同様（Console → Supabase Auth） |
| Frontend → BaaS | 直接（SDK + RLS） | ✅ 同様（Console → Supabase DB） |

**結論:** MCPistの通信パターンは一般的なベストプラクティスに準拠している。

---

## 補足: 管理者判定について

現在の`mcpist.users`テーブルには`role`列がない。管理者判定が必要な場合、以下のいずれかで対応:

### Option A: usersテーブルにrole列を追加

```sql
ALTER TABLE mcpist.users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'
  CHECK (role IN ('user', 'admin'));
```

### Option B: 別テーブルで管理

```sql
CREATE TABLE mcpist.admins (
    user_id UUID PRIMARY KEY REFERENCES mcpist.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Option C: 環境変数で管理（開発段階）

```go
// 管理者UUIDを環境変数で指定
adminUserIDs := strings.Split(os.Getenv("ADMIN_USER_IDS"), ",")
```

**推奨**: 初期段階ではOption C、本番運用開始時にOption Aに移行

---

## 参考資料

- [Supabase OAuth 2.1 Server](https://supabase.com/docs/guides/auth/oauth-server)
- [Supabase MCP向け設定](https://supabase.com/docs/guides/auth/oauth-server/mcp)
- [RFC 9728 - OAuth 2.0 Protected Resource Metadata](https://datatracker.ietf.org/doc/rfc9728/)
- [dwhbi console認証実装](C:\Users\m_fukuda\Documents\dwhbi\packages\console\src\app\api\mcp\lib\auth.ts)
- [mcpist Entitlement Store](C:\Users\m_fukuda\Documents\mcpist\supabase\migrations\00000000000001_entitlement_store.sql)
