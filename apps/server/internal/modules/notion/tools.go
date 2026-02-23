package notion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-faster/jx"

	"mcpist/server/internal/modules"
	gen "mcpist/server/pkg/notionapi/gen"
)

var toJSON = modules.ToJSON

// toolDefinitions returns all Notion tool definitions
func toolDefinitions() []modules.Tool {
	return []modules.Tool{
		// Search
		{
			ID:   "notion:search",
			Name: "search",
			Descriptions: modules.LocalizedText{
				"en-US": "Search pages and databases in Notion by title. Returns pages and databases shared with the integration.",
				"ja-JP": "Notionのページとデータベースをタイトルで検索します。インテグレーションと共有されているページとデータベースを返します。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"query": {
						Type:        "string",
						Description: "Search query to match against page/database titles. If empty, returns all shared content.",
					},
					"filter_type": {
						Type:        "string",
						Description: "Filter by object type: \"page\" or \"database\"",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of results (1-100, default 10)",
					},
				},
			},
		},
		// Pages
		{
			ID:   "notion:get_page",
			Name: "get_page",
			Descriptions: modules.LocalizedText{
				"en-US": "Retrieve a Notion page's properties and metadata by ID. Use get_page_content to read block content.",
				"ja-JP": "IDでNotionページのプロパティとメタデータを取得します。ブロックコンテンツを読むにはget_page_contentを使用してください。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "Page ID (UUID format, e.g. \"a1b2c3d4-e5f6-...\")",
					},
				},
				Required: []string{"page_id"},
			},
		},
		{
			ID:   "notion:get_page_content",
			Name: "get_page_content",
			Descriptions: modules.LocalizedText{
				"en-US": "Get the block content of a Notion page. Returns child blocks (text, headings, lists, etc.).",
				"ja-JP": "Notionページのブロックコンテンツを取得します。子ブロック（テキスト、見出し、リストなど）を返します。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "Page ID (UUID format)",
					},
					"page_size": {
						Type:        "number",
						Description: "Blocks per request (1-100, default 100)",
					},
					"fetch_all": {
						Type:        "boolean",
						Description: "Fetch all blocks via pagination (default false)",
					},
				},
				Required: []string{"page_id"},
			},
		},
		{
			ID:   "notion:create_page",
			Name: "create_page",
			Descriptions: modules.LocalizedText{
				"en-US": "Create a new Notion page. Specify exactly one of parent_page_id or parent_database_id.",
				"ja-JP": "Notionページを新規作成します。parent_page_idまたはparent_database_idのいずれか1つを指定してください。",
			},
			Annotations: modules.AnnotateCreate,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"title": {
						Type:        "string",
						Description: "Page title text",
					},
					"parent_page_id": {
						Type:        "string",
						Description: "Create as child of this page. Mutually exclusive with parent_database_id.",
					},
					"parent_database_id": {
						Type:        "string",
						Description: "Create as row in this database. Mutually exclusive with parent_page_id.",
					},
					"properties": {
						Type:        "object",
						Description: "Database row properties (only when using parent_database_id). Keys are property names, values follow Notion property value format.",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			ID:   "notion:update_page",
			Name: "update_page",
			Descriptions: modules.LocalizedText{
				"en-US": "Update a Notion page's properties. Use append_blocks to modify block content.",
				"ja-JP": "Notionページのプロパティを更新します。ブロックコンテンツの変更にはappend_blocksを使用してください。",
			},
			Annotations: modules.AnnotateUpdate,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "Page ID to update (UUID format)",
					},
					"properties": {
						Type:        "object",
						Description: "Properties to update. Keys are property names, values follow Notion property value format.",
					},
				},
				Required: []string{"page_id", "properties"},
			},
		},
		// Databases
		{
			ID:   "notion:get_database",
			Name: "get_database",
			Descriptions: modules.LocalizedText{
				"en-US": "Retrieve a Notion database's schema (property definitions) and metadata. Use query_database to get rows.",
				"ja-JP": "Notionデータベースのスキーマ（プロパティ定義）とメタデータを取得します。行の取得にはquery_databaseを使用してください。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"database_id": {
						Type:        "string",
						Description: "Database ID (UUID format)",
					},
				},
				Required: []string{"database_id"},
			},
		},
		{
			ID:   "notion:query_database",
			Name: "query_database",
			Descriptions: modules.LocalizedText{
				"en-US": "Query a Notion database to get rows (pages). Supports Notion filter and sort syntax.",
				"ja-JP": "Notionデータベースの行（ページ）を取得します。Notionフィルター・ソート構文に対応しています。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"database_id": {
						Type:        "string",
						Description: "Database ID (UUID format)",
					},
					"filter": {
						Type:        "object",
						Description: "Notion filter object. Example: {\"property\":\"Status\",\"select\":{\"equals\":\"Done\"}}",
					},
					"sorts": {
						Type:        "array",
						Description: "Sort array. Example: [{\"property\":\"Created\",\"direction\":\"descending\"}]",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of rows (1-100, default 10)",
					},
				},
				Required: []string{"database_id"},
			},
		},
		// Blocks
		{
			ID:   "notion:append_blocks",
			Name: "append_blocks",
			Descriptions: modules.LocalizedText{
				"en-US": "Append content blocks to a Notion page or block.",
				"ja-JP": "Notionページまたはブロックにコンテンツブロックを追加します。",
			},
			Annotations: modules.AnnotateCreate,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"block_id": {
						Type:        "string",
						Description: "Page or block ID to append to (UUID format)",
					},
					"blocks": {
						Type:        "array",
						Description: "Simplified block array. Each element: {\"type\":\"<block_type>\",\"content\":\"<text>\"}. Supported types: paragraph, heading_1, heading_2, heading_3, bulleted_list_item, numbered_list_item, to_do, toggle, code, quote, callout, divider.",
					},
				},
				Required: []string{"block_id", "blocks"},
			},
		},
		{
			ID:   "notion:delete_block",
			Name: "delete_block",
			Descriptions: modules.LocalizedText{
				"en-US": "Delete a block and all its children from Notion.",
				"ja-JP": "Notionからブロックとそのすべての子ブロックを削除します。",
			},
			Annotations: modules.AnnotateDelete,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"block_id": {
						Type:        "string",
						Description: "Block ID to delete (UUID format)",
					},
				},
				Required: []string{"block_id"},
			},
		},
		// Comments
		{
			ID:   "notion:list_comments",
			Name: "list_comments",
			Descriptions: modules.LocalizedText{
				"en-US": "List comments on a Notion page or block.",
				"ja-JP": "Notionページまたはブロックのコメントを一覧表示します。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"block_id": {
						Type:        "string",
						Description: "Page or block ID to list comments from (UUID format)",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of comments (1-100, default 50)",
					},
				},
				Required: []string{"block_id"},
			},
		},
		{
			ID:   "notion:add_comment",
			Name: "add_comment",
			Descriptions: modules.LocalizedText{
				"en-US": "Add a text comment to a Notion page.",
				"ja-JP": "Notionページにテキストコメントを追加します。",
			},
			Annotations: modules.AnnotateCreate,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "Page ID to comment on (UUID format)",
					},
					"content": {
						Type:        "string",
						Description: "Comment text",
					},
				},
				Required: []string{"page_id", "content"},
			},
		},
		// Users
		{
			ID:   "notion:list_users",
			Name: "list_users",
			Descriptions: modules.LocalizedText{
				"en-US": "List all users in the Notion workspace.",
				"ja-JP": "Notionワークスペース内のすべてのユーザーを一覧表示します。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_size": {
						Type:        "number",
						Description: "Number of users (1-100, default 50)",
					},
				},
			},
		},
		{
			ID:   "notion:get_user",
			Name: "get_user",
			Descriptions: modules.LocalizedText{
				"en-US": "Get information about a specific Notion user by ID.",
				"ja-JP": "IDで特定のNotionユーザーの情報を取得します。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"user_id": {
						Type:        "string",
						Description: "User ID (UUID format)",
					},
				},
				Required: []string{"user_id"},
			},
		},
		{
			ID:   "notion:get_bot_user",
			Name: "get_bot_user",
			Descriptions: modules.LocalizedText{
				"en-US": "Get information about the current integration bot user.",
				"ja-JP": "現在のインテグレーションボットユーザーの情報を取得します。",
			},
			Annotations: modules.AnnotateReadOnly,
			InputSchema: modules.InputSchema{
				Type:       "object",
				Properties: map[string]modules.Property{},
			},
		},
	}
}

// toolHandlers maps tool names to their handler functions
var toolHandlers = map[string]func(ctx context.Context, params map[string]any) (string, error){
	"search":           search,
	"get_page":         getPage,
	"get_page_content": getPageContent,
	"create_page":      createPage,
	"update_page":      updatePage,
	"get_database":     getDatabase,
	"query_database":   queryDatabase,
	"append_blocks":    appendBlocks,
	"delete_block":     deleteBlock,
	"list_comments":    listComments,
	"add_comment":      addComment,
	"list_users":       listUsers,
	"get_user":         getUser,
	"get_bot_user":     getBotUser,
}

// =============================================================================
// Search
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	req := gen.SearchRequest{}
	if query, ok := params["query"].(string); ok && query != "" {
		req.Query.SetTo(query)
	}
	if filterType, ok := params["filter_type"].(string); ok && filterType != "" {
		req.Filter = jx.Raw(fmt.Sprintf(`{"property":"object","value":"%s"}`, filterType))
	}
	pageSize := 10
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	req.PageSize.SetTo(pageSize)

	res, err := c.Search(ctx, &req)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Pages
// =============================================================================

func getPage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	res, err := c.GetPage(ctx, gen.GetPageParams{PageID: pageID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func getPageContent(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)

	pageSize := 100
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}

	fetchAll := false
	if fa, ok := params["fetch_all"].(bool); ok {
		fetchAll = fa
	}

	// Single request mode (default)
	if !fetchAll {
		p := gen.GetBlockChildrenParams{BlockID: pageID}
		p.PageSize.SetTo(pageSize)
		res, err := c.GetBlockChildren(ctx, p)
		if err != nil {
			return "", err
		}
		jsonStr, err := toJSON(res)
		if err != nil {
			return "", err
		}
		return jsonStr, nil
	}

	// Fetch all mode - loop until has_more is false
	var allResults []json.RawMessage
	nextCursor := ""

	for {
		p := gen.GetBlockChildrenParams{BlockID: pageID}
		p.PageSize.SetTo(pageSize)
		if nextCursor != "" {
			p.StartCursor.SetTo(nextCursor)
		}

		res, err := c.GetBlockChildren(ctx, p)
		if err != nil {
			return "", err
		}

		// Extract results from PaginatedList
		resJSON, err := json.Marshal(res)
		if err != nil {
			return "", err
		}
		var parsed struct {
			Results    []json.RawMessage `json:"results"`
			HasMore    bool              `json:"has_more"`
			NextCursor *string           `json:"next_cursor"`
		}
		json.Unmarshal(resJSON, &parsed)

		allResults = append(allResults, parsed.Results...)

		if !parsed.HasMore || parsed.NextCursor == nil {
			break
		}
		nextCursor = *parsed.NextCursor
	}

	// Build combined response
	result := map[string]any{
		"object":   "list",
		"results":  allResults,
		"has_more": false,
	}
	jsonStr, err := toJSON(result)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func createPage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	title, _ := params["title"].(string)
	parentPageID, hasParentPage := params["parent_page_id"].(string)
	parentDatabaseID, hasParentDB := params["parent_database_id"].(string)

	if !hasParentPage && !hasParentDB {
		return "", fmt.Errorf("either parent_page_id or parent_database_id is required")
	}

	// Build request as raw JSON since parent/properties are any types
	body := make(map[string]any)

	if hasParentDB && parentDatabaseID != "" {
		body["parent"] = map[string]any{"database_id": parentDatabaseID}
		properties := make(map[string]any)
		if props, ok := params["properties"].(map[string]any); ok {
			properties = props
		}
		if _, hasName := properties["Name"]; !hasName {
			if _, hasTitle := properties["Title"]; !hasTitle {
				properties["Name"] = map[string]any{
					"title": []map[string]any{
						{"text": map[string]any{"content": title}},
					},
				}
			}
		}
		body["properties"] = properties
	} else {
		body["parent"] = map[string]any{"page_id": parentPageID}
		body["properties"] = map[string]any{
			"title": map[string]any{
				"title": []map[string]any{
					{"text": map[string]any{"content": title}},
				},
			},
		}
	}

	// Marshal to gen.CreatePageRequest via JSON round-trip
	bodyJSON, _ := json.Marshal(body)
	var req gen.CreatePageRequest
	json.Unmarshal(bodyJSON, &req)

	res, err := c.CreatePage(ctx, &req)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func updatePage(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	properties, _ := params["properties"].(map[string]any)

	body := map[string]any{"properties": properties}
	bodyJSON, _ := json.Marshal(body)
	var req gen.UpdatePageRequest
	json.Unmarshal(bodyJSON, &req)

	res, err := c.UpdatePage(ctx, &req, gen.UpdatePageParams{PageID: pageID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Databases
// =============================================================================

func getDatabase(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	databaseID, _ := params["database_id"].(string)
	res, err := c.GetDatabase(ctx, gen.GetDatabaseParams{DatabaseID: databaseID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func queryDatabase(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	databaseID, _ := params["database_id"].(string)

	body := make(map[string]any)
	if filter, ok := params["filter"].(map[string]any); ok {
		body["filter"] = filter
	}
	if sorts, ok := params["sorts"].([]any); ok {
		body["sorts"] = sorts
	}
	pageSize := 10
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	body["page_size"] = pageSize

	bodyJSON, _ := json.Marshal(body)
	var req gen.QueryDatabaseRequest
	json.Unmarshal(bodyJSON, &req)

	res, err := c.QueryDatabase(ctx, &req, gen.QueryDatabaseParams{DatabaseID: databaseID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Blocks
// =============================================================================

func appendBlocks(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	blockID, _ := params["block_id"].(string)
	blocksInput, ok := params["blocks"].([]any)
	if !ok {
		return "", fmt.Errorf("blocks must be an array")
	}

	children := buildBlockChildren(blocksInput)

	body := map[string]any{"children": children}
	bodyJSON, _ := json.Marshal(body)
	var req gen.AppendBlockChildrenRequest
	json.Unmarshal(bodyJSON, &req)

	res, err := c.AppendBlockChildren(ctx, &req, gen.AppendBlockChildrenParams{BlockID: blockID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func buildBlockChildren(blocksInput []any) []map[string]any {
	children := make([]map[string]any, 0, len(blocksInput))

	for _, b := range blocksInput {
		block, ok := b.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)
		content, _ := block["content"].(string)
		checked, _ := block["checked"].(bool)
		language, _ := block["language"].(string)

		richText := []map[string]any{}
		if content != "" {
			richText = []map[string]any{
				{"type": "text", "text": map[string]any{"content": content}},
			}
		}

		var notionBlock map[string]any

		switch blockType {
		case "paragraph":
			notionBlock = map[string]any{"object": "block", "type": "paragraph", "paragraph": map[string]any{"rich_text": richText}}
		case "heading_1":
			notionBlock = map[string]any{"object": "block", "type": "heading_1", "heading_1": map[string]any{"rich_text": richText}}
		case "heading_2":
			notionBlock = map[string]any{"object": "block", "type": "heading_2", "heading_2": map[string]any{"rich_text": richText}}
		case "heading_3":
			notionBlock = map[string]any{"object": "block", "type": "heading_3", "heading_3": map[string]any{"rich_text": richText}}
		case "bulleted_list_item":
			notionBlock = map[string]any{"object": "block", "type": "bulleted_list_item", "bulleted_list_item": map[string]any{"rich_text": richText}}
		case "numbered_list_item":
			notionBlock = map[string]any{"object": "block", "type": "numbered_list_item", "numbered_list_item": map[string]any{"rich_text": richText}}
		case "to_do":
			notionBlock = map[string]any{"object": "block", "type": "to_do", "to_do": map[string]any{"rich_text": richText, "checked": checked}}
		case "toggle":
			notionBlock = map[string]any{"object": "block", "type": "toggle", "toggle": map[string]any{"rich_text": richText}}
		case "code":
			if language == "" {
				language = "plain text"
			}
			notionBlock = map[string]any{"object": "block", "type": "code", "code": map[string]any{"rich_text": richText, "language": language}}
		case "quote":
			notionBlock = map[string]any{"object": "block", "type": "quote", "quote": map[string]any{"rich_text": richText}}
		case "callout":
			notionBlock = map[string]any{"object": "block", "type": "callout", "callout": map[string]any{"rich_text": richText}}
		case "divider":
			notionBlock = map[string]any{"object": "block", "type": "divider", "divider": map[string]any{}}
		default:
			notionBlock = map[string]any{"object": "block", "type": "paragraph", "paragraph": map[string]any{"rich_text": richText}}
		}

		children = append(children, notionBlock)
	}
	return children
}

func deleteBlock(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	blockID, _ := params["block_id"].(string)
	res, err := c.DeleteBlock(ctx, gen.DeleteBlockParams{BlockID: blockID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Comments
// =============================================================================

func listComments(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	blockID, _ := params["block_id"].(string)
	p := gen.ListCommentsParams{BlockID: blockID}
	if ps, ok := params["page_size"].(float64); ok {
		pageSize := int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
		p.PageSize.SetTo(pageSize)
	} else {
		p.PageSize.SetTo(50)
	}

	res, err := c.ListComments(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func addComment(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	pageID, _ := params["page_id"].(string)
	content, _ := params["content"].(string)

	body := map[string]any{
		"parent": map[string]any{"page_id": pageID},
		"rich_text": []map[string]any{
			{"text": map[string]any{"content": content}},
		},
	}
	bodyJSON, _ := json.Marshal(body)
	var req gen.AddCommentRequest
	json.Unmarshal(bodyJSON, &req)

	res, err := c.AddComment(ctx, &req)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Users
// =============================================================================

func listUsers(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.ListUsersParams{}
	if ps, ok := params["page_size"].(float64); ok {
		pageSize := int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
		p.PageSize.SetTo(pageSize)
	} else {
		p.PageSize.SetTo(50)
	}

	res, err := c.ListUsers(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func getUser(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	userID, _ := params["user_id"].(string)
	res, err := c.GetUser(ctx, gen.GetUserParams{UserID: userID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func getBotUser(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetBotUser(ctx)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}
