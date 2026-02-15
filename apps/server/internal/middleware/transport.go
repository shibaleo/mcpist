package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"mcpist/server/internal/jsonrpc"
)

// RequestProcessor processes JSON-RPC requests.
// Implemented by the MCP handler.
type RequestProcessor interface {
	ProcessRequest(ctx context.Context, req *jsonrpc.Request) (interface{}, *jsonrpc.Error)
}

// session represents an SSE connection session.
type session struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
	messages chan []byte
}

// transport manages SSE/Inline transport for MCP.
type transport struct {
	processor RequestProcessor
	sessions  map[string]*session
	mu        sync.RWMutex
}

// Transport creates an http.Handler that manages SSE and Inline JSON-RPC transport.
// It delegates request processing to the given RequestProcessor.
func Transport(processor RequestProcessor) http.Handler {
	return &transport{
		processor: processor,
		sessions:  make(map[string]*session),
	}
}

func (t *transport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		t.handleSSE(w, r)
	case http.MethodPost:
		t.handleMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (t *transport) handleSSE(w http.ResponseWriter, r *http.Request) {
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

	s := &session{
		id:       sessionID,
		writer:   w,
		flusher:  flusher,
		done:     make(chan struct{}),
		messages: make(chan []byte, 100),
	}

	t.mu.Lock()
	t.sessions[sessionID] = s
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.sessions, sessionID)
		t.mu.Unlock()
		close(s.done)
	}()

	// Send endpoint event (MCP SSE protocol)
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp?sessionId=%s\n\n", sessionID)
	flusher.Flush()
	log.Printf("SSE connection established, session=%s", sessionID)

	// Keep connection open and send messages
	for {
		select {
		case msg := <-s.messages:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			log.Printf("SSE connection closed, session=%s", sessionID)
			return
		}
	}
}

func (t *transport) handleMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		t.handleInlineMessage(w, r)
		return
	}

	t.mu.RLock()
	s, ok := t.sessions[sessionID]
	t.mu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req jsonrpc.Request
	if err := json.Unmarshal(body, &req); err != nil {
		t.sendToSession(s, nil, &jsonrpc.Error{Code: jsonrpc.ParseError, Message: "Parse error"})
		w.WriteHeader(http.StatusAccepted)
		return
	}

	log.Printf("Received request: method=%s id=%v session=%s", req.Method, req.ID, sessionID)

	result, rpcErr := t.processor.ProcessRequest(r.Context(), &req)
	if rpcErr != nil {
		t.sendToSession(s, req.ID, rpcErr)
	} else if req.ID != nil {
		t.sendResultToSession(s, req.ID, result)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (t *transport) handleInlineMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req jsonrpc.Request
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		resp := jsonrpc.Response{JSONRPC: "2.0", Error: &jsonrpc.Error{Code: jsonrpc.ParseError, Message: "Parse error"}}
		json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("Received inline request: method=%s id=%v", req.Method, req.ID)

	result, rpcErr := t.processor.ProcessRequest(r.Context(), &req)

	w.Header().Set("Content-Type", "application/json")
	var resp jsonrpc.Response
	if rpcErr != nil {
		resp = jsonrpc.Response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}
	} else {
		resp = jsonrpc.Response{JSONRPC: "2.0", ID: req.ID, Result: result}
	}
	json.NewEncoder(w).Encode(resp)
}

func (t *transport) sendToSession(s *session, id interface{}, err *jsonrpc.Error) {
	resp := jsonrpc.Response{JSONRPC: "2.0", ID: id, Error: err}
	data, _ := json.Marshal(resp)
	select {
	case s.messages <- data:
	default:
		log.Printf("Session message buffer full")
	}
}

func (t *transport) sendResultToSession(s *session, id interface{}, result interface{}) {
	resp := jsonrpc.Response{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(resp)
	select {
	case s.messages <- data:
	default:
		log.Printf("Session message buffer full")
	}
}
