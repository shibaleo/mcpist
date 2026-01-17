package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/modules"
)

// toolDefinitions returns all Notion tool definitions
func toolDefinitions() []modules.Tool {
	return []modules.Tool{
		// Search
		{
			Name:        "search",
			Description: "Search pages and databases in Notion by title. Returns pages and databases shared with the integration.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"query": {
						Type:        "string",
						Description: "Search query to match against page/database titles. If empty, returns all shared content.",
					},
					"filter_type": {
						Type:        "string",
						Description: "Filter results to only pages or only databases (page or database)",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of results to return (max 100, default 10)",
					},
				},
			},
		},
		// Pages
		{
			Name:        "get_page",
			Description: "Retrieve a Notion page by ID. Returns page properties and metadata.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "The ID of the page to retrieve (UUID format)",
					},
				},
				Required: []string{"page_id"},
			},
		},
		{
			Name:        "get_page_content",
			Description: "Get the content (blocks) of a Notion page. Use this to read the actual text content.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "The ID of the page to get content from",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of blocks to return per request (max 100, default 100)",
					},
					"fetch_all": {
						Type:        "boolean",
						Description: "If true, automatically fetches all blocks using pagination (default false)",
					},
				},
				Required: []string{"page_id"},
			},
		},
		{
			Name:        "create_page",
			Description: "Create a new page in Notion. Can create as a child of another page or in a database.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"parent_page_id": {
						Type:        "string",
						Description: "Parent page ID (use this OR parent_database_id)",
					},
					"parent_database_id": {
						Type:        "string",
						Description: "Parent database ID (use this OR parent_page_id)",
					},
					"title": {
						Type:        "string",
						Description: "Page title",
					},
					"properties": {
						Type:        "object",
						Description: "Page properties (for database pages). Keys are property names.",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "update_page",
			Description: "Update a Notion page's properties.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "The ID of the page to update",
					},
					"properties": {
						Type:        "object",
						Description: "Properties to update. Keys are property names.",
					},
				},
				Required: []string{"page_id", "properties"},
			},
			Dangerous: true,
		},
		// Databases
		{
			Name:        "get_database",
			Description: "Retrieve a Notion database schema and metadata.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"database_id": {
						Type:        "string",
						Description: "The ID of the database to retrieve",
					},
				},
				Required: []string{"database_id"},
			},
		},
		{
			Name:        "query_database",
			Description: "Query a Notion database with optional filters and sorts. Returns pages in the database.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"database_id": {
						Type:        "string",
						Description: "The ID of the database to query",
					},
					"filter": {
						Type:        "object",
						Description: "Filter object (Notion filter format)",
					},
					"sorts": {
						Type:        "array",
						Description: "Sort specifications array",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of results to return (max 100, default 10)",
					},
				},
				Required: []string{"database_id"},
			},
		},
		// Blocks
		{
			Name:        "append_blocks",
			Description: "Append content blocks to a page or block. Use to add text, headings, lists, etc.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"block_id": {
						Type:        "string",
						Description: "The ID of the page or block to append to",
					},
					"blocks": {
						Type:        "array",
						Description: "Array of block objects to append. Each block needs type (paragraph, heading_1, heading_2, heading_3, bulleted_list_item, numbered_list_item, to_do, toggle, code, quote, callout, divider) and content (text).",
					},
				},
				Required: []string{"block_id", "blocks"},
			},
			Dangerous: true,
		},
		{
			Name:        "delete_block",
			Description: "Delete a block from Notion. This also deletes all children of the block.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"block_id": {
						Type:        "string",
						Description: "The ID of the block to delete",
					},
				},
				Required: []string{"block_id"},
			},
			Dangerous: true,
		},
		// Comments
		{
			Name:        "list_comments",
			Description: "List comments on a Notion page or block.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"block_id": {
						Type:        "string",
						Description: "The ID of the page or block to get comments from",
					},
					"page_size": {
						Type:        "number",
						Description: "Number of comments to return (max 100, default 50)",
					},
				},
				Required: []string{"block_id"},
			},
		},
		{
			Name:        "add_comment",
			Description: "Add a comment to a Notion page.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_id": {
						Type:        "string",
						Description: "The ID of the page to comment on",
					},
					"content": {
						Type:        "string",
						Description: "Comment text content",
					},
				},
				Required: []string{"page_id", "content"},
			},
		},
		// Users
		{
			Name:        "list_users",
			Description: "List all users in the Notion workspace.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"page_size": {
						Type:        "number",
						Description: "Number of users to return (max 100, default 50)",
					},
				},
			},
		},
		{
			Name:        "get_user",
			Description: "Get information about a Notion user.",
			InputSchema: modules.InputSchema{
				Type: "object",
				Properties: map[string]modules.Property{
					"user_id": {
						Type:        "string",
						Description: "The ID of the user to retrieve",
					},
				},
				Required: []string{"user_id"},
			},
		},
		{
			Name:        "get_bot_user",
			Description: "Get information about the current integration bot user.",
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
	endpoint := notionAPIBase + "/search"

	body := make(map[string]interface{})

	if query, ok := params["query"].(string); ok && query != "" {
		body["query"] = query
	}

	if filterType, ok := params["filter_type"].(string); ok && filterType != "" {
		body["filter"] = map[string]interface{}{
			"property": "object",
			"value":    filterType,
		}
	}

	pageSize := 10
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	body["page_size"] = pageSize

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Pages
// =============================================================================

func getPage(ctx context.Context, params map[string]any) (string, error) {
	pageID, ok := params["page_id"].(string)
	if !ok {
		return "", fmt.Errorf("page_id must be a string")
	}

	endpoint := fmt.Sprintf("%s/pages/%s", notionAPIBase, pageID)

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func getPageContent(ctx context.Context, params map[string]any) (string, error) {
	pageID, ok := params["page_id"].(string)
	if !ok {
		return "", fmt.Errorf("page_id must be a string")
	}

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
		endpoint := fmt.Sprintf("%s/blocks/%s/children?page_size=%d", notionAPIBase, pageID, pageSize)
		respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
		if err != nil {
			return "", err
		}
		return httpclient.PrettyJSON(respBody), nil
	}

	// Fetch all mode - loop until has_more is false
	var allBlocks []interface{}
	nextCursor := ""

	for {
		endpoint := fmt.Sprintf("%s/blocks/%s/children?page_size=%d", notionAPIBase, pageID, pageSize)
		if nextCursor != "" {
			endpoint += "&start_cursor=" + nextCursor
		}

		respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
		if err != nil {
			return "", err
		}

		var response struct {
			Results    []interface{} `json:"results"`
			HasMore    bool          `json:"has_more"`
			NextCursor *string       `json:"next_cursor"`
		}

		if err := json.Unmarshal(respBody, &response); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		allBlocks = append(allBlocks, response.Results...)

		if !response.HasMore || response.NextCursor == nil {
			break
		}
		nextCursor = *response.NextCursor
	}

	// Build combined response
	result := map[string]interface{}{
		"object":   "list",
		"results":  allBlocks,
		"has_more": false,
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

func createPage(ctx context.Context, params map[string]any) (string, error) {
	title, ok := params["title"].(string)
	if !ok {
		return "", fmt.Errorf("title must be a string")
	}

	parentPageID, hasParentPage := params["parent_page_id"].(string)
	parentDatabaseID, hasParentDB := params["parent_database_id"].(string)

	if !hasParentPage && !hasParentDB {
		return "", fmt.Errorf("either parent_page_id or parent_database_id is required")
	}

	body := make(map[string]interface{})

	if hasParentDB && parentDatabaseID != "" {
		body["parent"] = map[string]interface{}{
			"database_id": parentDatabaseID,
		}
		// For database pages, set title in Name or Title property
		properties := make(map[string]interface{})
		if props, ok := params["properties"].(map[string]interface{}); ok {
			properties = props
		}
		// Add Name property with title if not already set
		if _, hasName := properties["Name"]; !hasName {
			if _, hasTitle := properties["Title"]; !hasTitle {
				properties["Name"] = map[string]interface{}{
					"title": []map[string]interface{}{
						{"text": map[string]interface{}{"content": title}},
					},
				}
			}
		}
		body["properties"] = properties
	} else {
		body["parent"] = map[string]interface{}{
			"page_id": parentPageID,
		}
		body["properties"] = map[string]interface{}{
			"title": map[string]interface{}{
				"title": []map[string]interface{}{
					{"text": map[string]interface{}{"content": title}},
				},
			},
		}
	}

	endpoint := notionAPIBase + "/pages"

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func updatePage(ctx context.Context, params map[string]any) (string, error) {
	pageID, ok := params["page_id"].(string)
	if !ok {
		return "", fmt.Errorf("page_id must be a string")
	}

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("properties must be an object")
	}

	body := map[string]interface{}{
		"properties": properties,
	}

	endpoint := fmt.Sprintf("%s/pages/%s", notionAPIBase, pageID)

	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Databases
// =============================================================================

func getDatabase(ctx context.Context, params map[string]any) (string, error) {
	databaseID, ok := params["database_id"].(string)
	if !ok {
		return "", fmt.Errorf("database_id must be a string")
	}

	endpoint := fmt.Sprintf("%s/databases/%s", notionAPIBase, databaseID)

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func queryDatabase(ctx context.Context, params map[string]any) (string, error) {
	databaseID, ok := params["database_id"].(string)
	if !ok {
		return "", fmt.Errorf("database_id must be a string")
	}

	body := make(map[string]interface{})

	if filter, ok := params["filter"].(map[string]interface{}); ok {
		body["filter"] = filter
	}

	if sorts, ok := params["sorts"].([]interface{}); ok {
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

	endpoint := fmt.Sprintf("%s/databases/%s/query", notionAPIBase, databaseID)

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Blocks
// =============================================================================

func appendBlocks(ctx context.Context, params map[string]any) (string, error) {
	blockID, ok := params["block_id"].(string)
	if !ok {
		return "", fmt.Errorf("block_id must be a string")
	}

	blocksInput, ok := params["blocks"].([]interface{})
	if !ok {
		return "", fmt.Errorf("blocks must be an array")
	}

	children := make([]map[string]interface{}, 0, len(blocksInput))

	for _, b := range blocksInput {
		block, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)
		content, _ := block["content"].(string)
		checked, _ := block["checked"].(bool)
		language, _ := block["language"].(string)

		richText := []map[string]interface{}{}
		if content != "" {
			richText = []map[string]interface{}{
				{"type": "text", "text": map[string]interface{}{"content": content}},
			}
		}

		var notionBlock map[string]interface{}

		switch blockType {
		case "paragraph":
			notionBlock = map[string]interface{}{
				"object":    "block",
				"type":      "paragraph",
				"paragraph": map[string]interface{}{"rich_text": richText},
			}
		case "heading_1":
			notionBlock = map[string]interface{}{
				"object":    "block",
				"type":      "heading_1",
				"heading_1": map[string]interface{}{"rich_text": richText},
			}
		case "heading_2":
			notionBlock = map[string]interface{}{
				"object":    "block",
				"type":      "heading_2",
				"heading_2": map[string]interface{}{"rich_text": richText},
			}
		case "heading_3":
			notionBlock = map[string]interface{}{
				"object":    "block",
				"type":      "heading_3",
				"heading_3": map[string]interface{}{"rich_text": richText},
			}
		case "bulleted_list_item":
			notionBlock = map[string]interface{}{
				"object":             "block",
				"type":               "bulleted_list_item",
				"bulleted_list_item": map[string]interface{}{"rich_text": richText},
			}
		case "numbered_list_item":
			notionBlock = map[string]interface{}{
				"object":             "block",
				"type":               "numbered_list_item",
				"numbered_list_item": map[string]interface{}{"rich_text": richText},
			}
		case "to_do":
			notionBlock = map[string]interface{}{
				"object": "block",
				"type":   "to_do",
				"to_do":  map[string]interface{}{"rich_text": richText, "checked": checked},
			}
		case "toggle":
			notionBlock = map[string]interface{}{
				"object": "block",
				"type":   "toggle",
				"toggle": map[string]interface{}{"rich_text": richText},
			}
		case "code":
			if language == "" {
				language = "plain text"
			}
			notionBlock = map[string]interface{}{
				"object": "block",
				"type":   "code",
				"code":   map[string]interface{}{"rich_text": richText, "language": language},
			}
		case "quote":
			notionBlock = map[string]interface{}{
				"object": "block",
				"type":   "quote",
				"quote":  map[string]interface{}{"rich_text": richText},
			}
		case "callout":
			notionBlock = map[string]interface{}{
				"object":  "block",
				"type":    "callout",
				"callout": map[string]interface{}{"rich_text": richText},
			}
		case "divider":
			notionBlock = map[string]interface{}{
				"object":  "block",
				"type":    "divider",
				"divider": map[string]interface{}{},
			}
		default:
			// Default to paragraph
			notionBlock = map[string]interface{}{
				"object":    "block",
				"type":      "paragraph",
				"paragraph": map[string]interface{}{"rich_text": richText},
			}
		}

		children = append(children, notionBlock)
	}

	body := map[string]interface{}{
		"children": children,
	}

	endpoint := fmt.Sprintf("%s/blocks/%s/children", notionAPIBase, blockID)

	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func deleteBlock(ctx context.Context, params map[string]any) (string, error) {
	blockID, ok := params["block_id"].(string)
	if !ok {
		return "", fmt.Errorf("block_id must be a string")
	}

	endpoint := fmt.Sprintf("%s/blocks/%s", notionAPIBase, blockID)

	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Comments
// =============================================================================

func listComments(ctx context.Context, params map[string]any) (string, error) {
	blockID, ok := params["block_id"].(string)
	if !ok {
		return "", fmt.Errorf("block_id must be a string")
	}

	pageSize := 50
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}

	query := url.Values{}
	query.Set("block_id", blockID)
	query.Set("page_size", fmt.Sprintf("%d", pageSize))

	endpoint := fmt.Sprintf("%s/comments?%s", notionAPIBase, query.Encode())

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func addComment(ctx context.Context, params map[string]any) (string, error) {
	pageID, ok := params["page_id"].(string)
	if !ok {
		return "", fmt.Errorf("page_id must be a string")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content must be a string")
	}

	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"page_id": pageID,
		},
		"rich_text": []map[string]interface{}{
			{"text": map[string]interface{}{"content": content}},
		},
	}

	endpoint := notionAPIBase + "/comments"

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Users
// =============================================================================

func listUsers(ctx context.Context, params map[string]any) (string, error) {
	pageSize := 50
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}

	endpoint := fmt.Sprintf("%s/users?page_size=%d", notionAPIBase, pageSize)

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func getUser(ctx context.Context, params map[string]any) (string, error) {
	userID, ok := params["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user_id must be a string")
	}

	endpoint := fmt.Sprintf("%s/users/%s", notionAPIBase, userID)

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func getBotUser(ctx context.Context, params map[string]any) (string, error) {
	endpoint := notionAPIBase + "/users/me"

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// Ensure json package is used
var _ = json.Marshal
