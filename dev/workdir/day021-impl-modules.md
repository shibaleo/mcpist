# DAY021 追加実装計画: 新規モジュール

## 概要

5つのモジュールを追加実装する。既存モジュール（google_tasks）をテンプレートとして使用。

---

## 追加モジュール一覧

| モジュール | 認証方式 | API | 優先度 | 難易度 |
|-----------|---------|-----|--------|--------|
| Todoist | OAuth 2.0 | REST API v2 | 1 | 低 |
| Google Docs | OAuth 2.0 | REST API v1 | 2 | 中 |
| Trello | OAuth 1.0a / API Key | REST API | 3 | 低 |
| Asana | OAuth 2.0 | REST API v1 | 4 | 中 |
| PostgreSQL | Connection String | pg driver | 5 | 中 |

---

## Phase 1: Todoist モジュール

### 1.1 ファイル構成

```
apps/server/internal/modules/todoist/
├── module.go      # Module インターフェース実装
└── tools.go       # ツール定義とハンドラー (オプション: module.go に統合可)
```

### 1.2 認証

- **OAuth 2.0** (Authorization Code Flow)
- Token URL: `https://todoist.com/oauth/access_token`
- Scopes: `data:read_write`
- Refresh Token: なし（アクセストークンは永続的）

### 1.3 ツール定義

| ツール | 説明 | メソッド | エンドポイント |
|--------|------|---------|---------------|
| list_projects | プロジェクト一覧 | GET | /rest/v2/projects |
| get_project | プロジェクト取得 | GET | /rest/v2/projects/{id} |
| list_tasks | タスク一覧 | GET | /rest/v2/tasks |
| get_task | タスク取得 | GET | /rest/v2/tasks/{id} |
| create_task | タスク作成 | POST | /rest/v2/tasks |
| update_task | タスク更新 | POST | /rest/v2/tasks/{id} |
| complete_task | タスク完了 | POST | /rest/v2/tasks/{id}/close |
| delete_task | タスク削除 | DELETE | /rest/v2/tasks/{id} |

### 1.4 実装ステップ

1. [ ] `modules/todoist/module.go` 作成
2. [ ] OAuth アプリ設定を Supabase に登録（todoist）
3. [ ] `main.go` に RegisterModule 追加
4. [ ] Console に Todoist OAuth 連携 UI 追加
5. [ ] 動作確認

---

## Phase 2: Google Docs モジュール

### 2.1 認証

- **OAuth 2.0** (既存 Google OAuth 基盤を流用)
- Scopes: `https://www.googleapis.com/auth/documents`
- Token リフレッシュ: google_tasks と共通ロジック

### 2.2 ツール定義

| ツール | 説明 | メソッド | エンドポイント |
|--------|------|---------|---------------|
| list_documents | ドキュメント一覧 (Drive API) | GET | /drive/v3/files |
| get_document | ドキュメント取得 | GET | /documents/v1/documents/{id} |
| create_document | ドキュメント作成 | POST | /documents/v1/documents |
| batch_update | ドキュメント更新 | POST | /documents/v1/documents/{id}:batchUpdate |

### 2.3 実装ステップ

1. [ ] `modules/google_docs/module.go` 作成
2. [ ] Google Cloud Console でスコープ追加
3. [ ] `main.go` に RegisterModule 追加
4. [ ] 動作確認

---

## Phase 3: Trello モジュール

### 3.1 認証

- **API Key + Token** (OAuth 1.0a より簡易)
- API Key: アプリ固定
- Token: ユーザーごとに発行（有効期限なし or 30日）

### 3.2 ツール定義

| ツール | 説明 | メソッド | エンドポイント |
|--------|------|---------|---------------|
| list_boards | ボード一覧 | GET | /1/members/me/boards |
| get_board | ボード取得 | GET | /1/boards/{id} |
| list_lists | リスト一覧 | GET | /1/boards/{id}/lists |
| list_cards | カード一覧 | GET | /1/lists/{id}/cards |
| get_card | カード取得 | GET | /1/cards/{id} |
| create_card | カード作成 | POST | /1/cards |
| update_card | カード更新 | PUT | /1/cards/{id} |
| delete_card | カード削除 | DELETE | /1/cards/{id} |

### 3.3 実装ステップ

1. [ ] `modules/trello/module.go` 作成
2. [ ] Trello Power-Up / API Key 取得
3. [ ] `main.go` に RegisterModule 追加
4. [ ] Console に Trello API Key 入力 UI 追加
5. [ ] 動作確認

---

## Phase 4: Asana モジュール

### 4.1 認証

- **OAuth 2.0** (Authorization Code Flow)
- Token URL: `https://app.asana.com/-/oauth_token`
- Refresh Token: あり

### 4.2 ツール定義

| ツール | 説明 | メソッド | エンドポイント |
|--------|------|---------|---------------|
| list_workspaces | ワークスペース一覧 | GET | /workspaces |
| list_projects | プロジェクト一覧 | GET | /projects |
| get_project | プロジェクト取得 | GET | /projects/{id} |
| list_tasks | タスク一覧 | GET | /tasks |
| get_task | タスク取得 | GET | /tasks/{id} |
| create_task | タスク作成 | POST | /tasks |
| update_task | タスク更新 | PUT | /tasks/{id} |
| complete_task | タスク完了 | POST | /tasks/{id} (completed: true) |
| delete_task | タスク削除 | DELETE | /tasks/{id} |

### 4.3 実装ステップ

1. [ ] `modules/asana/module.go` 作成
2. [ ] Asana Developer Console でアプリ登録
3. [ ] OAuth アプリ設定を Supabase に登録（asana）
4. [ ] `main.go` に RegisterModule 追加
5. [ ] Console に Asana OAuth 連携 UI 追加
6. [ ] 動作確認

---

## Phase 5: PostgreSQL モジュール

### 5.1 認証

- **Connection String** (Basic 認証相当)
- 形式: `postgres://user:password@host:port/dbname?sslmode=require`

### 5.2 ツール定義

| ツール | 説明 | 備考 |
|--------|------|------|
| query | 読み取りクエリ実行 | SELECT のみ許可 |
| list_tables | テーブル一覧 | information_schema から |
| describe_table | テーブル構造 | カラム情報 |
| execute | 書き込みクエリ実行 | INSERT/UPDATE/DELETE（要確認ダイアログ） |

### 5.3 セキュリティ考慮

- **SQL インジェクション対策**: プリペアドステートメント必須
- **接続制限**: 1接続/リクエスト、タイムアウト設定
- **実行制限**: 1回のクエリで返す行数制限 (max_rows)
- **危険操作**: DROP/TRUNCATE/ALTER は禁止

### 5.4 実装ステップ

1. [ ] `modules/postgresql/module.go` 作成
2. [ ] `pgx` ドライバー追加
3. [ ] `main.go` に RegisterModule 追加
4. [ ] Console に接続文字列入力 UI 追加
5. [ ] 動作確認

---

## 共通作業

### Console UI 更新

1. [ ] `connections/page.tsx` に新モジュールの認証 UI 追加
2. [ ] `tools/page.tsx` に新モジュールのツール設定 UI 追加

### Supabase 更新

1. [ ] `modules` テーブルに新モジュール追加（マイグレーション）
2. [ ] OAuth アプリ設定追加（todoist, asana）

### ドキュメント

1. [ ] README にモジュール一覧更新
2. [ ] バックログ更新

---

## 見積もり時間

| Phase | モジュール | 見積もり |
|-------|-----------|---------|
| 1 | Todoist | 1-2時間 |
| 2 | Google Docs | 1-2時間 |
| 3 | Trello | 1-2時間 |
| 4 | Asana | 1-2時間 |
| 5 | PostgreSQL | 2-3時間 |
| - | Console UI・共通 | 1時間 |

**合計: 7-12時間**

---

## 着手順序（推奨）

1. **Todoist** - OAuth がシンプル（リフレッシュトークンなし）
2. **Trello** - API Key 方式で最も簡単
3. **Google Docs** - 既存 OAuth 基盤流用
4. **Asana** - 標準的な OAuth 2.0
5. **PostgreSQL** - セキュリティ考慮が必要

---

## 参照

- [Todoist REST API](https://developer.todoist.com/rest/v2/)
- [Google Docs API](https://developers.google.com/docs/api/reference/rest)
- [Trello REST API](https://developer.atlassian.com/cloud/trello/rest/)
- [Asana API](https://developers.asana.com/docs)
- [pgx - PostgreSQL driver](https://github.com/jackc/pgx)
