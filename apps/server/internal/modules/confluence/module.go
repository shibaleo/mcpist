package confluence

import (
	"context"
	"fmt"
	"regexp"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/confluenceapi"
	gen "mcpist/server/pkg/confluenceapi/gen"
)

const (
	confluenceAPIVersion = "v2"
)

// ConfluenceModule implements the Module interface for Confluence API
type ConfluenceModule struct{}

// New creates a new ConfluenceModule instance
func New() *ConfluenceModule {
	return &ConfluenceModule{}
}

// Module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Confluence API - Wiki operations (Space, Page, Search, Comment, Label)",
	"ja-JP": "Confluence API - Wiki操作（スペース、ページ、検索、コメント、ラベル）",
}

// Name returns the module name
func (m *ConfluenceModule) Name() string {
	return "confluence"
}

// Descriptions returns the module descriptions in all languages
func (m *ConfluenceModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *ConfluenceModule) Description() string {
	return moduleDescriptions["en-US"]
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

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *ConfluenceModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for Confluence)
func (m *ConfluenceModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *ConfluenceModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// ogen client helper
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "confluence")
	if err != nil {
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}

	switch creds.AuthType {
	case broker.AuthTypeBasic:
		domain, _ := creds.Metadata["domain"].(string)
		if domain == "" {
			return nil, fmt.Errorf("confluence domain not configured")
		}
		serverURL := fmt.Sprintf("https://%s", domain)
		return confluenceapi.NewBasicClient(serverURL, creds.Username, creds.Password)
	default:
		// OAuth 2.0
		cloudID, _ := creds.Metadata["cloud_id"].(string)
		if cloudID == "" {
			return nil, fmt.Errorf("confluence cloud_id not configured")
		}
		serverURL := fmt.Sprintf("https://api.atlassian.com/ex/confluence/%s", cloudID)
		return confluenceapi.NewBearerClient(serverURL, creds.AccessToken)
	}
}

var toJSON = modules.ToJSON

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	{
		ID:   "confluence:list_spaces",
		Name: "list_spaces",
		Descriptions: modules.LocalizedText{
			"en-US": "List all Confluence spaces accessible to the current user.",
			"ja-JP": "現在のユーザーがアクセス可能なすべてのConfluenceスペースを一覧表示します。",
		},
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
		ID:   "confluence:get_space",
		Name: "get_space",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific Confluence space by ID or key.",
			"ja-JP": "IDまたはキーで特定のConfluenceスペースの詳細を取得します。",
		},
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
		ID:   "confluence:get_pages",
		Name: "get_pages",
		Descriptions: modules.LocalizedText{
			"en-US": "List pages in a Confluence space.",
			"ja-JP": "Confluenceスペース内のページを一覧表示します。",
		},
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
		ID:   "confluence:get_page",
		Name: "get_page",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a Confluence page by ID with its content.",
			"ja-JP": "IDでConfluenceページとそのコンテンツを取得します。",
		},
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
		ID:   "confluence:create_page",
		Name: "create_page",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new Confluence page.",
			"ja-JP": "新しいConfluenceページを作成します。",
		},
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
		ID:   "confluence:update_page",
		Name: "update_page",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing Confluence page.",
			"ja-JP": "既存のConfluenceページを更新します。",
		},
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
		ID:   "confluence:delete_page",
		Name: "delete_page",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a Confluence page.",
			"ja-JP": "Confluenceページを削除します。",
		},
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
		ID:   "confluence:search",
		Name: "search",
		Descriptions: modules.LocalizedText{
			"en-US": "Search Confluence content using CQL (Confluence Query Language). Example: 'type=page AND space=MYSPACE AND text~\"keyword\"'",
			"ja-JP": "CQL（Confluence Query Language）を使用してConfluenceコンテンツを検索します。例：'type=page AND space=MYSPACE AND text~\"keyword\"'",
		},
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
		ID:   "confluence:get_page_comments",
		Name: "get_page_comments",
		Descriptions: modules.LocalizedText{
			"en-US": "Get comments on a Confluence page.",
			"ja-JP": "Confluenceページのコメントを取得します。",
		},
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
		ID:   "confluence:add_page_comment",
		Name: "add_page_comment",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a comment to a Confluence page.",
			"ja-JP": "Confluenceページにコメントを追加します。",
		},
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
		ID:   "confluence:get_page_labels",
		Name: "get_page_labels",
		Descriptions: modules.LocalizedText{
			"en-US": "Get labels on a Confluence page.",
			"ja-JP": "Confluenceページのラベルを取得します。",
		},
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
		ID:   "confluence:add_page_label",
		Name: "add_page_label",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a label to a Confluence page.",
			"ja-JP": "Confluenceページにラベルを追加します。",
		},
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
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.ListSpacesParams{}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		p.Cursor.SetTo(cursor)
	}
	res, err := c.ListSpaces(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

var numericRegex = regexp.MustCompile(`^\d+$`)

func getSpace(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spaceIDOrKey, _ := params["space_id_or_key"].(string)

	if numericRegex.MatchString(spaceIDOrKey) {
		res, err := c.GetSpaceById(ctx, gen.GetSpaceByIdParams{SpaceId: spaceIDOrKey})
		if err != nil {
			return "", err
		}
		return toJSON(res)
	}
	res, err := c.GetSpaceByKey(ctx, gen.GetSpaceByKeyParams{SpaceKey: spaceIDOrKey})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Pages
// =============================================================================

func getPages(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spaceID, _ := params["space_id"].(string)
	p := gen.GetPagesParams{SpaceId: spaceID}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		p.Cursor.SetTo(cursor)
	}
	res, err := c.GetPages(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getPage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	p := gen.GetPageParams{PageId: pageID}
	if bf, ok := params["body_format"].(string); ok && bf != "" {
		p.BodyFormat.SetTo(bf)
	} else {
		p.BodyFormat.SetTo("storage")
	}
	res, err := c.GetPage(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createPage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spaceID, _ := params["space_id"].(string)
	title, _ := params["title"].(string)
	body, _ := params["body"].(string)

	req := gen.CreatePageRequest{
		SpaceId: spaceID,
		Title:   title,
		Status:  "current",
		Body: gen.PageBody{
			Representation: gen.NewOptString("storage"),
			Value:          gen.NewOptString(body),
		},
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		req.ParentId.SetTo(parentID)
	}

	res, err := c.CreatePage(ctx, &req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updatePage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	title, _ := params["title"].(string)
	body, _ := params["body"].(string)
	version, _ := params["version"].(float64)

	req := gen.UpdatePageRequest{
		ID:     pageID,
		Title:  title,
		Status: "current",
		Body: gen.PageBody{
			Representation: gen.NewOptString("storage"),
			Value:          gen.NewOptString(body),
		},
		Version: gen.VersionInput{
			Number: gen.NewOptInt(int(version)),
		},
	}

	res, err := c.UpdatePage(ctx, &req, gen.UpdatePageParams{PageId: pageID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deletePage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	err = c.DeletePage(ctx, gen.DeletePageParams{PageId: pageID})
	if err != nil {
		return "", err
	}
	return `{"deleted":true}`, nil
}

// =============================================================================
// Search
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cql, _ := params["cql"].(string)
	p := gen.SearchContentParams{Cql: cql}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}
	if s, ok := params["start"].(float64); ok {
		p.Start.SetTo(int(s))
	}
	res, err := c.SearchContent(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Comments
// =============================================================================

func getPageComments(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	p := gen.GetPageCommentsParams{PageId: pageID}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		p.Cursor.SetTo(cursor)
	}
	res, err := c.GetPageComments(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func addPageComment(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	body, _ := params["body"].(string)

	req := gen.CreateCommentRequest{
		PageId: pageID,
		Body: gen.PageBody{
			Representation: gen.NewOptString("storage"),
			Value:          gen.NewOptString(body),
		},
	}
	res, err := c.AddPageComment(ctx, &req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Labels
// =============================================================================

func getPageLabels(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	res, err := c.GetPageLabels(ctx, gen.GetPageLabelsParams{PageId: pageID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func addPageLabel(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	label, _ := params["label"].(string)

	req := gen.AddLabelRequestArray{{Name: label}}
	res, err := c.AddPageLabel(ctx, req, gen.AddPageLabelParams{PageId: pageID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}
