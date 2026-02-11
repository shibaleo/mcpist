# Asana Module

## Status

- **Status**: Implemented (ogen)
- **Date**: 2026-02-11
- **API Version**: 1.0
- **Client**: ogen generated (`pkg/asanaapi/`)

## Endpoint Catalog

### User

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `get_me` | GET | `/users/me` | (none) |

### Workspaces

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_workspaces` | GET | `/workspaces` | (none) |
| `get_workspace` | GET | `/workspaces/{workspace_gid}` | **workspace_gid** |

### Projects

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_projects` | GET | `/workspaces/{workspace_gid}/projects` | **workspace_gid**, team_gid, archived |
| | GET | `/teams/{team_gid}/projects` | (team_gid 指定時に自動選択) |
| `get_project` | GET | `/projects/{project_gid}` | **project_gid** |
| `create_project` | POST | `/projects` | **name**, workspace_gid, team_gid, notes, color, default_view, due_on |
| `update_project` | PUT | `/projects/{project_gid}` | **project_gid**, name, notes, color, default_view, due_on, archived |
| `delete_project` | DELETE | `/projects/{project_gid}` | **project_gid** |

### Sections

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_sections` | GET | `/projects/{project_gid}/sections` | **project_gid** |
| `create_section` | POST | `/projects/{project_gid}/sections` | **project_gid**, **name** |

### Tasks

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_tasks` | GET | `/projects/{project_gid}/tasks` | project_gid, section_gid, assignee_gid, workspace_gid, completed |
| | GET | `/sections/{section_gid}/tasks` | (section_gid 指定時に自動選択) |
| | GET | `/tasks` | (assignee_gid + workspace_gid 指定時に自動選択) |
| `get_task` | GET | `/tasks/{task_gid}` | **task_gid** |
| `create_task` | POST | `/tasks` | **name**, workspace_gid, projects, notes, html_notes, due_on, due_at, start_on, assignee_gid, tags, parent_gid, section_gid |
| `update_task` | PUT | `/tasks/{task_gid}` | **task_gid**, name, notes, html_notes, due_on, due_at, start_on, completed, assignee_gid |
| `complete_task` | PUT | `/tasks/{task_gid}` | **task_gid** |
| `delete_task` | DELETE | `/tasks/{task_gid}` | **task_gid** |

### Subtasks

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_subtasks` | GET | `/tasks/{task_gid}/subtasks` | **task_gid** |
| `create_subtask` | POST | `/tasks/{task_gid}/subtasks` | **parent_gid**, **name**, notes, due_on, assignee_gid |

### Stories (Comments)

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_stories` | GET | `/tasks/{task_gid}/stories` | **task_gid** |
| `add_comment` | POST | `/tasks/{task_gid}/stories` | **task_gid**, **text** |

### Tags

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_tags` | GET | `/workspaces/{workspace_gid}/tags` | **workspace_gid** |
| `create_tag` | POST | `/tags` | **workspace_gid**, **name**, color |

### Search

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `search_tasks` | GET | `/workspaces/{workspace_gid}/tasks/search` | **workspace_gid**, text, completed, is_subtask, assignee_gid, projects_gid, due_on_before, due_on_after, sort_by, sort_ascending |

## Summary

- **Total**: 23 tools (GET: 12, POST: 7, PUT: 3, DELETE: 2)
- **ogen operations**: 26 (list_projects=2, list_tasks=3, addTaskToSection=1 の分岐含む)

## Tool Classification

| 分類 | ツール | 説明 |
|------|--------|------|
| **ogen 単体** | get_me, list_workspaces, get_workspace, get_project, list_sections, get_task, list_subtasks, list_stories, list_tags, search_tasks | ogen client の 1 メソッド呼出 → `toJSON(res.Data)` |
| **ogen 条件分岐** | list_projects, list_tasks | パラメータに応じて異なる ogen メソッドに分岐 |
| **ogen POST/PUT** | create_project, update_project, create_section, create_task, update_task, complete_task, create_subtask, add_comment, create_tag | リクエストボディ構築 → ogen メソッド呼出 → `toJSON(res.Data)` |
| **ogen DELETE** | delete_project, delete_task | 戻り値無視 → `{"success":true,"message":"..."}` |
| **2段階処理** | create_task (section_gid 指定時) | `CreateTask` → `AddTaskToSection` の 2 API 呼出 |

## Response Schemas

subset spec で定義しているスキーマ:

| Schema | Used by |
|--------|---------|
| User | getMe |
| Workspace | listWorkspaces, getWorkspace |
| Project | listProjectsByWorkspace, listProjectsByTeam, getProject, createProject, updateProject |
| Section | listSections, createSection |
| Task | listTasksByProject, listTasksBySection, listTasksByAssignee, getTask, createTask, updateTask, searchTasks |
| Story | listStories, createStory |
| Tag | listTags, createTag |
| EmptyDataResponse | deleteProject, deleteTask |
| CreateProjectRequest, CreateProjectRequestData | createProject |
| UpdateProjectRequest, UpdateProjectRequestData | updateProject |
| CreateSectionRequest, CreateSectionRequestData | createSection |
| CreateTaskRequest, CreateTaskRequestData | createTask |
| UpdateTaskRequest, UpdateTaskRequestData | updateTask |
| CreateSubtaskRequest, CreateSubtaskRequestData | createSubtask |
| CreateStoryRequest, CreateStoryRequestData | createStory |
| CreateTagRequest, CreateTagRequestData | createTag |
| AddTaskToSectionRequest, AddTaskToSectionRequestData | addTaskToSection |

## Notes

### Data Envelope パターン

Asana API の全レスポンスは `{"data": ...}` でラップされる。subset spec で `*Response` / `*ListResponse` ラッパースキーマを定義:

```yaml
TaskResponse:
  type: object
  properties:
    data:
      $ref: '#/components/schemas/Task'

TaskListResponse:
  type: object
  properties:
    data:
      type: array
      items:
        $ref: '#/components/schemas/Task'
```

ハンドラは `res.Data` でアンラップし `toJSON(res.Data)` で返す。

### リクエストボディも Data Envelope

POST/PUT リクエストも `{"data": {...}}` でラップが必要:

```go
res, err := c.CreateTask(ctx, &gen.CreateTaskRequest{Data: reqData})
```

### 条件付きエンドポイント

`list_projects` と `list_tasks` はパラメータに応じて異なる ogen operation を呼び分ける:

- **list_projects**: `team_gid` が指定されていれば `ListProjectsByTeam`、そうでなければ `ListProjectsByWorkspace`
- **list_tasks**: `section_gid` → `ListTasksBySection`、`project_gid` → `ListTasksByProject`、`assignee_gid + workspace_gid` → `ListTasksByAssignee`

### create_task の2段階処理

`section_gid` が指定された場合、タスク作成後に `AddTaskToSection` を追加呼出:

```go
res, err := c.CreateTask(ctx, &gen.CreateTaskRequest{Data: reqData})
// ...
if hasSection && sectionGID != "" {
    if taskData, ok := res.Data.Get(); ok {
        if taskGID, ok := taskData.Gid.Get(); ok && taskGID != "" {
            addReqData := gen.AddTaskToSectionRequestData{}
            addReqData.Task.SetTo(taskGID)
            c.AddTaskToSection(ctx, &gen.AddTaskToSectionRequest{Data: addReqData}, ...)
        }
    }
}
```

### complete_task は updateTask のラッパー

`complete_task` ツールは `updateTask` operation を `completed: true` で呼ぶだけの便利ツール。

### search_tasks のドット表記パラメータ

Asana の検索 API は `assignee.any`, `projects.any`, `due_on.before`, `due_on.after` というドット区切りのクエリパラメータを使う。ogen がこれを自動サニタイズし `AssigneeAny`, `ProjectsAny`, `DueOnBefore`, `DueOnAfter` として生成する。

### search_tasks は Premium プラン必要

Asana の `/workspaces/{workspace_gid}/tasks/search` エンドポイントは Premium プラン以上が必要。Free プランでは HTTP 402 が返る。これはコード側の問題ではなく API の制約。

### OAuth トークンリフレッシュ

Asana は OAuth 2.0 認証を使用。`getCredentials()` でトークン取得 → 有効期限チェック → 必要に応じて `refreshToken()` で自動更新。更新したトークンは `newOgenClient()` に渡す。

### Nullable フィールド

`due_on`, `due_at`, `start_on`, `assignee`, `parent`, `color`, `completed_at` は nullable。subset spec で `nullable: true` を指定し、ogen が `OptNilString` / `OptNilTaskAssignee` 等の型を生成。
