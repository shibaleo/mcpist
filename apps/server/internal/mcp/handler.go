package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/observability"
	"mcpist/server/internal/broker"
)

type Handler struct {
	sessions  map[string]*Session
	mu        sync.RWMutex
	userStore *broker.UserStore
}

type Session struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
	messages chan []byte
}

func NewHandler(userStore *broker.UserStore) *Handler {
	return &Handler{
		sessions:  make(map[string]*Session),
		userStore: userStore,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleSSE(w, r)
	case http.MethodPost:
		h.handleMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create session with cryptographic random ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		http.Error(w, "failed to generate session ID", http.StatusInternalServerError)
		return
	}
	sessionID := hex.EncodeToString(idBytes)

	session := &Session{
		id:       sessionID,
		writer:   w,
		flusher:  flusher,
		done:     make(chan struct{}),
		messages: make(chan []byte, 100),
	}

	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionID)
		h.mu.Unlock()
		close(session.done)
	}()

	// Send endpoint event (MCP SSE protocol)
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp?sessionId=%s\n\n", sessionID)
	flusher.Flush()
	log.Printf("SSE connection established, session=%s", sessionID)

	// Keep connection open and send messages
	for {
		select {
		case msg := <-session.messages:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			log.Printf("SSE connection closed, session=%s", sessionID)
			return
		}
	}
}

func (h *Handler) handleMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		h.handleInlineMessage(w, r)
		return
	}

	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		h.sendToSession(session, nil, &Error{Code: ParseError, Message: "Parse error"})
		w.WriteHeader(http.StatusAccepted)
		return
	}

	log.Printf("Received request: method=%s id=%v session=%s", req.Method, req.ID, sessionID)

	result, rpcErr := h.processRequest(r.Context(), &req)
	if rpcErr != nil {
		h.sendToSession(session, req.ID, rpcErr)
	} else if req.ID != nil {
		h.sendResultToSession(session, req.ID, result)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) handleInlineMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		resp := Response{JSONRPC: "2.0", Error: &Error{Code: ParseError, Message: "Parse error"}}
		json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("Received inline request: method=%s id=%v", req.Method, req.ID)

	result, rpcErr := h.processRequest(r.Context(), &req)

	w.Header().Set("Content-Type", "application/json")
	var resp Response
	if rpcErr != nil {
		resp = Response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}
	} else {
		resp = Response{JSONRPC: "2.0", ID: req.ID, Result: result}
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) sendToSession(session *Session, id interface{}, err *Error) {
	resp := Response{JSONRPC: "2.0", ID: id, Error: err}
	data, _ := json.Marshal(resp)
	select {
	case session.messages <- data:
	default:
		log.Printf("Session message buffer full")
	}
}

func (h *Handler) sendResultToSession(session *Session, id interface{}, result interface{}) {
	resp := Response{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(resp)
	select {
	case session.messages <- data:
	default:
		log.Printf("Session message buffer full")
	}
}

func (h *Handler) processRequest(ctx context.Context, req *Request) (interface{}, *Error) {
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
		return nil, &Error{Code: MethodNotFound, Message: "Method not found"}
	}
}

func (h *Handler) handleInitialize(req *Request) *InitializeResult {
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

func (h *Handler) handlePromptsList(ctx context.Context) (*PromptsListResult, *Error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
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

func (h *Handler) handlePromptsGet(ctx context.Context, req *Request) (*PromptsGetResult, *Error) {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return nil, &Error{Code: InvalidParams, Message: "Invalid params"}
	}

	var params PromptsGetParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return nil, &Error{Code: InvalidParams, Message: "Invalid params structure"}
	}

	if params.Name == "" {
		return nil, &Error{Code: InvalidParams, Message: "name is required"}
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
	}

	prompt, err := h.userStore.GetUserPromptByName(authCtx.UserID, params.Name)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: fmt.Sprintf("failed to get prompt: %v", err)}
	}

	if prompt == nil {
		return nil, &Error{Code: InvalidParams, Message: fmt.Sprintf("prompt not found: %s", params.Name)}
	}

	// Check if prompt is enabled
	if !prompt.Enabled {
		return nil, &Error{Code: InvalidParams, Message: fmt.Sprintf("prompt is disabled: %s", params.Name)}
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

func (h *Handler) handleToolsList(ctx context.Context) (*ToolsListResult, *Error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
	}
	return &ToolsListResult{Tools: modules.DynamicMetaTools(authCtx.EnabledModules, authCtx.Language)}, nil
}

func (h *Handler) handleToolCall(ctx context.Context, req *Request) (*ToolCallResult, *Error) {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return nil, &Error{Code: InvalidParams, Message: "Invalid params"}
	}

	var params ToolCallParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return nil, &Error{Code: InvalidParams, Message: "Invalid params structure"}
	}

	switch params.Name {
	case "get_module_schema":
		return h.handleGetModuleSchema(ctx, params.Arguments)
	case "run":
		return h.handleRun(ctx, params.Arguments)
	case "batch":
		return h.handleBatch(ctx, params.Arguments)
	default:
		return nil, &Error{Code: InvalidParams, Message: fmt.Sprintf("Unknown tool: %s", params.Name)}
	}
}

func (h *Handler) handleGetModuleSchema(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error) {
	var moduleNames []string

	switch v := args["module"].(type) {
	case []interface{}:
		// Array input: ["notion", "jira"]
		for _, item := range v {
			name, ok := item.(string)
			if !ok {
				return nil, &Error{Code: InvalidParams, Message: "module array must contain strings"}
			}
			moduleNames = append(moduleNames, name)
		}
	case string:
		// Backward compatible: single string "notion"
		moduleNames = []string{v}
	default:
		return nil, &Error{Code: InvalidParams, Message: "module must be a string or array of strings"}
	}

	if len(moduleNames) == 0 {
		return nil, &Error{Code: InvalidParams, Message: "module must not be empty"}
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
	}

	result, err := modules.GetModuleSchemas(moduleNames, authCtx.EnabledModules, authCtx.EnabledTools, authCtx.Language, authCtx.ModuleDescriptions)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: err.Error()}
	}

	return result, nil
}

func (h *Handler) handleRun(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error) {
	moduleName, ok := args["module"].(string)
	if !ok {
		return nil, &Error{Code: InvalidParams, Message: "module must be a string"}
	}

	toolName, ok := args["tool"].(string)
	if !ok {
		return nil, &Error{Code: InvalidParams, Message: "tool must be a string"}
	}

	params, _ := args["params"].(map[string]interface{})
	if params == nil {
		params = make(map[string]interface{})
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
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
		return nil, &Error{Code: InternalError, Message: err.Error()}
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

func (h *Handler) handleBatch(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error) {
	commands, ok := args["commands"].(string)
	if !ok {
		return nil, &Error{Code: InvalidParams, Message: "commands must be a string"}
	}

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
	}

	// All-or-Nothing: pre-check all commands before execution
	requestID := middleware.GetRequestID(ctx)
	if mcpErr := checkBatchPermissions(requestID, authCtx, commands); mcpErr != nil {
		return nil, mcpErr
	}

	batchResult, err := modules.Batch(ctx, commands)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: err.Error()}
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
func checkBatchPermissions(requestID string, authCtx *middleware.AuthContext, commands string) *Error {
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
		return &Error{
			Code:    InvalidParams,
			Message: fmt.Sprintf("batch too large: %d commands (max %d)", toolCount, maxBatchSize),
		}
	}

	if len(deniedDetails) > 0 {
		// Layer 3: Detection log (server-side only, not exposed to client)
		observability.LogSecurityEvent(requestID, authCtx.UserID, "batch_permission_denied", map[string]any{
			"denied_tools": deniedDetails,
		})

		return &Error{
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
		return &Error{
			Code:    ErrUsageLimitExceeded,
			Message: fmt.Sprintf("Daily usage limit exceeded. Used: %d, Limit: %d.%s", authCtx.DailyUsed, authCtx.DailyLimit, upgradeURL),
		}
	}

	return nil
}

// authErrorToRPC maps middleware.AuthError to the appropriate JSON-RPC error code.
func authErrorToRPC(err error) *Error {
	authErr, ok := err.(*middleware.AuthError)
	if !ok {
		return &Error{Code: InternalError, Message: err.Error()}
	}
	switch authErr.Code {
	case "USAGE_LIMIT_EXCEEDED":
		return &Error{Code: ErrUsageLimitExceeded, Message: authErr.Message}
	case "MODULE_NOT_ENABLED", "TOOL_DISABLED":
		return &Error{Code: ErrPermissionDenied, Message: authErr.Message}
	default:
		return &Error{Code: InternalError, Message: authErr.Message}
	}
}
