package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/observability"
	"mcpist/server/internal/store"
)

type Handler struct {
	sessions  map[string]*Session
	mu        sync.RWMutex
	userStore *store.UserStore
}

type Session struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
	messages chan []byte
}

func NewHandler(userStore *store.UserStore) *Handler {
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

	// Create session
	sessionID := fmt.Sprintf("%p", r)

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
	default:
		return nil, &Error{Code: MethodNotFound, Message: "Method not found"}
	}
}

func (h *Handler) handleInitialize(req *Request) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: "2025-03-26",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "mcpist",
			Version: "0.1.0",
		},
	}
}

func (h *Handler) handleToolsList(ctx context.Context) (*ToolsListResult, *Error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
	}
	return &ToolsListResult{Tools: modules.DynamicMetaTools(authCtx.EnabledModules, authCtx.DisabledTools)}, nil
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

	result, err := modules.GetModuleSchemas(moduleNames, authCtx.DisabledTools)
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

	// Currently 1 credit per tool. To support per-tool pricing,
	// replace with e.g. modules.CreditCost(moduleName, toolName).
	creditCost := 1

	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, &Error{Code: InternalError, Message: "auth context missing"}
	}

	if err := authCtx.CanAccessTool(moduleName, toolName, creditCost); err != nil {
		authErr, ok := err.(*middleware.AuthError)
		if ok {
			return nil, &Error{Code: InvalidRequest, Message: authErr.Message}
		}
		return nil, &Error{Code: InvalidRequest, Message: err.Error()}
	}

	result, err := modules.Run(ctx, moduleName, toolName, params)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: err.Error()}
	}

	// Consume credits after successful call
	consumeResult, err := h.userStore.ConsumeCredit(
		authCtx.UserID,
		moduleName,
		toolName,
		creditCost,
		middleware.GetRequestID(ctx),
		nil, // no task_id for single calls
	)
	if err != nil {
		log.Printf("Failed to consume credits: %v", err)
	} else if !consumeResult.Success {
		log.Printf("Credit consumption failed: %s", consumeResult.Error)
	}

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

	// Consume credits for successful tool executions
	if batchResult.SuccessCount > 0 {
		creditCost := batchResult.SuccessCount // 1 credit per successful tool call
		consumeResult, err := h.userStore.ConsumeCredit(
			authCtx.UserID,
			"batch",
			"batch",
			creditCost,
			requestID,
			nil,
		)
		if err != nil {
			log.Printf("Failed to consume credits for batch: %v", err)
		} else if !consumeResult.Success {
			log.Printf("Credit consumption failed for batch: %s", consumeResult.Error)
		}
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

	if len(deniedDetails) > 0 {
		// Layer 3: Detection log (server-side only, not exposed to client)
		observability.LogSecurityEvent(requestID, authCtx.UserID, "batch_permission_denied", map[string]any{
			"denied_tools": deniedDetails,
		})

		return &Error{
			Code:    InvalidRequest,
			Message: "batch rejected: one or more tools are not permitted",
		}
	}

	// Credit balance check: currently 1 credit per tool.
	// To support per-tool pricing, replace toolCount with a sum of
	// per-tool costs (e.g. modules.CreditCost(module, tool)) accumulated in the loop above.
	if authCtx.TotalCredits() < toolCount {
		return &Error{
			Code:    InvalidRequest,
			Message: fmt.Sprintf("Insufficient credits. Required: %d, Available: %d", toolCount, authCtx.TotalCredits()),
		}
	}

	return nil
}

