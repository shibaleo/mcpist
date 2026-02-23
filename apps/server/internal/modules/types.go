package modules

import "context"

// =============================================================================
// Localization (used by Console UI for multilingual tool descriptions)
// =============================================================================

// LocalizedText holds multilingual text.
// key: BCP47 language code (en-US, ja-JP)
type LocalizedText map[string]string

// =============================================================================
// Module Interface
// =============================================================================

// Module defines the interface that all modules must implement.
// Each module provides Tools and Resources (MCP primitives).
// Note: Prompts are managed at user level, not per-module.
type Module interface {
	// Metadata
	Name() string
	Description() string                 // English description (for MCP schema)
	Descriptions() LocalizedText         // Multilingual descriptions (synced to DB for Console UI)
	APIVersion() string

	// Tools - LLM executes, has side effects
	Tools() []Tool
	ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error)

	// Resources - LLM reads, no side effects
	Resources() []Resource
	ReadResource(ctx context.Context, uri string) (string, error)
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

// ToolAnnotations describes the tool's behavior hints per MCP spec (2025-11-25).
type ToolAnnotations struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

// Helper to create *bool for annotation fields
func boolPtr(v bool) *bool { return &v }

// Pre-built annotation sets for common tool patterns
var (
	// AnnotateReadOnly: list, get, search, query tools
	AnnotateReadOnly = &ToolAnnotations{
		ReadOnlyHint:  boolPtr(true),
		OpenWorldHint: boolPtr(false),
	}
	// AnnotateCreate: create, add, append tools (non-idempotent write)
	AnnotateCreate = &ToolAnnotations{
		ReadOnlyHint:    boolPtr(false),
		DestructiveHint: boolPtr(false),
		IdempotentHint:  boolPtr(false),
		OpenWorldHint:   boolPtr(false),
	}
	// AnnotateUpdate: update, transition tools (idempotent write)
	AnnotateUpdate = &ToolAnnotations{
		ReadOnlyHint:    boolPtr(false),
		DestructiveHint: boolPtr(false),
		IdempotentHint:  boolPtr(true),
		OpenWorldHint:   boolPtr(false),
	}
	// AnnotateDelete: delete tools (destructive, idempotent)
	AnnotateDelete = &ToolAnnotations{
		ReadOnlyHint:    boolPtr(false),
		DestructiveHint: boolPtr(true),
		IdempotentHint:  boolPtr(true),
		OpenWorldHint:   boolPtr(false),
	}
	// AnnotateDestructive: run_query, apply_migration (destructive, non-idempotent)
	AnnotateDestructive = &ToolAnnotations{
		ReadOnlyHint:    boolPtr(false),
		DestructiveHint: boolPtr(true),
		IdempotentHint:  boolPtr(false),
		OpenWorldHint:   boolPtr(false),
	}
)

// Tool represents an MCP tool definition
type Tool struct {
	ID           string           `json:"id,omitempty"`            // Stable ID (e.g., "notion:search")
	Name         string           `json:"name"`                    // Display name / execution key
	Description  string           `json:"description"`             // Runtime description (after language selection)
	Descriptions LocalizedText    `json:"descriptions,omitempty"`  // Multilingual descriptions (for export)
	InputSchema  InputSchema      `json:"inputSchema"`
	Annotations  *ToolAnnotations `json:"annotations,omitempty"`
}

// InputSchema defines the input parameters for a tool
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property defines a single property in the input schema
type Property struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Items       *Property `json:"items,omitempty"`
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
