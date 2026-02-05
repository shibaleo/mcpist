package trello

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	trelloAPIBase = "https://api.trello.com/1"
	trelloVersion = "1"
)

var client = httpclient.New()

// TrelloModule implements the Module interface for Trello API
type TrelloModule struct{}

// New creates a new TrelloModule instance
func New() *TrelloModule {
	return &TrelloModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Trello API - Manage boards, lists, cards, and checklists",
	"ja-JP": "Trello API - ボード、リスト、カード、チェックリストの管理",
}

// Name returns the module name
func (m *TrelloModule) Name() string {
	return "trello"
}

// Descriptions returns the module descriptions in all languages
func (m *TrelloModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *TrelloModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Trello API version
func (m *TrelloModule) APIVersion() string {
	return trelloVersion
}

// Tools returns all available tools
func (m *TrelloModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *TrelloModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Trello)
func (m *TrelloModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *TrelloModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[trello] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "trello")
	if err != nil {
		log.Printf("[trello] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[trello] Got credentials: auth_type=%s", credentials.AuthType)
	return credentials
}

// addAuth adds API key and token to the URL query parameters
func addAuth(endpoint string, ctx context.Context) string {
	creds := getCredentials(ctx)
	if creds == nil {
		return endpoint
	}

	// Trello uses API Key + Token as query parameters
	apiKey := creds.APIKey
	token := creds.AccessToken

	if apiKey == "" || token == "" {
		log.Printf("[trello] Missing API key or token")
		return endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	q := u.Query()
	q.Set("key", apiKey)
	q.Set("token", token)
	u.RawQuery = q.Encode()

	return u.String()
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Boards
	{
		ID:   "trello:list_boards",
		Name: "list_boards",
		Descriptions: modules.LocalizedText{
			"en-US": "List all boards for the authenticated user.",
			"ja-JP": "認証されたユーザーのすべてのボードを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "trello:get_board",
		Name: "get_board",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific board.",
			"ja-JP": "特定のボードの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"board_id": {Type: "string", Description: "Board ID"},
			},
			Required: []string{"board_id"},
		},
	},
	// Lists
	{
		ID:   "trello:get_lists",
		Name: "get_lists",
		Descriptions: modules.LocalizedText{
			"en-US": "Get all lists on a board.",
			"ja-JP": "ボード上のすべてのリストを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"board_id": {Type: "string", Description: "Board ID"},
			},
			Required: []string{"board_id"},
		},
	},
	// Cards
	{
		ID:   "trello:get_cards",
		Name: "get_cards",
		Descriptions: modules.LocalizedText{
			"en-US": "Get all cards on a board or in a specific list.",
			"ja-JP": "ボード上またはリスト内のすべてのカードを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"board_id": {Type: "string", Description: "Board ID (required if list_id not specified)"},
				"list_id":  {Type: "string", Description: "List ID (optional, filters cards by list)"},
			},
		},
	},
	{
		ID:   "trello:get_card",
		Name: "get_card",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific card.",
			"ja-JP": "特定のカードの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id": {Type: "string", Description: "Card ID"},
			},
			Required: []string{"card_id"},
		},
	},
	{
		ID:   "trello:create_card",
		Name: "create_card",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new card in a list.",
			"ja-JP": "リストに新しいカードを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id":     {Type: "string", Description: "List ID to add the card to"},
				"name":        {Type: "string", Description: "Card name/title"},
				"desc":        {Type: "string", Description: "Card description"},
				"pos":         {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
				"due":         {Type: "string", Description: "Due date (ISO 8601 format)"},
				"labels":      {Type: "string", Description: "Comma-separated label IDs"},
				"member_ids":  {Type: "string", Description: "Comma-separated member IDs"},
			},
			Required: []string{"list_id", "name"},
		},
	},
	{
		ID:   "trello:update_card",
		Name: "update_card",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing card.",
			"ja-JP": "既存のカードを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id":  {Type: "string", Description: "Card ID"},
				"name":     {Type: "string", Description: "New card name"},
				"desc":     {Type: "string", Description: "New card description"},
				"closed":   {Type: "boolean", Description: "Archive the card"},
				"due":      {Type: "string", Description: "Due date (ISO 8601 format)"},
				"list_id":  {Type: "string", Description: "Move to different list"},
			},
			Required: []string{"card_id"},
		},
	},
	{
		ID:   "trello:move_card",
		Name: "move_card",
		Descriptions: modules.LocalizedText{
			"en-US": "Move a card to a different list.",
			"ja-JP": "カードを別のリストに移動します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id": {Type: "string", Description: "Card ID"},
				"list_id": {Type: "string", Description: "Target list ID"},
				"pos":     {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
			},
			Required: []string{"card_id", "list_id"},
		},
	},
	{
		ID:   "trello:delete_card",
		Name: "delete_card",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a card permanently.",
			"ja-JP": "カードを完全に削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id": {Type: "string", Description: "Card ID"},
			},
			Required: []string{"card_id"},
		},
	},
	// Checklists
	{
		ID:   "trello:get_checklists",
		Name: "get_checklists",
		Descriptions: modules.LocalizedText{
			"en-US": "Get all checklists on a card.",
			"ja-JP": "カード上のすべてのチェックリストを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id": {Type: "string", Description: "Card ID"},
			},
			Required: []string{"card_id"},
		},
	},
	{
		ID:   "trello:create_checklist",
		Name: "create_checklist",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new checklist on a card.",
			"ja-JP": "カードに新しいチェックリストを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id": {Type: "string", Description: "Card ID"},
				"name":    {Type: "string", Description: "Checklist name"},
				"pos":     {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
			},
			Required: []string{"card_id", "name"},
		},
	},
	{
		ID:   "trello:delete_checklist",
		Name: "delete_checklist",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a checklist.",
			"ja-JP": "チェックリストを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"checklist_id": {Type: "string", Description: "Checklist ID"},
			},
			Required: []string{"checklist_id"},
		},
	},
	// Checklist Items
	{
		ID:   "trello:get_checklist_items",
		Name: "get_checklist_items",
		Descriptions: modules.LocalizedText{
			"en-US": "Get all items in a checklist.",
			"ja-JP": "チェックリスト内のすべてのアイテムを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"checklist_id": {Type: "string", Description: "Checklist ID"},
			},
			Required: []string{"checklist_id"},
		},
	},
	{
		ID:   "trello:add_checklist_item",
		Name: "add_checklist_item",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a new item to a checklist.",
			"ja-JP": "チェックリストに新しいアイテムを追加します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"checklist_id": {Type: "string", Description: "Checklist ID"},
				"name":         {Type: "string", Description: "Item name/text"},
				"pos":          {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
				"checked":      {Type: "boolean", Description: "Initial checked state (default: false)"},
			},
			Required: []string{"checklist_id", "name"},
		},
	},
	{
		ID:   "trello:update_checklist_item",
		Name: "update_checklist_item",
		Descriptions: modules.LocalizedText{
			"en-US": "Update a checklist item (name or state).",
			"ja-JP": "チェックリストアイテムを更新します（名前または状態）。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"card_id":  {Type: "string", Description: "Card ID that contains the checklist"},
				"item_id":  {Type: "string", Description: "Checklist item ID"},
				"name":     {Type: "string", Description: "New item name"},
				"state":    {Type: "string", Description: "State: 'complete' or 'incomplete'"},
				"pos":      {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
			},
			Required: []string{"card_id", "item_id"},
		},
	},
	{
		ID:   "trello:delete_checklist_item",
		Name: "delete_checklist_item",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a checklist item.",
			"ja-JP": "チェックリストアイテムを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"checklist_id": {Type: "string", Description: "Checklist ID"},
				"item_id":      {Type: "string", Description: "Checklist item ID"},
			},
			Required: []string{"checklist_id", "item_id"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	// Boards
	"list_boards": listBoards,
	"get_board":   getBoard,
	// Lists
	"get_lists": getLists,
	// Cards
	"get_cards":    getCards,
	"get_card":     getCard,
	"create_card":  createCard,
	"update_card":  updateCard,
	"move_card":    moveCard,
	"delete_card":  deleteCard,
	// Checklists
	"get_checklists":        getChecklists,
	"create_checklist":      createChecklist,
	"delete_checklist":      deleteChecklist,
	"get_checklist_items":   getChecklistItems,
	"add_checklist_item":    addChecklistItem,
	"update_checklist_item": updateChecklistItem,
	"delete_checklist_item": deleteChecklistItem,
}

// =============================================================================
// Boards
// =============================================================================

func listBoards(ctx context.Context, params map[string]any) (string, error) {
	endpoint := addAuth(trelloAPIBase+"/members/me/boards?fields=id,name,desc,url,closed", ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getBoard(ctx context.Context, params map[string]any) (string, error) {
	boardID, _ := params["board_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/boards/%s?fields=id,name,desc,url,closed", trelloAPIBase, url.PathEscape(boardID)), ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Lists
// =============================================================================

func getLists(ctx context.Context, params map[string]any) (string, error) {
	boardID, _ := params["board_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/boards/%s/lists?fields=id,name,closed,pos", trelloAPIBase, url.PathEscape(boardID)), ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Cards
// =============================================================================

func getCards(ctx context.Context, params map[string]any) (string, error) {
	listID, hasListID := params["list_id"].(string)
	boardID, hasBoardID := params["board_id"].(string)

	var endpoint string
	if hasListID && listID != "" {
		endpoint = fmt.Sprintf("%s/lists/%s/cards?fields=id,name,desc,due,closed,pos,labels", trelloAPIBase, url.PathEscape(listID))
	} else if hasBoardID && boardID != "" {
		endpoint = fmt.Sprintf("%s/boards/%s/cards?fields=id,name,desc,due,closed,pos,labels,idList", trelloAPIBase, url.PathEscape(boardID))
	} else {
		return "", fmt.Errorf("either board_id or list_id is required")
	}

	endpoint = addAuth(endpoint, ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getCard(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/cards/%s?fields=id,name,desc,due,closed,pos,labels,idList,idBoard&checklists=all", trelloAPIBase, url.PathEscape(cardID)), ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createCard(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	name, _ := params["name"].(string)

	query := url.Values{}
	query.Set("idList", listID)
	query.Set("name", name)

	if desc, ok := params["desc"].(string); ok && desc != "" {
		query.Set("desc", desc)
	}
	if pos, ok := params["pos"].(string); ok && pos != "" {
		query.Set("pos", pos)
	}
	if due, ok := params["due"].(string); ok && due != "" {
		query.Set("due", due)
	}
	if labels, ok := params["labels"].(string); ok && labels != "" {
		query.Set("idLabels", labels)
	}
	if memberIDs, ok := params["member_ids"].(string); ok && memberIDs != "" {
		query.Set("idMembers", memberIDs)
	}

	endpoint := addAuth(trelloAPIBase+"/cards?"+query.Encode(), ctx)
	respBody, err := client.DoJSON("POST", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateCard(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)

	query := url.Values{}

	if name, ok := params["name"].(string); ok && name != "" {
		query.Set("name", name)
	}
	if desc, ok := params["desc"].(string); ok {
		query.Set("desc", desc)
	}
	if closed, ok := params["closed"].(bool); ok {
		if closed {
			query.Set("closed", "true")
		} else {
			query.Set("closed", "false")
		}
	}
	if due, ok := params["due"].(string); ok && due != "" {
		query.Set("due", due)
	}
	if listID, ok := params["list_id"].(string); ok && listID != "" {
		query.Set("idList", listID)
	}

	endpoint := addAuth(fmt.Sprintf("%s/cards/%s?%s", trelloAPIBase, url.PathEscape(cardID), query.Encode()), ctx)
	respBody, err := client.DoJSON("PUT", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func moveCard(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)
	listID, _ := params["list_id"].(string)

	query := url.Values{}
	query.Set("idList", listID)

	if pos, ok := params["pos"].(string); ok && pos != "" {
		query.Set("pos", pos)
	}

	endpoint := addAuth(fmt.Sprintf("%s/cards/%s?%s", trelloAPIBase, url.PathEscape(cardID), query.Encode()), ctx)
	respBody, err := client.DoJSON("PUT", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteCard(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/cards/%s", trelloAPIBase, url.PathEscape(cardID)), ctx)
	_, err := client.DoJSON("DELETE", endpoint, nil, nil)
	if err != nil {
		// Check if it's a 200 OK (success)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 200 {
			return `{"success": true, "message": "Card deleted"}`, nil
		}
		return "", err
	}
	return `{"success": true, "message": "Card deleted"}`, nil
}

// =============================================================================
// Checklists
// =============================================================================

func getChecklists(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/cards/%s/checklists", trelloAPIBase, url.PathEscape(cardID)), ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createChecklist(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)
	name, _ := params["name"].(string)

	query := url.Values{}
	query.Set("idCard", cardID)
	query.Set("name", name)

	if pos, ok := params["pos"].(string); ok && pos != "" {
		query.Set("pos", pos)
	}

	endpoint := addAuth(trelloAPIBase+"/checklists?"+query.Encode(), ctx)
	respBody, err := client.DoJSON("POST", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteChecklist(ctx context.Context, params map[string]any) (string, error) {
	checklistID, _ := params["checklist_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/checklists/%s", trelloAPIBase, url.PathEscape(checklistID)), ctx)
	_, err := client.DoJSON("DELETE", endpoint, nil, nil)
	if err != nil {
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 200 {
			return `{"success": true, "message": "Checklist deleted"}`, nil
		}
		return "", err
	}
	return `{"success": true, "message": "Checklist deleted"}`, nil
}

// =============================================================================
// Checklist Items
// =============================================================================

func getChecklistItems(ctx context.Context, params map[string]any) (string, error) {
	checklistID, _ := params["checklist_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/checklists/%s/checkItems", trelloAPIBase, url.PathEscape(checklistID)), ctx)
	respBody, err := client.DoJSON("GET", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func addChecklistItem(ctx context.Context, params map[string]any) (string, error) {
	checklistID, _ := params["checklist_id"].(string)
	name, _ := params["name"].(string)

	query := url.Values{}
	query.Set("name", name)

	if pos, ok := params["pos"].(string); ok && pos != "" {
		query.Set("pos", pos)
	}
	if checked, ok := params["checked"].(bool); ok && checked {
		query.Set("checked", "true")
	}

	endpoint := addAuth(fmt.Sprintf("%s/checklists/%s/checkItems?%s", trelloAPIBase, url.PathEscape(checklistID), query.Encode()), ctx)
	respBody, err := client.DoJSON("POST", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateChecklistItem(ctx context.Context, params map[string]any) (string, error) {
	cardID, _ := params["card_id"].(string)
	itemID, _ := params["item_id"].(string)

	query := url.Values{}

	if name, ok := params["name"].(string); ok && name != "" {
		query.Set("name", name)
	}
	if state, ok := params["state"].(string); ok && state != "" {
		query.Set("state", state)
	}
	if pos, ok := params["pos"].(string); ok && pos != "" {
		query.Set("pos", pos)
	}

	endpoint := addAuth(fmt.Sprintf("%s/cards/%s/checkItem/%s?%s", trelloAPIBase, url.PathEscape(cardID), url.PathEscape(itemID), query.Encode()), ctx)
	respBody, err := client.DoJSON("PUT", endpoint, nil, nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteChecklistItem(ctx context.Context, params map[string]any) (string, error) {
	checklistID, _ := params["checklist_id"].(string)
	itemID, _ := params["item_id"].(string)
	endpoint := addAuth(fmt.Sprintf("%s/checklists/%s/checkItems/%s", trelloAPIBase, url.PathEscape(checklistID), url.PathEscape(itemID)), ctx)
	_, err := client.DoJSON("DELETE", endpoint, nil, nil)
	if err != nil {
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 200 {
			return `{"success": true, "message": "Checklist item deleted"}`, nil
		}
		return "", err
	}
	return `{"success": true, "message": "Checklist item deleted"}`, nil
}
