# Sprint 005 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-005 |
| 期間 | 2026-01-25 〜 |
| マイルストーン | M4: RPC実装・リファクタリング・UI要件定義 |

---

## Sprint目標

**テーブル定義に基づくRPC関数の実装、各コンポーネントのRPC呼び出しリファクタ、パスルーティング設計、ユーザーコンソール要件定義**

---

## タスク一覧

### Phase 1: RPC関数実装 (Supabase)

設計書 [dtl-dsn-rpc.md](../../docs/design/dtl-dsn-rpc.md) に基づく実装

#### MCP Server向け（service_role）

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-001 | lookup_user_by_key_hash 実装 | ✅ 完了 | APIキーハッシュ → user_id |
| S5-002 | get_user_context 実装 | ✅ 完了 | ツール実行用ユーザー情報 |
| S5-003 | consume_credit 実装 | ✅ 完了 | クレジット消費・履歴記録 |
| S5-004 | get_module_token 実装 | ✅ 完了 | モジュール用トークン取得 |
| S5-005 | update_module_token 実装 | ⬜ 未着手 | リフレッシュ後トークン保存 |

#### Console Frontend向け（authenticated）

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-006 | generate_api_key 実装 | ✅ 完了 | APIキー生成（既存を統合） |
| S5-007 | list_api_keys 実装 | ✅ 完了 | APIキー一覧取得 |
| S5-008 | revoke_api_key 実装 | ✅ 完了 | APIキー論理削除 |
| S5-009 | list_service_connections 実装 | ✅ 完了 | サービス接続一覧 |
| S5-010 | upsert_service_token 実装 | ✅ 完了 | トークン登録/更新 |
| S5-011 | delete_service_token 実装 | ✅ 完了 | トークン削除 |
| S5-014 | list_oauth_consents 実装 | ✅ 完了 | OAuth認可済みクライアント一覧 |
| S5-015 | revoke_oauth_consent 実装 | ✅ 完了 | OAuth認可取り消し |
| S5-016 | list_all_oauth_consents 実装 | ✅ 完了 | 全ユーザーOAuth認可一覧（管理者用） |

#### Console API Routes向け / Cron

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-012 | add_paid_credits 実装 | ✅ 完了 | Webhook用クレジット加算 |
| S5-013 | reset_free_credits 実装 | ✅ 完了 | 月次リセット（pg_cron） |

---

### Phase 2: RPC呼び出しリファクタ

#### Console (Next.js)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-020 | api-keys ページをRPC使用に統一 | ⬜ 未着手 | generate/list/revoke |
| S5-021 | connections ページをRPC使用に統一 | ⬜ 未着手 | list/upsert/delete |
| S5-022 | dashboard クレジット表示をRPC化 | ⬜ 未着手 | 直接テーブル参照 → RPC |
| S5-023 | token-vault API Route リファクタ | ⬜ 未着手 | upsert_service_token RPC使用 |
| S5-024 | database.types.ts にRPC型定義追加 | ⬜ 未着手 | 新規RPC全てに対応 |

#### Worker (Cloudflare)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-030 | APIキー検証を lookup_user_by_key_hash 使用に統一 | ⬜ 未着手 | 既存実装の整理 |
| S5-031 | キャッシュ無効化ロジック確認 | ⬜ 未着手 | /internal/invalidate-api-key |

#### MCP Server (Go)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-040 | token/store.go を get_module_token RPC使用に変更 | ⬜ 未着手 | 現在: 直接クエリ |
| S5-041 | クレジット消費を consume_credit RPC使用に変更 | ⬜ 未着手 | 現在: 直接クエリ |
| S5-042 | get_user_context RPC呼び出し実装 | ⬜ 未着手 | アカウント状態・設定取得 |
| S5-043 | update_module_token RPC呼び出し実装 | ⬜ 未着手 | トークンリフレッシュ時 |

---

### Phase 3: パスルーティング設計

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-050 | 現行ルート構造の整理 | ⬜ 未着手 | 既存ページ・APIの棚卸し |
| S5-051 | ルーティング設計書作成 (dsn-route.md) | ⬜ 未着手 | URL設計、認証要件 |
| S5-052 | 不要ルートの削除・統合 | ⬜ 未着手 | dev/mcp-client等 |

---

### Phase 4: ユーザーコンソール要件定義

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-060 | UI要求仕様書作成 (spc-ui.md) | ⬜ 未着手 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | ⬜ 未着手 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | ⬜ 未着手 | 認証後のナビゲーション |

---

## 現行ルート構造（棚卸し）

### ページ (page.tsx)

| パス | 用途 | 状態 |
|------|------|------|
| `/` | ランディング | 🔄 |
| `/login` | ログイン | ✅ |
| `/dashboard` | ダッシュボード | 🔄 |
| `/my/api-keys` | APIキー管理 | ✅ |
| `/my/connections` | サービス接続管理 | 🔄 |
| `/my/mcp-connection` | MCP接続テスト | 🔄 要整理 |
| `/my/preferences` | 設定 | 🔄 |
| `/settings` | 設定（重複?） | 🔄 要整理 |
| `/billing` | 課金 | ⬜ 未実装 |
| `/marketplace` | マーケットプレイス | ⬜ 未実装 |
| `/admin` | 管理者 | 🔄 |
| `/dev/mcp-client` | 開発用MCPクライアント | 🔄 要整理 |
| `/dev/mcp-client/callback` | 開発用callback | 🔄 要整理 |
| `/oauth/consent` | OAuth同意画面 | ✅ |
| `/oauth/callback` | OAuthコールバック | ✅ |

### API Routes (route.ts)

| パス | 用途 | 状態 |
|------|------|------|
| `/api/token-vault` | トークン保存 | 🔄 要リファクタ |
| `/api/validate-token` | トークン検証 | 🔄 要確認 |
| `/auth/callback` | Supabase Auth callback | ✅ |
| `/.well-known/oauth-authorization-server` | OAuth metadata | ✅ |
| `/.well-known/oauth-protected-resource` | OAuth resource metadata | ✅ |

---

## RPC関数一覧（設計書より）

### MCP Server向け（service_role）

| RPC関数 | 用途 | 参照テーブル | 状態 |
|---------|------|-------------|------|
| lookup_user_by_key_hash | APIキー検証 | api_keys | ✅ |
| get_user_context | ユーザー情報取得 | users, credits, module_settings, tool_settings | ✅ |
| consume_credit | クレジット消費 | credits, credit_transactions | ✅ |
| get_module_token | トークン取得 | service_tokens, vault.secrets | ✅ |
| update_module_token | トークン更新 | service_tokens, vault.secrets | ⬜ |

### Console Frontend向け（authenticated）

| RPC関数 | 用途 | 参照テーブル | 状態 |
|---------|------|-------------|------|
| generate_api_key | APIキー生成 | api_keys | ✅ |
| list_api_keys | APIキー一覧 | api_keys | ✅ |
| revoke_api_key | APIキー削除 | api_keys | ✅ |
| list_service_connections | 接続一覧 | service_tokens | ✅ |
| upsert_service_token | トークン登録/更新 | service_tokens, vault.secrets | ✅ |
| delete_service_token | トークン削除 | service_tokens, vault.secrets | ✅ |
| list_oauth_consents | OAuth認可一覧 | auth.oauth_consents | ✅ |
| revoke_oauth_consent | OAuth認可取り消し | auth.oauth_consents | ✅ |
| list_all_oauth_consents | 全ユーザーOAuth認可一覧（admin） | auth.oauth_consents, auth.users | ✅ |

### Console API Routes / Cron

| RPC関数 | 用途 | 参照テーブル | 状態 |
|---------|------|-------------|------|
| add_paid_credits | クレジット加算 | credits, credit_transactions, processed_webhook_events | ✅ |
| reset_free_credits | 月次リセット | credits, credit_transactions | ✅ |

---

## 完了条件

### Phase 1: RPC関数実装
- [x] 設計書に記載されたRPC関数がSupabaseに実装されている（15/16完了、update_module_token未実装）
- [x] 各RPCがRLSポリシーに準拠している
- [x] SQLマイグレーションファイルが作成されている

### Phase 2: RPC呼び出しリファクタ
- [ ] Console: 直接テーブル参照がRPC呼び出しに置き換えられている
- [ ] Worker: lookup_user_by_key_hash を使用している
- [ ] MCP Server: 全RPC関数を呼び出せる

### Phase 3: パスルーティング設計
- [ ] dsn-route.md が作成されている
- [ ] 不要ルートが削除/整理されている

### Phase 4: ユーザーコンソール要件定義
- [ ] spc-ui.md が作成されている
- [ ] ユーザーフロー図が作成されている
- [ ] 画面遷移図が作成されている

---

## 技術メモ

### RPC関数のセキュリティ

| 呼び出し元 | Supabaseキー | 備考 |
|-----------|-------------|------|
| Console Frontend | anon key | RLS適用、auth.uid()で制御 |
| Console API Routes | service_role key | RLSバイパス、Webhook処理 |
| MCP Server (Go) | service_role key | ツール実行、クレジット消費 |
| Cloudflare Worker | service_role key | APIキー検証のみ |

### 既存RPC関数（確認必要）

現在Supabaseに存在するRPC関数を確認し、設計書との差分を洗い出す。

---

## 依存関係

```
Phase 1 (RPC実装)
    ↓
Phase 2 (リファクタ)
    ↓
Phase 3 (ルーティング設計) ←─ Phase 4 (UI要件定義)
```

Phase 1完了後、Phase 2-4は並行作業可能。

---

## 参考資料

- [dsn-rpc.md](../../docs/design/dsn-rpc.md) - RPC関数設計書
- [dtl-dsn-rpc.md](../../docs/design/dtl-dsn-rpc.md) - RPC関数詳細設計書
- [dtl-dsn-tbl.md](../../docs/design/dtl-dsn-tbl.md) - テーブル詳細設計書
- [DAY013 review.md](../DAY013/review.md) - 前スプリントレビュー
- [DAY014 backlog.md](./backlog.md) - バックログ

---

## 作業ログ

### 2026-01-25: OAuth Consent管理機能の実装

#### 実装内容

OAuth認証で接続したMCPクライアント（認可済みクライアント）を表示・管理する機能を実装。

#### 作成・修正ファイル

| ファイル | 変更内容 |
|---------|---------|
| `supabase/migrations/00000000000009_rpc_oauth_consents.sql` | OAuth Consent用RPC関数（新規） |
| `apps/console/src/lib/oauth-consents.ts` | API呼び出しライブラリ（新規） |
| `apps/console/src/lib/supabase/database.types.ts` | RPC型定義追加 |
| `apps/console/src/app/(console)/my/mcp-connection/page.tsx` | 認可済みクライアント表示・取り消し機能追加 |
| `apps/console/src/app/(console)/admin/page.tsx` | 全ユーザーのOAuth認可状況表示追加 |

#### RPC関数（auth.oauth_consents用）

| 関数名 | 用途 | 権限 |
|-------|------|------|
| `list_oauth_consents` | ユーザー自身の認可済みクライアント一覧 | authenticated |
| `revoke_oauth_consent` | 認可の取り消し（論理削除） | authenticated |
| `list_all_oauth_consents` | 全ユーザーの認可状況（管理者用） | admin only |

#### 技術メモ

- `auth.oauth_consents`テーブルはSupabase OAuthサーバーが管理するスキーマのため、SECURITY DEFINER関数でアクセス
- 管理者チェックは`raw_app_meta_data->>'role' = 'admin'`で実施
- 取り消しは物理削除ではなく`revoked_at`を更新する論理削除

#### 未完了

- [ ] Supabaseにマイグレーションをプッシュ（`supabase db push`）
- [ ] 本番環境での動作確認

#### コミットメッセージ

```
feat: add OAuth consent management to MCP connection and admin pages

- Add RPC functions for OAuth consent management (list, revoke)
- Display authorized OAuth clients in MCP connection page (OAuth tab)
- Add consent revocation feature for users
- Show all users' OAuth consents in admin panel
- Create oauth-consents.ts library for API functions
```

---

### 次のタスク: Phase 3 パスルーティング設計

現行のルート構造を整理し、設計書を作成する。

#### 作業内容

1. **dsn-route.md の作成**
   - 全ページ・APIルートのURL設計
   - 認証要件（public / authenticated / admin）
   - 各ルートの責務・用途

2. **不要ルートの整理**
   - `/dev/mcp-client` - 開発用、本番では不要
   - `/settings` と `/my/preferences` の重複解消
   - `/my/api-keys` と `/my/mcp-connection` の統合検討

3. **ルート命名規則の統一**
   - `/my/*` - ユーザー個人の設定・管理
   - `/admin/*` - 管理者機能
   - `/oauth/*` - OAuth関連
   - `/api/*` - API Routes
