# テーブル詳細設計書（dtl-dsn-tbl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Table Detail Design |

---

## 概要

本ドキュメントは、MCPistのデータベーステーブルの詳細設計を定義する。

- テーブル仕様: [spc-tbl.md](../../002_specification/spc-tbl.md)
- テーブル設計: [dsn-tbl.md](./dsn-tbl.md)

---

## Enum定義

### mcpist.account_status

```sql
CREATE TYPE mcpist.account_status AS ENUM (
    'active',      -- アクティブ
    'suspended',   -- 一時停止
    'disabled'     -- 無効化
);
```

### mcpist.module_status

```sql
CREATE TYPE mcpist.module_status AS ENUM (
    'active',       -- 利用可能
    'coming_soon',  -- 近日公開
    'maintenance',  -- メンテナンス中
    'beta',         -- ベータ版
    'deprecated',   -- 非推奨
    'disabled'      -- 無効
);
```

### mcpist.credit_transaction_type

```sql
CREATE TYPE mcpist.credit_transaction_type AS ENUM (
    'consume',       -- クレジット消費
    'purchase',      -- クレジット購入
    'monthly_reset'  -- 月次リセット
);
```

---

## テーブル定義

### mcpist.users

ユーザー情報を管理する。auth.usersと1:1で紐づく。

```sql
CREATE TABLE mcpist.users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    account_status mcpist.account_status NOT NULL DEFAULT 'active',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_users_account_status ON mcpist.users(account_status);

-- 更新トリガー
CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON mcpist.users
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | - | PK、auth.usersへのFK |
| account_status | account_status | NO | 'active' | アカウント状態 |
| preferences | JSONB | YES | '{}' | UI設定（theme, background, accent） |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

**preferences構造:**

```json
{
  "theme": "light" | "dark" | "system",
  "background": "default" | "gradient" | "image",
  "accent": "#3B82F6"
}
```

---

### mcpist.credits

ユーザーのクレジット残高を管理する。

```sql
CREATE TABLE mcpist.credits (
    user_id UUID PRIMARY KEY REFERENCES mcpist.users(id) ON DELETE CASCADE,
    free_credits INTEGER NOT NULL DEFAULT 1000 CHECK (free_credits >= 0 AND free_credits <= 1000),
    paid_credits INTEGER NOT NULL DEFAULT 0 CHECK (paid_credits >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 更新トリガー
CREATE TRIGGER set_credits_updated_at
    BEFORE UPDATE ON mcpist.credits
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();
```

| 列 | 型 | NULL | デフォルト | 制約 | 説明 |
|----|-----|------|-----------|------|------|
| user_id | UUID | NO | - | PK, FK | usersへの参照 |
| free_credits | INTEGER | NO | 1000 | 0-1000 | 無料クレジット |
| paid_credits | INTEGER | NO | 0 | >= 0 | 有料クレジット |
| updated_at | TIMESTAMPTZ | NO | NOW() | - | 更新日時 |

---

### mcpist.credit_transactions

クレジットの増減履歴を記録する。

```sql
CREATE TABLE mcpist.credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    type mcpist.credit_transaction_type NOT NULL,
    amount INTEGER NOT NULL,
    credit_type TEXT CHECK (credit_type IN ('free', 'paid')),
    module TEXT,
    tool TEXT,
    request_id TEXT,
    task_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_credit_transactions_user_id ON mcpist.credit_transactions(user_id);
CREATE INDEX idx_credit_transactions_created_at ON mcpist.credit_transactions(created_at DESC);
CREATE INDEX idx_credit_transactions_type ON mcpist.credit_transactions(type);
CREATE INDEX idx_credit_transactions_request_id ON mcpist.credit_transactions(request_id) WHERE request_id IS NOT NULL;

-- 冪等性のためのUNIQUE制約（consume時のみ使用）
CREATE UNIQUE INDEX idx_credit_transactions_idempotency
    ON mcpist.credit_transactions(user_id, request_id, COALESCE(task_id, ''))
    WHERE request_id IS NOT NULL;
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | gen_random_uuid() | PK |
| user_id | UUID | NO | - | usersへのFK |
| type | credit_transaction_type | NO | - | トランザクション種別 |
| amount | INTEGER | NO | - | 増減量（消費は負、加算は正） |
| credit_type | TEXT | YES | - | 'free' or 'paid'（消費時に記録） |
| module | TEXT | YES | - | モジュール名（消費時） |
| tool | TEXT | YES | - | ツール名（消費時） |
| request_id | TEXT | YES | - | リクエストID（追跡用） |
| task_id | TEXT | YES | - | タスクID（batch時） |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |

---

### mcpist.modules

モジュール定義（マスタデータ）。

```sql
CREATE TABLE mcpist.modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    status mcpist.module_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_modules_status ON mcpist.modules(status);
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | gen_random_uuid() | PK |
| name | TEXT | NO | - | モジュール名（UK） |
| status | module_status | NO | 'active' | モジュール状態 |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |

---

### mcpist.module_settings

ユーザーごとのモジュール有効/無効設定。

```sql
CREATE TABLE mcpist.module_settings (
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id)
);

-- インデックス
CREATE INDEX idx_module_settings_user_id ON mcpist.module_settings(user_id);
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| user_id | UUID | NO | - | PK（複合）、usersへのFK |
| module_id | UUID | NO | - | PK（複合）、modulesへのFK |
| enabled | BOOLEAN | NO | true | 有効/無効 |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |

---

### mcpist.tool_settings

ユーザーごとのツール有効/無効設定。

```sql
CREATE TABLE mcpist.tool_settings (
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE RESTRICT,
    tool_name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id, tool_name)
);

-- インデックス
CREATE INDEX idx_tool_settings_user_module ON mcpist.tool_settings(user_id, module_id);
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| user_id | UUID | NO | - | PK（複合）、usersへのFK |
| module_id | UUID | NO | - | PK（複合）、modulesへのFK |
| tool_name | TEXT | NO | - | PK（複合）、ツール名 |
| enabled | BOOLEAN | NO | true | 有効/無効 |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |

---

### mcpist.prompts

ユーザー定義プロンプト。

```sql
CREATE TABLE mcpist.prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    module_id UUID REFERENCES mcpist.modules(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, module_id, name)
);

-- インデックス
CREATE INDEX idx_prompts_user_id ON mcpist.prompts(user_id);
CREATE INDEX idx_prompts_user_module ON mcpist.prompts(user_id, module_id);

-- 更新トリガー
CREATE TRIGGER set_prompts_updated_at
    BEFORE UPDATE ON mcpist.prompts
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | gen_random_uuid() | PK |
| user_id | UUID | NO | - | usersへのFK |
| module_id | UUID | YES | - | modulesへのFK（NULLは全モジュール共通） |
| name | TEXT | NO | - | プロンプト名 |
| content | TEXT | NO | - | プロンプト内容 |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

---

### mcpist.api_keys

MCP接続用APIキー。

```sql
CREATE TABLE mcpist.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    name TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_api_keys_user_id ON mcpist.api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON mcpist.api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_expires_at ON mcpist.api_keys(expires_at) WHERE expires_at IS NOT NULL;
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | gen_random_uuid() | PK |
| user_id | UUID | NO | - | usersへのFK |
| key_hash | TEXT | NO | - | SHA-256ハッシュ（UK） |
| key_prefix | TEXT | NO | - | プレフィックス（表示用） |
| name | TEXT | NO | - | キー名（ユーザー管理用） |
| expires_at | TIMESTAMPTZ | YES | - | 有効期限（NULLは無期限） |
| last_used_at | TIMESTAMPTZ | YES | - | 最終使用日時 |
| revoked_at | TIMESTAMPTZ | YES | - | 削除日時（論理削除） |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |

---

### mcpist.service_tokens

外部サービスのトークン管理。Vaultのsecret IDを参照する。

```sql
CREATE TABLE mcpist.service_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    credentials_secret_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, service)
);

-- インデックス
CREATE INDEX idx_service_tokens_user_id ON mcpist.service_tokens(user_id);

-- 更新トリガー
CREATE TRIGGER set_service_tokens_updated_at
    BEFORE UPDATE ON mcpist.service_tokens
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.trigger_set_updated_at();
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | gen_random_uuid() | PK |
| user_id | UUID | NO | - | usersへのFK |
| service | TEXT | NO | - | サービス名（notion, github等） |
| credentials_secret_id | UUID | NO | - | vault.secretsへの参照 |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

**Vault JSON形式:**

```json
{
  "access_token": "xxx",
  "refresh_token": "yyy",
  "client_id": "...",
  "client_secret": "...",
  "auth_type": "oauth2",
  "expires_at": "2024-01-01T00:00:00+00:00"
}
```

| auth_type | 説明 |
|------------|------|
| oauth2 | OAuth 2.0トークン（access_token, refresh_token等） |
| api_key | 長期トークン/APIキー（token） |

---

### mcpist.oauth_apps

OAuthアプリ設定（プロバイダ別のOAuthクライアント情報）。Vaultに暗号化保存する。

```sql
CREATE TABLE mcpist.oauth_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL UNIQUE,
    secret_id UUID REFERENCES vault.secrets(id) ON DELETE SET NULL,
    redirect_uri TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| id | UUID | NO | gen_random_uuid() | PK |
| provider | TEXT | NO | - | プロバイダ名（google, microsoft等） |
| secret_id | UUID | YES | - | vault.secretsへの参照（client_id/secret） |
| redirect_uri | TEXT | NO | - | OAuthコールバックURL |
| enabled | BOOLEAN | YES | true | 有効フラグ |
| created_at | TIMESTAMPTZ | YES | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | YES | NOW() | 更新日時 |

---

### mcpist.processed_webhook_events

PSP Webhook冪等性チェック用。

```sql
CREATE TABLE mcpist.processed_webhook_events (
    event_id TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES mcpist.users(id) ON DELETE CASCADE,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_processed_webhook_events_user_id ON mcpist.processed_webhook_events(user_id);
CREATE INDEX idx_processed_webhook_events_processed_at ON mcpist.processed_webhook_events(processed_at DESC);
```

| 列 | 型 | NULL | デフォルト | 説明 |
|----|-----|------|-----------|------|
| event_id | TEXT | NO | - | PK（PSPのevent.id） |
| user_id | UUID | NO | - | usersへのFK |
| processed_at | TIMESTAMPTZ | NO | NOW() | 処理日時 |

---

## 共通関数

### trigger_set_updated_at

```sql
CREATE OR REPLACE FUNCTION mcpist.trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

---

## Row Level Security (RLS)

### コンポーネント別Supabaseキー

| コンポーネント | Supabaseキー | 用途 |
|---------------|-------------|------|
| User Console (Frontend) | anon key | ユーザー操作（RLS適用） |
| User Console (API Routes) | service_role key | Webhook処理、管理操作 |
| Cloudflare Worker | - | API Gateway（認証・ルーティングのみ） |
| MCP Server (Go) | service_role key | ツール実行、クレジット消費 |

**備考:**
- `anon key`: RLSポリシーに従ったアクセスのみ可能
- `service_role key`: RLSをバイパス、サーバーサイドでのみ使用

### ポリシー方針

| テーブル                     | SELECT | INSERT       | UPDATE       | DELETE |
| ------------------------ | ------ | ------------ | ------------ | ------ |
| users                    | 自分のみ   | -            | 自分のみ         | -      |
| credits                  | 自分のみ   | -            | service_role | -      |
| credit_transactions      | 自分のみ   | service_role | -            | -      |
| modules                  | 全員     | -            | -            | -      |
| module_settings          | 自分のみ   | 自分のみ         | 自分のみ         | 自分のみ   |
| tool_settings            | 自分のみ   | 自分のみ         | 自分のみ         | 自分のみ   |
| prompts                  | 自分のみ   | 自分のみ         | 自分のみ         | 自分のみ   |
| api_keys                 | 自分のみ   | 自分のみ         | 自分のみ         | 自分のみ   |
| service_tokens           | 自分のみ   | 自分のみ         | 自分のみ         | 自分のみ   |
| oauth_apps               | service_role | service_role | service_role | service_role |
| processed_webhook_events | -      | service_role | -            | -      |

**service_role操作の実行元:**
- `credits` UPDATE: MCP Server（クレジット消費）、User Console API Routes（購入処理）
- `credit_transactions` INSERT: MCP Server（消費記録）、User Console API Routes（購入記録）
- `processed_webhook_events` INSERT: User Console API Routes（Webhook冪等性）
- `service_tokens` SELECT/UPDATE: MCP Server（トークン取得・更新）※RPCを経由
- `oauth_apps` ALL: User Console API Routes（管理操作）※service_roleのみ

### RLS有効化

```sql
-- 全テーブルでRLS有効化
ALTER TABLE mcpist.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.credits ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.credit_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.module_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.tool_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.service_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.oauth_apps ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcpist.processed_webhook_events ENABLE ROW LEVEL SECURITY;
```

### ポリシー定義

```sql
-- users
CREATE POLICY users_select ON mcpist.users FOR SELECT USING (auth.uid() = id);
CREATE POLICY users_update ON mcpist.users FOR UPDATE USING (auth.uid() = id);

-- credits
CREATE POLICY credits_select ON mcpist.credits FOR SELECT USING (auth.uid() = user_id);

-- credit_transactions
CREATE POLICY credit_transactions_select ON mcpist.credit_transactions FOR SELECT USING (auth.uid() = user_id);

-- modules（全員読み取り可）
CREATE POLICY modules_select ON mcpist.modules FOR SELECT USING (true);

-- module_settings
CREATE POLICY module_settings_select ON mcpist.module_settings FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY module_settings_insert ON mcpist.module_settings FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY module_settings_update ON mcpist.module_settings FOR UPDATE USING (auth.uid() = user_id);
CREATE POLICY module_settings_delete ON mcpist.module_settings FOR DELETE USING (auth.uid() = user_id);

-- tool_settings
CREATE POLICY tool_settings_select ON mcpist.tool_settings FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY tool_settings_insert ON mcpist.tool_settings FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY tool_settings_update ON mcpist.tool_settings FOR UPDATE USING (auth.uid() = user_id);
CREATE POLICY tool_settings_delete ON mcpist.tool_settings FOR DELETE USING (auth.uid() = user_id);

-- prompts
CREATE POLICY prompts_select ON mcpist.prompts FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY prompts_insert ON mcpist.prompts FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY prompts_update ON mcpist.prompts FOR UPDATE USING (auth.uid() = user_id);
CREATE POLICY prompts_delete ON mcpist.prompts FOR DELETE USING (auth.uid() = user_id);

-- api_keys
CREATE POLICY api_keys_select ON mcpist.api_keys FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY api_keys_insert ON mcpist.api_keys FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY api_keys_update ON mcpist.api_keys FOR UPDATE USING (auth.uid() = user_id);
CREATE POLICY api_keys_delete ON mcpist.api_keys FOR DELETE USING (auth.uid() = user_id);

-- service_tokens
CREATE POLICY service_tokens_select ON mcpist.service_tokens FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY service_tokens_insert ON mcpist.service_tokens FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY service_tokens_update ON mcpist.service_tokens FOR UPDATE USING (auth.uid() = user_id);
CREATE POLICY service_tokens_delete ON mcpist.service_tokens FOR DELETE USING (auth.uid() = user_id);

-- oauth_apps（service_roleのみ）
CREATE POLICY oauth_apps_all ON mcpist.oauth_apps
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- processed_webhook_events（service_roleのみINSERT可）
CREATE POLICY processed_webhook_events_insert ON mcpist.processed_webhook_events
    FOR INSERT WITH CHECK (auth.jwt() ->> 'role' = 'service_role');
```

---

## 初期データ

### modules

```sql
INSERT INTO mcpist.modules (name, status) VALUES
    ('notion', 'active'),
    ('github', 'active'),
    ('jira', 'active'),
    ('confluence', 'active'),
    ('supabase', 'beta'),
    ('google_calendar', 'active'),
    ('microsoft_todo', 'active'),
    ('rag', 'active');
```

---

## ユーザー作成時の初期化

auth.usersにユーザーが作成されたときのトリガー処理。

```sql
CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
    -- usersテーブルに挿入
    INSERT INTO mcpist.users (id)
    VALUES (NEW.id);

    -- creditsテーブルに挿入
    INSERT INTO mcpist.credits (user_id)
    VALUES (NEW.id);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- auth.usersへのトリガー
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION mcpist.handle_new_user();
```

---

## 将来実装: credit_transactionsパーティショニング

credit_transactionsは全ユーザーのツール実行履歴を記録するため、レコード数が膨大になる。
監査・請求根拠として7年間の保持が必要（SOX準拠）。

### 現状（MVP）

通常テーブルとして実装。規模拡大時に以下を実装する。

### 将来設計

```
┌─────────────────────────────────────────────────────┐
│ credit_transactions (パーティション親テーブル)        │
│                                                     │
│  [0-12ヶ月] PostgreSQL 月次パーティション            │
│     └─ 高速クエリ、RLS適用                          │
│                                                     │
│  [12ヶ月-7年] アーカイブストレージ                   │
│     └─ S3/GCS (Parquet形式)                        │
│     └─ 必要時のみ復元                               │
│                                                     │
│  [7年経過] 削除                                     │
└─────────────────────────────────────────────────────┘
```

### 実装時の変更点

1. **月次パーティション化**
   ```sql
   CREATE TABLE mcpist.credit_transactions (
       id UUID DEFAULT gen_random_uuid(),
       -- ...
       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
       PRIMARY KEY (id, created_at)
   ) PARTITION BY RANGE (created_at);
   ```

2. **パーティション自動作成（Cron）**
   - 毎月1日に翌月パーティションを作成

3. **アーカイブ処理（Cron）**
   - 12ヶ月経過したパーティションをS3/GCSへエクスポート
   - PostgreSQLからDETACH/DROP

4. **削除処理（Cron）**
   - 7年経過したアーカイブを削除

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-tbl.md](../../002_specification/spc-tbl.md) | テーブル仕様書 |
| [dsn-tbl.md](./dsn-tbl.md) | テーブル設計書 |
| [dtl-spc-credit-model.md](../../002_specification/dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
