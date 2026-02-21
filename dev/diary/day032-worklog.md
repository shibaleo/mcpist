# DAY032 作業ログ

## 日付

2026-02-19

---

## 実施内容

### PostgREST → Go Server (GORM) + Clerk 認証 移行の継続

前回の大規模リファクタ (PostgREST/Supabase Auth → GORM 直接 DB + Clerk) に続き、
ローカル E2E 検証で発見されたバグの修正と、本番 (dev) 環境のインフラ設定を実施。

### 1. GetMCPContext クレデンシャルチェック修正

**問題:** MCP `tools/list` が、ユーザーが接続していないモジュール (例: supabase) のツールも返していた。
**原因:** `GetMCPContext` が `tool_settings` のみを確認し、`user_credentials` の存在をチェックしていなかった。
**修正:** `repo_user.go` の `GetMCPContext` クエリに `user_credentials` への JOIN を追加。

### 2. API キーの物理削除

**変更:** `RevokeAPIKey` をソフトデリート (`revoked_at` タイムスタンプ更新) から物理削除 (`DELETE`) に変更。
- `APIKey` モデルから `RevokedAt` フィールドを削除
- `ListAPIKeys` から `revoked_at IS NULL` フィルタを削除
- `APIKeyResponse` からも `RevokedAt` を削除

### 3. クレデンシャル upsert 時のツール自動有効化

**変更:** `UpsertCredential` に自動ツール有効化ロジックを追加。
- モジュールに初めて接続した際、`destructiveHint != true` のツールを自動で有効化
- 既に `tool_settings` が存在する場合はスキップ (再接続時に設定を保持)

### 4. Console の module.id (UUID) → module.name 修正

**問題:** Console の複数ページで `module.id` (UUID) を使って比較・ルックアップしていたが、
API レスポンスや DB のキーは `module.name` (テキスト識別子) を使用。

**影響:**
- サービス名がダイアログに表示されない
- 全アイコンがデフォルトの Wrench になる
- ツール設定ページが「接続済みサービスがありません」と表示

**修正ファイル:**
- `services/page.tsx` — ダイアログ lookup、接続/未接続フィルタ、ソート
- `tools/page.tsx` — connectedModules フィルタ、selectedModule lookup、combobox
- `page.tsx` (ホーム) — ModuleIcon 呼び出し
- `onboarding/page.tsx` — ModuleIcon 呼び出し

### 5. ModuleIcon prop リネーム

**変更:** `ModuleIcon` コンポーネントの prop を `moduleId` → `moduleName` にリネーム。
- `module-icon.tsx` — コンポーネント定義
- 全 4 ページ (services, tools, home, onboarding) の呼び出し箇所を更新

### 6. OAuth メタデータ・クライアントの Clerk 対応更新

**変更:**
- `.well-known/oauth-authorization-server` — Clerk discovery URL に向ける
- `.well-known/oauth-protected-resource` — 更新
- `lib/oauth/client.ts` — Clerk ベースに更新
- `auth-context.tsx` — Clerk `useUser` / `useClerk` ベースに統一

### 7. dev 環境インフラ設定

**Cloudflare Workers (dev):**
- `wrangler secret put` で 4 つのシークレットを設定:
  - `PRIMARY_API_URL` = `https://mcpist-api-dev.onrender.com`
  - `API_SERVER_JWKS_URL` = `https://mcpist-api-dev.onrender.com/.well-known/jwks.json`
  - `CLERK_JWKS_URL` = Clerk JWKS URL
  - `GATEWAY_SECRET`
- `wrangler deploy -e dev` でデプロイ

**Render (Go Server dev):**
- `POSTGREST_URL`, `POSTGREST_API_KEY` → 削除
- `DATABASE_URL`, `API_KEY_PRIVATE_KEY`, `ADMIN_EMAILS` → 追加

**Supabase (リモート dev):**
- `supabase db reset --linked` で DB を初期化
- 旧 PostgREST RPC 関数を全て削除、新 baseline マイグレーションを適用

---

## 未完了・次ステップ

- 本番 (dev) 環境での E2E 動作確認がまだ。リファクタの影響で稼働していない
- Clerk ダッシュボードが一時的に 500 エラーを返していた（Clerk 側の問題、復旧済み）
- MCP OAuth 2.1 認可フロー (Clerk ネイティブ) は次フェーズ

---

## コミット

ステージ済み (19 ファイル変更、253 追加、129 削除):

```
fix: correct module identification (UUID→name) across Console, harden credential/API key lifecycle
```
