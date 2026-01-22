package notion

import (
	"context"
	"fmt"

	"mcpist/server/internal/modules"
)

// NotionModule implements the Module interface for Notion API
type NotionModule struct{}

// New creates a new NotionModule instance
func New() *NotionModule {
	return &NotionModule{}
}

// Name returns the module name
func (m *NotionModule) Name() string {
	return "notion"
}

// Description returns the module description
func (m *NotionModule) Description() string {
	return "Notion API - ページ・データベース・ブロック操作"
}

// APIVersion returns the Notion API version
func (m *NotionModule) APIVersion() string {
	return notionVersion
}

// Tools returns all available tools
func (m *NotionModule) Tools() []modules.Tool {
	return toolDefinitions()
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *NotionModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	// Execute and get JSON response
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format (TOON or Markdown)
// Implements modules.CompactConverter interface
func (m *NotionModule) ToCompact(toolName string, jsonResult string) string {
	return ToTOON(toolName, jsonResult)
}

// Resources returns all available resources
func (m *NotionModule) Resources() []modules.Resource {
	return resourceDefinitions()
}

// ReadResource reads a resource by URI
func (m *NotionModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return readResource(ctx, uri)
}

// Prompts returns all available prompts
func (m *NotionModule) Prompts() []modules.Prompt {
	return promptDefinitions()
}

// GetPrompt generates a prompt with arguments
func (m *NotionModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	return getPrompt(ctx, name, args)
}
