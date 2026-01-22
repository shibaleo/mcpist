package modules

import "context"

// =============================================================================
// Module Interface
// =============================================================================

// Module defines the interface that all modules must implement.
// Each module provides Tools, Resources, and Prompts (MCP 3 primitives).
type Module interface {
	// Metadata
	Name() string
	Description() string
	APIVersion() string

	// Tools - LLM executes, has side effects
	Tools() []Tool
	ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error)

	// Resources - LLM reads, no side effects
	Resources() []Resource
	ReadResource(ctx context.Context, uri string) (string, error)

	// Prompts - Templates/workflows
	Prompts() []Prompt
	GetPrompt(ctx context.Context, name string, args map[string]any) (string, error)
}

// CompactConverter provides optional compact format conversion (TOON/Markdown)
// Modules that implement this can convert their JSON output to token-efficient formats
type CompactConverter interface {
	// ToCompact converts JSON result to compact format (TOON or Markdown)
	// toolName is used to select the appropriate format for each tool
	ToCompact(toolName string, jsonResult string) string
}

// =============================================================================
// Tool Definition
// =============================================================================

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
	Dangerous   bool        `json:"dangerous,omitempty"` // Requires confirmation
}

// InputSchema defines the input parameters for a tool
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property defines a single property in the input schema
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// =============================================================================
// Resource Definition
// =============================================================================

// Resource represents an MCP resource definition
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// =============================================================================
// Prompt Definition
// =============================================================================

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// =============================================================================
// Result Types
// =============================================================================

// ToolCallResult represents the result of a tool call
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in the result
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
