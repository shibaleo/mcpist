# Plan: get_module_schema の複数モジュール対応 + ユーザー設定によるツールフィルタリング

作成日: 2026-01-27

---

## 変更概要

`get_module_schema` メタツールを以下の3点で改善する:

1. `module` 引数を配列（string array）に変更し、複数モジュールを1回で取得可能にする
2. レスポンスに含まれるツールを、ユーザーのツール設定（`DisabledTools`）でフィルタリングする
3. `tools/list` のレスポンスを動的化し、接続済み＆有効ツール>0のモジュールのみdescriptionに含める

---

## 現状のコード構造

### handler.go (mcp/handler.go)

```
handleToolCall() → switch params.Name:
  "get_module_schema" → handleGetModuleSchema(args)  // 単一module、authなし
  "run"               → handleRun(ctx, args)          // authCtx あり
  "batch"             → handleBatch(ctx, args)        // authCtx あり

handleToolsList() → MetaTools() // 静的定義をそのまま返す
```

### modules.go (modules/modules.go)

```
MetaTools()         → 静的な[]Tool（get_module_schema/run/batch）
GetModuleSchema()   → 単一モジュール名→全ツール含むModuleSchemaを返す
```

### authz.go (middleware/authz.go)

```
AuthContext {
    EnabledModules []string            // 接続済みモジュール
    DisabledTools  map[string][]string // module → disabled tool names
}
```

---

## 変更ファイルと変更内容

### 1. `apps/server/internal/modules/types.go` ✅ 実施済み

Property構造体に `Items` フィールドを追加:

```go
type Property struct {
    Type        string    `json:"type"`
    Description string    `json:"description"`
    Items       *Property `json:"items,omitempty"`
}
```

### 2. `apps/server/internal/modules/modules.go`

#### MetaTools() → DynamicMetaTools(enabledModules []string, disabledTools map[string][]string)

- 引数にenabledModulesとdisabledToolsを追加
- `get_module_schema` の `module` プロパティ型を `array` (items: string) に変更
- Description内のモジュール一覧を、enabledModulesかつ有効ツール>0のモジュールのみに動的生成
- `run` のDescriptionも同様に動的化
- 引数がnilの場合（認証なし）は全モジュール表示（後方互換）

**get_module_schema InputSchema変更:**
```go
Properties: map[string]Property{
    "module": {
        Type:        "array",
        Description: "モジュール名の配列 (例: [\"notion\", \"jira\"])",
        Items:       &Property{Type: "string"},
    },
},
```

#### GetModuleSchema() → GetModuleSchemas()

```go
func GetModuleSchemas(moduleNames []string, disabledTools map[string][]string) (*ToolCallResult, error)
```

- 複数モジュール名をループ
- 各モジュールのToolsをdisabledToolsでフィルタリング
- `[]ModuleSchema` のJSON配列として返す
- 未知のモジュール名はエラーメッセージ付きでスキップ（部分成功）

### 3. `apps/server/internal/mcp/handler.go`

#### handleToolsList() の動的化

```go
func (h *Handler) handleToolsList(ctx context.Context) *ToolsListResult {
    authCtx := middleware.GetAuthContext(ctx)
    if authCtx != nil {
        return &ToolsListResult{Tools: modules.DynamicMetaTools(authCtx.EnabledModules, authCtx.DisabledTools)}
    }
    return &ToolsListResult{Tools: modules.DynamicMetaTools(nil, nil)}
}
```

- processRequest()からctxを渡す

#### handleGetModuleSchema() の変更

```go
func (h *Handler) handleGetModuleSchema(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error)
```

- `args["module"]` を `[]interface{}` (配列) として取得
- 後方互換: 単一文字列が渡された場合も `[]string{name}` に変換
- authContextから`DisabledTools`を取得
- `modules.GetModuleSchemas(names, disabledTools)` を呼び出す

#### processRequest() の変更

```go
case "tools/list":
    return h.handleToolsList(ctx), nil  // ctxを渡す
```

#### handleToolCall() の変更

```go
case "get_module_schema":
    return h.handleGetModuleSchema(ctx, params.Arguments)  // ctxを渡す
```

---

## レスポンス形式

常に`[]ModuleSchema`の配列形式で返す:

```json
[
  {
    "module": "notion",
    "description": "ページ・データベース操作",
    "api_version": "v0",
    "tools": [ ... (enabled tools only) ... ]
  },
  {
    "module": "jira",
    "description": "Issue/Project操作",
    "api_version": "v0",
    "tools": [ ... ]
  }
]
```

---

## ツールフィルタリングのロジック

```go
func filterTools(moduleName string, tools []Tool, disabledTools map[string][]string) []Tool {
    if disabledTools == nil {
        return tools // 認証なし: 全ツール返却
    }
    disabled, ok := disabledTools[moduleName]
    if !ok {
        return tools // このモジュールに無効ツールなし
    }
    disabledSet := make(map[string]bool, len(disabled))
    for _, t := range disabled {
        disabledSet[t] = true
    }
    var filtered []Tool
    for _, tool := range tools {
        if !disabledSet[tool.Name] {
            filtered = append(filtered, tool)
        }
    }
    return filtered
}
```

---

## tools/list 動的モジュール一覧のロジック

```go
func availableModuleNames(enabledModules []string, disabledTools map[string][]string) []string {
    if enabledModules == nil {
        return ListModules() // 全モジュール
    }
    var available []string
    for _, name := range enabledModules {
        m, ok := registry[name]
        if !ok { continue }
        // 有効ツールが1つ以上あるか確認
        filtered := filterTools(name, m.Tools(), disabledTools)
        if len(filtered) > 0 {
            available = append(available, name)
        }
    }
    return available
}
```

---

## 実装進捗

| ステップ | 状態 |
|----------|------|
| types.go: Property.Items 追加 | ✅ 完了 |
| modules.go: DynamicMetaTools() 実装 | ⬜ 未着手 |
| modules.go: GetModuleSchemas() 実装 | ⬜ 未着手 |
| modules.go: filterTools(), availableModuleNames() 実装 | ⬜ 未着手 |
| handler.go: handleToolsList(ctx) 動的化 | ⬜ 未着手 |
| handler.go: handleGetModuleSchema(ctx, args) 配列対応 | ⬜ 未着手 |
| handler.go: processRequest/handleToolCall の引数変更 | ⬜ 未着手 |
| go build 確認 | ⬜ 未着手 |

---

## 検証方法

1. `go build ./...` でビルド成功を確認
2. MCPサーバー経由で `get_module_schema` を呼び出し:
   - 配列: `{"module": ["notion", "jira"]}` → 両方のスキーマが返る
   - 後方互換: `{"module": "notion"}` → 動作する
   - 無効ツールがフィルタリングされていることを確認
3. `tools/list` で返るdescription内のモジュール一覧が接続状況を反映していることを確認
