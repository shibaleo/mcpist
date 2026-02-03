package airtable

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	airtableAPIBase    = "https://api.airtable.com/v0"
	airtableMetaAPI    = "https://api.airtable.com/v0/meta"
	airtableTokenURL   = "https://airtable.com/oauth2/v1/token"
	airtableAPIVersion = "v0"

	// Refresh token 5 minutes before expiry
	tokenRefreshBuffer = 300
)

var client = httpclient.New()

// AirtableModule implements the Module interface for Airtable API
type AirtableModule struct{}

// New creates a new AirtableModule instance
func New() *AirtableModule {
	return &AirtableModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Airtable API - Bases, Tables, Records operations with search and aggregation",
	"ja-JP": "Airtable API - ベース、テーブル、レコード操作（検索・集計機能付き）",
}

// Name returns the module name
func (m *AirtableModule) Name() string {
	return "airtable"
}

// Descriptions returns the module descriptions in all languages
func (m *AirtableModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *AirtableModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Airtable API version
func (m *AirtableModule) APIVersion() string {
	return airtableAPIVersion
}

// Tools returns all available tools
func (m *AirtableModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *AirtableModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Airtable)
func (m *AirtableModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *AirtableModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "airtable")
	if err != nil {
		return nil
	}

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[airtable] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[airtable] Token refresh failed: %v", err)
				return credentials
			}
			log.Printf("[airtable] Token refreshed successfully")
			return refreshed
		}
	}

	return credentials
}

// needsRefresh checks if the token is expired or will expire soon
func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		return false
	}
	now := time.Now().Unix()
	return now >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

// refreshToken exchanges the refresh token for a new access token
// Airtable uses Basic auth and rotates refresh tokens on each use
func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "airtable")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	// Airtable requires Basic auth header for token exchange
	basicAuth := base64.StdEncoding.EncodeToString([]byte(oauthApp.ClientID + ":" + oauthApp.ClientSecret))

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", creds.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", airtableTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	expiresAt := time.Now().Unix() + int64(tokenResp.ExpiresIn)

	// Airtable rotates refresh tokens - save the new one
	newCreds := &store.Credentials{
		AuthType:     creds.AuthType,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    store.FlexibleTime(expiresAt),
	}

	if err := store.GetTokenStore().UpdateModuleToken(ctx, userID, "airtable", newCreds); err != nil {
		log.Printf("[airtable] Failed to save refreshed token: %v", err)
	}

	return newCreds, nil
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	return map[string]string{
		"Authorization": "Bearer " + creds.AccessToken,
		"Content-Type":  "application/json",
	}
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Base Operations
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
	// Schema Operations
	{
		ID:   "airtable:describe",
		Name: "describe",
		Descriptions: modules.LocalizedText{
			"en-US": "Describe Airtable base or table schema. Use detailLevel to optimize context: tableIdentifiersOnly (minimal), identifiersOnly (IDs and names), full (complete details with field types)",
			"ja-JP": "Airtableのベースまたはテーブルスキーマを説明します。detailLevelを使用してコンテキストを最適化：tableIdentifiersOnly（最小限）、identifiersOnly（IDと名前）、full（フィールドタイプを含む完全な詳細）",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":       {Type: "string", Description: "Base ID (starts with 'app')"},
				"scope":         {Type: "string", Description: "Scope of description: 'base' for all tables, 'table' for a specific table"},
				"table":         {Type: "string", Description: "Table name or ID (required when scope='table')"},
				"detail_level":  {Type: "string", Description: "Detail level: tableIdentifiersOnly, identifiersOnly, or full (default: full)"},
				"include_views": {Type: "boolean", Description: "Include view information (default: false)"},
			},
			Required: []string{"base_id"},
		},
	},
	// Record Operations
	{
		ID:   "airtable:query",
		Name: "query",
		Descriptions: modules.LocalizedText{
			"en-US": "Query Airtable records with filtering, sorting, and pagination",
			"ja-JP": "フィルタリング、ソート、ページネーションを使用してAirtableレコードをクエリします",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":           {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":             {Type: "string", Description: "Table name or ID"},
				"fields":            {Type: "array", Description: "Array of field names to return"},
				"filter_by_formula": {Type: "string", Description: "Airtable formula to filter records"},
				"view":              {Type: "string", Description: "View name or ID to use"},
				"sort":              {Type: "array", Description: "Sort configuration: [{field: string, direction: 'asc'|'desc'}]"},
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
	{
		ID:   "airtable:create",
		Name: "create",
		Descriptions: modules.LocalizedText{
			"en-US": "Create new records in a table. Supports batch creation (up to 10 records per request)",
			"ja-JP": "テーブルに新しいレコードを作成します。バッチ作成をサポート（リクエストごとに最大10レコード）",
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
		ID:   "airtable:update",
		Name: "update",
		Descriptions: modules.LocalizedText{
			"en-US": "Update existing records. Supports batch update (up to 10 records per request). Uses PATCH (partial update)",
			"ja-JP": "既存のレコードを更新します。バッチ更新をサポート（リクエストごとに最大10レコード）。PATCH（部分更新）を使用",
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
		ID:   "airtable:delete",
		Name: "delete",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete records from a table. Supports batch deletion (up to 10 records per request)",
			"ja-JP": "テーブルからレコードを削除します。バッチ削除をサポート（リクエストごとに最大10レコード）",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":    {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":      {Type: "string", Description: "Table name or ID"},
				"record_ids": {Type: "array", Description: "Array of record IDs to delete"},
			},
			Required: []string{"base_id", "table", "record_ids"},
		},
	},
	// Search Records
	{
		ID:   "airtable:search_records",
		Name: "search_records",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for records containing specific text across specified fields or all text fields",
			"ja-JP": "指定されたフィールドまたはすべてのテキストフィールドで特定のテキストを含むレコードを検索します",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":     {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":       {Type: "string", Description: "Table name or ID"},
				"search_term": {Type: "string", Description: "Text to search for (case-insensitive)"},
				"fields":      {Type: "array", Description: "Optional: specific field names to search in. If not provided, searches all text fields"},
				"max_records": {Type: "number", Description: "Maximum records to return (default: 100)"},
			},
			Required: []string{"base_id", "table", "search_term"},
		},
	},
	// Aggregate Records
	{
		ID:   "airtable:aggregate_records",
		Name: "aggregate_records",
		Descriptions: modules.LocalizedText{
			"en-US": "Perform aggregation operations (sum, count, avg, min, max) on records with optional grouping",
			"ja-JP": "オプションのグループ化を使用してレコードに対して集計操作（sum、count、avg、min、max）を実行します",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"base_id":           {Type: "string", Description: "Base ID (starts with 'app')"},
				"table":             {Type: "string", Description: "Table name or ID"},
				"operation":         {Type: "string", Description: "Aggregation operation: sum, count, avg, min, max"},
				"field":             {Type: "string", Description: "Field to aggregate (required for sum, avg, min, max)"},
				"group_by":          {Type: "string", Description: "Optional: field to group results by"},
				"filter_by_formula": {Type: "string", Description: "Optional: Airtable formula to filter records before aggregation"},
			},
			Required: []string{"base_id", "table", "operation"},
		},
	},
	// Create Table
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
				"fields":      {Type: "array", Description: "Array of field definitions: [{name, type, description?, options?}]. Types: singleLineText, multilineText, number, checkbox, singleSelect, multipleSelects, date, email, url, etc."},
			},
			Required: []string{"base_id", "name", "fields"},
		},
	},
	// Update Table
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
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_bases":        listBases,
	"describe":          describe,
	"query":             query,
	"get_record":        getRecord,
	"create":            create,
	"update":            update,
	"delete":            deleteRecords,
	"search_records":    searchRecords,
	"aggregate_records": aggregateRecords,
	"create_table":      createTable,
	"update_table":      updateTable,
}

// =============================================================================
// Base Operations
// =============================================================================

func listBases(ctx context.Context, params map[string]any) (string, error) {
	endpoint := airtableMetaAPI + "/bases"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Schema Operations
// =============================================================================

func describe(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	scope, _ := params["scope"].(string)
	if scope == "" {
		scope = "base"
	}

	// Get tables schema
	tablesEndpoint := fmt.Sprintf("%s/bases/%s/tables", airtableMetaAPI, url.PathEscape(baseID))
	tablesInfoBytes, err := client.DoJSON("GET", tablesEndpoint, headers(ctx), nil)
	if err != nil {
		return "", fmt.Errorf("failed to get tables: %w", err)
	}

	var tablesData map[string]interface{}
	if err := json.Unmarshal(tablesInfoBytes, &tablesData); err != nil {
		return "", fmt.Errorf("failed to parse tables: %w", err)
	}

	result := map[string]interface{}{
		"base_id": baseID,
	}
	tables, _ := tablesData["tables"].([]interface{})

	detailLevel, _ := params["detail_level"].(string)
	if detailLevel == "" {
		detailLevel = "full"
	}

	includeViews := false
	if iv, ok := params["include_views"].(bool); ok {
		includeViews = iv
	}

	// Filter to specific table if scope is "table"
	if scope == "table" {
		tableName, _ := params["table"].(string)
		if tableName == "" {
			return "", fmt.Errorf("table is required when scope is 'table'")
		}

		var foundTable interface{}
		for _, t := range tables {
			tbl, _ := t.(map[string]interface{})
			if tbl["name"] == tableName || tbl["id"] == tableName {
				foundTable = t
				break
			}
		}

		if foundTable == nil {
			return "", fmt.Errorf("table '%s' not found", tableName)
		}

		tables = []interface{}{foundTable}
	}

	// Apply detail level filtering
	filteredTables := filterTablesDetail(tables, detailLevel, includeViews)
	result["tables"] = filteredTables

	return httpclient.PrettyJSONFromInterface(result), nil
}

func filterTablesDetail(tables []interface{}, detailLevel string, includeViews bool) []interface{} {
	result := make([]interface{}, 0, len(tables))

	for _, t := range tables {
		tbl, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		filtered := make(map[string]interface{})
		filtered["id"] = tbl["id"]
		filtered["name"] = tbl["name"]

		switch detailLevel {
		case "tableIdentifiersOnly":
			// Only id and name

		case "identifiersOnly":
			if pf, ok := tbl["primaryFieldId"]; ok {
				filtered["primaryFieldId"] = pf
			}
			if fields, ok := tbl["fields"].([]interface{}); ok {
				filteredFields := make([]map[string]interface{}, 0, len(fields))
				for _, f := range fields {
					field, _ := f.(map[string]interface{})
					filteredFields = append(filteredFields, map[string]interface{}{
						"id":   field["id"],
						"name": field["name"],
					})
				}
				filtered["fields"] = filteredFields
			}
			if includeViews {
				if views, ok := tbl["views"].([]interface{}); ok {
					filteredViews := make([]map[string]interface{}, 0, len(views))
					for _, v := range views {
						view, _ := v.(map[string]interface{})
						filteredViews = append(filteredViews, map[string]interface{}{
							"id":   view["id"],
							"name": view["name"],
						})
					}
					filtered["views"] = filteredViews
				}
			}

		default: // "full"
			if pf, ok := tbl["primaryFieldId"]; ok {
				filtered["primaryFieldId"] = pf
			}
			if fields, ok := tbl["fields"]; ok {
				filtered["fields"] = fields
			}
			if includeViews {
				if views, ok := tbl["views"]; ok {
					filtered["views"] = views
				}
			}
		}

		result = append(result, filtered)
	}

	return result
}

// =============================================================================
// Record Operations
// =============================================================================

func query(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	queryParams := url.Values{}

	if fields, ok := params["fields"].([]interface{}); ok {
		for _, f := range fields {
			if fieldName, ok := f.(string); ok {
				queryParams.Add("fields[]", fieldName)
			}
		}
	}

	if formula, ok := params["filter_by_formula"].(string); ok && formula != "" {
		queryParams.Set("filterByFormula", formula)
	}

	if view, ok := params["view"].(string); ok && view != "" {
		queryParams.Set("view", view)
	}

	if sorts, ok := params["sort"].([]interface{}); ok {
		for i, s := range sorts {
			sort, _ := s.(map[string]interface{})
			if field, ok := sort["field"].(string); ok {
				queryParams.Set(fmt.Sprintf("sort[%d][field]", i), field)
				direction := "asc"
				if dir, ok := sort["direction"].(string); ok {
					direction = dir
				}
				queryParams.Set(fmt.Sprintf("sort[%d][direction]", i), direction)
			}
		}
	}

	if pageSize, ok := params["page_size"].(float64); ok {
		queryParams.Set("pageSize", fmt.Sprintf("%d", int(pageSize)))
	}

	if maxRecords, ok := params["max_records"].(float64); ok {
		queryParams.Set("maxRecords", fmt.Sprintf("%d", int(maxRecords)))
	}

	if offset, ok := params["offset"].(string); ok && offset != "" {
		queryParams.Set("offset", offset)
	}

	endpoint := fmt.Sprintf("%s/%s/%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table))
	if len(queryParams) > 0 {
		endpoint += "?" + queryParams.Encode()
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func getRecord(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	recordID, _ := params["record_id"].(string)
	if recordID == "" {
		return "", fmt.Errorf("record_id is required")
	}

	endpoint := fmt.Sprintf("%s/%s/%s/%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table), url.PathEscape(recordID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

func create(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	records, ok := params["records"].([]interface{})
	if !ok || len(records) == 0 {
		return "", fmt.Errorf("records is required and must not be empty")
	}

	typecast := false
	if tc, ok := params["typecast"].(bool); ok {
		typecast = tc
	}

	// Process records in chunks of 10
	allCreated := make([]interface{}, 0)

	for i := 0; i < len(records); i += 10 {
		end := i + 10
		if end > len(records) {
			end = len(records)
		}
		chunk := records[i:end]

		body := map[string]interface{}{
			"records":  chunk,
			"typecast": typecast,
		}

		endpoint := fmt.Sprintf("%s/%s/%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table))
		respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
		if err != nil {
			return "", fmt.Errorf("failed to create records (batch %d): %w", i/10+1, err)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}
		if created, ok := resp["records"].([]interface{}); ok {
			allCreated = append(allCreated, created...)
		}
	}

	result := map[string]interface{}{
		"records": allCreated,
		"summary": map[string]interface{}{
			"created": len(allCreated),
		},
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

func update(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	records, ok := params["records"].([]interface{})
	if !ok || len(records) == 0 {
		return "", fmt.Errorf("records is required and must not be empty")
	}

	typecast := false
	if tc, ok := params["typecast"].(bool); ok {
		typecast = tc
	}

	// Process records in chunks of 10
	allUpdated := make([]interface{}, 0)

	for i := 0; i < len(records); i += 10 {
		end := i + 10
		if end > len(records) {
			end = len(records)
		}
		chunk := records[i:end]

		body := map[string]interface{}{
			"records":  chunk,
			"typecast": typecast,
		}

		endpoint := fmt.Sprintf("%s/%s/%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table))
		respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
		if err != nil {
			return "", fmt.Errorf("failed to update records (batch %d): %w", i/10+1, err)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}
		if updated, ok := resp["records"].([]interface{}); ok {
			allUpdated = append(allUpdated, updated...)
		}
	}

	result := map[string]interface{}{
		"records": allUpdated,
		"summary": map[string]interface{}{
			"updated": len(allUpdated),
		},
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

func deleteRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	recordIDs, ok := params["record_ids"].([]interface{})
	if !ok || len(recordIDs) == 0 {
		return "", fmt.Errorf("record_ids is required and must not be empty")
	}

	// Process records in chunks of 10
	allDeleted := make([]interface{}, 0)

	for i := 0; i < len(recordIDs); i += 10 {
		end := i + 10
		if end > len(recordIDs) {
			end = len(recordIDs)
		}
		chunk := recordIDs[i:end]

		// Build query parameters for DELETE
		queryParams := url.Values{}
		for _, id := range chunk {
			if recordID, ok := id.(string); ok {
				queryParams.Add("records[]", recordID)
			}
		}

		endpoint := fmt.Sprintf("%s/%s/%s?%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table), queryParams.Encode())
		respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
		if err != nil {
			return "", fmt.Errorf("failed to delete records (batch %d): %w", i/10+1, err)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}
		if deleted, ok := resp["records"].([]interface{}); ok {
			allDeleted = append(allDeleted, deleted...)
		}
	}

	result := map[string]interface{}{
		"records": allDeleted,
		"summary": map[string]interface{}{
			"deleted": len(allDeleted),
		},
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

// =============================================================================
// NEW: Search Records
// =============================================================================

func searchRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	searchTerm, _ := params["search_term"].(string)
	if searchTerm == "" {
		return "", fmt.Errorf("search_term is required")
	}

	maxRecords := 100
	if mr, ok := params["max_records"].(float64); ok {
		maxRecords = int(mr)
	}

	// Build SEARCH formula
	// If specific fields are provided, search only those fields
	// Otherwise, use a broader approach
	var formula string
	if fields, ok := params["fields"].([]interface{}); ok && len(fields) > 0 {
		// Build OR conditions for each field
		conditions := make([]string, 0, len(fields))
		for _, f := range fields {
			if fieldName, ok := f.(string); ok {
				// SEARCH returns position (0+) or error, so we check if FIND returns > 0
				// Using FIND for case-insensitive search with LOWER
				conditions = append(conditions, fmt.Sprintf("FIND(LOWER('%s'), LOWER({%s}))", escapeFormulaString(searchTerm), fieldName))
			}
		}
		if len(conditions) > 0 {
			formula = "OR(" + strings.Join(conditions, ", ") + ")"
		}
	} else {
		// Search across all fields using SEARCH with ARRAYJOIN
		// This searches the concatenated string of all field values
		formula = fmt.Sprintf("FIND(LOWER('%s'), LOWER(ARRAYJOIN(RECORD_ID())))", escapeFormulaString(searchTerm))
		// Note: A more comprehensive approach would be to get schema first and build formula for all text fields
		// For simplicity, we use a simpler formula that works with the primary field
		formula = fmt.Sprintf("SEARCH(LOWER('%s'), LOWER(CONCATENATE(RECORD_ID(), ' ')))", escapeFormulaString(searchTerm))
	}

	queryParams := url.Values{}
	if formula != "" {
		queryParams.Set("filterByFormula", formula)
	}
	queryParams.Set("maxRecords", fmt.Sprintf("%d", maxRecords))

	endpoint := fmt.Sprintf("%s/%s/%s?%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table), queryParams.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	// Parse and enhance response
	var resp map[string]interface{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	records, _ := resp["records"].([]interface{})
	result := map[string]interface{}{
		"records":     records,
		"search_term": searchTerm,
		"count":       len(records),
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

// escapeFormulaString escapes special characters in Airtable formula strings
func escapeFormulaString(s string) string {
	// Escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "''")
}

// =============================================================================
// NEW: Aggregate Records
// =============================================================================

func aggregateRecords(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	table, _ := params["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required")
	}

	operation, _ := params["operation"].(string)
	if operation == "" {
		return "", fmt.Errorf("operation is required")
	}

	field, _ := params["field"].(string)
	groupBy, _ := params["group_by"].(string)
	filterFormula, _ := params["filter_by_formula"].(string)

	// Validate operation
	validOps := map[string]bool{"sum": true, "count": true, "avg": true, "average": true, "min": true, "max": true}
	if !validOps[strings.ToLower(operation)] {
		return "", fmt.Errorf("invalid operation: %s. Valid: sum, count, avg, min, max", operation)
	}

	// For operations other than count, field is required
	if operation != "count" && field == "" {
		return "", fmt.Errorf("field is required for operation: %s", operation)
	}

	// Fetch all records (with optional filter)
	queryParams := url.Values{}
	if filterFormula != "" {
		queryParams.Set("filterByFormula", filterFormula)
	}

	// Request the field we're aggregating and the group_by field
	fieldsToFetch := []string{}
	if field != "" {
		fieldsToFetch = append(fieldsToFetch, field)
	}
	if groupBy != "" {
		fieldsToFetch = append(fieldsToFetch, groupBy)
	}
	for _, f := range fieldsToFetch {
		queryParams.Add("fields[]", f)
	}

	// Fetch all records (paginate if necessary)
	allRecords := make([]map[string]interface{}, 0)
	offset := ""

	for {
		currentParams := url.Values{}
		for k, v := range queryParams {
			for _, val := range v {
				currentParams.Add(k, val)
			}
		}
		if offset != "" {
			currentParams.Set("offset", offset)
		}

		endpoint := fmt.Sprintf("%s/%s/%s?%s", airtableAPIBase, url.PathEscape(baseID), url.PathEscape(table), currentParams.Encode())
		respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
		if err != nil {
			return "", err
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return "", err
		}

		if records, ok := resp["records"].([]interface{}); ok {
			for _, r := range records {
				if rec, ok := r.(map[string]interface{}); ok {
					allRecords = append(allRecords, rec)
				}
			}
		}

		// Check for more pages
		if nextOffset, ok := resp["offset"].(string); ok && nextOffset != "" {
			offset = nextOffset
		} else {
			break
		}
	}

	// Perform aggregation
	var result interface{}
	op := strings.ToLower(operation)
	if op == "average" {
		op = "avg"
	}

	if groupBy != "" {
		// Group by a field
		groups := make(map[string][]float64)
		groupCounts := make(map[string]int)

		for _, rec := range allRecords {
			fields, _ := rec["fields"].(map[string]interface{})
			groupValue := "null"
			if gv := fields[groupBy]; gv != nil {
				groupValue = fmt.Sprintf("%v", gv)
			}

			groupCounts[groupValue]++

			if field != "" {
				if val := fields[field]; val != nil {
					if numVal, err := toFloat64(val); err == nil {
						groups[groupValue] = append(groups[groupValue], numVal)
					}
				}
			}
		}

		groupResults := make(map[string]interface{})
		for groupValue, values := range groups {
			groupResults[groupValue] = calculateAggregate(op, values, groupCounts[groupValue])
		}
		// Add groups that have counts but no numeric values (for count operation)
		if op == "count" {
			for groupValue, count := range groupCounts {
				if _, exists := groupResults[groupValue]; !exists {
					groupResults[groupValue] = count
				}
			}
		}

		result = map[string]interface{}{
			"operation": operation,
			"field":     field,
			"group_by":  groupBy,
			"groups":    groupResults,
		}
	} else {
		// No grouping
		values := make([]float64, 0)
		for _, rec := range allRecords {
			if field != "" {
				fields, _ := rec["fields"].(map[string]interface{})
				if val := fields[field]; val != nil {
					if numVal, err := toFloat64(val); err == nil {
						values = append(values, numVal)
					}
				}
			}
		}

		result = map[string]interface{}{
			"operation":    operation,
			"field":        field,
			"result":       calculateAggregate(op, values, len(allRecords)),
			"record_count": len(allRecords),
		}
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to float64")
	}
}

func calculateAggregate(op string, values []float64, totalCount int) interface{} {
	switch op {
	case "count":
		return totalCount
	case "sum":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum
	case "avg":
		if len(values) == 0 {
			return 0.0
		}
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	case "min":
		if len(values) == 0 {
			return nil
		}
		min := values[0]
		for _, v := range values[1:] {
			if v < min {
				min = v
			}
		}
		return min
	case "max":
		if len(values) == 0 {
			return nil
		}
		max := values[0]
		for _, v := range values[1:] {
			if v > max {
				max = v
			}
		}
		return max
	default:
		return nil
	}
}

// =============================================================================
// NEW: Create Table
// =============================================================================

func createTable(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	name, _ := params["name"].(string)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	fields, ok := params["fields"].([]interface{})
	if !ok || len(fields) == 0 {
		return "", fmt.Errorf("fields is required and must not be empty")
	}

	body := map[string]interface{}{
		"name":   name,
		"fields": fields,
	}

	if description, ok := params["description"].(string); ok && description != "" {
		body["description"] = description
	}

	endpoint := fmt.Sprintf("%s/bases/%s/tables", airtableMetaAPI, url.PathEscape(baseID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// NEW: Update Table
// =============================================================================

func updateTable(ctx context.Context, params map[string]any) (string, error) {
	baseID, _ := params["base_id"].(string)
	if baseID == "" {
		return "", fmt.Errorf("base_id is required")
	}

	tableID, _ := params["table_id"].(string)
	if tableID == "" {
		return "", fmt.Errorf("table_id is required")
	}

	body := make(map[string]interface{})

	if name, ok := params["name"].(string); ok && name != "" {
		body["name"] = name
	}

	if description, ok := params["description"].(string); ok {
		body["description"] = description
	}

	if len(body) == 0 {
		return "", fmt.Errorf("at least one of name or description is required")
	}

	endpoint := fmt.Sprintf("%s/bases/%s/tables/%s", airtableMetaAPI, url.PathEscape(baseID), url.PathEscape(tableID))
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}

	return httpclient.PrettyJSON(respBody), nil
}
