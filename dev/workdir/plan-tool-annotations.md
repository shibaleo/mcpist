# 計画書: MCP Tool Annotations 準拠 + defaultEnabled 廃止

## 日付

2026-01-27

---

## 背景

### 現状

- tools.json に独自フィールド `defaultEnabled`, `dangerous` をハードコード
- MCP仕様には `annotations` (readOnlyHint, destructiveHint, idempotentHint, openWorldHint) が定義済み
- OpenAI / Anthropic 共にディレクトリ審査で annotations 必須

### 方針決定

- `dangerous` → MCP annotations に置き換え（ハードコード）
- `defaultEnabled` → `readOnlyHint: true` なら有効、それ以外は無効（機械的に導出）
- 管理画面は不要。annotations からすべて決定する

---

## 設計

### ルール

```
readOnlyHint: true  → defaultEnabled = true  （読み取り専用は安全）
readOnlyHint: false → defaultEnabled = false （書き込み系は明示的に有効化が必要）
```

### 独自フィールドと MCP annotations の対応

| 現在（独自） | MCP annotations |
|---|---|
| `dangerous: false` | `readOnlyHint: true` or `destructiveHint: false` |
| `dangerous: true` | `destructiveHint: true` |
| `defaultEnabled: true` | 導出: `readOnlyHint: true` |
| `defaultEnabled: false` | 導出: `readOnlyHint: false` |

### tools.json 変更

```jsonc
// 変更前
{
  "id": "delete_event",
  "name": "delete_event",
  "description": "Delete an event from a calendar.",
  "dangerous": true,
  "defaultEnabled": false
}

// 変更後
{
  "id": "delete_event",
  "name": "delete_event",
  "description": "Delete an event from a calendar.",
  "annotations": {
    "readOnlyHint": false,
    "destructiveHint": true,
    "idempotentHint": true,
    "openWorldHint": false
  }
}
```

`defaultEnabled` と `dangerous` は削除。`annotations` から導出。

---

## 実装計画

### Step 1: tools.json を MCP annotations 形式に変換

`defaultEnabled`, `dangerous` を削除し、`annotations` を追加。

**変更ファイル:** `apps/console/src/lib/tools.json`

### Step 2: TypeScript 型定義を更新

**変更ファイル:** `apps/console/src/lib/module-data.ts`

```typescript
// 変更前
export interface ToolDef {
  id: string
  name: string
  description: string
  dangerous: boolean
  defaultEnabled: boolean
}

// 変更後
export interface ToolAnnotations {
  readOnlyHint?: boolean    // default: false
  destructiveHint?: boolean // default: true
  idempotentHint?: boolean  // default: false
  openWorldHint?: boolean   // default: true
}

export interface ToolDef {
  id: string
  name: string
  description: string
  annotations: ToolAnnotations
}
```

ヘルパー関数を追加:

```typescript
/** readOnlyHint: true のツールはデフォルト有効 */
export function isDefaultEnabled(tool: ToolDef): boolean {
  return tool.annotations.readOnlyHint === true
}

/** destructiveHint: true のツールは危険表示 */
export function isDangerous(tool: ToolDef): boolean {
  return tool.annotations.destructiveHint !== false
    && tool.annotations.readOnlyHint !== true
}
```

### Step 3: Console UI の参照を更新

`tool.defaultEnabled` → `isDefaultEnabled(tool)` に置換。
`tool.dangerous` → `isDangerous(tool)` に置換。

**変更ファイル:**

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/(console)/tools/page.tsx` | `tool.defaultEnabled` → `isDefaultEnabled(tool)` |
| `apps/console/src/app/(console)/tools/page.tsx` | `tool.dangerous` → `isDangerous(tool)` |
| `apps/console/src/lib/tool-settings.ts` | `saveDefaultToolSettings` 内の `tool.defaultEnabled` → `isDefaultEnabled(tool)` |

### Step 4: Go Server の tools.go に annotations を追加

Go 側の Tool 構造体に annotations を追加。MCP protocol で `get_module_schema` がクライアントに返す際に annotations を含める。

**変更ファイル:**

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/types.go` | `ToolAnnotations` 構造体追加、`Tool` に `Annotations` フィールド追加 |
| `apps/server/internal/modules/*/tools.go` | 各モジュールのツール定義に annotations を追加 |

```go
type ToolAnnotations struct {
    ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
    DestructiveHint *bool `json:"destructiveHint,omitempty"`
    IdempotentHint  *bool `json:"idempotentHint,omitempty"`
    OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

type Tool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    InputSchema json.RawMessage `json:"inputSchema"`
    Annotations *ToolAnnotations `json:"annotations,omitempty"`
}
```

---

## 全モジュール annotations 一覧

### 分類ルール

| 操作 | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list / get / search / query | true | false | - | false |
| create | false | false | false | false |
| update | false | false | true | false |
| delete | false | **true** | true | false |

### Google Calendar

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_calendars | true | - | - | false |
| get_calendar | true | - | - | false |
| list_events | true | - | - | false |
| get_event | true | - | - | false |
| create_event | false | false | false | false |
| update_event | false | false | true | false |
| delete_event | false | true | true | false |
| quick_add | false | false | false | false |

### Microsoft To Do

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_lists | true | - | - | false |
| get_list | true | - | - | false |
| create_list | false | false | false | false |
| update_list | false | false | true | false |
| delete_list | false | true | true | false |
| list_tasks | true | - | - | false |
| get_task | true | - | - | false |
| create_task | false | false | false | false |
| update_task | false | false | true | false |
| complete_task | false | false | true | false |
| delete_task | false | true | true | false |

### Notion

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| search | true | - | - | false |
| get_page | true | - | - | false |
| get_page_content | true | - | - | false |
| create_page | false | false | false | false |
| update_page | false | false | true | false |
| get_database | true | - | - | false |
| query_database | true | - | - | false |
| append_blocks | false | false | false | false |
| delete_block | false | true | true | false |
| list_comments | true | - | - | false |
| add_comment | false | false | false | false |
| list_users | true | - | - | false |
| get_user | true | - | - | false |
| get_bot_user | true | - | - | false |

### GitHub

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| get_user | true | - | - | false |
| list_repos | true | - | - | false |
| get_repo | true | - | - | false |
| list_branches | true | - | - | false |
| list_commits | true | - | - | false |
| get_file_content | true | - | - | false |
| list_issues | true | - | - | false |
| get_issue | true | - | - | false |
| create_issue | false | false | false | false |
| update_issue | false | false | true | false |
| add_issue_comment | false | false | false | false |
| list_prs | true | - | - | false |
| get_pr | true | - | - | false |
| create_pr | false | false | false | false |
| list_pr_files | true | - | - | false |
| search_repos | true | - | - | false |
| search_code | true | - | - | false |
| search_issues | true | - | - | false |
| list_workflows | true | - | - | false |
| list_workflow_runs | true | - | - | false |

### Jira

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| get_myself | true | - | - | false |
| list_projects | true | - | - | false |
| get_project | true | - | - | false |
| search | true | - | - | false |
| get_issue | true | - | - | false |
| create_issue | false | false | false | false |
| update_issue | false | false | true | false |
| get_transitions | true | - | - | false |
| transition_issue | false | false | true | false |
| get_comments | true | - | - | false |
| add_comment | false | false | false | false |

### Confluence

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_spaces | true | - | - | false |
| get_space | true | - | - | false |
| get_pages | true | - | - | false |
| get_page | true | - | - | false |
| create_page | false | false | false | false |
| update_page | false | false | true | false |
| delete_page | false | true | true | false |
| search | true | - | - | false |
| get_page_comments | true | - | - | false |
| add_page_comment | false | false | false | false |
| get_page_labels | true | - | - | false |
| add_page_label | false | false | false | false |

### Supabase

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_organizations | true | - | - | false |
| list_projects | true | - | - | false |
| get_project | true | - | - | false |
| list_tables | true | - | - | false |
| run_query | false | true | false | false |
| list_migrations | true | - | - | false |
| apply_migration | false | true | false | false |
| get_logs | true | - | - | false |
| get_security_advisors | true | - | - | false |
| get_performance_advisors | true | - | - | false |
| get_project_url | true | - | - | false |
| get_api_keys | true | - | - | false |
| generate_typescript_types | true | - | - | false |
| list_edge_functions | true | - | - | false |
| get_edge_function | true | - | - | false |
| list_storage_buckets | true | - | - | false |
| get_storage_config | true | - | - | false |

### Airtable

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_bases | true | - | - | false |
| describe | true | - | - | false |
| query | true | - | - | false |
| get_record | true | - | - | false |
| create | false | false | false | false |
| update | false | false | true | false |
| delete | false | true | true | false |
| search_records | true | - | - | false |
| aggregate_records | true | - | - | false |
| create_table | false | false | false | false |
| update_table | false | false | true | false |

---

## 実装順序

| Step | 内容 | 依存 |
|------|------|------|
| 1 | tools.json を annotations 形式に変換 | - |
| 2 | module-data.ts の型定義 + ヘルパー関数 | Step 1 |
| 3 | tools/page.tsx の参照を更新 | Step 2 |
| 4 | tool-settings.ts の saveDefaultToolSettings を更新 | Step 2 |
| 5 | Go Server の types.go に ToolAnnotations 追加 | - |
| 6 | Go Server の各モジュール tools.go に annotations 追加 | Step 5 |

Step 1-4 は Console 側、Step 5-6 は Go Server 側。並行して進められる。

---

## 変更ファイル一覧

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/lib/tools.json` | `defaultEnabled`/`dangerous` 削除、`annotations` 追加 |
| `apps/console/src/lib/module-data.ts` | `ToolAnnotations` 型追加、`isDefaultEnabled`/`isDangerous` 関数追加 |
| `apps/console/src/app/(console)/tools/page.tsx` | `defaultEnabled`/`dangerous` 参照を関数呼び出しに変更 |
| `apps/console/src/lib/tool-settings.ts` | `saveDefaultToolSettings` を annotations ベースに変更 |
| `apps/server/internal/modules/types.go` | `ToolAnnotations` 構造体追加 |
| `apps/server/internal/modules/*/tools.go` | 各モジュールに annotations 追加 |

---

## 備考

- 管理画面は不要。annotations はツール固有の性質でありハードコードが適切
- `defaultEnabled` の概念は annotations から導出するため、DB化も不要
- OpenAI / Anthropic 共に同一の MCP annotations 仕様に準拠しており、将来のディレクトリ申請にそのまま使える
- 参考: [report-tool-annotations.md](./report-tool-annotations.md)
