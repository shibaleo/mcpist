# ユーザートークン保管方式の移行計画

## 概要

Supabase Vault の用途を整理し、ユーザートークンと運営シークレットを論理的に分離する。

---

## 現状の問題

```
vault.secrets（全て混在）
├── user:xxx:google     ← ユーザーの OAuth トークン
├── user:xxx:microsoft  ← ユーザーの OAuth トークン
├── oauth_app_google    ← 運営の client_secret
└── oauth_app_microsoft ← 運営の client_secret
```

**問題点**:
1. ユーザートークンと運営シークレットが同じテーブルに混在
2. Supabase Vault の本来の用途（運営シークレット管理）と異なる
3. 監査・トラブルシューティングが困難

---

## 移行後のアーキテクチャ

```
┌─────────────────────────────────────────────────────────────────┐
│ ユーザートークン                                                  │
│ mcpist.user_credentials (pgsodium TCE で暗号化)                 │
│ ├── user_id, service, credentials (暗号化)                      │
│ └── RLS: 自分のデータのみアクセス可能                             │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 運営シークレット                                                  │
│ vault.secrets (Supabase Vault)                                  │
│ ├── oauth_app_google (client_id, client_secret)                 │
│ ├── oauth_app_microsoft                                         │
│ ├── stripe_secret_key ← 環境変数から移行                         │
│ └── stripe_webhook_secret ← 環境変数から移行                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 移行計画

### Phase 1: マイグレーション整理

現在36個のマイグレーションファイルを**9個**に統合：

| 新番号 | 内容 | 統合元 |
|--------|------|--------|
| 001 | スキーマ・Enum・トリガー関数 | 001 |
| 002 | コアテーブル（users, credits, modules, etc.） | 002（service_tokens 削除） |
| 003 | user_credentials テーブル（pgsodium TCE） | **新規** |
| 004 | RLS ポリシー | 003 |
| 005 | ユーザートリガー | 004 |
| 006 | MCP Server RPC | 005, 010-013, 018 |
| 007 | Console RPC | 006, 008, 019-028 |
| 008 | OAuth Apps（運営シークレット） | 014-017 |
| 009 | Stripe 連携 | 029-036 |

### Phase 2: user_credentials テーブル設計

```sql
-- 新テーブル: ユーザートークン専用
CREATE TABLE mcpist.user_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module TEXT NOT NULL,  -- 'google', 'microsoft', 'jira', etc.
    credentials TEXT NOT NULL,  -- pgsodium TCE で自動暗号化
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module)
);

-- pgsodium Transparent Column Encryption 設定
SELECT pgsodium.create_key(
    name := 'user_credentials_key',
    key_type := 'aead-det'
);

SECURITY LABEL FOR pgsodium ON COLUMN mcpist.user_credentials.credentials
    IS 'ENCRYPT WITH KEY ID <key_id> ASSOCIATED (id, user_id)';

-- RLS
ALTER TABLE mcpist.user_credentials ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_credentials_select ON mcpist.user_credentials
    FOR SELECT USING (auth.uid() = user_id);

-- service_role のみ INSERT/UPDATE/DELETE 可能（RPC 経由）
CREATE POLICY user_credentials_service_role ON mcpist.user_credentials
    FOR ALL TO service_role USING (true) WITH CHECK (true);
```

### Phase 3: RPC 関数の更新

**変更が必要な関数**:

| 旧関数名 | 新関数名 | 変更内容 |
|----------|----------|----------|
| `upsert_service_token` | `upsert_user_credential` | vault.secrets → mcpist.user_credentials |
| `delete_service_token` | `delete_my_credential` | vault.secrets → mcpist.user_credentials |
| `get_module_token` | `get_user_credential` | vault.decrypted_secrets → mcpist.user_credentials |
| `update_module_token` | `upsert_user_credential` | **廃止**: upsert_user_credential に統合 |
| `list_service_connections` | `list_my_credentials` | service_tokens → user_credentials |

**命名規則**:
- MCP Server 向け（`p_user_id`）: `{verb}_user_credential`
- Console 向け（`auth.uid()`）: `{verb}_my_credential`
- ユーザーID不要: `{verb}_{resource}`

**引数の統一**: すべてのRPC関数で `p_service` → `p_module` に変更

**変更不要な関数**:
- `get_oauth_app_credentials` — 運営シークレット、vault.secrets のまま
- `upsert_oauth_app` — 運営シークレット、vault.secrets のまま

### Phase 3.5: RPC 命名規則の統一

設計レビューで発見された命名規則の問題を修正：

| 旧関数名 | 新関数名 | 変更理由 |
|----------|----------|----------|
| `consume_credit` | `consume_credits` | `add_credits` と単数/複数形を統一 |

**戻り値の統一**:

| RPC | 旧 | 新 |
|-----|-----|-----|
| `consume_credits` | `remaining_free`, `remaining_paid` | `free_credits`, `paid_credits` |

**その他の統一**:

| RPC | 旧 | 新 |
|-----|-----|-----|
| `get_user_role` | `TEXT` を返す | `JSONB { role: 'admin' \| 'user' }` |

**`_my_` 命名規則の統一**:

Console から呼び出される RPC で `auth.uid()` を使用する関数は `_my_` プレフィックスを持つ。
`p_user_id` パラメータを取る関数は `_my_` を持たない。

| 旧関数名 | 新関数名 | 変更理由 |
|----------|----------|----------|
| `get_user_role` | `get_my_role` | `auth.uid()` 使用、`p_user_id` 不要 |
| `get_tool_settings` | `get_my_tool_settings` | Console 向け、`auth.uid()` 使用 |
| `upsert_tool_settings` | `upsert_my_tool_settings` | Console 向け、`auth.uid()` 使用 |
| `generate_api_key` | `generate_my_api_key` | Console 向け、`auth.uid()` 使用 |
| `list_api_keys` | `list_my_api_keys` | Console 向け、`auth.uid()` 使用 |
| `revoke_api_key` | `revoke_my_api_key` | Console 向け、`auth.uid()` 使用 |
| `delete_credential` | `delete_my_credential` | Console 向け、`auth.uid()` 使用 |
| `list_credentials` | `list_my_credentials` | Console 向け、`auth.uid()` 使用 |

**`_user_` 命名規則（MCP Server 向け）**:

| 旧関数名 | 新関数名 | 変更理由 |
|----------|----------|----------|
| `get_credential` | `get_user_credential` | MCP Server 向け、`p_user_id` 使用 |
| `upsert_credential` | `upsert_user_credential` | MCP Server 向け、`p_user_id` 使用 |

※ `get_tool_settings(p_user_id)` / `upsert_tool_settings(p_user_id, ...)` は MCP Server 向けに残す

### Phase 4: アプリケーションコードの変更

#### 4.1 影響範囲サマリ

| コンポーネント | 影響ファイル数 | 変更内容 |
|----------------|----------------|----------|
| Go Server (MCP) | 1 | RPC 関数名変更 |
| Console (Next.js) | 4 | RPC 関数名変更 + 型定義更新 |
| マイグレーション | 36 → 9 | 統合 + 新テーブル |

#### 4.2 Go Server の変更

**変更ファイル**: `apps/server/internal/store/token.go`

| メソッド | 旧RPC | 新RPC | 行番号 |
|----------|-------|-------|--------|
| `GetModuleToken()` | `get_module_token` | `get_user_credential` | L121 |
| `UpdateModuleToken()` | `update_module_token` | `upsert_user_credential` | L255 |

**レスポンス形式は維持**: Go 側の構造体 (`TokenResult`, `Credentials`) は変更不要。

#### 4.3 Console (Next.js) の変更

**変更ファイル一覧**:

| ファイル | 旧RPC | 新RPC |
|----------|-------|-------|
| `apps/console/src/lib/token-vault.ts` | `list_service_connections` | `list_my_credentials` |
| `apps/console/src/lib/token-vault.ts` | `upsert_service_token` | `upsert_my_credential` |
| `apps/console/src/lib/token-vault.ts` | `delete_service_token` | `delete_my_credential` |
| `apps/console/src/lib/credits.ts` | `list_service_connections` | `list_my_credentials` |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | `upsert_service_token` | `upsert_my_credential` |
| `apps/console/src/app/api/oauth/microsoft/callback/route.ts` | `upsert_service_token` | `upsert_my_credential` |
| `apps/console/src/lib/supabase/database.types.ts` | 型定義更新 | 自動生成で対応 |

**引数名の変更**: `p_service` → `p_module`

#### 4.4 ドキュメント更新（スコープ外）

以下のファイルは旧RPC名を参照しているが、実装完了後に更新：
- `docs/002_specification/details/itf-tvl.md`
- `docs/002_specification/interaction/dtl-itr-CON-TVL.md`
- `docs/003_design/interface/dsn-rpc.md`
- `docs/003_design/data/dtl-dsn-rpc.md`

---

## 実装手順

### Step 1: ローカルで新マイグレーション作成

```bash
# 作業ディレクトリ
cd supabase/migrations

# 既存ファイルをバックアップ
mkdir -p ../migrations_backup
mv *.sql ../migrations_backup/

# 新しい統合マイグレーションを作成（9ファイル）
```

### Step 2: pgsodium TCE キーの作成

```sql
-- キー作成（一度だけ）
SELECT id FROM pgsodium.create_key(
    name := 'user_credentials_key',
    key_type := 'aead-det'
);
-- 結果を控えておく: <key_id>
```

### Step 3: リモートDBリセット

```bash
# 本番DBを完全リセット（ユーザーなし前提）
npx supabase db reset --linked

# または
npx supabase db push --linked
```

### Step 4: OAuth App 設定の再登録

```sql
-- Google OAuth App
SELECT public.upsert_oauth_app(
    'google',
    '<GOOGLE_CLIENT_ID>',
    '<GOOGLE_CLIENT_SECRET>',
    'https://dev.mcpist.app/oauth/callback',
    true
);

-- Microsoft OAuth App
SELECT public.upsert_oauth_app(
    'microsoft',
    '<MICROSOFT_CLIENT_ID>',
    '<MICROSOFT_CLIENT_SECRET>',
    'https://dev.mcpist.app/oauth/callback',
    true
);
```

### Step 5: E2E テスト

1. 新規ユーザー登録
2. サービス接続（Google Calendar）
3. ツール実行
4. トークンリフレッシュ確認

---

## 新マイグレーションファイル構成

```
supabase/migrations/
├── 00000000000001_schema_and_enums.sql
├── 00000000000002_tables.sql
├── 00000000000003_user_credentials.sql       ← 新規
├── 00000000000004_rls_policies.sql
├── 00000000000005_user_triggers.sql
├── 00000000000006_rpc_mcp_server.sql
├── 00000000000007_rpc_console.sql
├── 00000000000008_oauth_apps.sql
└── 00000000000009_stripe_integration.sql
```

---

## 削除されるテーブル/カラム

| 削除対象 | 理由 |
|----------|------|
| `mcpist.service_tokens` | `mcpist.user_credentials` に置き換え |
| `service_tokens.credentials_secret_id` | vault.secrets への参照不要 |

---

## リスク評価

| リスク | 対策 |
|--------|------|
| pgsodium TCE の設定ミス | ローカルで十分テスト後にリモート適用 |
| RPC 関数のシグネチャ変更 | シグネチャは維持、内部実装のみ変更 |
| Go Server との互換性 | RPC レスポンス形式は維持 |

---

## 作業見積もり

| タスク | 見積もり |
|--------|----------|
| マイグレーション統合・作成 | 2h |
| pgsodium TCE 設定 | 0.5h |
| RPC 関数更新 | 1h |
| Go Server 変更 (`token.go`) | 0.5h |
| Console 変更 (4ファイル) | 1h |
| 型定義再生成 (`database.types.ts`) | 0.5h |
| ローカルテスト | 0.5h |
| リモートリセット・適用 | 0.5h |
| E2E テスト | 0.5h |
| **合計** | **7h** |

---

## 完了条件

### DB / マイグレーション
- [ ] マイグレーションが9ファイルに統合
- [ ] `mcpist.user_credentials` テーブルが pgsodium TCE で暗号化
- [ ] `mcpist.service_tokens` テーブルが削除
- [ ] vault.secrets にはユーザートークンが存在しない
- [ ] vault.secrets には運営シークレット（oauth_app_*, stripe_* など）のみ存在

### Go Server
- [ ] `token.go` が新RPC名 (`get_user_credential`, `upsert_user_credential`) を使用

### Console
- [ ] `token-vault.ts` が新RPC名を使用
- [ ] `credits.ts` が新RPC名を使用
- [ ] OAuth callback routes が新RPC名を使用
- [ ] `database.types.ts` が再生成済み

### E2E テスト
- [ ] 新規ユーザー登録
- [ ] サービス接続（Google Calendar / Microsoft To Do）
- [ ] ツール実行
- [ ] トークンリフレッシュ確認
