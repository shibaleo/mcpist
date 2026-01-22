package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Registry
// =============================================================================

// registry holds all registered modules
var registry = make(map[string]Module)

// RegisterModule adds a module to the registry
func RegisterModule(m Module) {
	registry[m.Name()] = m
}

// GetModule returns a module by name
func GetModule(name string) (Module, bool) {
	m, ok := registry[name]
	return m, ok
}

// ListModules returns all registered module names
func ListModules() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// Meta Tools
// =============================================================================

// MetaTools returns the three meta tools for lazy loading
func MetaTools() []Tool {
	return []Tool{
		{
			Name:        "get_module_schema",
			Description: "モジュールのツール定義を取得。重要: 各モジュールにつき1セッション1回のみ呼び出すこと。スキーマは会話履歴にキャッシュされるため、同一モジュールへの2回目以降の呼び出しはcallを直接使用すること。",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"module": {
						Type:        "string",
						Description: "モジュール名(notion, github, jira, confluence, supabase, google_calendar, microsoft_todo, rag)",
					},
				},
				Required: []string{"module"},
			},
		},
		{
			Name: "call",
			Description: `モジュールのツールを単発実行。

【利用可能モジュール】
- notion: ページ・データベース操作
- github: リポジトリ、Issue、PR操作
- jira: Issue/Project操作
- confluence: Wiki操作
- supabase: DB操作、ストレージ
- google_calendar: 予定の取得・作成
- microsoft_todo: タスク管理
- rag: ドキュメント検索

【使い方】
1. get_module_schema(module) でツール一覧とパラメータを確認
2. call(module, tool, params) で実行`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"module": {
						Type:        "string",
						Description: "モジュール名",
					},
					"tool": {
						Type:        "string",
						Description: "ツール名",
					},
					"params": {
						Type:        "object",
						Description: "ツールパラメータ",
					},
				},
				Required: []string{"module", "tool"},
			},
		},
		{
			Name: "batch",
			Description: `複数ツールを一括実行（JSONL形式、依存関係・並列実行対応）。

【フィールド】
- id (必須): タスク識別子
- module (必須): モジュール名
- tool (必須): ツール名
- params: パラメータ
- after: 依存タスクID配列（これらの完了を待ってから実行）
- output: trueでTOON/MD形式で結果を返却
- raw_output: trueでJSON形式で結果を返却（outputより優先）

【変数参照】${id.results[index].field} 形式でJSONPathアクセス

【例1: 並列取得】
{"id":"tasks","module":"microsoft_todo","tool":"list_tasks","params":{"listId":"AQMk..."},"output":true}
{"id":"daily","module":"microsoft_todo","tool":"list_tasks","params":{"listId":"AQMk..."},"output":true}

【例2: 連鎖処理】
{"id":"search","module":"notion","tool":"search","params":{"query":"設計"}}
{"id":"page","module":"notion","tool":"get_page_content","params":{"page_id":"${search.results[0].id}"},"after":["search"],"output":true}

【実行ルール】
- afterなし → goroutineで並列実行
- afterあり → 依存タスク完了後に実行
- 循環依存 → エラー
- 依存タスク失敗 → 依存先もスキップ`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"commands": {
						Type:        "string",
						Description: "JSONL形式のコマンド列",
					},
				},
				Required: []string{"commands"},
			},
		},
	}
}

// =============================================================================
// Schema Response
// =============================================================================

// ModuleSchema represents the schema response for get_module_schema
type ModuleSchema struct {
	Module      string     `json:"module"`
	Description string     `json:"description"`
	APIVersion  string     `json:"api_version"`
	Tools       []Tool     `json:"tools"`
	Resources   []Resource `json:"resources,omitempty"`
	Prompts     []Prompt   `json:"prompts,omitempty"`
}

// GetModuleSchema returns the schema for a module
func GetModuleSchema(moduleName string) (*ToolCallResult, error) {
	m, ok := registry[moduleName]
	if !ok {
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Unknown module: %s. Available: %v", moduleName, ListModules())}},
			IsError: true,
		}, nil
	}

	schema := ModuleSchema{
		Module:      m.Name(),
		Description: m.Description(),
		APIVersion:  m.APIVersion(),
		Tools:       m.Tools(),
		Resources:   m.Resources(),
		Prompts:     m.Prompts(),
	}

	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}

	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
	}, nil
}

// =============================================================================
// Tool Execution
// =============================================================================

// Call executes a single tool in a module
func Call(ctx context.Context, moduleName, toolName string, params map[string]interface{}) (*ToolCallResult, error) {
	start := time.Now()

	m, ok := registry[moduleName]
	if !ok {
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Unknown module: %s", moduleName)}},
			IsError: true,
		}, nil
	}

	result, err := m.ExecuteTool(ctx, toolName, params)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		log.Printf("[%s.%s] error (%dms): %s", moduleName, toolName, durationMs, err.Error())
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	log.Printf("[%s.%s] success (%dms)", moduleName, toolName, durationMs)
	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
	}, nil
}

// =============================================================================
// Batch Execution (DAG-based parallel execution)
// =============================================================================

// BatchCommand represents a single command in batch execution
type BatchCommand struct {
	ID        string                 `json:"id"`                   // Task identifier (required)
	Module    string                 `json:"module"`               // Module name (required)
	Tool      string                 `json:"tool"`                 // Tool name (required)
	Params    map[string]interface{} `json:"params,omitempty"`     // Tool parameters
	After     []string               `json:"after,omitempty"`      // Dependency task IDs
	Output    bool                   `json:"output,omitempty"`     // Include compact result (TOON/MD)
	RawOutput bool                   `json:"raw_output,omitempty"` // Include raw JSON result (overrides output)
}

// BatchResponse represents the batch execution response
type BatchResponse struct {
	Results map[string]string `json:"results"`          // ID -> result (for output:true tasks)
	Errors  map[string]string `json:"errors,omitempty"` // ID -> error message
}

// taskState holds execution state for a task
type taskState struct {
	cmd     BatchCommand
	result  string
	err     error
	done    chan struct{}
	skipped bool
}

// Batch executes multiple tools from JSONL input with DAG-based parallel execution
func Batch(ctx context.Context, commands string) (*ToolCallResult, error) {
	// Parse commands
	lines := strings.Split(strings.TrimSpace(commands), "\n")
	tasks := make(map[string]*taskState)
	order := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var cmd BatchCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			return &ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("JSON parse error: %v", err)}},
				IsError: true,
			}, nil
		}

		if cmd.ID == "" {
			return &ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: "id field is required for all commands"}},
				IsError: true,
			}, nil
		}

		if _, exists := tasks[cmd.ID]; exists {
			return &ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("duplicate id: %s", cmd.ID)}},
				IsError: true,
			}, nil
		}

		tasks[cmd.ID] = &taskState{
			cmd:  cmd,
			done: make(chan struct{}),
		}
		order = append(order, cmd.ID)
	}

	// Validate dependencies exist
	for _, state := range tasks {
		for _, dep := range state.cmd.After {
			if _, exists := tasks[dep]; !exists {
				return &ToolCallResult{
					Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("unknown dependency %s for task %s", dep, state.cmd.ID)}},
					IsError: true,
				}, nil
			}
		}
	}

	// Detect circular dependencies
	if cycle := detectCycle(tasks); cycle != "" {
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("circular dependency detected: %s", cycle)}},
			IsError: true,
		}, nil
	}

	// Execute tasks with goroutines
	var wg sync.WaitGroup
	resultStore := &sync.Map{} // Store results for variable substitution

	for _, id := range order {
		wg.Add(1)
		go func(taskID string) {
			defer wg.Done()
			executeTask(ctx, taskID, tasks, resultStore)
		}(id)
	}

	wg.Wait()

	// Build response
	response := BatchResponse{
		Results: make(map[string]string),
		Errors:  make(map[string]string),
	}

	for _, id := range order {
		state := tasks[id]
		if state.err != nil {
			response.Errors[id] = state.err.Error()
		} else if state.skipped {
			response.Errors[id] = "skipped due to dependency failure"
		} else if state.cmd.RawOutput {
			// raw_output: true -> return JSON as-is
			response.Results[id] = state.result
		} else if state.cmd.Output {
			// output: true -> convert to compact format (TOON/MD)
			if m, ok := registry[state.cmd.Module]; ok {
				if converter, ok := m.(CompactConverter); ok {
					response.Results[id] = converter.ToCompact(state.cmd.Tool, state.result)
				} else {
					response.Results[id] = state.result // No converter, return JSON
				}
			} else {
				response.Results[id] = state.result
			}
		}
	}

	// Clean up empty maps
	if len(response.Errors) == 0 {
		response.Errors = nil
	}
	if len(response.Results) == 0 {
		response.Results = nil
	}

	// Return JSON format
	jsonBytes, _ := json.Marshal(response)

	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
	}, nil
}

// detectCycle detects circular dependencies using DFS
func detectCycle(tasks map[string]*taskState) string {
	visited := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited
	var cyclePath []string

	var dfs func(id string) bool
	dfs = func(id string) bool {
		if visited[id] == 2 {
			return false
		}
		if visited[id] == 1 {
			// Found cycle
			cyclePath = append(cyclePath, id)
			return true
		}

		visited[id] = 1
		cyclePath = append(cyclePath, id)

		for _, dep := range tasks[id].cmd.After {
			if dfs(dep) {
				return true
			}
		}

		cyclePath = cyclePath[:len(cyclePath)-1]
		visited[id] = 2
		return false
	}

	for id := range tasks {
		cyclePath = nil
		if dfs(id) {
			return strings.Join(cyclePath, " -> ")
		}
	}
	return ""
}

// executeTask executes a single task after waiting for dependencies
func executeTask(ctx context.Context, taskID string, tasks map[string]*taskState, resultStore *sync.Map) {
	state := tasks[taskID]
	defer close(state.done)

	// Wait for dependencies
	for _, depID := range state.cmd.After {
		depState := tasks[depID]
		<-depState.done // Wait for dependency to complete

		// Check if dependency failed or was skipped
		if depState.err != nil || depState.skipped {
			state.skipped = true
			return
		}
	}

	// Resolve variable references in params
	resolvedParams := resolveVariables(state.cmd.Params, resultStore)

	// Execute the tool
	result, err := Call(ctx, state.cmd.Module, state.cmd.Tool, resolvedParams)
	if err != nil {
		state.err = err
		return
	}

	if result.IsError {
		state.err = fmt.Errorf("%s", result.Content[0].Text)
		return
	}

	state.result = result.Content[0].Text

	// Store result for variable substitution by dependent tasks
	resultStore.Store(taskID, state.result)
}

// resolveVariables replaces ${id.items[N].field} references with actual values
func resolveVariables(params map[string]interface{}, resultStore *sync.Map) map[string]interface{} {
	if params == nil {
		return nil
	}

	resolved := make(map[string]interface{})
	for key, value := range params {
		resolved[key] = resolveValue(value, resultStore)
	}
	return resolved
}

// resolveValue recursively resolves variable references in a value
func resolveValue(value interface{}, resultStore *sync.Map) interface{} {
	switch v := value.(type) {
	case string:
		return resolveStringVariables(v, resultStore)
	case map[string]interface{}:
		resolved := make(map[string]interface{})
		for k, val := range v {
			resolved[k] = resolveValue(val, resultStore)
		}
		return resolved
	case []interface{}:
		resolved := make([]interface{}, len(v))
		for i, val := range v {
			resolved[i] = resolveValue(val, resultStore)
		}
		return resolved
	default:
		return value
	}
}

// Variable reference pattern: ${taskId.results[index].field}
var varRefPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\.results\[(\d+)\]\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// resolveStringVariables resolves variable references in a string
// Format: ${taskId.results[index].field} - extracts from JSON results array
func resolveStringVariables(s string, resultStore *sync.Map) string {
	return varRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := varRefPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}

		taskID := parts[1]
		index := 0
		fmt.Sscanf(parts[2], "%d", &index)
		field := parts[3]

		// Get the result from store (always JSON format internally)
		resultVal, ok := resultStore.Load(taskID)
		if !ok {
			return match // Keep original if not found
		}

		resultStr, ok := resultVal.(string)
		if !ok {
			return match
		}

		// Parse JSON and extract value
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(resultStr), &data); err != nil {
			return match
		}

		results, ok := data["results"].([]interface{})
		if !ok || index >= len(results) {
			return match
		}

		item, ok := results[index].(map[string]interface{})
		if !ok {
			return match
		}

		val, ok := item[field]
		if !ok {
			return match
		}

		if strVal, ok := val.(string); ok {
			return strVal
		}
		return fmt.Sprintf("%v", val)
	})
}
