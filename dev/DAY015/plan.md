# DAY015 計画

## 日付

2026-01-27

---

## 概要

Sprint-005の継続。DAY014で残った課題（マイグレーション適用、RPC呼び出しリファクタ、ツール設定API）を進める。

---

## 前日までの状況（DAY014完了時点）

### 完了済み
- RPC関数実装: 16/17 (94%)
- モジュール拡張: Airtable 11ツール
- インフラ整備: Render/Koyeb GitHub連携、render.yaml追加
- 不要ファイル削除: .devcontainer/, compose/, infra/
- モジュール自動同期: sync_modules RPC

### 未着手
- Phase 2: RPC呼び出しリファクタ (0%)
- Phase 4: UI要件定義 (0%)
- Phase 5: ツール設定API (25%)

---

## 本日の目標

**マイグレーション適用 + RPC呼び出しリファクタ開始**

---

## タスク一覧

### 優先度: 高

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D15-001 | マイグレーションpush | ⬜ | `supabase db push` でsync_modules RPC適用 |
| D15-002 | database.types.ts にRPC型定義追加 | ✅ | 既に完了済み |
| D15-003 | api-keys ページをRPC使用に統一 | ✅ | RPC利用済み |
| D15-004 | connections ページをRPC使用に統一 | ✅ | RPC利用済み |
| D15-005 | dashboard クレジット表示をRPC化 | ✅ | RPC利用済み |

### 優先度: 中

| ID      | タスク                     | 状態  | 備考                                 |
| ------- | ----------------------- | --- | ---------------------------------- |
| D15-006 | next.config.ts デバッグログ削除 | ⬜   | console.log削除                      |
| D15-007 | Worker側RPC呼び出しリファクタ     | ⬜   | lookup_user_by_key_hash確認          |
| D15-008 | Go Server側RPC呼び出しリファクタ  | ⬜   | get_module_token, consume_credit確認 |

### 優先度: 低

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D15-009 | tool_settingsテーブル作成 | ⬜ | S5-072: Phase 5継続 |
| D15-010 | /tools ページをtools.json使用に変更 | ⬜ | S5-075: ハードコード削除 |

---

## 技術メモ

### マイグレーション手順

```bash
cd supabase
supabase db push
```

### RPC型定義追加手順

1. `supabase/migrations/` にあるRPC定義を確認
2. `apps/console/src/database.types.ts` に型追加
3. Console側でRPC呼び出しコードを更新

### 対象RPC関数（型定義追加対象）

| RPC | 戻り値 |
|-----|--------|
| list_oauth_consents | consent[] |
| revoke_oauth_consent | void |
| list_all_oauth_consents | consent[] |
| sync_modules | module[] |

---

## 確認事項

- [ ] マイグレーション適用前にSupabaseダッシュボードで現在の状態を確認
- [ ] RPC型定義追加後、`pnpm exec next build` で型チェック
- [ ] リファクタ後、各ページの動作確認

---

## 参考

- [DAY014/backlog.md](../DAY014/backlog.md) - 残タスク一覧
- [DAY014/sprint-005.md](../DAY014/sprint-005.md) - 詳細タスク
- [dsn-rpc.md](../../docs/design/dsn-rpc.md) - RPC設計書

---

## Phase 6: OAuth認証モジュール実装

### 概要

Google Calendar、Microsoft TodoのOAuth 2.0認証モジュールを実装する。dwhbiの実装を参考にGoサーバー側で実装。

### 参考実装（dwhbi）

- `dwhbi/packages/console/src/app/api/mcp/modules/google-calendar/`
- `dwhbi/packages/console/src/app/api/mcp/modules/microsoft-todo/`

### アーキテクチャ

```
┌─────────────────────────────────────────────────────────────┐
│                    Admin Console                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ OAuth App Settings                                   │    │
│  │ - Google: Client ID, Client Secret, Redirect URI    │    │
│  │ - Microsoft: Client ID, Client Secret, Redirect URI │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────┘
                           │ save to
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Supabase                                  │
│  ┌──────────────────┐  ┌────────────────────────────────┐  │
│  │ oauth_apps       │  │ vault.secrets                   │  │
│  │ - provider       │  │ - client_id                     │  │
│  │ - secret_id (FK) │  │ - client_secret                 │  │
│  │ - redirect_uri   │  │ - (encrypted)                   │  │
│  └──────────────────┘  └────────────────────────────────┘  │
└──────────────────────────┬──────────────────────────────────┘
                           │ read via RPC
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Go Server                                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ modules/google_calendar/                            │    │
│  │   - client.go (OAuth refresh, Calendar API)         │    │
│  │   - schema.go (Tool definitions)                    │    │
│  │   - tools.go (Tool handlers)                        │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ modules/microsoft_todo/                             │    │
│  │   - client.go (OAuth refresh, Graph API)            │    │
│  │   - schema.go (Tool definitions)                    │    │
│  │   - tools.go (Tool handlers)                        │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### 実装タスク

#### 6.1 データベース設計

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D15-011 | oauth_appsテーブル作成 | ⬜ | provider, secret_id, redirect_uri, enabled |
| D15-012 | get_oauth_app_credentials RPC作成 | ⬜ | Vaultからclient_id/secretを取得 |
| D15-013 | upsert_oauth_app RPC作成 | ⬜ | 管理者用の設定保存 |

**oauth_appsテーブル設計:**
```sql
CREATE TABLE mcpist.oauth_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL UNIQUE,  -- 'google', 'microsoft'
    secret_id UUID REFERENCES vault.secrets(id),
    redirect_uri TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Vaultに保存するシークレット形式:**
```json
{
  "client_id": "xxx.apps.googleusercontent.com",
  "client_secret": "GOCSPX-xxx"
}
```

#### 6.2 Admin Console設定画面

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D15-014 | /admin/oauth-apps ページ作成 | ⬜ | OAuth設定一覧 |
| D15-015 | OAuthAppForm コンポーネント | ⬜ | Client ID/Secret入力フォーム |
| D15-016 | database.types.ts 型定義追加 | ⬜ | oauth_apps, RPC型 |

**設定画面UI:**
```
┌────────────────────────────────────────────────────┐
│ OAuth App Settings                                 │
├────────────────────────────────────────────────────┤
│ Google Calendar                          [Enabled] │
│ ┌────────────────────────────────────────────────┐ │
│ │ Client ID:     [________________________]      │ │
│ │ Client Secret: [________________________]      │ │
│ │ Redirect URI:  https://mcpist.app/callback/    │ │
│ │                google                          │ │
│ └────────────────────────────────────────────────┘ │
│                                          [Save]    │
├────────────────────────────────────────────────────┤
│ Microsoft Todo                           [Enabled] │
│ ┌────────────────────────────────────────────────┐ │
│ │ Client ID:     [________________________]      │ │
│ │ Client Secret: [________________________]      │ │
│ │ Redirect URI:  https://mcpist.app/callback/    │ │
│ │                microsoft                       │ │
│ └────────────────────────────────────────────────┘ │
│                                          [Save]    │
└────────────────────────────────────────────────────┘
```

#### 6.3 Go Server OAuth Client実装

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D15-017 | store/oauth.go 作成 | ⬜ | get_oauth_app_credentials RPC呼び出し |
| D15-018 | modules/google_calendar/client.go | ⬜ | OAuth refresh + Calendar API |
| D15-019 | modules/google_calendar/schema.go | ⬜ | 10ツール定義 |
| D15-020 | modules/google_calendar/tools.go | ⬜ | ツールハンドラー |
| D15-021 | modules/microsoft_todo/client.go | ⬜ | OAuth refresh + Graph API |
| D15-022 | modules/microsoft_todo/schema.go | ⬜ | 11ツール定義 |
| D15-023 | modules/microsoft_todo/tools.go | ⬜ | ツールハンドラー |

**OAuth Refreshフロー（dwhbi参考）:**
```go
// client.go
type Client struct {
    userID      string
    tokenStore  *store.TokenStore
    oauthStore  *store.OAuthStore
    httpClient  *http.Client
}

func (c *Client) ensureValidToken(ctx context.Context) (*store.Credentials, error) {
    // 1. TokenStoreからユーザートークン取得
    token, err := c.tokenStore.GetModuleToken(ctx, c.userID, "google_calendar")

    // 2. 期限切れチェック
    if token.Credentials.ExpiresAt <= time.Now().Unix() {
        // 3. OAuthStoreからClient ID/Secret取得
        appCreds, err := c.oauthStore.GetOAuthAppCredentials(ctx, "google")

        // 4. トークンリフレッシュ
        newToken, err := c.refreshAccessToken(appCreds, token.Credentials.RefreshToken)

        // 5. 新トークンを保存
        err = c.tokenStore.UpdateModuleToken(ctx, c.userID, "google_calendar", newToken)
    }

    return token.Credentials, nil
}
```

#### 6.4 Google Calendarツール一覧（dwhbi参考）

| ツール名 | 説明 |
|---------|------|
| list_calendars | カレンダー一覧取得 |
| get_calendar | カレンダー詳細取得 |
| list_events | イベント一覧取得 |
| get_event | イベント詳細取得 |
| create_event | イベント作成 |
| update_event | イベント更新 |
| delete_event | イベント削除 |
| quick_add_event | 自然言語でイベント追加 |
| list_colors | カラーパレット取得 |
| get_freebusy | 空き時間検索 |

#### 6.5 Microsoft Todoツール一覧（dwhbi参考）

| ツール名 | 説明 |
|---------|------|
| list_task_lists | タスクリスト一覧 |
| get_task_list | タスクリスト詳細 |
| create_task_list | タスクリスト作成 |
| delete_task_list | タスクリスト削除 |
| list_tasks | タスク一覧 |
| get_task | タスク詳細 |
| create_task | タスク作成 |
| update_task | タスク更新 |
| delete_task | タスク削除 |
| complete_task | タスク完了 |
| list_linked_resources | リンクリソース一覧 |

### 優先順位

1. **D15-011〜013**: データベース・RPC（依存なし）
2. **D15-014〜016**: Admin Console（D15-011〜013完了後）
3. **D15-017**: store/oauth.go（D15-012完了後）
4. **D15-018〜020**: Google Calendar（D15-017完了後）
5. **D15-021〜023**: Microsoft Todo（D15-017完了後）

### 見積もり

| フェーズ | タスク数 |
|---------|---------|
| 6.1 DB設計 | 3 |
| 6.2 Admin UI | 3 |
| 6.3 Go Server | 7 |
| **合計** | **13** |
