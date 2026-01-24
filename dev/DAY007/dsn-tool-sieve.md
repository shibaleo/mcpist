# MCPist 認証・認可アーキテクチャ

## 概要

MCPサーバーにおける認証（Authentication）と認可（Authorization）の設計。

## 全体構成
```
MCPクライアント（Claude Desktop等）
    │
    │ Authorization Code Flow（初回）
    │ Token Refresh（定期）
    ↓
┌─────────────────────────────────────┐
│  Supabase Auth                      │
│  ├─ OAuth 2.0 Authorization Server  │
│  ├─ JWKS公開（/.well-known/jwks）   │
│  └─ Supabase DB と同居              │
└─────────────────────────────────────┘
    │
    │ Access Token (JWT)
    ↓
┌─────────────────────────────────────┐
│  MCPサーバー (Go)                    │
│                                     │
│  AuthMiddleware                     │
│    └─ JWT署名検証（JWKSキャッシュ）  │
│    └─ user_id抽出                   │
│            ↓                        │
│  SieveMiddleware                    │
│    └─ Sieve.GetAllowedTools(userID) │
│    └─ ゲート判定（許可/拒否）        │
│            ↓                        │
│  mcp.HandleRequest                  │
│    ├─ tools/list → 2ツールを返す    │
│    ├─ get_module_schema             │
│    │     └─ Sieve参照 → フィルタ    │
│    └─ call_module_tool              │
│          └─ モジュール呼び出し       │
└─────────────────────────────────────┘
    │
    ↓
┌─────────────────────────────────────┐
│  Supabase DB                        │
│  ├─ ユーザー権限テーブル            │
│  └─ 外部APIトークン（暗号化）       │
└─────────────────────────────────────┘
```

## コンポーネント配置

| コンポーネント | 場所 | 理由 |
|---------------|------|------|
| Auth | Supabase（DBと同居） | マネージド、JWKS提供 |
| MCPサーバー | 別ホスト（Koyeb等） | スケール独立、計算リソース |
| DB | Supabase | 権限・トークン永続化 |
| UI | Vercel | 静的配信、Edge最適化 |

## 認証フロー

### JWT検証方式

ローカル検証（署名検証）を採用。認証サーバーへの問い合わせ不要。
```
1. 起動時: 認証サーバーの公開鍵を取得
   GET https://auth.example.com/.well-known/jwks.json

2. リクエストごと: ローカルで検証
   - 署名の正当性
   - 有効期限（exp）
   - 発行者（iss）
```

| 方式 | メリット | デメリット |
|------|----------|------------|
| ローカル検証 | 高速、認証サーバー障害に強い | トークン失効を即座に反映できない |
| イントロスペクション | リアルタイムで失効反映 | レイテンシ、認証サーバー依存 |

### トークンリフレッシュ

MCP 2025-03-26仕様でOAuth 2.0をサポート。MCPクライアントがリフレッシュを行う。
```
MCPクライアント
    ├─ 初回: Authorization Code Flow → Access Token + Refresh Token取得
    └─ 以降: Access Token期限切れ時 → 認証サーバーにRefresh Token送信
```

## 認可（Tool Sieve）

### 設計方針

- Sieve: 純粋なDBキャッシュに徹する
- Middleware: ゲートロジックを持つ
- Handler: フィルタロジックを持つ

### Sieve（DBキャッシュ）
```go
type Sieve struct {
    cache sync.Map  // user_id → *CacheEntry
    db    *supabase.Client
}

type CacheEntry struct {
    tools     []string
    expiresAt time.Time
}

// ユーザーが使えるツール一覧を返す（キャッシュ優先）
func (s *Sieve) GetAllowedTools(userID string) []string {
    if entry, ok := s.cache.Load(userID); ok {
        if time.Now().Before(entry.(*CacheEntry).expiresAt) {
            return entry.(*CacheEntry).tools
        }
    }
    tools := s.fetchFromDB(userID)
    s.cache.Store(userID, &CacheEntry{
        tools:     tools,
        expiresAt: time.Now().Add(5 * time.Minute),
    })
    return tools
}

// キャッシュクリア（課金変更時等）
func (s *Sieve) InvalidateUser(userID string) {
    s.cache.Delete(userID)
}

func (s *Sieve) InvalidateAll() {
    s.cache = sync.Map{}
}
```

### SieveMiddleware（ゲート）
```go
type SieveMiddleware struct {
    sieve *Sieve
}

func (m *SieveMiddleware) Handle(ctx context.Context, req Request) Response {
    userID := GetUserID(ctx)
    
    if req.Method == "tools/call" && req.Params.Name == "call_module_tool" {
        allowed := m.sieve.GetAllowedTools(userID)
        toolName := req.Params.Module + ":" + req.Params.ToolName
        
        if !contains(allowed, toolName) {
            return ErrorResponse("tool not permitted")
        }
    }
    
    return next(ctx, req)
}
```

### ModuleSchemaHandler（フィルタ）
```go
type ModuleSchemaHandler struct {
    sieve    *Sieve
    registry *modules.Registry
}

func (h *ModuleSchemaHandler) Handle(ctx context.Context, req Request) Response {
    userID := GetUserID(ctx)
    module := req.Params.Module
    
    allowed := h.sieve.GetAllowedTools(userID)
    allSchema := h.registry.GetSchema(module)
    
    filtered := filter(allSchema, allowed)
    return Response{Schema: filtered}
}
```

## キャッシュ設計

| キャッシュ | TTL | 場所 | 用途 |
|-----------|-----|------|------|
| JWKS | 1時間 | AuthMiddleware | JWT公開鍵 |
| 権限 | 5分 | Sieve | ユーザー許可ツール |

### メモリ消費見積もり

- ユーザーあたり: user_id(36byte) + ツール名リスト(50ツール × 30byte) ≈ 1.5KB
- 100ユーザー: 150KB
- 1000ユーザー: 1.5MB

TTLで自動期限切れするため、アクティブユーザー分のみ。

### キャッシュ更新トリガー

- TTL経過: 自動期限切れ → 次回アクセス時にDB再取得
- 課金変更: Webhook → `InvalidateUser(userID)` → 即時反映

## ツール構成

| ツール | 役割 | Sieve関与 |
|--------|------|-----------|
| `get_module_schema` | ユーザーが使えるツールのスキーマ返却 | 照会（フィルタ） |
| `call_module_tool` | 実際のツール呼び出し | ゲート（許可チェック） |

## 処理フロー
```
MCPクライアント
    │ tools/call("call_module_tool", {module: "notion", tool: "create_page"})
    ↓
AuthMiddleware
    └─ JWT検証 → user_id抽出
          ↓
SieveMiddleware
    ├─ Sieve.GetAllowedTools(userID) → ["notion:search", "notion:get_page"]
    ├─ "notion:create_page" in allowed?
    │     └─ No → 拒否（エラー返却）
    └─ Yes → 通過
          ↓
mcp.HandleRequest
    └─ call_module_tool実行
```
```
MCPクライアント
    │ tools/call("get_module_schema", {module: "notion"})
    ↓
AuthMiddleware → SieveMiddleware（通過）
    ↓
ModuleSchemaHandler
    ├─ Sieve.GetAllowedTools(userID) → ["notion:search", "notion:get_page"]
    ├─ Registry.GetSchema("notion") → 全14ツールのスキーマ
    └─ filter → 2ツールのスキーマのみ返却
```