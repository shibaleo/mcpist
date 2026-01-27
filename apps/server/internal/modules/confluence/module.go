package confluence

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	confluenceAPIV2      = "/wiki/api/v2"
	confluenceAPIV1      = "/wiki/rest/api"
	confluenceAPIVersion = "v2"
)

var client = httpclient.New()

// ConfluenceModule implements the Module interface for Confluence API
type ConfluenceModule struct{}

// New creates a new ConfluenceModule instance
func New() *ConfluenceModule {
	return &ConfluenceModule{}
}

// Name returns the module name
func (m *ConfluenceModule) Name() string {
	return "confluence"
}

// Description returns the module description
func (m *ConfluenceModule) Description() string {
	return "Confluence API - Wiki operations (Space, Page, Search, Comment, Label)"
}

// APIVersion returns the Confluence API version
func (m *ConfluenceModule) APIVersion() string {
	return confluenceAPIVersion
}

// Tools returns all available tools
func (m *ConfluenceModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *ConfluenceModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Confluence)
func (m *ConfluenceModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *ConfluenceModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for Confluence)
func (m *ConfluenceModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *ConfluenceModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	return "", fmt.Errorf("prompts not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "confluence")
	if err != nil {
		return nil
	}
	return credentials
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Accept": "application/json",
	}

	switch creds.AuthType {
	case store.AuthTypeBasic:
		// Basic auth: username:password
		auth := base64.StdEncoding.EncodeToString([]byte(creds.Username + ":" + creds.Password))
		h["Authorization"] = "Basic " + auth
	case store.AuthTypeOAuth2:
		// Bearer token (OAuth 2.0)
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}

func baseURLV2(ctx context.Context) string {
	creds := getCredentials(ctx)
	if creds == nil {
		return ""
	}
	domain := creds.Metadata["domain"]
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%s%s", domain, confluenceAPIV2)
}

func baseURLV1(ctx context.Context) string {
	creds := getCredentials(ctx)
	if creds == nil {
		return ""
	}
	domain := creds.Metadata["domain"]
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%s%s", domain, confluenceAPIV1)
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	{
		Name:        "list_spaces",
		Description: "List all Confluence spaces accessible to the current user.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"limit":  {Type: "number", Description: "Maximum results to return. Default: 25"},
				"cursor": {Type: "string", Description: "Pagination cursor for next page"},
			},
		},
	},
	{
		Name:        "get_space",
		Description: "Get details of a specific Confluence space by ID or key.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"space_id_or_key": {Type: "string", Description: "Space ID (numeric) or key (e.g., 'MYSPACE')"},
			},
			Required: []string{"space_id_or_key"},
		},
	},
	{
		Name:        "get_pages",
		Description: "List pages in a Confluence space.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"space_id": {Type: "string", Description: "Space ID (numeric). Use get_space to get ID from key."},
				"limit":    {Type: "number", Description: "Maximum results to return. Default: 25"},
				"cursor":   {Type: "string", Description: "Pagination cursor for next page"},
			},
			Required: []string{"space_id"},
		},
	},
	{
		Name:        "get_page",
		Description: "Get a Confluence page by ID with its content.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id":     {Type: "string", Description: "Page ID"},
				"body_format": {Type: "string", Description: "Body format: storage (XHTML) or atlas_doc_format. Default: storage"},
			},
			Required: []string{"page_id"},
		},
	},
	{
		Name:        "create_page",
		Description: "Create a new Confluence page.",
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"space_id":  {Type: "string", Description: "Space ID (numeric)"},
				"title":     {Type: "string", Description: "Page title"},
				"body":      {Type: "string", Description: "Page body in storage format (XHTML)"},
				"parent_id": {Type: "string", Description: "Parent page ID for nested pages"},
			},
			Required: []string{"space_id", "title", "body"},
		},
	},
	{
		Name:        "update_page",
		Description: "Update an existing Confluence page.",
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id": {Type: "string", Description: "Page ID"},
				"title":   {Type: "string", Description: "New page title"},
				"body":    {Type: "string", Description: "New page body in storage format (XHTML)"},
				"version": {Type: "number", Description: "Current version number (must be incremented)"},
			},
			Required: []string{"page_id", "title", "body", "version"},
		},
	},
	{
		Name:        "delete_page",
		Description: "Delete a Confluence page.",
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id": {Type: "string", Description: "Page ID"},
			},
			Required: []string{"page_id"},
		},
	},
	{
		Name:        "search",
		Description: "Search Confluence content using CQL (Confluence Query Language). Example: 'type=page AND space=MYSPACE AND text~\"keyword\"'",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"cql":   {Type: "string", Description: "CQL query string"},
				"limit": {Type: "number", Description: "Maximum results to return. Default: 25"},
				"start": {Type: "number", Description: "Starting index for pagination. Default: 0"},
			},
			Required: []string{"cql"},
		},
	},
	{
		Name:        "get_page_comments",
		Description: "Get comments on a Confluence page.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id": {Type: "string", Description: "Page ID"},
				"limit":   {Type: "number", Description: "Maximum results to return. Default: 25"},
				"cursor":  {Type: "string", Description: "Pagination cursor for next page"},
			},
			Required: []string{"page_id"},
		},
	},
	{
		Name:        "add_page_comment",
		Description: "Add a comment to a Confluence page.",
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id": {Type: "string", Description: "Page ID"},
				"body":    {Type: "string", Description: "Comment body in storage format (XHTML)"},
			},
			Required: []string{"page_id", "body"},
		},
	},
	{
		Name:        "get_page_labels",
		Description: "Get labels on a Confluence page.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id": {Type: "string", Description: "Page ID"},
			},
			Required: []string{"page_id"},
		},
	},
	{
		Name:        "add_page_label",
		Description: "Add a label to a Confluence page.",
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_id": {Type: "string", Description: "Page ID"},
				"label":   {Type: "string", Description: "Label name"},
			},
			Required: []string{"page_id", "label"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_spaces":       listSpaces,
	"get_space":         getSpace,
	"get_pages":         getPages,
	"get_page":          getPage,
	"create_page":       createPage,
	"update_page":       updatePage,
	"delete_page":       deletePage,
	"search":            search,
	"get_page_comments": getPageComments,
	"add_page_comment":  addPageComment,
	"get_page_labels":   getPageLabels,
	"add_page_label":    addPageLabel,
}

// =============================================================================
// Spaces
// =============================================================================

func listSpaces(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}
	limit := 25
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query.Set("limit", fmt.Sprintf("%d", limit))
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		query.Set("cursor", cursor)
	}

	endpoint := fmt.Sprintf("%s/spaces?%s", baseURLV2(ctx), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getSpace(ctx context.Context, params map[string]any) (string, error) {
	spaceIDOrKey, _ := params["space_id_or_key"].(string)
	numericRegex := regexp.MustCompile(`^\d+$`)
	var endpoint string

	if numericRegex.MatchString(spaceIDOrKey) {
		endpoint = fmt.Sprintf("%s/spaces/%s", baseURLV2(ctx), spaceIDOrKey)
	} else {
		endpoint = fmt.Sprintf("%s/space/%s", baseURLV1(ctx), spaceIDOrKey)
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Pages
// =============================================================================

func getPages(ctx context.Context, params map[string]any) (string, error) {
	spaceID, _ := params["space_id"].(string)
	query := url.Values{}
	limit := 25
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query.Set("limit", fmt.Sprintf("%d", limit))
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		query.Set("cursor", cursor)
	}

	endpoint := fmt.Sprintf("%s/spaces/%s/pages?%s", baseURLV2(ctx), spaceID, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getPage(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	bodyFormat := "storage"
	if bf, ok := params["body_format"].(string); ok && bf != "" {
		bodyFormat = bf
	}

	endpoint := fmt.Sprintf("%s/pages/%s?body-format=%s", baseURLV2(ctx), pageID, bodyFormat)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createPage(ctx context.Context, params map[string]any) (string, error) {
	spaceID, _ := params["space_id"].(string)
	title, _ := params["title"].(string)
	body, _ := params["body"].(string)

	payload := map[string]interface{}{
		"spaceId": spaceID,
		"title":   title,
		"status":  "current",
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          body,
		},
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		payload["parentId"] = parentID
	}

	endpoint := baseURLV2(ctx) + "/pages"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), payload)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updatePage(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	title, _ := params["title"].(string)
	body, _ := params["body"].(string)
	version, _ := params["version"].(float64)

	payload := map[string]interface{}{
		"id":     pageID,
		"title":  title,
		"status": "current",
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          body,
		},
		"version": map[string]interface{}{
			"number": int(version),
		},
	}

	endpoint := fmt.Sprintf("%s/pages/%s", baseURLV2(ctx), pageID)
	respBody, err := client.DoJSON("PUT", endpoint, headers(ctx), payload)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deletePage(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	endpoint := fmt.Sprintf("%s/pages/%s", baseURLV2(ctx), pageID)
	_, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return `{"deleted": true}`, nil
}

// =============================================================================
// Search
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	cql, _ := params["cql"].(string)
	query := url.Values{}
	query.Set("cql", cql)

	limit := 25
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query.Set("limit", fmt.Sprintf("%d", limit))

	start := 0
	if s, ok := params["start"].(float64); ok {
		start = int(s)
	}
	query.Set("start", fmt.Sprintf("%d", start))

	endpoint := fmt.Sprintf("%s/search?%s", baseURLV1(ctx), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Comments
// =============================================================================

func getPageComments(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	query := url.Values{}
	limit := 25
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query.Set("limit", fmt.Sprintf("%d", limit))
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		query.Set("cursor", cursor)
	}

	endpoint := fmt.Sprintf("%s/pages/%s/footer-comments?%s", baseURLV2(ctx), pageID, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func addPageComment(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	body, _ := params["body"].(string)

	payload := map[string]interface{}{
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          body,
		},
	}

	endpoint := fmt.Sprintf("%s/pages/%s/footer-comments", baseURLV2(ctx), pageID)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), payload)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Labels
// =============================================================================

func getPageLabels(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	endpoint := fmt.Sprintf("%s/pages/%s/labels", baseURLV2(ctx), pageID)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func addPageLabel(ctx context.Context, params map[string]any) (string, error) {
	pageID, _ := params["page_id"].(string)
	label, _ := params["label"].(string)

	payload := map[string]interface{}{
		"name": label,
	}

	endpoint := fmt.Sprintf("%s/pages/%s/labels", baseURLV2(ctx), pageID)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), payload)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
