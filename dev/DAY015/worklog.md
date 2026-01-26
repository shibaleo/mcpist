# DAY015 作業ログ

## 日付

2026-01-27

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

### Phase 2 残タスク（Worker/Go側）

Console側は完了済みのため、Phase 2の残りはWorker/Go Serverのリファクタ:

| ID | タスク | 対象 | 状態 |
|----|--------|------|------|
| S5-030 | `lookup_user_by_key_hash` 使用 | Worker | ⬜ 未着手 |
| S5-040 | `get_module_token` RPC使用 | Go Server | ⬜ 未着手 |
| S5-041 | `consume_credit` RPC使用 | Go Server | ⬜ 未着手 |
| S5-042 | `get_user_context` RPC呼び出し | Go Server | ⬜ 未着手 |

---

## 結論

- **Console (Next.js)**: RPC呼び出しリファクタ完了
- **Worker/Go Server**: リファクタ未着手（Phase 2残タスク）
- **database.types.ts**: 型定義完了、追加不要

Sprint-005 Phase 2の進捗を更新:
- 旧: 0/9 (0%)
- 新: 5/9 (56%) - Console側5タスク完了

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

## 未解決の問題

### vault.secrets への UPDATE 権限付与が失敗

`update_module_token` RPC は `SECURITY DEFINER` として `postgres` ユーザーで実行されるが、`vault.secrets` テーブルへの UPDATE 権限が付与できない。

#### 状況

1. マイグレーションファイル `00000000000016_fix_update_module_token_grant.sql` を作成
   ```sql
   GRANT UPDATE ON vault.secrets TO postgres;
   ```

2. `supabase db push` で適用 → "Remote database is up to date" と表示されるが実際には適用されていない

3. Supabase Dashboard SQL Editor で直接実行（Role: postgres）→ "Success" と表示されるが権限が付与されない

4. 権限確認:
   ```sql
   SELECT relacl FROM pg_class WHERE relname = 'secrets' AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'vault');
   ```
   結果: `postgres=r*d*D*x*` (UPDATE=w がない)

#### 原因

`vault.secrets` テーブルの所有者は `supabase_admin` であり、`postgres` ロールから GRANT を実行しても権限が付与されない。

#### 影響

トークンリフレッシュ自体は成功するが、リフレッシュ後の新しいトークンを Vault に保存できない（403エラー）。
次回のAPI呼び出し時に再度リフレッシュが発生する（機能的には動作するが非効率）。

#### 試した解決策

1. `supabase db push` → マイグレーション記録はされるが SQL が実行されない
2. Dashboard で Role: postgres として GRANT → 成功表示されるが反映されない
3. `supabase` モジュールの `run_query` で GRANT → 成功表示されるが反映されない

#### 次のステップ

1. Supabase Dashboard で Role を `supabase_admin` に変更して GRANT を実行
2. または、関数の所有者を `supabase_admin` に変更:
   ```sql
   ALTER FUNCTION mcpist.update_module_token(UUID, TEXT, JSONB) OWNER TO supabase_admin;
   ```

---

## デバッグ用設定（要修正）

テスト用に `tokenRefreshBuffer` を1年に設定してあるため、本番前に修正が必要:

```go
// apps/server/internal/modules/google_calendar/module.go:23
tokenRefreshBuffer = 365 * 24 * 60 * 60 // DEBUG: Force refresh (1 year buffer)
```

本番値: `tokenRefreshBuffer = 5 * 60` （5分）

---

## 次のアクション

1. vault.secrets への UPDATE 権限問題を解決
2. tokenRefreshBuffer を本番値（5分）に戻す
3. デバッグログの削除
