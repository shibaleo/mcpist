package trello

import (
	"context"
	"fmt"
	"log"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/trelloapi"
	gen "mcpist/server/pkg/trelloapi/gen"
)

const trelloVersion = "1"

var toJSON = modules.ToJSON

// TrelloModule implements the Module interface for Trello API
type TrelloModule struct{}

// New creates a new TrelloModule instance
func New() *TrelloModule {
	return &TrelloModule{}
}

// Module descriptions
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

// Description returns the module description (English)
func (m *TrelloModule) Description() string {
	return moduleDescriptions["en-US"]
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

// ToCompact converts JSON result to compact format.
// Implements modules.CompactConverter interface.
func (m *TrelloModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for Trello)
func (m *TrelloModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *TrelloModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// =============================================================================
// Client / Auth
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[trello] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "trello")
	if err != nil {
		log.Printf("[trello] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	if creds.ConsumerKey == "" || creds.AccessToken == "" {
		return nil, fmt.Errorf("missing API key or token")
	}
	return trelloapi.NewClient(creds.ConsumerKey, creds.AccessToken)
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
			Type: "object",
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
				"list_id":    {Type: "string", Description: "List ID to add the card to"},
				"name":       {Type: "string", Description: "Card name/title"},
				"desc":       {Type: "string", Description: "Card description"},
				"pos":        {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
				"due":        {Type: "string", Description: "Due date (ISO 8601 format)"},
				"labels":     {Type: "string", Description: "Comma-separated label IDs"},
				"member_ids": {Type: "string", Description: "Comma-separated member IDs"},
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
				"card_id": {Type: "string", Description: "Card ID"},
				"name":    {Type: "string", Description: "New card name"},
				"desc":    {Type: "string", Description: "New card description"},
				"closed":  {Type: "boolean", Description: "Archive the card"},
				"due":     {Type: "string", Description: "Due date (ISO 8601 format)"},
				"list_id": {Type: "string", Description: "Move to different list"},
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
				"pos": {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
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
				"pos": {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
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
				"checked": {Type: "boolean", Description: "Initial checked state (default: false)"},
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
				"card_id": {Type: "string", Description: "Card ID that contains the checklist"},
				"item_id": {Type: "string", Description: "Checklist item ID"},
				"name":    {Type: "string", Description: "New item name"},
				"state":   {Type: "string", Description: "State: 'complete' or 'incomplete'"},
				"pos": {Type: "string", Description: "Position: 'top', 'bottom', or a positive number"},
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
	"list_boards":           listBoards,
	"get_board":             getBoard,
	"get_lists":             getLists,
	"get_cards":             getCards,
	"get_card":              getCard,
	"create_card":           createCard,
	"update_card":           updateCard,
	"move_card":             moveCard,
	"delete_card":           deleteCard,
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
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.ListBoardsParams{}
	p.Fields.SetTo("id,name,desc,url,closed")
	res, err := c.ListBoards(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func getBoard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	boardID, _ := params["board_id"].(string)
	p := gen.GetBoardParams{BoardId: boardID}
	p.Fields.SetTo("id,name,desc,url,closed")
	res, err := c.GetBoard(ctx, p)
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
// Lists
// =============================================================================

func getLists(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	boardID, _ := params["board_id"].(string)
	p := gen.GetListsParams{BoardId: boardID}
	p.Fields.SetTo("id,name,closed,pos")
	res, err := c.GetLists(ctx, p)
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
// Cards
// =============================================================================

func getCards(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	listID, hasListID := params["list_id"].(string)
	boardID, hasBoardID := params["board_id"].(string)

	if hasListID && listID != "" {
		p := gen.GetCardsByListParams{ListId: listID}
		p.Fields.SetTo("id,name,desc,due,closed,pos,labels")
		res, err := c.GetCardsByList(ctx, p)
		if err != nil {
			return "", err
		}
		jsonStr, err := toJSON(res)
		if err != nil {
			return "", err
		}
		return jsonStr, nil
	}

	if hasBoardID && boardID != "" {
		p := gen.GetCardsByBoardParams{BoardId: boardID}
		p.Fields.SetTo("id,name,desc,due,closed,pos,labels,idList")
		res, err := c.GetCardsByBoard(ctx, p)
		if err != nil {
			return "", err
		}
		jsonStr, err := toJSON(res)
		if err != nil {
			return "", err
		}
		return jsonStr, nil
	}

	return "", fmt.Errorf("either board_id or list_id is required")
}

func getCard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)
	p := gen.GetCardParams{CardId: cardID}
	p.Fields.SetTo("id,name,desc,due,closed,pos,labels,idList,idBoard")
	p.Checklists.SetTo("all")
	res, err := c.GetCard(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func createCard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	listID, _ := params["list_id"].(string)
	name, _ := params["name"].(string)

	p := gen.CreateCardParams{IdList: listID, Name: name}
	if v, ok := params["desc"].(string); ok && v != "" {
		p.Desc.SetTo(v)
	}
	if v, ok := params["pos"].(string); ok && v != "" {
		p.Pos.SetTo(v)
	}
	if v, ok := params["due"].(string); ok && v != "" {
		p.Due.SetTo(v)
	}
	if v, ok := params["labels"].(string); ok && v != "" {
		p.IdLabels.SetTo(v)
	}
	if v, ok := params["member_ids"].(string); ok && v != "" {
		p.IdMembers.SetTo(v)
	}

	res, err := c.CreateCard(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func updateCard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)

	p := gen.UpdateCardParams{CardId: cardID}
	if v, ok := params["name"].(string); ok && v != "" {
		p.Name.SetTo(v)
	}
	if v, ok := params["desc"].(string); ok {
		p.Desc.SetTo(v)
	}
	if closed, ok := params["closed"].(bool); ok {
		if closed {
			p.Closed.SetTo("true")
		} else {
			p.Closed.SetTo("false")
		}
	}
	if v, ok := params["due"].(string); ok && v != "" {
		p.Due.SetTo(v)
	}
	if v, ok := params["list_id"].(string); ok && v != "" {
		p.IdList.SetTo(v)
	}

	res, err := c.UpdateCard(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func moveCard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)
	listID, _ := params["list_id"].(string)

	p := gen.UpdateCardParams{CardId: cardID}
	p.IdList.SetTo(listID)
	if v, ok := params["pos"].(string); ok && v != "" {
		p.Pos.SetTo(v)
	}

	res, err := c.UpdateCard(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func deleteCard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)
	err = c.DeleteCard(ctx, gen.DeleteCardParams{CardId: cardID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Card deleted"}`, nil
}

// =============================================================================
// Checklists
// =============================================================================

func getChecklists(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)
	res, err := c.GetChecklists(ctx, gen.GetChecklistsParams{CardId: cardID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func createChecklist(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)
	name, _ := params["name"].(string)

	p := gen.CreateChecklistParams{IdCard: cardID, Name: name}
	if v, ok := params["pos"].(string); ok && v != "" {
		p.Pos.SetTo(v)
	}

	res, err := c.CreateChecklist(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func deleteChecklist(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	checklistID, _ := params["checklist_id"].(string)
	err = c.DeleteChecklist(ctx, gen.DeleteChecklistParams{ChecklistId: checklistID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Checklist deleted"}`, nil
}

// =============================================================================
// Checklist Items
// =============================================================================

func getChecklistItems(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	checklistID, _ := params["checklist_id"].(string)
	res, err := c.GetChecklistItems(ctx, gen.GetChecklistItemsParams{ChecklistId: checklistID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func addChecklistItem(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	checklistID, _ := params["checklist_id"].(string)
	name, _ := params["name"].(string)

	p := gen.AddChecklistItemParams{ChecklistId: checklistID, Name: name}
	if v, ok := params["pos"].(string); ok && v != "" {
		p.Pos.SetTo(v)
	}
	if checked, ok := params["checked"].(bool); ok && checked {
		p.Checked.SetTo("true")
	}

	res, err := c.AddChecklistItem(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func updateChecklistItem(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	cardID, _ := params["card_id"].(string)
	itemID, _ := params["item_id"].(string)

	p := gen.UpdateChecklistItemParams{CardId: cardID, CheckItemId: itemID}
	if v, ok := params["name"].(string); ok && v != "" {
		p.Name.SetTo(v)
	}
	if v, ok := params["state"].(string); ok && v != "" {
		p.State.SetTo(v)
	}
	if v, ok := params["pos"].(string); ok && v != "" {
		p.Pos.SetTo(v)
	}

	res, err := c.UpdateChecklistItem(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func deleteChecklistItem(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	checklistID, _ := params["checklist_id"].(string)
	itemID, _ := params["item_id"].(string)
	err = c.DeleteChecklistItem(ctx, gen.DeleteChecklistItemParams{ChecklistId: checklistID, CheckItemId: itemID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Checklist item deleted"}`, nil
}
