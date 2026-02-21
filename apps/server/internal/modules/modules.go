package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/observability"
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
// Tool Filtering
// =============================================================================

// filterTools returns tools that are enabled for a given module (whitelist approach).
// If enabledTools is nil (no auth context), all tools are returned.
func filterTools(moduleName string, tools []Tool, enabledTools map[string][]string) []Tool {
	if enabledTools == nil {
		return tools
	}
	enabled, ok := enabledTools[moduleName]
	if !ok {
		return nil // No tools enabled for this module
	}
	enabledSet := make(map[string]bool, len(enabled))
	for _, t := range enabled {
		enabledSet[t] = true
	}
	var filtered []Tool
	for _, tool := range tools {
		// Check both tool.ID (new format: module:tool_name) and tool.Name (legacy)
		if enabledSet[tool.ID] || enabledSet[moduleName+":"+tool.Name] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// availableModuleNames returns module names that are enabled and registered in the server.
// If enabledModules is nil (no auth context), all registered modules are returned.
func availableModuleNames(enabledModules []string) []string {
	if enabledModules == nil {
		return ListModules()
	}
	var available []string
	for _, name := range enabledModules {
		// Only include if module is registered in the server
		if _, ok := registry[name]; ok {
			available = append(available, name)
		}
	}
	return available
}

// =============================================================================
// Meta Tools
// =============================================================================

// DynamicMetaTools returns meta tools with dynamic module lists based on user's enabled modules.
// If enabledModules is nil, all modules are listed.
func DynamicMetaTools(enabledModules []string) []Tool {
	available := availableModuleNames(enabledModules)
	moduleList := strings.Join(available, ", ")

	// Build module description lines for run tool
	var moduleLines []string
	for _, name := range available {
		m, ok := registry[name]
		if !ok {
			continue
		}
		moduleLines = append(moduleLines, fmt.Sprintf("- %s: %s", name, m.Description()))
	}
	moduleDesc := strings.Join(moduleLines, "\n")

	getSchemaDesc := "Get tool definitions for modules. Important: Call only once per module per session. Schemas are cached in conversation history, so use run directly for subsequent calls to the same module."
	getSchemaModuleDesc := fmt.Sprintf("Array of module names (e.g. [\"notion\", \"jira\"]). Available: %s", moduleList)
	runDesc := fmt.Sprintf(`Execute a single module tool.

[Available Modules]
%s

[Usage]
1. get_module_schema(module) to check available tools and parameters
2. run(module, tool, params) to execute

[Response Format]
Results are returned in compact format (CSV/MD) by default. For full JSON response, add format: "json" to params.`, moduleDesc)
	batchDesc := `Execute multiple tools in batch (JSONL format, with dependency and parallel execution support).

[Fields]
- id (required): Task identifier
- module (required): Module name
- tool (required): Tool name
- params: Parameters
- after: Dependency task ID array (waits for these to complete before executing)
- output: If true, includes result in response (default: compact format)

[Response Format]
Tasks with output: true return compact format (CSV/MD) by default. For full JSON response, add format: "json" to params.

[Variable References] Access via JSONPath: ${id.results[index].field}

[Example 1: Parallel Fetch]
{"id":"tasks","module":"microsoft_todo","tool":"list_tasks","params":{"listId":"AQMk..."},"output":true}
{"id":"daily","module":"microsoft_todo","tool":"list_tasks","params":{"listId":"AQMk..."},"output":true}

[Example 2: Chained Processing]
{"id":"search","module":"notion","tool":"search","params":{"query":"design"}}
{"id":"page","module":"notion","tool":"get_page_content","params":{"page_id":"${search.results[0].id}"},"after":["search"],"output":true}

[Limits]
- Maximum 10 commands per batch

[Execution Rules]
- No after -> parallel execution via goroutines
- With after -> executes after dependent tasks complete
- Circular dependency -> error
- Dependent task failure -> dependents are skipped`
	batchCommandsDesc := "Commands in JSONL format"

	return []Tool{
		{
			Name:        "get_module_schema",
			Description: getSchemaDesc,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"module": {
						Type:        "array",
						Description: getSchemaModuleDesc,
						Items:       &Property{Type: "string"},
					},
				},
				Required: []string{"module"},
			},
		},
		{
			Name:        "run",
			Description: runDesc,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"module": {
						Type:        "string",
						Description: "Module name",
					},
					"tool": {
						Type:        "string",
						Description: "Tool name",
					},
					"params": {
						Type:        "object",
						Description: "Tool parameters",
					},
				},
				Required: []string{"module", "tool"},
			},
		},
		{
			Name:        "batch",
			Description: batchDesc,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"commands": {
						Type:        "string",
						Description: batchCommandsDesc,
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
	}

	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}

	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
	}, nil
}

// GetModuleSchemas returns schemas for multiple modules with tool filtering.
// Modules with zero enabled tools are treated as unknown (not exposed to client).
// Unknown module names are reported as errors in the response but don't prevent other modules from returning.
// moduleDescriptions is a map of module_name -> custom description to prepend to schema output.
func GetModuleSchemas(moduleNames []string, enabledModules []string, enabledTools map[string][]string, moduleDescriptions map[string]string) (*ToolCallResult, error) {
	var schemas []ModuleSchema
	var errors []string
	var userNotes []string

	for _, name := range moduleNames {
		m, ok := registry[name]
		if !ok {
			errors = append(errors, fmt.Sprintf("Unknown module: %s", name))
			continue
		}

		tools := filterTools(name, m.Tools(), enabledTools)
		if len(tools) == 0 {
			errors = append(errors, fmt.Sprintf("Unknown module: %s", name))
			continue
		}

		// Collect module-level custom description for header
		if customDesc, ok := moduleDescriptions[name]; ok && customDesc != "" {
			userNotes = append(userNotes, fmt.Sprintf("[%s] %s", name, customDesc))
		}

		// Set English description for each tool
		enTools := make([]Tool, len(tools))
		for i, t := range tools {
			enTools[i] = t
			enTools[i].Description = t.Descriptions["en-US"]
			enTools[i].Descriptions = nil // Don't expose all languages to client
		}

		schemas = append(schemas, ModuleSchema{
			Module:      m.Name(),
			Description: m.Description(),
			APIVersion:  m.APIVersion(),
			Tools:       enTools,
			Resources:   m.Resources(),
		})
	}

	// If all modules were unknown or had no enabled tools, return error with available list
	if len(schemas) == 0 && len(errors) > 0 {
		available := availableModuleNames(enabledModules)
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: strings.Join(errors, "; ") + fmt.Sprintf(". Available: %v", available)}},
			IsError: true,
		}, nil
	}

	jsonBytes, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return nil, err
	}

	// Build output text with user notes at the beginning
	var textParts []string
	if len(errors) > 0 {
		textParts = append(textParts, fmt.Sprintf("⚠ %s", strings.Join(errors, "; ")))
	}
	if len(userNotes) > 0 {
		textParts = append(textParts, "[User Note]\n"+strings.Join(userNotes, "\n"))
	}
	textParts = append(textParts, string(jsonBytes))

	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: strings.Join(textParts, "\n\n")}},
	}, nil
}

// =============================================================================
// Tool Execution
// =============================================================================

// toolTimeout is the maximum duration for a single tool execution.
const toolTimeout = 30 * time.Second

// Run executes a single tool in a module
func Run(ctx context.Context, moduleName, toolName string, params map[string]interface{}) (*ToolCallResult, error) {
	start := time.Now()

	m, ok := registry[moduleName]
	if !ok {
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Unknown module: %s", moduleName)}},
			IsError: true,
		}, nil
	}

	// Validate params against tool's InputSchema
	if tool, found := findTool(m.Tools(), toolName); found {
		validated, err := ValidateParams(tool.InputSchema, params)
		if err != nil {
			return &ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		params = validated
	}

	// Apply timeout to prevent external API calls from hanging indefinitely
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	result, err := m.ExecuteTool(ctx, toolName, params)
	durationMs := time.Since(start).Milliseconds()
	requestID := middleware.GetRequestID(ctx)
	authCtx := middleware.GetAuthContext(ctx)
	userID := ""
	if authCtx != nil {
		userID = authCtx.UserID
	}

	if err != nil {
		errMsg := err.Error()
		if ctx.Err() == context.DeadlineExceeded {
			errMsg = fmt.Sprintf("Request to %s timed out after %s. The external service did not respond in time.", moduleName, toolTimeout)
		}
		observability.LogToolCall(requestID, userID, moduleName, toolName, durationMs, "error", errMsg)
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: errMsg}},
			IsError: true,
		}, nil
	}

	observability.LogToolCall(requestID, userID, moduleName, toolName, durationMs, "success", "")
	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
	}, nil
}

// ApplyCompact converts a JSON result to compact format (CSV/MD) for a given module and tool.
// Returns the original JSON if the module has no CompactConverter.
func ApplyCompact(moduleName, toolName, jsonResult string) string {
	m, ok := registry[moduleName]
	if !ok {
		return jsonResult
	}
	if converter, ok := m.(CompactConverter); ok {
		return converter.ToCompact(toolName, jsonResult)
	}
	return jsonResult
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
	Output    bool                   `json:"output,omitempty"`     // Include result in response
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

// SuccessfulTask represents a successfully executed task for credit tracking
type SuccessfulTask struct {
	TaskID string
	Module string
	Tool   string
}

// BatchResult contains the tool call result and success count for credit consumption
type BatchResult struct {
	Result          *ToolCallResult
	SuccessCount    int
	SuccessfulTasks []SuccessfulTask // Individual task info for per-tool credit tracking
}

// Batch executes multiple tools from JSONL input with DAG-based parallel execution
// Returns the result and the count of successful tool executions for credit consumption
func Batch(ctx context.Context, commands string) (*BatchResult, error) {
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
			return &BatchResult{
				Result: &ToolCallResult{
					Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("JSON parse error: %v", err)}},
					IsError: true,
				},
				SuccessCount: 0,
			}, nil
		}

		if cmd.ID == "" {
			return &BatchResult{
				Result: &ToolCallResult{
					Content: []ContentBlock{{Type: "text", Text: "id field is required for all commands"}},
					IsError: true,
				},
				SuccessCount: 0,
			}, nil
		}

		if _, exists := tasks[cmd.ID]; exists {
			return &BatchResult{
				Result: &ToolCallResult{
					Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("duplicate id: %s", cmd.ID)}},
					IsError: true,
				},
				SuccessCount: 0,
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
				return &BatchResult{
					Result: &ToolCallResult{
						Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("unknown dependency %s for task %s", dep, state.cmd.ID)}},
						IsError: true,
					},
					SuccessCount: 0,
				}, nil
			}
		}
	}

	// Detect circular dependencies
	if cycle := detectCycle(tasks); cycle != "" {
		return &BatchResult{
			Result: &ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("circular dependency detected: %s", cycle)}},
				IsError: true,
			},
			SuccessCount: 0,
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

	// Build response and count successful executions
	response := BatchResponse{
		Results: make(map[string]string),
		Errors:  make(map[string]string),
	}
	successCount := 0
	var successfulTasks []SuccessfulTask

	for _, id := range order {
		state := tasks[id]
		if state.err != nil {
			response.Errors[id] = state.err.Error()
		} else if state.skipped {
			response.Errors[id] = "skipped due to dependency failure"
		} else {
			// Successful execution
			successCount++
			successfulTasks = append(successfulTasks, SuccessfulTask{
				TaskID: id,
				Module: state.cmd.Module,
				Tool:   state.cmd.Tool,
			})
			if state.cmd.Output {
				// output: true -> apply compact unless params.format == "json"
				f, _ := state.cmd.Params["format"].(string)
				if f == "json" {
					response.Results[id] = state.result
				} else {
					response.Results[id] = ApplyCompact(state.cmd.Module, state.cmd.Tool, state.result)
				}
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

	// Return JSON format with success count
	jsonBytes, _ := json.Marshal(response)

	return &BatchResult{
		Result: &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
		},
		SuccessCount:    successCount,
		SuccessfulTasks: successfulTasks,
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
	result, err := Run(ctx, state.cmd.Module, state.cmd.Tool, resolvedParams)
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
		// Result can be a JSON array [...] or an object with "results" key {"results": [...]}
		var results []interface{}
		if err := json.Unmarshal([]byte(resultStr), &results); err != nil {
			// Try as object with "results" key
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(resultStr), &data); err != nil {
				return match
			}
			var ok bool
			results, ok = data["results"].([]interface{})
			if !ok {
				return match
			}
		}

		if index >= len(results) {
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
