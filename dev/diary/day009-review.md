# DAY9 レビュー

## 概要

DAY9では、Sprint-001（M1: 基盤構築）を実施した。新規リポジトリ`mcpist`を作成し、モノレポ構成・CI/CD・Supabaseマイグレーションを完了した。

---

## 成果物

### 新規リポジトリ: mcpist

| 項目 | 内容 |
|------|------|
| リポジトリ | `mcpist`（Private） |
| 構成 | Turborepo + pnpm モノレポ |
| 場所 | `C:\Users\shiba\HOBBY\mcpist` |

### ディレクトリ構成

```
mcpist/
├── apps/
│   ├── server/         # MCP Server (Go 1.23)
│   ├── console/        # User Console (Next.js 15)
│   └── worker/         # API Gateway (Cloudflare Worker)
├── packages/           # 共有パッケージ（将来用）
├── supabase/           # マイグレーション
├── docs/specification/ # 仕様書（DAY8から移行）
├── .github/workflows/  # CI/CD
├── package.json
├── pnpm-workspace.yaml
├── turbo.json
└── docker-compose.yml
```

### マイグレーション

| ファイル | 内容 | 状態 |
|---------|------|------|
| 00000000000000_setup.sql | スキーマ初期化 | 適用済み |
| 00000000000001_entitlement_store.sql | ENTテーブル（12テーブル） | 適用済み |
| 00000000000002_token_vault.sql | TVLテーブル | 適用済み |
| 00000000000003_rls_policies.sql | RLSポリシー | 適用済み |

### CI/CD

| ワークフロー | 内容 |
|-------------|------|
| ci.yml | Lint/Test/Build（Go, Next.js, Worker） |

---

## 主要な決定事項

### リポジトリ分離

| 項目 | 決定 |
|------|------|
| 本番リポジトリ | `mcpist`（新規作成） |
| プロトタイプリポジトリ | `go-mcp-dev`（既存、保持） |

**理由:**
- プロトタイプと本番コードを分離
- クリーンな状態から開始
- 仕様書は`mcpist/docs/specification/`に移行

### 技術選定

| 項目 | 選定 |
|------|------|
| モノレポツール | Turborepo + pnpm |
| UUID生成 | `gen_random_uuid()`（PostgreSQL組み込み） |

**学び:**
- Supabaseは`uuid-ossp`拡張を`extensions`スキーマに配置するため、`uuid_generate_v4()`は直接呼び出せない
- `gen_random_uuid()`はPostgreSQL 13+の組み込み関数で、拡張不要

---

## 解決した問題

### マイグレーションエラー

| 問題 | 原因 | 解決 |
|------|------|------|
| `init`マイグレーションがスキップ | Supabaseが`init`という名前を特別扱い | `setup`にリネーム |
| `uuid_generate_v4() does not exist` | uuid-ossp拡張がextensionsスキーマ | `gen_random_uuid()`に変更 |

---

## 残課題

### Sprint-001完了後の状態

| 項目 | 状態 |
|------|------|
| `pnpm install` | 完了 |
| ローカルSupabase | 起動済み（`supabase start`） |
| ローカル開発環境 | 動作確認済み |
| メール/パスワード認証 | 実装済み |
| auth.users → mcpist.users トリガー | 適用済み |
| adminユーザー設定 | shiba.dog.leo.private@gmail.com に設定済み |

---

## ユーザー管理方針

### ロール管理

| 項目 | 方針 |
|------|------|
| デフォルトロール | 新規ユーザー登録時は `role = 'user'` |
| admin昇格 | 手動SQL（マイグレーション）でのみ可能 |
| ロールの用途 | 管理画面へのアクセス権限のみ（サービス利用は課金プランで制御） |

**設計思想:**
- adminでもMCPサービス利用は通常通り課金される
- 「adminだから無制限」という法人資源の私的流用を防止
- 開発・検証用には別途 `dev-unlimited` プランを使用

### auth.users と mcpist.users の同期

**トリガー設定:**
```sql
-- auth.users 作成時に mcpist.users も自動作成
CREATE OR REPLACE FUNCTION mcpist.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO mcpist.users (id, display_name, status, role)
  VALUES (
    NEW.id,
    COALESCE(NEW.raw_user_meta_data->>'name', NEW.email),
    'active',
    'user'  -- デフォルトはuser
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION mcpist.handle_new_user();
```

### admin昇格手順

```sql
-- 特定ユーザーをadminに昇格（手動実行）
UPDATE mcpist.users
SET role = 'admin'
WHERE display_name = 'target-user@example.com';
```

### ローカル開発環境

| 項目 | 設定 |
|------|------|
| Supabase URL | `http://127.0.0.1:54321` |
| Mailpit | `http://127.0.0.1:54324` |
| メール確認 | `enable_confirmations = false`（スキップ） |
| Studio | `http://127.0.0.1:54323` |

**環境切り替え:**
- ローカル: `apps/console/.env.local` でローカルSupabaseを指定
- 本番: `.env.production` または Vercel環境変数で本番Supabaseを指定

---

## 次のスプリント候補

### M2: MCP Server

| タスク | 成果物 |
|--------|--------|
| MCP Protocol Handler | JSON-RPC over SSE実装 |
| Module Registry | `get_module_schema`, `call`, `batch` |
| Auth Middleware | JWT検証、X-User-ID抽出 |

### M6: User Console

| タスク | 成果物 |
|--------|--------|
| 認証画面 | ログイン/ログアウト |
| ダッシュボード | 使用量表示 |

---

## 振り返り

### うまくいったこと

- 新規リポジトリでクリーンスタートできた
- Turborepo + pnpmでモノレポ構成が簡潔
- マイグレーションがリモートSupabaseに適用成功

### 改善点

- マイグレーションファイル名に`init`を使わない（Supabaseの特別扱い）
- Supabase固有の制約（uuid-ossp拡張の配置）を事前に把握

### 学び

- Supabase CLIは`init`という名前のマイグレーションをスキップする
- PostgreSQL 13+では`gen_random_uuid()`が組み込みで使用可能
- リモートプッシュ前にローカルでテストすべき

---

## 重要なバグ修正: Supabase Auth デッドロック問題

### 問題

`@supabase/supabase-js` v2.90.1 で `client.auth.getSession()` が無限にハングする問題が発生。
ダッシュボードが "Loading..." のまま表示されない。

**原因:** Web Locks APIのデッドロック問題
- 参照: https://github.com/supabase/supabase-js/issues/1594

### 解決策

**1. noOpLock ワークアラウンド（client.ts）**

```typescript
// Workaround for Web Locks API deadlock issue
// https://github.com/supabase/supabase-js/issues/1594
const noOpLock = async <T>(
  _name: string,
  _acquireTimeout: number,
  fn: () => Promise<T>
): Promise<T> => {
  return await fn()
}

client = createSupabaseClient(supabaseUrl, supabaseKey, {
  auth: {
    lock: noOpLock,
    // ...
  },
})
```

**2. Auth初期化の順序が重要（auth-context.tsx）**

```typescript
// IMPORTANT: Auth initialization order matters to avoid deadlocks
// See: https://github.com/supabase/supabase-js/issues/1594
//
// 1. Set up onAuthStateChange BEFORE calling getSession()
//    - This ensures the listener is ready when session events fire
//
// 2. Use .then() instead of await for getSession()
//    - Prevents blocking the event loop during initialization
//
// 3. Wrap async operations (like buildUser) in setTimeout(..., 0)
//    - Defers execution to next tick, avoiding deadlocks with Supabase internals
//    - The Web Locks API can cause hangs if async calls happen synchronously
//      within onAuthStateChange callback

const { data: { subscription } } = client.auth.onAuthStateChange(async (_event, session) => {
  // ...
  if (sbUser) {
    // setTimeout prevents deadlock - do not remove!
    setTimeout(async () => {
      // async operations here
    }, 0)
  }
})

// Use .then() not await - prevents blocking during init
client.auth.getSession().then(({ data: { session } }) => {
  // ...
})
```

### 注意事項

- `@supabase/ssr` の `createBrowserClient` でも同様の問題が発生
- 本番環境でも同じワークアラウンドが必要
- 将来のSupabaseアップデートで修正される可能性あり

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [sprint-001.md](./sprint-001.md) | Sprint-001計画書 |
| [spc-dev.md](../DAY8/spc-dev.md) | 開発計画書 |
| [spc-tbl.md](../DAY8/spc-tbl.md) | テーブル仕様書 |

---

## DAY9 (2026-01-19) レビュー

### 本日の成果

| 項目 | 内容 |
|------|------|
| コミット | `2c0031e feat(auth): add API Key authentication with audit trail` |
| 主な機能 | API Key認証、トークン履歴、UI改善 |

### 実装したマイグレーション

| ファイル | 内容 |
|---------|------|
| `20260118140000_validate_api_key_rpc.sql` | API Key検証RPC |
| `20260118150000_get_masked_api_key.sql` | マスクされたAPI Key取得RPC |
| `20260118160000_fix_upsert_token.sql` | Vault重複問題修正 |
| `20260118170000_token_history.sql` | トークン履歴テーブル・RPC |

---

### 反省・気づき

#### 1. Supabase Vault のシークレット名はユニーク制約がある

**問題:** API Key再生成時に500エラー
- `vault.create_secret()` は同じ名前のシークレットを作成できない
- 古いシークレットを削除しても、名前が同じだと衝突

**解決:** シークレット名にタイムスタンプを追加
```sql
'oauth_access_' || p_service || '_' || v_user_id || '_' || v_timestamp
```

**学び:** Vault操作は冪等性を考慮した設計が必要

#### 2. RPC関数のテーブル名・カラム名の確認不足

**問題:** `get_masked_api_key` が `mpt_**...**` しか返さない
- 誤: `oauth_connections.vault_secret_id`
- 正: `oauth_tokens.access_token_secret_id`

**学び:** RPCを書く前にテーブル構造を再確認する

#### 3. UIコンポーネントの適切な選択

**問題:** ボタンの色が意図と異なる
- `CardHeader` のgridレイアウトと `CardAction` の位置関係
- `variant="outline"` だけでは背景色が継承される

**解決:** `CardAction` コンポーネント + 明示的な背景色指定
```tsx
className="bg-white dark:bg-zinc-800"
```

**学び:** shadcn/uiのコンポーネントは内部構造を理解して使う

#### 4. セキュリティ設計: トークン履歴の重要性

**気づき:**
- API Keyローテーション時、古いキーの履歴を残すべき
- 「誰がいつからいつまでどのキーを使っていたか」の監査ログ
- `expired_reason` で `rotated` / `revoked` を区別

**実装:**
```sql
INSERT INTO oauth_token_history (
  user_id, service, access_token_secret_id,
  created_at, expired_at, expired_reason
) VALUES (..., NOW(), 'rotated');
```

#### 5. UIの情報設計

**学び:**
- 「接続方法」セクションはAPI Keyを使う人向け
- OAuth 2.0は自動なのでシンプルな設定で十分
- 上級者向け機能は折りたたみで隠す

---

### 今日うまくいったこと

- API Key認証のE2E実装（生成→保存→検証→無効化）
- トークン履歴による監査ログ設計
- UIの段階的な改善（ユーザーフィードバックを即反映）

### 改善が必要なこと

- マイグレーション作成時のテーブル構造確認
- コンポーネントの内部構造理解
- エラーログの詳細化（500エラーの原因特定に時間がかかった）
