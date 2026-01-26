# Sprint-005: OAuth認証実装 (Google Calendar)

## 概要
外部サービス連携のためのOAuth認証フローを実装。第一弾としてGoogle Calendarを対応。

## 完了タスク

### 1. データベース設計 (Supabase)

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

### 2. Console (Next.js) 実装

#### 管理者設定ページ
- `apps/console/src/app/(console)/admin/oauth/page.tsx`
- OAuth App（Google/Microsoft）の設定UI
- Client ID, Client Secret, Redirect URIの管理

#### OAuth認可フロー
- `apps/console/src/app/api/oauth/google/authorize/route.ts`
  - Google OAuth認可URLの生成
  - CSRF対策のstateパラメータ生成

- `apps/console/src/app/api/oauth/google/callback/route.ts`
  - 認可コードをアクセストークンに交換
  - トークンをVaultに保存（`upsert_service_token` RPC使用）

#### Connections ページ更新
- `apps/console/src/app/(console)/connections/page.tsx`
- OAuthサービス（google_calendar, microsoft_todo）の接続フロー
- クエリパラメータでの成功/エラー通知

### 3. Go Server 実装

#### google_calendar モジュール
- `apps/server/internal/modules/google_calendar/module.go`

**ツール一覧:**
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

#### モジュール登録
- `apps/server/cmd/server/main.go` に google_calendar モジュールを追加

## アーキテクチャ

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

## 環境変数

```env
# Console (.env.local)
SUPABASE_SECRET_KEY=your-service-role-key  # OAuth認証情報取得に必要
```

## テスト確認

### サーバー起動ログ
```
Registered modules: [google_calendar notion github jira confluence supabase airtable]
```

### MCP tools/list レスポンス
google_calendarが利用可能モジュールとして表示:
```
モジュール名(notion, github, jira, confluence, supabase, google_calendar, microsoft_todo, rag)
```

## 今後の作業 (Sprint-006予定)

1. **Microsoft OAuth実装**
   - `/api/oauth/microsoft/authorize`
   - `/api/oauth/microsoft/callback`
   - microsoft_todo モジュール

2. **トークンリフレッシュ**
   - アクセストークン有効期限切れ時の自動更新

3. **エラーハンドリング強化**
   - OAuth認可エラーの詳細表示
   - トークン取り消し対応
