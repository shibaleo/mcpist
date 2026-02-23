package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/airtableapi"
	gen "mcpist/server/pkg/airtableapi/gen"
)

const (
	airtableAPIVersion = "v0"
)

// AirtableModule implements the Module interface for Airtable API
type AirtableModule struct{}

func New() *AirtableModule { return &AirtableModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Airtable API - Bases, Tables, Records operations with search",
	"ja-JP": "Airtable API - ベース、テーブル、レコード操作（検索機能付き）",
}

func (m *AirtableModule) Name() string                        { return "airtable" }
func (m *AirtableModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *AirtableModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *AirtableModule) APIVersion() string           { return airtableAPIVersion }
func (m *AirtableModule) Tools() []modules.Tool        { return toolDefinitions }
func (m *AirtableModule) Resources() []modules.Resource { return nil }
func (m *AirtableModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *AirtableModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

func (m *AirtableModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "airtable")
	if err != nil {
		return nil
	}
	return credentials
}

// =============================================================================
// ogen client helpers
// =============================================================================

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return airtableapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Base
	{
		ID:   "airtable:list_bases",
		Name: "list_bases",
		Descriptions: modules.LocalizedText{
			"en-US": "List all accessible Airtable bases with their names, IDs, and permission levels",
			"ja-JP": "アクセス可能なすべてのAirtableベースをその名前、ID、権限レベルとともに一覧表示します",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	// Schema
	{
		ID:   "airtable:get_base_tables",
		Name: "get_base_tables",
		Descriptions: modules.LocalizedText{
			"en-US": "List all tables in a base (id and name only)",
			"ja-JP": "ベース内のすべてのテーブルを一覧表示します（IDと名前のみ）",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id": {Type: "string", Description: "Base ID (starts with 'app')"},
			},
			Required: []string{"base_id"},
		},
	},
	{
		ID:   "airtable:get_table_fields",
		Name: "get_table_fields",
		Descriptions: modules.LocalizedText{
			"en-US": "Get field definitions for a specific table (id, name, type, options)",
			"ja-JP": "特定テーブルのフィールド定義を取得します（ID、名前、タイプ、オプション）",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id": {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":   {Type: "string", Description: "Table name or ID"},
			},
			Required: []string{"base_id", "table"},
		},
	},
	{
		ID:   "airtable:get_table_views",
		Name: "get_table_views",
		Descriptions: modules.LocalizedText{
			"en-US": "Get views for a specific table (id, name, type)",
			"ja-JP": "特定テーブルのビューを取得します（ID、名前、タイプ）",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id": {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":   {Type: "string", Description: "Table name or ID"},
			},
			Required: []string{"base_id", "table"},
		},
	},
	{
		ID:   "airtable:create_table",
		Name: "create_table",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new table in a base with specified fields",
			"ja-JP": "指定されたフィールドでベースに新しいテーブルを作成します",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":     {Type: "string", Description: "Base ID (starts with 'app')"},
				"name":        {Type: "string", Description: "Table name"},
				"description": {Type: "string", Description: "Optional table description"},
				"fields":      {Type: "array", Description: "Array of field definitions: [{name, type, description?, options?}]"},
			},
			Required: []string{"base_id", "name", "fields"},
		},
	},
	{
		ID:   "airtable:update_table",
		Name: "update_table",
		Descriptions: modules.LocalizedText{
			"en-US": "Update table metadata (name and/or description)",
			"ja-JP": "テーブルのメタデータ（名前および/または説明）を更新します",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":     {Type: "string", Description: "Base ID (starts with 'app')"},
				"table_id":    {Type: "string", Description: "Table ID (starts with 'tbl')"},
				"name":        {Type: "string", Description: "New table name"},
				"description": {Type: "string", Description: "New table description"},
			},
			Required: []string{"base_id", "table_id"},
		},
	},
	// Record Read
	{
		ID:   "airtable:list_records",
		Name: "list_records",
		Descriptions: modules.LocalizedText{
			"en-US": "List records in a table with filtering, sorting, and pagination",
			"ja-JP": "フィルタリング、ソート、ページネーションを使用してテーブル内のレコードを一覧表示します",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":           {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":             {Type: "string", Description: "Table name or ID"},
				"filter_by_formula": {Type: "string", Description: "Airtable formula to filter records"},
				"view":              {Type: "string", Description: "View name or ID to use"},
				"sort":              {Type: "array", Description: "Sort configuration: [{field: string, direction: 'asc'|'desc'}] (max 3)"},
				"page_size":         {Type: "number", Description: "Number of records per page (max 100)"},
				"max_records":       {Type: "number", Description: "Maximum number of records to return"},
				"offset":            {Type: "string", Description: "Pagination offset from previous response"},
			},
			Required: []string{"base_id", "table"},
		},
	},
	{
		ID:   "airtable:get_record",
		Name: "get_record",
		Descriptions: modules.LocalizedText{
			"en-US": "Retrieve a single record by ID",
			"ja-JP": "IDで単一のレコードを取得します",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":   {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":     {Type: "string", Description: "Table name or ID"},
				"record_id": {Type: "string", Description: "Record ID (starts with 'rec')"},
			},
			Required: []string{"base_id", "table", "record_id"},
		},
	},
	// Record Write
	{
		ID:   "airtable:create_records",
		Name: "create_records",
		Descriptions: modules.LocalizedText{
			"en-US": "Create new records in a table (max 10 per request)",
			"ja-JP": "テーブルに新しいレコードを作成します（1リクエストあたり最大10件）",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":  {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":    {Type: "string", Description: "Table name or ID"},
				"records":  {Type: "array", Description: "Array of records to create. Each record is {fields: {fieldName: value}}"},
				"typecast": {Type: "boolean", Description: "Automatically typecast field values (default: false)"},
			},
			Required: []string{"base_id", "table", "records"},
		},
	},
	{
		ID:   "airtable:update_records",
		Name: "update_records",
		Descriptions: modules.LocalizedText{
			"en-US": "Update existing records (max 10 per request). Uses PATCH (partial update)",
			"ja-JP": "既存のレコードを更新します（1リクエストあたり最大10件）。PATCH（部分更新）を使用",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":  {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":    {Type: "string", Description: "Table name or ID"},
				"records":  {Type: "array", Description: "Array of records to update. Each record is {id: recordId, fields: {fieldName: value}}"},
				"typecast": {Type: "boolean", Description: "Automatically typecast field values (default: false)"},
			},
			Required: []string{"base_id", "table", "records"},
		},
	},
	{
		ID:   "airtable:delete_records",
		Name: "delete_records",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete records from a table (max 10 per request)",
			"ja-JP": "テーブルからレコードを削除します（1リクエストあたり最大10件）",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":    {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":      {Type: "string", Description: "Table name or ID"},
				"record_ids": {Type: "array", Description: "Array of record IDs to delete (max 10)"},
			},
			Required: []string{"base_id", "table", "record_ids"},
		},
	},
	// Search
	{
		ID:   "airtable:search_records",
		Name: "search_records",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for records containing specific text across specified fields",
			"ja-JP": "指定されたフィールドで特定のテキストを含むレコードを検索します",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":     {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":       {Type: "string", Description: "Table name or ID"},
				"search_term": {Type: "string", Description: "Text to search for (case-insensitive)"},
				"fields":      {Type: "array", Description: "Specific field names to search in (required)"},
				"max_records": {Type: "number", Description: "Maximum records to return (default: 100)"},
			},
			Required: []string{"base_id", "table", "search_term", "fields"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_bases":       listBases,
	"get_base_tables":  getBaseTables,
	"get_table_fields": getTableFields,
	"get_table_views":  getTableViews,
	"create_table":     createTable,
	"update_table":     updateTable,
	"list_records":     listRecords,
	"get_record":       getRecord,
	"create_records":   createRecords,
	"update_records":   updateRecords,
	"delete_records":   deleteRecords,
	"search_records":   searchRecords,
}

// =============================================================================
// Base Operations
// =============================================================================

func listBases(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListBases(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res.Bases)
}

// =============================================================================
// Schema Operations
// =============================================================================

func getBaseTables(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListTables(ctx, gen.ListTablesParams{BaseId: baseID})
	if err != nil {
		return "", err
	}
	// Extract only id and name from each table
	tables := make([]map[string]string, 0, len(res.Tables))
	for _, t := range res.Tables {
		tables = append(tables, map[string]string{
			"id":   t.ID.Value,
			"name": t.Name.Value,
		})
	}
	return toJSON(tables)
}

func getTableFields(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	tableName, _ := params["table"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListTables(ctx, gen.ListTablesParams{BaseId: baseID})
	if err != nil {
		return "", err
	}
	for _, t := range res.Tables {
		if t.ID.Value == tableName || t.Name.Value == tableName {
			return toJSON(t.Fields)
		}
	}
	return "", fmt.Errorf("table '%s' not found", tableName)
}

func getTableViews(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	tableName, _ := params["table"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListTables(ctx, gen.ListTablesParams{BaseId: baseID})
	if err != nil {
		return "", err
	}
	for _, t := range res.Tables {
		if t.ID.Value == tableName || t.Name.Value == tableName {
			return toJSON(t.Views)
		}
	}
	return "", fmt.Errorf("table '%s' not found", tableName)
}

func createTable(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	name, _ := params["name"].(string)
	fieldsRaw, _ := params["fields"].([]interface{})

	// Convert fields to ogen types
	fields := make([]gen.Field, 0, len(fieldsRaw))
	for _, f := range fieldsRaw {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		field := gen.Field{}
		if n, ok := fm["name"].(string); ok {
			field.Name.SetTo(n)
		}
		if t, ok := fm["type"].(string); ok {
			field.Type.SetTo(t)
		}
		if d, ok := fm["description"].(string); ok {
			field.Description.SetTo(d)
		}
		fields = append(fields, field)
	}

	req := &gen.CreateTableReq{
		Name:   name,
		Fields: fields,
	}
	if desc, ok := params["description"].(string); ok && desc != "" {
		req.Description.SetTo(desc)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateTable(ctx, req, gen.CreateTableParams{BaseId: baseID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateTable(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	tableID, _ := params["table_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := &gen.UpdateTableReq{}
	if name, ok := params["name"].(string); ok && name != "" {
		req.Name.SetTo(name)
	}
	if desc, ok := params["description"].(string); ok {
		req.Description.SetTo(desc)
	}
	res, err := c.UpdateTable(ctx, req, gen.UpdateTableParams{BaseId: baseID, TableId: tableID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Record Operations
// =============================================================================

func listRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	table, _ := params["table"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.ListRecordsParams{BaseId: baseID, TableIdOrName: table}
	if formula, ok := params["filter_by_formula"].(string); ok && formula != "" {
		p.FilterByFormula.SetTo(formula)
	}
	if view, ok := params["view"].(string); ok && view != "" {
		p.View.SetTo(view)
	}
	if pageSize, ok := params["page_size"].(float64); ok {
		p.PageSize.SetTo(int(pageSize))
	}
	if maxRecords, ok := params["max_records"].(float64); ok {
		p.MaxRecords.SetTo(int(maxRecords))
	}
	if offset, ok := params["offset"].(string); ok && offset != "" {
		p.Offset.SetTo(offset)
	}
	// Sort parameters (max 3)
	if sorts, ok := params["sort"].([]interface{}); ok {
		for i, s := range sorts {
			if i >= 3 {
				break
			}
			sort, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			field, _ := sort["field"].(string)
			direction := "asc"
			if dir, ok := sort["direction"].(string); ok {
				direction = dir
			}
			switch i {
			case 0:
				p.Sort0Field.SetTo(field)
				p.Sort0Direction.SetTo(direction)
			case 1:
				p.Sort1Field.SetTo(field)
				p.Sort1Direction.SetTo(direction)
			case 2:
				p.Sort2Field.SetTo(field)
				p.Sort2Direction.SetTo(direction)
			}
		}
	}

	res, err := c.ListRecords(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getRecord(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	table, _ := params["table"].(string)
	recordID, _ := params["record_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetRecord(ctx, gen.GetRecordParams{
		BaseId: baseID, TableIdOrName: table, RecordId: recordID,
	})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	table, _ := params["table"].(string)
	recordsRaw, ok := params["records"].([]interface{})
	if !ok || len(recordsRaw) == 0 {
		return "", fmt.Errorf("records is required and must not be empty")
	}

	// Build ogen request - records contain {fields: {...}} objects
	// RecordFields is map[string]jx.Raw with JSON marshal support, so use JSON roundtrip.
	records := make([]gen.CreateRecordsReqRecordsItem, 0, len(recordsRaw))
	for _, r := range recordsRaw {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		item := gen.CreateRecordsReqRecordsItem{}
		if _, ok := rm["fields"]; ok {
			fieldsJSON, _ := json.Marshal(rm["fields"])
			var rf gen.RecordFields
			_ = json.Unmarshal(fieldsJSON, &rf)
			item.Fields.SetTo(rf)
		}
		records = append(records, item)
	}

	req := &gen.CreateRecordsReq{Records: records}
	if typecast, ok := params["typecast"].(bool); ok {
		req.Typecast.SetTo(typecast)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateRecords(ctx, req, gen.CreateRecordsParams{
		BaseId: baseID, TableIdOrName: table,
	})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	table, _ := params["table"].(string)
	recordsRaw, ok := params["records"].([]interface{})
	if !ok || len(recordsRaw) == 0 {
		return "", fmt.Errorf("records is required and must not be empty")
	}

	records := make([]gen.UpdateRecordsReqRecordsItem, 0, len(recordsRaw))
	for _, r := range recordsRaw {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		item := gen.UpdateRecordsReqRecordsItem{}
		if id, ok := rm["id"].(string); ok {
			item.ID.SetTo(id)
		}
		if _, ok := rm["fields"]; ok {
			fieldsJSON, _ := json.Marshal(rm["fields"])
			var rf gen.RecordFields
			_ = json.Unmarshal(fieldsJSON, &rf)
			item.Fields.SetTo(rf)
		}
		records = append(records, item)
	}

	req := &gen.UpdateRecordsReq{Records: records}
	if typecast, ok := params["typecast"].(bool); ok {
		req.Typecast.SetTo(typecast)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.UpdateRecords(ctx, req, gen.UpdateRecordsParams{
		BaseId: baseID, TableIdOrName: table,
	})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	table, _ := params["table"].(string)
	recordIDsRaw, ok := params["record_ids"].([]interface{})
	if !ok || len(recordIDsRaw) == 0 {
		return "", fmt.Errorf("record_ids is required and must not be empty")
	}

	ids := make([]string, 0, len(recordIDsRaw))
	for _, id := range recordIDsRaw {
		if s, ok := id.(string); ok {
			ids = append(ids, s)
		}
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.DeleteRecords(ctx, gen.DeleteRecordsParams{
		BaseId: baseID, TableIdOrName: table, Records: ids,
	})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Search
// =============================================================================

func searchRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	table, _ := params["table"].(string)
	searchTerm, _ := params["search_term"].(string)

	fields, ok := params["fields"].([]interface{})
	if !ok || len(fields) == 0 {
		return "", fmt.Errorf("fields is required for search")
	}

	maxRecords := 100
	if mr, ok := params["max_records"].(float64); ok {
		maxRecords = int(mr)
	}

	// Build FIND formula for case-insensitive search across specified fields
	conditions := make([]string, 0, len(fields))
	for _, f := range fields {
		if fieldName, ok := f.(string); ok {
			conditions = append(conditions, fmt.Sprintf(
				"FIND(LOWER('%s'), LOWER({%s}))",
				escapeFormulaString(searchTerm), fieldName,
			))
		}
	}
	formula := "OR(" + strings.Join(conditions, ", ") + ")"

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.ListRecordsParams{
		BaseId:        baseID,
		TableIdOrName: table,
	}
	p.FilterByFormula.SetTo(formula)
	p.MaxRecords.SetTo(maxRecords)

	res, err := c.ListRecords(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func escapeFormulaString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
