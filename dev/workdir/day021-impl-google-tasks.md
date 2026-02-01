# Google Tasks モジュール実装計画

## 概要

Google Tasks API を呼び出す MCP モジュールを `apps/server/internal/modules/google_tasks/` に実装する。

## 設計方針

### OAuth 構成

```
Google Cloud Console OAuth App: "google" (共有)
  │
  ├── google_calendar モジュール
  │     └── トークン (scope: calendar)
  │
  └── google_tasks モジュール (NEW)
        └── トークン (scope: tasks)
```

- **OAuth App（client_id/client_secret）**: `GetOAuthAppCredentials(ctx, "google")` で共有
- **トークン（access_token/refresh_token）**: `GetModuleToken(ctx, userID, "google_tasks")` でモジュール固有

### ファイル構成

```
apps/server/internal/modules/google_tasks/
└── module.go   # 単一ファイル（google_calendar と同じパターン）
```

## 実装するツール

| ツール | 説明 | HTTP | エンドポイント | Annotation |
|--------|------|------|----------------|------------|
| list_task_lists | タスクリスト一覧 | GET | /tasks/v1/users/@me/lists | ReadOnly |
| get_task_list | タスクリスト詳細 | GET | /tasks/v1/users/@me/lists/{tasklist} | ReadOnly |
| list_tasks | タスク一覧 | GET | /tasks/v1/lists/{tasklist}/tasks | ReadOnly |
| get_task | タスク詳細 | GET | /tasks/v1/lists/{tasklist}/tasks/{task} | ReadOnly |
| create_task | タスク作成 | POST | /tasks/v1/lists/{tasklist}/tasks | Create |
| update_task | タスク更新 | PATCH | /tasks/v1/lists/{tasklist}/tasks/{task} | Update |
| delete_task | タスク削除 | DELETE | /tasks/v1/lists/{tasklist}/tasks/{task} | Delete |
| complete_task | タスク完了トグル | PATCH | (status変更) | Update |
| clear_completed | 完了タスク一括削除 | POST | /tasks/v1/lists/{tasklist}/clear | Delete |

## 実装手順

### Step 1: module.go 作成

1. `apps/server/internal/modules/google_tasks/module.go` を作成
2. google_calendar/module.go をベースに以下を実装:
   - `GoogleTasksModule` struct
   - `New()`, `Name()`, `Descriptions()`, `APIVersion()` 等のメタデータ
   - `getCredentials()` - モジュール名を `"google_tasks"` に変更
   - `refreshToken()` - OAuth App は `"google"` を共有、保存先は `"google_tasks"`
   - `headers()` - Bearer トークン設定

### Step 2: ツール定義

`toolDefinitions` を定義:
- 各ツールの InputSchema（パラメータ定義）
- 日英のローカライズ説明

### Step 3: ツールハンドラ実装

```go
var toolHandlers = map[string]toolHandler{
    "list_task_lists":  listTaskLists,
    "get_task_list":    getTaskList,
    "list_tasks":       listTasks,
    "get_task":         getTask,
    "create_task":      createTask,
    "update_task":      updateTask,
    "delete_task":      deleteTask,
    "complete_task":    completeTask,
    "clear_completed":  clearCompleted,
}
```

### Step 4: モジュール登録

`apps/server/cmd/server/main.go` に追加:

```go
import "mcpist/server/internal/modules/google_tasks"

func init() {
    // ...existing modules...
    modules.RegisterModule(google_tasks.New())
}
```

### Step 5: tools-export への追加

`apps/server/cmd/tools-export/main.go` にも同様に追加

## API エンドポイント詳細

Base URL: `https://tasks.googleapis.com`

### list_tasks パラメータ

| パラメータ | 型 | 説明 | デフォルト |
|-----------|-----|------|-----------|
| task_list_id | string | タスクリストID | (必須) |
| max_results | number | 最大件数 | 100 |
| show_completed | boolean | 完了タスクを含む | true |
| show_hidden | boolean | 非表示タスクを含む | false |

### create_task パラメータ

| パラメータ | 型 | 説明 | 必須 |
|-----------|-----|------|------|
| task_list_id | string | タスクリストID | ✓ |
| title | string | タスクタイトル | ✓ |
| notes | string | メモ | |
| due | string | 期限 (RFC3339) | |
| parent | string | 親タスクID | |

### update_task パラメータ

| パラメータ | 型 | 説明 | 必須 |
|-----------|-----|------|------|
| task_list_id | string | タスクリストID | ✓ |
| task_id | string | タスクID | ✓ |
| title | string | 新しいタイトル | |
| notes | string | 新しいメモ | |
| due | string | 新しい期限 | |
| status | string | needsAction / completed | |

## 変更対象ファイル

### Server (Go) - ✅ 完了

| ファイル | 変更内容 | 状態 |
|----------|----------|------|
| `apps/server/internal/modules/google_tasks/module.go` | 新規作成 | ✅ |
| `apps/server/cmd/server/main.go` | import追加、RegisterModule追加 | ✅ |
| `apps/server/cmd/tools-export/main.go` | import追加、RegisterModule追加 | ✅ |

### Console (Next.js) - ✅ 完了

**注意:** 当初計画の google-tasks 専用エンドポイントは作成せず、共有コールバック方式に変更

| ファイル | 変更内容 | 状態 |
|----------|----------|------|
| `apps/console/src/app/api/oauth/google/authorize/route.ts` | module パラメータ対応、モジュール別スコープ定義 | ✅ |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | state から module を取り出し、適切なモジュールにトークン保存 | ✅ |
| `apps/console/src/lib/oauth-apps.ts` | google-tasks を google authorize にルーティング | ✅ |
| `apps/console/src/app/(console)/tools/page.tsx` | google_tasks の authConfig 追加 | ✅ |

### Database - ✅ 完了

| ファイル | 変更内容 | 状態 |
|----------|----------|------|
| `supabase/migrations/00000000000013_fix_oauth_apps_vault_update.sql` | vault.update_secret → delete+create パターンに修正 | ✅ |

## Console OAuth 実装詳細（実績）

### OAuth scope

```
https://www.googleapis.com/auth/tasks
```

### 共有コールバック方式

**理由:** Google Cloud Console に登録済みの redirect_uri は `/api/oauth/google/callback` のみ

**実装:**
1. `authorize/route.ts` で `module` パラメータを受け取り、モジュール別スコープを使用
2. state パラメータに `module` を含める（base64url エンコード）
3. `callback/route.ts` で state から `module` を取り出し、適切なモジュールにトークン保存

### clear_completed の挙動修正

**問題:** Google Tasks API の `/clear` エンドポイントは `hidden=true` を設定するだけで、UI からタスクが消えない

**解決:** 完了タスクを取得して各々を DELETE する実装に変更

## 検証方法

1. **ビルド確認**
   ```bash
   cd apps/server && go build ./...
   ```

2. **ローカル動作確認**
   - サーバー起動
   - MCP Inspector または Claude Web で google_tasks モジュールのスキーマ取得
   - list_task_lists, list_tasks, create_task を実行

3. **E2E テスト** (Console OAuth 設定後)
   - Claude Web で Google Tasks 連携テスト

## 参考

- [Google Tasks API Reference](https://developers.google.com/tasks/reference/rest)
- 既存実装: `apps/server/internal/modules/google_calendar/module.go`
