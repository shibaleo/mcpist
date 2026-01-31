# DAY021 計画

## 日付

2026-02-01

---

## 概要

Sprint-006 5日目。DAY020でMCP Primitives（resources, prompts, elicitation）の調査・計画が完了。本日はGoogle Tasks モジュール実装とprompts MCP実装を開始する。

---

## DAY020 の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| database.types.ts 再生成 | RPC名変更後の型チェック通過 |
| Claude Web E2E テスト | Notion search + get_page_content 成功 |
| Liam ERD セットアップ | `pnpm erd:build`, `pnpm erd:serve` 追加 |
| MCP Primitives 調査・計画 | resources, prompts, elicitation の仕様調査 |

### 教訓（day020-review.md より）

- 設計が固まっていると、大きな変更でも復旧が早い
- 影響範囲の見積もりは、コードベースの理解度に依存する
- 緊張する作業こそ、事前の計画と段階的な実行が重要

---

## 本日のタスク

### Phase 1: Google Tasks モジュール実装（優先度：高）

| ID | タスク | 備考 |
|----|--------|------|
| D21-001 | Google Tasks API 調査 | OAuth scope、エンドポイント確認 |
| D21-002 | google_tasks モジュール作成 | `apps/server/internal/modules/google_tasks/` |
| D21-003 | ツール実装 | list_tasks, create_task, update_task, delete_task |
| D21-004 | Console OAuth 設定 | Google Tasks scope 追加 |
| D21-005 | E2E テスト | Claude Web で動作確認 |

### Phase 2: prompts MCP 実装（優先度：高）

| ID | タスク | 備考 |
|----|--------|------|
| D21-006 | `get_user_prompts` RPC 作成 | API Server 用（`p_user_id` 引数） |
| D21-007 | `get_user_prompt` RPC 作成 | API Server 用（単体取得） |
| D21-008 | handler.go に prompts/list 追加 | MCP プロトコル対応 |
| D21-009 | handler.go に prompts/get 追加 | 引数展開処理含む |
| D21-010 | Capability 宣言更新 | `prompts: { listChanged: false }` |

### Phase 3: Console プロンプト管理 UI（stretch）

| ID | タスク | 備考 |
|----|--------|------|
| D21-011 | /prompts ページ作成 | プロンプト一覧 |
| D21-012 | プロンプト作成/編集フォーム | name, description, content, arguments |
| D21-013 | プロンプト削除機能 | 確認ダイアログ付き |

### Phase 4: 仕様書整備（時間があれば）

| ID | タスク | BL ID | 備考 |
|----|--------|-------|------|
| D21-014 | JWT `aud` チェック要件整理 | BL-011 | 実装では明示チェックなし |
| D21-015 | MCP 拡張エラーコード整理 | BL-012 | JSON-RPC 標準コードのみに更新 |

---

## Google Tasks モジュール設計

### OAuth Scope

```
https://www.googleapis.com/auth/tasks
https://www.googleapis.com/auth/tasks.readonly
```

### ツール一覧

| ツール | 説明 | HTTP メソッド |
|--------|------|---------------|
| list_task_lists | タスクリスト一覧取得 | GET /users/@me/lists |
| list_tasks | タスク一覧取得 | GET /lists/{tasklist}/tasks |
| get_task | タスク詳細取得 | GET /lists/{tasklist}/tasks/{task} |
| create_task | タスク作成 | POST /lists/{tasklist}/tasks |
| update_task | タスク更新 | PATCH /lists/{tasklist}/tasks/{task} |
| delete_task | タスク削除 | DELETE /lists/{tasklist}/tasks/{task} |
| complete_task | タスク完了 | PATCH (status: completed) |

### ファイル構成

```
apps/server/internal/modules/google_tasks/
├── module.go      # Module インターフェース実装
├── tools.go       # ツール定義（多言語）
├── client.go      # Google Tasks API クライアント
└── types.go       # 型定義
```

---

## prompts MCP 設計

### RPC 設計

| RPC | 呼び出し元 | 引数 | 戻り値 |
|-----|------------|------|--------|
| `get_user_prompts` | API Server | `p_user_id UUID` | `TABLE (id, name, description, content, arguments, created_at)` |
| `get_user_prompt` | API Server | `p_user_id UUID, p_prompt_id UUID` | 単一行 |

### handler.go 追加メソッド

```go
// prompts/list ハンドラ
func (h *Handler) handlePromptsList(ctx context.Context) (*PromptsListResult, *Error)

// prompts/get ハンドラ
func (h *Handler) handlePromptsGet(ctx context.Context, args map[string]interface{}) (*PromptsGetResult, *Error)
```

### レスポンス形式

**prompts/list:**
```json
{
  "prompts": [
    {
      "name": "daily_tasks",
      "description": "Get daily tasks from MS Todo and Google Tasks",
      "arguments": [
        { "name": "date", "description": "Target date (YYYY-MM-DD)", "required": false }
      ]
    }
  ]
}
```

**prompts/get:**
```json
{
  "description": "Daily tasks prompt",
  "messages": [
    {
      "role": "user",
      "content": { "type": "text", "text": "Show my tasks for today..." }
    }
  ]
}
```

---

## 完了条件

- [ ] Google Tasks モジュールが動作（list_tasks, create_task）
- [ ] prompts/list, prompts/get が MCP プロトコルで動作
- [ ] Claude Web で prompts 一覧が表示される
- [ ] （stretch）Console でプロンプト作成が可能

---

## タイムライン

| 時間帯 | タスク |
|--------|--------|
| 午前 | Phase 1: Google Tasks モジュール (D21-001〜005) |
| 午後前半 | Phase 2: prompts MCP 実装 (D21-006〜010) |
| 午後後半 | Phase 3: Console UI or Phase 4: 仕様書整備 |

---

## 参考

- [day020-backlog.md](./day020-backlog.md) - バックログ
- [day020-plan-mcp-primitives.md](./day020-plan-mcp-primitives.md) - MCP Primitives 計画
- [day020-review.md](./day020-review.md) - 前日レビュー
- [Google Tasks API](https://developers.google.com/tasks/reference/rest)
- [MCP Specification - Prompts](https://modelcontextprotocol.io/specification/2025-11-25/server/prompts)
