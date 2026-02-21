package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/jsonrpc"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/observability"
)

type Handler struct {
	userStore *broker.UserBroker
}

func NewHandler(userStore *broker.UserBroker) *Handler {
	return &Handler{
		userStore: userStore,
	}
}

// ProcessRequest routes a JSON-RPC request to the appropriate handler.
// Called by the transport middleware.
func (h *Handler) ProcessRequest(ctx context.Context, req *jsonrpc.Request) (interface{}, *jsonrpc.Error) {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(req), nil
	case "initialized":
		return nil, nil
	case "tools/list":
		return h.handleToolsList(ctx)
	case "tools/call":
		return h.handleToolCall(ctx, req)
	case "prompts/list":
		return h.handlePromptsList(ctx)
	case "prompts/get":
		return h.handlePromptsGet(ctx, req)
	default:
		return nil, &jsonrpc.Error{Code: MethodNotFound, Message: "Method not found"}
	}
}

func (h *Handler) handleInitialize(req *jsonrpc.Request) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: "2025-03-26",
		Capabilities: ServerCapabilities{
			Tools:   &ToolsCapability{},
			Prompts: &PromptsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "mcpist",
			Version: "0.1.0",
		},
	}
}

func (h *Handler) handlePromptsList(ctx context.Context) (*PromptsListResult, *jsonrpc.Error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: "auth context missing"}
	}

	prompts, err := h.userStore.GetUserPrompts(authCtx.UserID)
	if err != nil {
		log.Printf("Failed to get user prompts: %v", err)
		return &PromptsListResult{Prompts: []PromptInfo{}}, nil
	}

	var promptInfos []PromptInfo
	for _, p := range prompts {
		desc := ""
		if p.Description != nil {
			desc = *p.Description
		}
		promptInfos = append(promptInfos, PromptInfo{
			Name:        p.Name,
			Description: desc,
		})
	}

	if promptInfos == nil {
		promptInfos = []PromptInfo{}
	}

	return &PromptsListResult{Prompts: promptInfos}, nil
}

func (h *Handler) handlePromptsGet(ctx context.Context, req *jsonrpc.Request) (*PromptsGetResult, *jsonrpc.Error) {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "Invalid params"}
	}

	var params PromptsGetParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "Invalid params structure"}
	}

	if params.Name == "" {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "name is required"}
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: "auth context missing"}
	}

	prompt, err := h.userStore.GetUserPromptByName(authCtx.UserID, params.Name)
	if err != nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: fmt.Sprintf("failed to get prompt: %v", err)}
	}

	if prompt == nil {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: fmt.Sprintf("prompt not found: %s", params.Name)}
	}

	// Check if prompt is enabled
	if !prompt.Enabled {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: fmt.Sprintf("prompt is disabled: %s", params.Name)}
	}

	return &PromptsGetResult{
		Messages: []PromptMessage{
			{
				Role: "user",
				Content: PromptContent{
					Type: "text",
					Text: prompt.Content,
				},
			},
		},
	}, nil
}

func (h *Handler) handleToolsList(ctx context.Context) (*ToolsListResult, *jsonrpc.Error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: "auth context missing"}
	}
	return &ToolsListResult{Tools: modules.DynamicMetaTools(authCtx.EnabledModules)}, nil
}

func (h *Handler) handleToolCall(ctx context.Context, req *jsonrpc.Request) (*ToolCallResult, *jsonrpc.Error) {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "Invalid params"}
	}

	var params ToolCallParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "Invalid params structure"}
	}

	switch params.Name {
	case "get_module_schema":
		return h.handleGetModuleSchema(ctx, params.Arguments)
	case "run":
		return h.handleRun(ctx, params.Arguments)
	case "batch":
		return h.handleBatch(ctx, params.Arguments)
	default:
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: fmt.Sprintf("Unknown tool: %s", params.Name)}
	}
}

func (h *Handler) handleGetModuleSchema(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *jsonrpc.Error) {
	var moduleNames []string

	switch v := args["module"].(type) {
	case []interface{}:
		// Array input: ["notion", "jira"]
		for _, item := range v {
			name, ok := item.(string)
			if !ok {
				return nil, &jsonrpc.Error{Code: InvalidParams, Message: "module array must contain strings"}
			}
			moduleNames = append(moduleNames, name)
		}
	case string:
		// Backward compatible: single string "notion"
		moduleNames = []string{v}
	default:
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "module must be a string or array of strings"}
	}

	if len(moduleNames) == 0 {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "module must not be empty"}
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: "auth context missing"}
	}

	result, err := modules.GetModuleSchemas(moduleNames, authCtx.EnabledModules, authCtx.EnabledTools, authCtx.ModuleDescriptions)
	if err != nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: err.Error()}
	}

	return result, nil
}

func (h *Handler) handleRun(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *jsonrpc.Error) {
	moduleName, ok := args["module"].(string)
	if !ok {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "module must be a string"}
	}

	toolName, ok := args["tool"].(string)
	if !ok {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "tool must be a string"}
	}

	params, _ := args["params"].(map[string]interface{})
	if params == nil {
		params = make(map[string]interface{})
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: "auth context missing"}
	}

	if err := authCtx.CanAccessTool(moduleName, toolName, 1); err != nil {
		observability.LogSecurityEvent(middleware.GetRequestID(ctx), authCtx.UserID, "run_permission_denied", map[string]any{
			"module": moduleName,
			"tool":   toolName,
			"reason": err.Error(),
		})
		return nil, authErrorToRPC(err)
	}

	result, err := modules.Run(ctx, moduleName, toolName, params)
	if err != nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: err.Error()}
	}

	// Apply compact format unless format=json is explicitly requested
	if !result.IsError {
		if f, _ := params["format"].(string); f != "json" {
			result.Content[0].Text = modules.ApplyCompact(moduleName, toolName, result.Content[0].Text)
		}
	}

	// Record usage asynchronously (fire-and-forget)
	h.userStore.RecordUsage(
		authCtx.UserID,
		"run",
		middleware.GetRequestID(ctx),
		[]broker.ToolDetail{{Module: moduleName, Tool: toolName}},
	)

	return result, nil
}

func (h *Handler) handleBatch(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *jsonrpc.Error) {
	commands, ok := args["commands"].(string)
	if !ok {
		return nil, &jsonrpc.Error{Code: InvalidParams, Message: "commands must be a string"}
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: "auth context missing"}
	}

	// All-or-Nothing: pre-check all commands before execution
	requestID := middleware.GetRequestID(ctx)
	if mcpErr := checkBatchPermissions(requestID, authCtx, commands); mcpErr != nil {
		return nil, mcpErr
	}

	batchResult, err := modules.Batch(ctx, commands)
	if err != nil {
		return nil, &jsonrpc.Error{Code: InternalError, Message: err.Error()}
	}

	// Record usage asynchronously for all successful tool executions
	if len(batchResult.SuccessfulTasks) > 0 {
		details := make([]broker.ToolDetail, len(batchResult.SuccessfulTasks))
		for i, task := range batchResult.SuccessfulTasks {
			details[i] = broker.ToolDetail{
				TaskID: task.TaskID,
				Module: task.Module,
				Tool:   task.Tool,
			}
		}

		h.userStore.RecordUsage(
			authCtx.UserID,
			"batch",
			requestID,
			details,
		)
	}

	return batchResult.Result, nil
}

// checkBatchPermissions parses batch JSONL and checks all tools are permitted.
// Returns an MCP error if any tool is denied (All-or-Nothing).
// Client receives a vague message; server log records specific denied tools (Layer 3: Detection).
func checkBatchPermissions(requestID string, authCtx *middleware.AuthContext, commands string) *jsonrpc.Error {
	lines := strings.Split(strings.TrimSpace(commands), "\n")

	var deniedDetails []string // for server log only
	toolCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var cmd struct {
			Module string `json:"module"`
			Tool   string `json:"tool"`
		}
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			continue // JSON parse errors are handled later by modules.Batch
		}
		if cmd.Module == "" || cmd.Tool == "" {
			continue // validation handled by modules.Batch
		}

		toolCount++

		// creditCost=0: skip credit check (credits are consumed after execution)
		if err := authCtx.CanAccessTool(cmd.Module, cmd.Tool, 0); err != nil {
			if authErr, ok := err.(*middleware.AuthError); ok {
				deniedDetails = append(deniedDetails, fmt.Sprintf("%s:%s(%s)", cmd.Module, cmd.Tool, authErr.Code))
			} else {
				deniedDetails = append(deniedDetails, fmt.Sprintf("%s:%s", cmd.Module, cmd.Tool))
			}
		}
	}

	// Batch size limit
	const maxBatchSize = 10
	if toolCount > maxBatchSize {
		return &jsonrpc.Error{
			Code:    InvalidParams,
			Message: fmt.Sprintf("batch too large: %d commands (max %d)", toolCount, maxBatchSize),
		}
	}

	if len(deniedDetails) > 0 {
		// Layer 3: Detection log (server-side only, not exposed to client)
		observability.LogSecurityEvent(requestID, authCtx.UserID, "batch_permission_denied", map[string]any{
			"denied_tools": deniedDetails,
		})

		return &jsonrpc.Error{
			Code:    ErrPermissionDenied,
			Message: "batch rejected: one or more tools are not permitted",
		}
	}

	// Daily usage limit check
	if !authCtx.WithinDailyLimit(toolCount) {
		consoleURL := os.Getenv("CONSOLE_URL")
		upgradeURL := ""
		if consoleURL != "" {
			upgradeURL = fmt.Sprintf(" Upgrade your plan at: %s/plan", consoleURL)
		}
		return &jsonrpc.Error{
			Code:    ErrUsageLimitExceeded,
			Message: fmt.Sprintf("Daily usage limit exceeded. Used: %d, Limit: %d.%s", authCtx.DailyUsed, authCtx.DailyLimit, upgradeURL),
		}
	}

	return nil
}

// authErrorToRPC maps middleware.AuthError to the appropriate JSON-RPC error code.
func authErrorToRPC(err error) *jsonrpc.Error {
	authErr, ok := err.(*middleware.AuthError)
	if !ok {
		return &jsonrpc.Error{Code: InternalError, Message: err.Error()}
	}
	switch authErr.Code {
	case "USAGE_LIMIT_EXCEEDED":
		return &jsonrpc.Error{Code: ErrUsageLimitExceeded, Message: authErr.Message}
	case "MODULE_NOT_ENABLED", "TOOL_DISABLED":
		return &jsonrpc.Error{Code: ErrPermissionDenied, Message: authErr.Message}
	default:
		return &jsonrpc.Error{Code: InternalError, Message: authErr.Message}
	}
}
