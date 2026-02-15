package mcp

import (
	"mcpist/server/internal/jsonrpc"
	"mcpist/server/internal/modules"
)

// Re-export JSON-RPC types for backward compatibility within this package
type Request = jsonrpc.Request
type Response = jsonrpc.Response
type Error = jsonrpc.Error

// Re-export JSON-RPC error codes
const (
	ParseError            = jsonrpc.ParseError
	InvalidRequest        = jsonrpc.InvalidRequest
	MethodNotFound        = jsonrpc.MethodNotFound
	InvalidParams         = jsonrpc.InvalidParams
	InternalError         = jsonrpc.InternalError
	ErrPermissionDenied   = jsonrpc.ErrPermissionDenied
	ErrUsageLimitExceeded = jsonrpc.ErrUsageLimitExceeded
)

// MCP Protocol Types
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct{}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools   *ToolsCapability   `json:"tools,omitempty"`
	Prompts *PromptsCapability `json:"prompts,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolsListResult struct {
	Tools []modules.Tool `json:"tools"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Use modules types
type ToolCallResult = modules.ToolCallResult
type ContentBlock = modules.ContentBlock

// =============================================================================
// Prompts Types (MCP 2025-11-25)
// =============================================================================

// PromptsListResult represents the result of prompts/list
type PromptsListResult struct {
	Prompts    []PromptInfo `json:"prompts"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

// PromptInfo represents a prompt in the list
type PromptInfo struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptsGetParams represents the parameters for prompts/get
type PromptsGetParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PromptsGetResult represents the result of prompts/get
type PromptsGetResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in the prompt result
type PromptMessage struct {
	Role    string        `json:"role"`
	Content PromptContent `json:"content"`
}

// PromptContent represents the content of a prompt message
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
