# リファクタリング実行計画（migration-plan）

## 概要

本ドキュメントは、既存実装を新RPC設計・テーブル設計に移行するための実行計画を定義する。

### 前提条件

- **Supabaseにユーザーデータは存在しない**
- `supabase db reset --remote`で全マイグレーションをクリーン適用する
- 不要なマイグレーションファイルは削除し、新設計に基づいたファイルのみを残す

### スコープ外

以下は本計画のスコープ外とし、Google Calendar OAuth連携実装時に対応する：

- **トークンリフレッシュ機構**
  - `update_module_token` RPC関数の実装
  - `internal/token/refresh.go` の実装
  - OAuthトークン更新の検証

現状はNotionのAPI Secret（長期トークン）のみが実例であり、リフレッシュ不要。
テーブル定義（`service_tokens`）およびVault JSON形式は本計画で整備する。

### 参照ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| gap-analysis-rpc.md | 既存実装とのギャップ分析 |
| dsn-tbl.md | テーブル設計書 |
| dsn-rpc.md / dtl-dsn-rpc.md | RPC関数設計書 |

---

## 開発環境構成

```
┌─────────────────────────────────────────────────────────────────┐
│                         ローカル開発                              │
├─────────────────────────────────────────────────────────────────┤
│  Next.js (Console)     Worker (Wrangler)    Go Server           │
│  localhost:3000        localhost:8787        localhost:8080      │
└────────┬───────────────────┬────────────────────┬───────────────┘
         │                   │                    │
         ▼                   ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Supabase Cloud (dev)                        │
│                                                                   │
│  PostgreSQL + Auth + Vault                                       │
│  https://xxx.supabase.co                                         │
└─────────────────────────────────────────────────────────────────┘
```

### デプロイフロー

```
ローカル開発 → テスト → デプロイ

1. Supabase: supabase db reset --remote（クリーン適用）
2. Console:  vercel deploy
3. Worker:   wrangler deploy
4. Server:   docker build → Koyeb/Render deploy
```

---

## 実行フェーズ

### Phase 1: Supabaseマイグレーション整理（クリーン適用）

**目的:** 不要なマイグレーションを削除し、新設計に基づいたスキーマを適用

#### 1.1 削除対象マイグレーションファイル

以下のファイルは新設計で不要となるため削除:

| ファイル | 削除理由 |
|---------|---------|
| `xxxx_plans.sql` | plansテーブル不要 |
| `xxxx_subscriptions.sql` | subscriptionsテーブル不要 |
| `xxxx_usage.sql` | credit_transactionsで代替 |
| `xxxx_tool_costs.sql` | 1クレジット固定のため不要 |
| `xxxx_user_module_preferences.sql` | module_settings/tool_settingsに統合 |
| `xxxx_mcp_tokens.sql` | api_keysに統合 |
| `xxxx_oauth_tokens.sql` | service_tokens + Vaultに移行 |
| `xxxx_rpc_functions.sql`（oauth_tokens版validate_api_key含む） | 新RPC設計で置き換え |

#### 1.2 残す/修正するマイグレーションファイル

| ファイル | 対応 |
|---------|------|
| `xxxx_init.sql` | スキーマ作成、残す |
| `xxxx_users.sql` | 残す |
| `xxxx_credits.sql` | **修正**: free_credits/paid_credits構造に変更 |
| `xxxx_credit_transactions.sql` | **修正**: module, tool, request_id, task_id追加 |
| `xxxx_modules.sql` | 残す |
| `xxxx_api_keys.sql` | 残す |
| `xxxx_prompts.sql` | 残す |
| `xxxx_processed_webhook_events.sql` | 残す |

#### 1.3 新規作成マイグレーションファイル

| ファイル | 内容 |
|---------|------|
| `xxxx_service_tokens.sql` | service_tokensテーブル |
| `xxxx_module_settings.sql` | module_settingsテーブル |
| `xxxx_tool_settings.sql` | tool_settingsテーブル |
| `xxxx_rpc_mcp_server.sql` | lookup_user_by_key_hash, get_user_context, consume_credit, get_module_token |
| `xxxx_rpc_console.sql` | list_service_connections, upsert_service_token, delete_service_token |
| `xxxx_rpc_webhook.sql` | add_paid_credits |
| `xxxx_rpc_cron.sql` | reset_free_credits |

> **スコープ外:** `update_module_token` RPCはテーブル定義のみ行い、実装はGoogle Calendar OAuth連携時に対応。

#### 1.4 シードデータ

以下のシードファイルは維持:

| ファイル | 内容 |
|---------|------|
| `seed.sql` | modulesマスタデータ（notion, github, jira, confluence, supabase, google_calendar, microsoft_todo, rag） |

#### 1.5 適用コマンド

```bash
# 1. 不要ファイル削除後、リモートDBをリセット（マイグレーション + シード自動適用）
supabase db reset --remote

# 2. RPC動作確認（Supabase Studio または psql）
SELECT mcpist.lookup_user_by_key_hash('テストハッシュ');
SELECT mcpist.get_user_context('テストUUID');
```

---

### Phase 2: Worker実装修正（Cloudflare Workers / TypeScript）

**目的:** APIキー認証をハッシュベース + キャッシュに変更

#### 2.1 変更内容

| ファイル | 変更内容 |
|---------|---------|
| `src/auth/api-key.ts` | SHA-256ハッシュ計算、キャッシュ参照、RPC呼び出し変更 |
| `src/lib/cache.ts` | KVキャッシュヘルパー追加（既存なら流用） |

#### 2.2 実装手順

```typescript
// 1. SHA-256ハッシュ計算関数
async function sha256Hex(key: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(key);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(hash))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

// 2. キャッシュ付きAPIキー検証
async function validateApiKey(token: string, env: Env): Promise<ValidateResult> {
  const keyHash = await sha256Hex(token);

  // キャッシュ参照
  const cached = await env.KV.get(`apikey:${keyHash}`, 'json');
  if (cached) {
    return cached as ValidateResult;
  }

  // RPC呼び出し
  const result = await supabase.rpc('lookup_user_by_key_hash', {
    p_key_hash: keyHash,
  });

  // キャッシュ保存（5分TTL）
  if (result.valid) {
    await env.KV.put(`apikey:${keyHash}`, JSON.stringify(result), {
      expirationTtl: 300,
    });
  }

  return result;
}
```

#### 2.3 ローカルテスト

```bash
# Wranglerでローカル起動
wrangler dev

# curlでテスト
curl -H "Authorization: Bearer mpt_xxx" http://localhost:8787/health
```

#### 2.4 デプロイ

```bash
wrangler deploy
```

---

### Phase 3: MCP Server実装修正（Go）

**目的:** RPC呼び出しを新設計に変更、トークン取得対応

> **スコープ外:** トークンリフレッシュ機構（`update_module_token` RPC、`internal/token/refresh.go`）は本フェーズでは実装しない。
> 現状はNotionのAPI Secret（長期トークン）のみが実例であり、リフレッシュ不要。
> Google Calendar等のOAuth連携実装時に対応する。

#### 3.1 変更内容

| パッケージ/ファイル | 変更内容 |
|-------------------|---------|
| `internal/entitlement/store.go` | get_user_entitlement → get_user_context |
| `internal/entitlement/types.go` | UserContext構造体定義 |
| `internal/credit/store.go` | deduct_credits → consume_credit |
| `internal/token/store.go` | get_module_token（新規）※ update_module_tokenは後日実装 |

#### 3.2 実装手順

**3.2.1 UserContext構造体**

```go
// internal/entitlement/types.go
type UserContext struct {
    AccountStatus   string              `json:"account_status"`
    FreeCredits     int                 `json:"free_credits"`
    PaidCredits     int                 `json:"paid_credits"`
    EnabledModules  []string            `json:"enabled_modules"`
    DisabledTools   map[string][]string `json:"disabled_tools"`
}
```

**3.2.2 consume_credit呼び出し**

```go
// internal/credit/store.go
func (s *Store) ConsumeCredit(ctx context.Context, params ConsumeParams) (*ConsumeResult, error) {
    result, err := s.supabase.Rpc("consume_credit", "", map[string]interface{}{
        "p_user_id":    params.UserID,
        "p_module":     params.Module,
        "p_tool":       params.Tool,
        "p_amount":     params.Amount,
        "p_request_id": params.RequestID,
        "p_task_id":    params.TaskID,
    })
    // ...
}
```

**3.2.3 トークン取得**

```go
// internal/token/store.go
func (s *Store) GetModuleToken(ctx context.Context, userID, module string) (*Credentials, error) {
    result, err := s.supabase.Rpc("get_module_token", "", map[string]interface{}{
        "p_user_id": userID,
        "p_module":  module,
    })
    // ... Vault JSON全体を返却
}

// NOTE: UpdateModuleToken は Google Calendar OAuth実装時に追加
```

#### 3.3 ローカルテスト

```bash
# ローカル起動
go run ./cmd/server

# curlでテスト（Workerを経由しないダイレクトテスト用）
curl -X POST http://localhost:8080/test/consume-credit \
  -H "Content-Type: application/json" \
  -d '{"user_id":"xxx","module":"notion","tool":"search","amount":1}'
```

#### 3.4 ビルド・デプロイ

```bash
# Dockerビルド
docker build -t mcpist-server:latest .

# Koyebデプロイ
koyeb service update mcp-server --docker mcpist-server:latest

# Renderデプロイ（自動 or 手動トリガー）
# Render Dashboard から Deploy trigger
```

---

### Phase 4: Console実装修正（Next.js）

**目的:** クレジット表示の分離、サービス接続画面対応

#### 4.1 変更内容

| ファイル | 変更内容 |
|---------|---------|
| `app/dashboard/page.tsx` | free_credits / paid_credits分離表示 |
| `app/settings/connections/page.tsx` | サービス接続一覧（新規） |
| `lib/supabase/rpc.ts` | 新RPC呼び出しヘルパー |

#### 4.2 ローカルテスト

```bash
npm run dev
# http://localhost:3000 で確認
```

#### 4.3 デプロイ

```bash
vercel deploy --prod
```

---

### Phase 5: 動作確認・検証

**目的:** 全コンポーネントの統合動作確認

#### 5.1 検証項目

| 項目 | 確認内容 |
|------|---------|
| APIキー認証 | Worker経由でMCP Serverにリクエスト、キャッシュ動作確認 |
| クレジット消費 | ツール実行時のクレジット減算、トランザクション記録 |
| トークン取得 | モジュール初期化時のVaultトークン取得（Notion API Secret） |
| Console表示 | free_credits/paid_credits分離表示 |

> **スコープ外:** トークン更新（OAuthリフレッシュ）はGoogle Calendar実装時に検証。

**注記:** `supabase db reset --remote`でクリーン適用するため、旧テーブル・旧RPC関数の削除マイグレーションは不要。

---

## 実行順序サマリー

```
Phase 1: Supabaseマイグレーション整理
    ├── 1.1 不要ファイル削除
    ├── 1.2 既存ファイル修正
    ├── 1.3 新規ファイル作成
    └── 1.4 supabase db reset --remote
           │
           ▼
Phase 2: Worker実装修正
    ├── 2.1-2.2 実装
    ├── 2.3 ローカルテスト
    └── 2.4 デプロイ
           │
           ▼
Phase 3: MCP Server実装修正
    ├── 3.1-3.2 実装
    ├── 3.3 ローカルテスト
    └── 3.4 ビルド・デプロイ
           │
           ▼
Phase 4: Console実装修正
    ├── 4.1 実装
    ├── 4.2 ローカルテスト
    └── 4.3 デプロイ
           │
           ▼
Phase 5: 動作確認・検証
    └── 5.1 統合テスト
```

---

## リスク・考慮事項

### クリーン適用の利点

| 項目 | 説明 |
|------|------|
| データ移行不要 | ユーザーデータがないため、既存データの互換性を考慮する必要がない |
| マイグレーション簡素化 | 増分マイグレーションではなく、新設計を直接適用 |
| 旧実装削除不要 | 不要なマイグレーションファイルを削除するだけで対応完了 |

### ロールバック計画

| フェーズ | ロールバック方法 |
|---------|-----------------|
| Phase 1 | supabase db reset --remote（再適用） |
| Phase 2 | wrangler rollback または前バージョンdeploy |
| Phase 3 | Koyeb/Render で前バージョンにロールバック |
| Phase 4 | Vercel で前バージョンにロールバック |

---

## 工数見積もり

### フェーズ別見積もり

| フェーズ | 作業内容 | 見積もり |
|---------|---------|---------|
| **Phase 1** | Supabaseマイグレーション整理 | |
| 1.1 | 不要ファイル削除（8ファイル） | 0.5h |
| 1.2 | 既存ファイル修正（credits, credit_transactions） | 1h |
| 1.3 | 新規テーブル作成（service_tokens, module_settings, tool_settings） | 1h |
| 1.3 | RPC実装（11関数）※ update_module_tokenはスコープ外 | 5h |
| 1.5 | 適用・検証 | 0.5h |
| | **Phase 1 小計** | **8h** |
| **Phase 2** | Worker実装修正 | |
| 2.1-2.2 | SHA-256ハッシュ計算、KVキャッシュ、RPC呼び出し変更 | 2h |
| 2.3 | ローカルテスト | 0.5h |
| 2.4 | デプロイ | 0.5h |
| | **Phase 2 小計** | **3h** |
| **Phase 3** | MCP Server実装修正 | |
| 3.1-3.2 | entitlement/store.go, credit/store.go, token/store.go ※ refresh.goはスコープ外 | 3h |
| 3.3 | ローカルテスト | 1h |
| 3.4 | ビルド・デプロイ | 0.5h |
| | **Phase 3 小計** | **4.5h** |
| **Phase 4** | Console実装修正 | |
| 4.1 | ダッシュボード（free/paid分離）、サービス接続画面 | 3h |
| 4.2 | ローカルテスト | 0.5h |
| 4.3 | デプロイ | 0.5h |
| | **Phase 4 小計** | **4h** |
| **Phase 5** | 動作確認・検証 | |
| 5.1 | 統合テスト | 2h |
| | **Phase 5 小計** | **2h** |

### 総合計

| 項目 | 時間 |
|------|------|
| 作業時間合計 | 21.5h |
| バッファ（+20%） | 4.5h |
| **総見積もり** | **26h** |

### 日程目安（1日6-7時間作業想定）

| 日程 | フェーズ | 作業時間 |
|------|---------|---------|
| Day 1 | Phase 1 | 8h |
| Day 2 | Phase 2 + Phase 3 | 7.5h |
| Day 3 | Phase 4 + Phase 5 | 6h |
| Day 4 | バッファ | - |

**完了見込み: 3日間（バッファ込み4日間）**

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [gap-analysis-rpc.md](./gap-analysis-rpc.md) | ギャップ分析 |
| [dsn-rpc.md](../../docs/design/dsn-rpc.md) | RPC設計書 |
| [dtl-dsn-rpc.md](../../docs/design/dtl-dsn-rpc.md) | RPC詳細設計書 |
| [dsn-tbl.md](../../docs/design/dsn-tbl.md) | テーブル設計書 |
| [dtl-dsn-tbl.md](../../docs/design/dtl-dsn-tbl.md) | テーブル詳細設計書 |
