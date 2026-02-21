package notion

import (
	"context"
	"fmt"
	"log"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/notionapi"
	gen "mcpist/server/pkg/notionapi/gen"
)

const (
	notionVersion = "2022-06-28"
)

// NotionModule implements the Module interface for Notion API
type NotionModule struct{}

// New creates a new NotionModule instance
func New() *NotionModule {
	return &NotionModule{}
}

// Module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Notion API - Page, Database, Block, Comment, and User operations",
	"ja-JP": "Notion API - ページ、データベース、ブロック、コメント、ユーザー操作",
}

// Name returns the module name
func (m *NotionModule) Name() string {
	return "notion"
}

// Descriptions returns the module descriptions in all languages
func (m *NotionModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *NotionModule) Description() string {
	return moduleDescriptions["en-US"]
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
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *NotionModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources
func (m *NotionModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI
func (m *NotionModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// =============================================================================
// Client / Auth
// =============================================================================

// getCredentials retrieves credentials from Vault via RPC for the given user
// and refreshes the token if needed (for OAuth2)
func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[notion] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "notion")
	if err != nil {
		log.Printf("[notion] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

// newOgenClient creates a new ogen-generated Notion API client
func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return notionapi.NewClient(creds.AccessToken, notionVersion)
}
