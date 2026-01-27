# DAY015 作業ログ

## 日付

2026-01-26

---

## 作業内容

### RPC呼び出しリファクタ調査

Console (Next.js) のRPC実装状況を調査した結果、**既にRPC化が完了**していることを確認。

#### Console側のRPC使用状況

| ページ | 状態 | 使用RPC |
|--------|------|---------|
| Dashboard | ✅ RPC利用済 | `get_user_context`, `list_service_connections` |
| Connections | ✅ RPC利用済 | `list_service_connections`, `upsert_service_token`, `delete_service_token` |
| MCP (API Keys) | ✅ RPC利用済 | `list_api_keys`, `generate_api_key`, `revoke_api_key` |
| OAuth Consents | ✅ RPC利用済 | `list_oauth_consents`, `revoke_oauth_consent` |

#### 実装ファイル

| ファイル | 役割 |
|----------|------|
| `apps/console/src/lib/api-keys.ts` | APIキー管理 |
| `apps/console/src/lib/token-vault.ts` | トークン・Vault管理 |
| `apps/console/src/lib/credits.ts` | クレジット情報取得 |
| `apps/console/src/lib/oauth-consents.ts` | OAuthコンセント管理 |
| `apps/console/src/app/(console)/mcp/actions.ts` | Server Action |

#### database.types.ts

全てのConsole向けRPC関数の型定義が存在。追加不要。

定義済みRPC:
- `get_user_context`
- `list_service_connections`
- `upsert_service_token`
- `delete_service_token`
- `get_user_role`
- `list_api_keys`
- `generate_api_key`
- `revoke_api_key`
- `get_service_token`
- `list_oauth_consents`
- `revoke_oauth_consent`
- `list_all_oauth_consents`

---

### Phase 2 残タスク（Worker/Go側） ✅

Console側は完了済みのため、Phase 2の残りはWorker/Go Serverのリファクタ:

| ID     | タスク                          | 対象        | 状態    |
| ------ | ---------------------------- | --------- | ----- |
| S5-030 | `lookup_user_by_key_hash` 使用 | Worker    | ✅ 完了 |
| S5-040 | `get_module_token` RPC使用     | Go Server | ✅ 完了 |
| S5-041 | `consume_credit` RPC使用       | Go Server | ✅ 完了 |
| S5-042 | `get_user_context` RPC呼び出し   | Go Server | ✅ 完了 |

---

## 結論

- **Console (Next.js)**: RPC呼び出しリファクタ完了
- **Worker**: RPC呼び出し完了（lookup_user_by_key_hash）
- **Go Server**: RPC呼び出し完了（get_user_context, consume_credit, get_module_token）
- **database.types.ts**: 型定義完了、追加不要

Sprint-005 Phase 2の進捗を更新:
- 旧: 0/9 (0%)
- 新: 9/9 (100%) - 全タスク完了

---

---

### Worker/Go Server RPC調査

Worker側とGo Server側の調査も完了。**既に全てRPC利用済み**。

#### Worker側 (Cloudflare Worker)

| RPC | ファイル | 状態 |
|-----|----------|------|
| `lookup_user_by_key_hash` | apps/worker/src/index.ts:559-606 | ✅ RPC使用済み |

#### Go Server側

| RPC | ファイル | 状態 |
|-----|----------|------|
| `get_user_context` | apps/server/internal/store/user.go:115-186 | ✅ RPC使用済み |
| `consume_credit` | apps/server/internal/store/user.go:189-245 | ✅ RPC使用済み |
| `get_module_token` | apps/server/internal/store/token.go:96-156 | ✅ RPC使用済み |

---

### 未実装RPC: update_module_token

設計書に定義されていた `update_module_token` RPCが未実装だったため実装した。

#### 用途
OAuth2トークンリフレッシュ後に、新しいトークンをVaultに保存する。

#### 実装ファイル
- `supabase/migrations/00000000000013_rpc_update_module_token.sql` - RPC関数定義
- `apps/server/internal/store/token.go` - Go側 `UpdateModuleToken()` 追加

---

## 結論

- **Console (Next.js)**: RPC呼び出しリファクタ完了
- **Worker**: RPC呼び出し完了（lookup_user_by_key_hash）
- **Go Server**: RPC呼び出し完了（get_user_context, consume_credit, get_module_token）
- **新規実装**: `update_module_token` RPC

Phase 2: RPC呼び出しリファクタ = **完了 (9/9 = 100%)**
Phase 1: RPC関数実装 = **完了 (17/17 = 100%)**

---

---

## Google Calendar OAuth トークンリフレッシュ実装

### 実装内容

Google Calendar モジュールで、API呼び出し時にOAuth2トークンの有効期限をチェックし、期限切れ間近であれば自動的にリフレッシュする機能を実装。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/store/token.go` | `GetOAuthAppCredentials()` - OAuth App認証情報取得、`UpdateModuleToken()` - リフレッシュ後のトークン保存 |
| `apps/server/internal/modules/google_calendar/module.go` | `getCredentials()` にトークンリフレッシュロジック追加、`needsRefresh()`, `refreshToken()` 関数追加 |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | `expires_at` をUnix timestamp（秒）で保存するように修正 |

#### トークンリフレッシュフロー

1. `getCredentials()` でVaultからトークン取得
2. `needsRefresh()` で有効期限チェック（デフォルト: 期限切れ5分前）
3. 期限切れ間近なら `refreshToken()` を呼び出し
4. `GetOAuthAppCredentials()` でOAuth App（client_id/client_secret）取得（service_role権限）
5. Google OAuth Token Endpointで新しいaccess_token取得
6. `UpdateModuleToken()` で新しいトークンをVaultに保存

#### 権限設計

| RPC | 必要権限 | 理由 |
|-----|----------|------|
| `get_module_token` | anon/authenticated | ユーザー自身のトークン取得 |
| `get_oauth_app_credentials` | service_role | OAuth App Secret は機密情報 |
| `update_module_token` | service_role | Vaultへの書き込み |

---

## 解決済みの問題

### vault.secrets への UPDATE 権限問題

#### 問題

`update_module_token` RPC が `vault.secrets` テーブルを UPDATE できなかった（403エラー）。

#### 原因

`vault.secrets` テーブルの所有者は `supabase_admin` であり、`postgres` ロールから GRANT UPDATE を実行しても権限が付与されない。

#### 解決策

`upsert_service_token` RPC と同じ方法を採用：
- 直接 UPDATE ではなく、DELETE + `vault.create_secret()` で新しいシークレットを作成
- `vault.create_secret()` は適切な権限で動作する

#### 実装

`supabase/migrations/00000000000018_fix_update_module_token_delete_insert.sql`:
```sql
-- 古いシークレットを削除
DELETE FROM vault.secrets WHERE id = v_secret_id;

-- 新しいシークレットを作成 (vault.create_secret を使用)
SELECT vault.create_secret(
    p_credentials::TEXT,
    v_secret_name,
    'Service credentials for ' || p_module
) INTO v_new_secret_id;

-- service_tokensのcredentials_secret_idを更新
UPDATE mcpist.service_tokens
SET credentials_secret_id = v_new_secret_id,
    updated_at = NOW()
WHERE user_id = p_user_id AND service = p_module;
```

#### 結果

✅ `update_module_token` RPC が正常動作することを確認

---

### OAuth App redirect_uri 設定問題

#### 問題

Google OAuth で `redirect_uri_mismatch` エラーが発生。

#### 原因

`mcpist.oauth_apps` テーブルの `redirect_uri` がローカル開発用（`http://localhost:3000/...`）のままだった。

#### 解決策

Console 管理画面（OAuth App Settings）で `redirect_uri` を本番URL（`https://dev.mcpist.app/api/oauth/google/callback`）に更新。

#### 結果

✅ Google Calendar OAuth 接続成功

---

## テスト結果

### トークンリフレッシュ動作確認

```
2026/01/26 07:58:52 [google_calendar] Got credentials: auth_type=oauth2, has_access_token=true
2026/01/26 07:58:52 [google_calendar] Token expired or expiring soon, refreshing...
2026/01/26 07:58:53 [google_calendar] Token refreshed successfully
2026/01/26 07:58:53 [google_calendar.list_calendars] success (1697ms)
```

✅ トークンリフレッシュ → API呼び出し成功の一連の流れを確認

---

## 完了したアクション

1. ✅ vault.secrets への UPDATE 権限問題を解決（DELETE + vault.create_secret 方式）
2. ✅ tokenRefreshBuffer を本番値（5分）に戻す
3. ✅ OAuth App redirect_uri を本番URLに更新
4. ✅ Google Calendar 接続・トークンリフレッシュの動作確認

---

## 最終結論

**Google Calendar OAuth トークンリフレッシュ機能 = 完了**

- トークン有効期限チェック（5分前にリフレッシュ）
- Google OAuth Token Endpoint でトークンリフレッシュ
- `update_module_token` RPC でVaultに新しいトークンを保存
- API呼び出し成功

---

## Sprint-005 全体サマリー

### 概要

外部サービス連携のためのOAuth認証フローを実装。第一弾としてGoogle Calendarを対応。

### データベース設計 (Supabase)

#### oauth_apps テーブル

管理者がOAuthクライアント情報を管理するためのテーブル

```sql
CREATE TABLE oauth_apps (
  id UUID PRIMARY KEY,
  provider TEXT NOT NULL UNIQUE,  -- 'google', 'microsoft'
  client_id TEXT NOT NULL,
  client_secret_id UUID,          -- vault.secrets への参照
  redirect_uri TEXT NOT NULL,
  scopes TEXT[],
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
);
```

#### RPC関数

- `upsert_oauth_app`: OAuth App設定の保存（client_secretはVaultに暗号化保存）
- `get_oauth_app_credentials`: OAuth認証情報の取得（service_role権限必須）
- `delete_oauth_app`: OAuth App設定の削除
- `update_module_token`: トークンリフレッシュ後の保存

### Console (Next.js) 実装

#### 管理者設定ページ

- `apps/console/src/app/(console)/admin/oauth/page.tsx`
- OAuth App（Google/Microsoft）の設定UI
- Client ID, Client Secret, Redirect URIの管理

#### OAuth認可フロー

- `apps/console/src/app/api/oauth/google/authorize/route.ts` - 認可URL生成
- `apps/console/src/app/api/oauth/google/callback/route.ts` - トークン交換・保存

### Go Server 実装

#### google_calendar モジュール

| ツール | 説明 |
|--------|------|
| list_calendars | カレンダー一覧取得 |
| get_calendar | カレンダー詳細取得 |
| list_events | イベント一覧取得 |
| get_event | イベント詳細取得 |
| create_event | イベント作成 |
| update_event | イベント更新 |
| delete_event | イベント削除 |
| quick_add | クイックイベント追加 |

#### トークンリフレッシュ

- `needsRefresh()` - 有効期限チェック（5分前）
- `refreshToken()` - Google OAuth Token Endpoint でリフレッシュ
- `UpdateModuleToken()` - Vault に保存

### アーキテクチャ

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Console   │────▶│ /api/oauth/  │────▶│ Google OAuth    │
│  (Next.js)  │     │   google/    │     │ (accounts.google│
└─────────────┘     └──────────────┘     │   .com)         │
       │                   │              └─────────────────┘
       │                   │
       ▼                   ▼
┌─────────────┐     ┌──────────────┐
│ Connections │     │  Supabase    │
│   Page      │     │   Vault      │
│             │     │ (tokens暗号化)│
└─────────────┘     └──────────────┘
                          │
                          ▼
                   ┌──────────────┐     ┌─────────────────┐
                   │  Go Server   │────▶│ Google Calendar │
                   │ (MCP Handler)│     │     API         │
                   └──────────────┘     └─────────────────┘
```

### 今後の作業

1. ~~**Microsoft OAuth実装**~~ ✅ 完了
2. **エラーハンドリング強化**
   - OAuth認可エラーの詳細表示
   - トークン取り消し対応

---

## Microsoft To Do OAuth実装

### 実装内容

Microsoft To Do モジュールを実装。Google Calendarと同様のパターンでOAuth認証フローとトークンリフレッシュ機能を実装。

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/microsoft/authorize/route.ts` | 認可URL生成（新規作成） |
| `apps/console/src/app/api/oauth/microsoft/callback/route.ts` | トークン交換・保存（修正: service_role権限、expires_at形式） |
| `apps/server/internal/modules/microsoft_todo/module.go` | Microsoft To Doモジュール（新規作成） |
| `apps/server/cmd/server/main.go` | モジュール登録追加 |
| `apps/server/cmd/tools-export/main.go` | tools-export対応 |
| `apps/console/src/lib/services.json` | Microsoft To Do追加 |
| `apps/console/src/lib/tools.json` | Microsoft To Do追加 |

### 実装したツール (11個)

| カテゴリ | ツール | 説明 |
|----------|--------|------|
| Lists | list_lists | タスクリスト一覧取得 |
| Lists | get_list | タスクリスト詳細取得 |
| Lists | create_list | タスクリスト作成 |
| Lists | update_list | タスクリスト更新 |
| Lists | delete_list | タスクリスト削除 (dangerous) |
| Tasks | list_tasks | タスク一覧取得 |
| Tasks | get_task | タスク詳細取得 |
| Tasks | create_task | タスク作成 |
| Tasks | update_task | タスク更新 |
| Tasks | complete_task | タスク完了 |
| Tasks | delete_task | タスク削除 (dangerous) |

### トークンリフレッシュ

Google Calendarと同様のパターン:
- `needsRefresh()` - 有効期限チェック（5分前）
- `refreshToken()` - Microsoft OAuth Token Endpoint でリフレッシュ
- `UpdateModuleToken()` - Vault に保存
- Microsoftは新しいrefresh_tokenを返す場合があるため、それも保存

### Microsoft Entra ID (Azure AD) 設定

1. Azure Portal → Microsoft Entra ID → アプリの登録
2. サポートされているアカウントの種類: 「任意の組織ディレクトリ内のアカウントと個人のMicrosoftアカウント」
3. リダイレクトURI: `https://dev.mcpist.app/api/oauth/microsoft/callback`
4. APIのアクセス許可: `Tasks.ReadWrite`, `offline_access`

### テスト結果

```
mcp__mcpist-dev__run microsoft_todo list_lists
→ 5件のタスクリストを取得成功 (Tasks, Daily Routine, Shopping, Wishlist, Flagged Emails)
```

✅ OAuth認証、トークン保存、API呼び出しすべて正常動作

---

## ツール設定のデータ永続化

### 実装内容

Console のツール設定（各ツールの有効/無効）をSupabase DBに保存する機能を実装。

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `supabase/migrations/00000000000019_rpc_tool_settings.sql` | tool_settings RPC関数（新規作成） |
| `apps/console/src/lib/tool-settings.ts` | ツール設定API（新規作成） |
| `apps/console/src/app/(console)/tools/page.tsx` | DBからの読み込み・保存機能追加 |

### RPC関数

| 関数 | 説明 |
|------|------|
| `get_tool_settings(p_user_id, p_module_name)` | ユーザーのツール設定取得 |
| `upsert_tool_settings(p_user_id, p_module_name, p_enabled_tools, p_disabled_tools)` | ツール設定の一括更新 |
| `get_my_tool_settings(p_module_name)` | 認証ユーザー自身の設定取得 |
| `upsert_my_tool_settings(p_module_name, p_enabled_tools, p_disabled_tools)` | 認証ユーザー自身の設定更新 |

### テスト結果

Airtableモジュールでツール設定を保存・確認:
- aggregate_records: true
- create: true
- create_table: false (dangerous)
- delete: false (dangerous)
- describe: true
- get_record: true
- list_bases: true
- query: true
- search_records: true
- update: true
- update_table: false (dangerous)

✅ DBへの保存・読み込み正常動作

---

## バックログ

### 未完了タスク

| ID | タスク | 優先度 | 備考 |
|----|--------|--------|------|
| BL-001 | サービス接続時にデフォルトツール設定を自動保存 | 高 | 現状は手動で「設定を保存」が必要。接続成功時にtools.jsonのdefaultEnabledを元に自動保存すべき |

### BL-001 詳細

**現状の問題:**
- サービス接続時にツール設定はDBに保存されない
- UIではtools.jsonのdefaultEnabledが表示されるが、DBには未保存
- ユーザーが明示的に「設定を保存」を押す必要がある

**期待する動作:**
- サービス接続成功後、自動的にデフォルトツール設定をDBに保存
- tools.jsonのdefaultEnabledを元に、enabled/disabledを設定

**実装箇所:**
- `apps/console/src/app/(console)/tools/page.tsx` の `handleConnectionConfirm()` または接続成功後の処理
- OAuth callback後のリダイレクト時にも対応が必要
