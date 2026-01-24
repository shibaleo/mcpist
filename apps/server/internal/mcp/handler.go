package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"mcpist/server/internal/entitlement"
	"mcpist/server/internal/modules"
)

type Handler struct {
	sessions         map[string]*Session
	mu               sync.RWMutex
	entitlementStore *entitlement.Store
}

type Session struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
	messages chan []byte
}

func NewHandler(store *entitlement.Store) *Handler {
	return &Handler{
		sessions:         make(map[string]*Session),
		entitlementStore: store,
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
		return h.handleToolsList(), nil
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

func (h *Handler) handleToolsList() *ToolsListResult {
	// Return only meta tools (lazy loading)
	return &ToolsListResult{Tools: modules.MetaTools()}
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
		return h.handleGetModuleSchema(params.Arguments)
	case "call":
		return h.handleCall(ctx, params.Arguments)
	case "batch":
		return h.handleBatch(ctx, params.Arguments)
	default:
		return nil, &Error{Code: InvalidParams, Message: fmt.Sprintf("Unknown tool: %s", params.Name)}
	}
}

func (h *Handler) handleGetModuleSchema(args map[string]interface{}) (*ToolCallResult, *Error) {
	moduleName, ok := args["module"].(string)
	if !ok {
		return nil, &Error{Code: InvalidParams, Message: "module must be a string"}
	}

	result, err := modules.GetModuleSchema(moduleName)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: err.Error()}
	}

	return result, nil
}

func (h *Handler) handleCall(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error) {
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

	// Get tool cost (default: 1 credit per call)
	creditCost := 1

	// Check module/tool access authorization
	authCtx := entitlement.GetAuthContext(ctx)
	if authCtx != nil {
		if err := authCtx.CanAccessTool(moduleName, toolName, creditCost); err != nil {
			authErr, ok := err.(*entitlement.AuthError)
			if ok {
				return nil, &Error{Code: InvalidRequest, Message: authErr.Message}
			}
			return nil, &Error{Code: InvalidRequest, Message: err.Error()}
		}
	}

	result, err := modules.Call(ctx, moduleName, toolName, params)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: err.Error()}
	}

	// Consume credits after successful call
	if authCtx != nil {
		// Generate a unique request ID for idempotency
		requestID := generateRequestID()
		consumeResult, err := h.entitlementStore.ConsumeCredit(
			authCtx.UserID,
			moduleName,
			toolName,
			creditCost,
			requestID,
			nil, // no task_id for single calls
		)
		if err != nil {
			log.Printf("Failed to consume credits: %v", err)
		} else if !consumeResult.Success {
			log.Printf("Credit consumption failed: %s", consumeResult.Error)
		}
	}

	return result, nil
}

func (h *Handler) handleBatch(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error) {
	commands, ok := args["commands"].(string)
	if !ok {
		return nil, &Error{Code: InvalidParams, Message: "commands must be a string"}
	}

	// Note: Batch authorization and credit consumption is handled per-command inside modules.Batch
	// We pass context so individual commands can be checked and credits consumed

	result, err := modules.Batch(ctx, commands)
	if err != nil {
		return nil, &Error{Code: InternalError, Message: err.Error()}
	}

	return result, nil
}

// generateRequestID generates a unique request ID for idempotency
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%1000000)
}
