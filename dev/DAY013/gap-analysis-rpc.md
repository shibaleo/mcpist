# RPC実装ギャップ分析

## 概要

本ドキュメントは、既存実装と新RPC設計（dsn-rpc.md）の差異を分析し、マイグレーション時に必要な変更を明確にする。

---

## 1. 既存RPCと新RPC設計の対応

### 1.1 RPC関数の対応表

| 新RPC設計 | 既存実装 | 状態 |
|-----------|----------|------|
| lookup_user_by_key_hash | public.validate_api_key (api_keys版) | ⚠️ 要リネーム・修正（ハッシュ入力に変更） |
| get_user_context | public.get_user_entitlement | ⚠️ 要リネーム・修正 |
| consume_credit | public.deduct_credits | ⚠️ 要リネーム・修正 |
| get_module_token | - | ❌ 未実装 |
| update_module_token | - | ❌ 未実装 |
| add_paid_credits | - | ❌ 未実装 |
| reset_free_credits | - | ❌ 未実装 |
| generate_api_key | public.generate_api_key | ✅ 存在 |
| list_api_keys | public.list_api_keys | ✅ 存在 |
| revoke_api_key | public.revoke_api_key | ✅ 存在 |
| list_service_connections | - | ❌ 未実装 |
| upsert_service_token | - | ❌ 未実装 |
| delete_service_token | - | ❌ 未実装 |

---

## 2. lookup_user_by_key_hash（旧validate_api_key）

### 2.1 設計変更

新設計では`validate_api_key` → `lookup_user_by_key_hash`にリネーム。

**変更理由:**
- Worker側でSHA-256ハッシュを計算し、キャッシュと組み合わせてVault呼び出しを削減
- RPC側ではハッシュを受け取って検索するのみ

### 2.2 二重定義問題（既存）

現在、`validate_api_key`が2つのマイグレーションで定義されている:

#### oauth_tokens版（00000000000004_rpc_functions.sql）
```sql
CREATE OR REPLACE FUNCTION public.validate_api_key(
  p_api_key TEXT,
  p_service TEXT
)
RETURNS JSONB
-- oauth_tokensテーブルから検索
-- 返却: {valid, user_id, error}
```

#### api_keys版（00000000000007_api_keys.sql）
```sql
CREATE OR REPLACE FUNCTION public.validate_api_key(p_key TEXT)
RETURNS JSONB
-- api_keysテーブルから検索（SHA-256ハッシュをRPC内で計算）
-- 返却: {valid, user_id, key_name, scopes, error}
```

### 2.3 Go実装の問題

`apps/mcp-server/internal/middleware/middleware.go:157-198`:
```go
result, err := m.supabase.Rpc("validate_api_key", "", map[string]interface{}{
    "p_api_key": token,
    "p_service": "mcpist",
})
```

**問題**: Go実装はoauth_tokens版（2引数）を呼び出している。

### 2.4 対応方針

1. oauth_tokens版の`validate_api_key`を削除
2. api_keys版を`lookup_user_by_key_hash`にリネーム
3. 入力を`p_key` → `p_key_hash`に変更（Worker側でハッシュ計算）
4. Go実装を修正:
   - Worker側でSHA-256ハッシュ計算
   - ハッシュをキーとしてキャッシュ参照
   - キャッシュミス時のみRPC呼び出し

---

## 3. creditsテーブルの構造差異

### 3.1 現行構造

```sql
CREATE TABLE mcpist.credits (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    balance INTEGER NOT NULL DEFAULT 0,  -- 単一残高
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

### 3.2 新設計

```sql
-- free_credits, paid_credits の分離が必要
-- 新設計では：
-- - free_credits: 月次無料クレジット
-- - paid_credits: 購入クレジット
-- - 消費順序: free_credits優先
```

### 3.3 マイグレーション案

```sql
ALTER TABLE mcpist.credits
    ADD COLUMN free_credits INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN paid_credits INTEGER NOT NULL DEFAULT 0;

-- 既存balanceをfree_creditsに移行
UPDATE mcpist.credits SET free_credits = balance;

-- balance列は互換性のため残すか、削除するか検討
```

---

## 4. 廃止予定テーブル

以下のテーブルは新設計では不要または統合される:

| テーブル | 理由 |
|---------|------|
| mcpist.plans | 料金プラン機能は実装しない（個人運用のため） |
| mcpist.subscriptions | 同上 |
| mcpist.usage | credit_transactionsで代替 |
| mcpist.tool_costs | モジュール単位の課金は1クレジット固定 |
| mcpist.user_module_preferences | module_settings/tool_settingsに統合 |
| mcpist.mcp_tokens | api_keysに統合 |

---

## 5. 新規作成が必要なテーブル/カラム

### 5.1 creditsテーブルの修正

- `free_credits` カラム追加
- `paid_credits` カラム追加

### 5.2 credit_transactionsテーブルの修正

現行:
```sql
transaction_type IN ('purchase', 'consume', 'refund', 'bonus', 'expire')
```

新設計:
```sql
type IN ('consume', 'purchase', 'monthly_reset')
```

追加カラム:
- `module TEXT`
- `tool TEXT`
- `request_id TEXT`
- `task_id TEXT`

### 5.3 新規テーブル

- `mcpist.module_settings` - モジュール有効化設定
- `mcpist.tool_settings` - ツール無効化設定
- `mcpist.service_tokens` - サービストークン管理（user_id + service → credentials_secret_id）
- `mcpist.processed_webhook_events` - ✅ 既に存在

### 5.4 service_tokensテーブル設計

```sql
CREATE TABLE mcpist.service_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id),
    service TEXT NOT NULL,  -- モジュール名（notion, github等）
    credentials_secret_id UUID NOT NULL,  -- vault.secretsへの参照
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, service)
);
```

**設計ポイント:**
- 1ユーザー×1サービス = 1レコード（1トークンタイプのみ）
- credentials_secret_idでVaultのJSON secret全体を参照
- `_auth_type`（oauth2/api_key）はVault JSON内で管理

---

## 6. 実装の変更箇所

### 6.1 Worker（Cloudflare Workers / TypeScript）

**現行**:
WorkerでAPIキー認証を処理。RPCを呼び出してuser_idを取得。

```typescript
// 現行: validate_api_keyを呼び出し（RPC側でハッシュ計算）
const result = await supabase.rpc('validate_api_key', {
    p_key: token,
});
```

**変更後**:
Worker側でハッシュ計算 + キャッシュを追加。

```typescript
// Worker側でハッシュ計算 + キャッシュ
const keyHash = await sha256Hex(token);
const cached = await cache.get(keyHash);
if (cached) {
    return cached.userId;
}

// キャッシュミス時のみRPC呼び出し
const result = await supabase.rpc('lookup_user_by_key_hash', {
    p_key_hash: keyHash,
});
// 成功時はキャッシュに保存
await cache.put(keyHash, { userId: result.user_id }, { expirationTtl: 300 });
```

**変更点:**
- RPC名: `validate_api_key` → `lookup_user_by_key_hash`
- 入力: 生キー → SHA-256ハッシュ
- キャッシュ機構の追加（ハッシュをキーとして）

### 6.3 entitlement/store.go（MCP Server / Go）

**現行**: `get_user_entitlement`を呼び出し

**変更後**: `get_user_context`に変更

返却値の構造も変更:
```go
// 現行
type Entitlement struct {
    PlanName      string
    RateLimitRPM  int
    QuotaMonthly  int
    // ...
}

// 新設計
type UserContext struct {
    AccountStatus   string
    FreeCredits     int
    PaidCredits     int
    EnabledModules  []string
    DisabledTools   map[string][]string
}
```

### 6.4 クレジット消費処理（MCP Server / Go）

**現行**: `deduct_credits`

**変更後**: `consume_credit`

引数変更:
```go
// 現行
params := map[string]interface{}{
    "p_user_id": userID,
    "p_amount":  amount,
}

// 新設計
params := map[string]interface{}{
    "p_user_id":    userID,
    "p_module":     module,
    "p_tool":       tool,
    "p_amount":     amount,
    "p_request_id": requestID,
    "p_task_id":    taskID,  // optional
}
```

### 6.5 トークン取得・更新処理（MCP Server / Go・新規）

**新規追加**: `get_module_token` / `update_module_token`

```go
// トークン取得（モジュール初期化時）
result, err := m.supabase.Rpc("get_module_token", "", map[string]interface{}{
    "p_user_id": userID,
    "p_module":  module,
})
// result: Vault JSON全体（_auth_type, _expires_at含む）

// リフレッシュ後のトークン保存
newCredentials := map[string]interface{}{
    "access_token":  newToken,
    "refresh_token": refreshToken,
    "client_id":     clientID,
    "client_secret": clientSecret,
    "_auth_type":    "oauth2",
    "_expires_at":   expiresAt.Format(time.RFC3339),
}
result, err := m.supabase.Rpc("update_module_token", "", map[string]interface{}{
    "p_user_id":     userID,
    "p_module":      module,
    "p_credentials": newCredentials,  // 全体を上書き
})
```

---

## 7. マイグレーション優先順位

### Phase 1: 基盤整備
1. creditsテーブルにfree_credits/paid_credits追加
2. credit_transactionsにmodule/tool/request_id/task_id追加
3. module_settings/tool_settingsテーブル作成
4. service_tokensテーブル作成

### Phase 2: RPC関数
1. oauth_tokens版validate_api_keyを削除
2. api_keys版を`lookup_user_by_key_hash`にリネーム（p_key → p_key_hash）
3. get_user_context作成
4. consume_credit作成（冪等性対応）
5. get_module_token / update_module_token作成
6. list_service_connections / upsert_service_token / delete_service_token作成

### Phase 3: 実装修正
1. Worker（TypeScript）: ハッシュ計算 + キャッシュ + lookup_user_by_key_hash呼び出し
2. MCP Server（Go）entitlement/store.go: get_user_contextへ変更
3. MCP Server（Go）クレジット消費処理の更新
4. MCP Server（Go）トークン取得・リフレッシュ処理の追加

### Phase 4: クリーンアップ
1. 不要テーブルの削除（plans, subscriptions等）
2. 旧RPC関数の削除（validate_api_key, get_user_entitlement, deduct_credits等）

---

## 8. Console Frontend影響

APIキー関連のRPC（generate_api_key, list_api_keys, revoke_api_key）は既存実装と互換性があり、変更不要。

ただし、以下の新機能追加時は対応が必要:
- クレジット表示（free_credits / paid_credits分離）
- モジュール設定画面
- ツール設定画面

---

## 関連ドキュメント

| ドキュメント | 場所 |
|-------------|------|
| RPC設計書 | mcpist/docs/design/dsn-rpc.md |
| テーブル設計書 | mcpist/docs/design/dtl-dsn-tbl.md |
| クレジットモデル仕様 | mcpist/docs/specification/dtl-spc/dtl-spc-credit-model.md |
