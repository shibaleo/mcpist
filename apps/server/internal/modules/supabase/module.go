package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/supabaseapi"
	gen "mcpist/server/pkg/supabaseapi/gen"
)

// SupabaseModule implements the Module interface for Supabase API
type SupabaseModule struct{}

// New creates a new SupabaseModule instance
func New() *SupabaseModule {
	return &SupabaseModule{}
}

// Name returns the module name
func (m *SupabaseModule) Name() string {
	return "supabase"
}

// moduleDescriptions holds module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Supabase Management API - Project management, DB operations, Migrations, Logs, and Storage",
	"ja-JP": "Supabase Management API - プロジェクト管理、DB操作、マイグレーション、ログ、ストレージ",
}

// Descriptions returns multilingual module descriptions
func (m *SupabaseModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *SupabaseModule) Description() string {
	return moduleDescriptions["en-US"]
}

// APIVersion returns the Supabase API version
func (m *SupabaseModule) APIVersion() string {
	return "v1"
}

// Tools returns all available tools
func (m *SupabaseModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *SupabaseModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *SupabaseModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for Supabase)
func (m *SupabaseModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *SupabaseModule) ReadResource(ctx context.Context, uri string) (string, error) {
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
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "supabase")
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
	return supabaseapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Account Tools
	{
		ID:   "supabase:list_organizations",
		Name: "list_organizations",
		Descriptions: modules.LocalizedText{
			"en-US": "List all organizations you have access to.",
			"ja-JP": "アクセス可能なすべての組織を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "supabase:list_projects",
		Name: "list_projects",
		Descriptions: modules.LocalizedText{
			"en-US": "List all Supabase projects you have access to. Use this first to get project_ref for other operations.",
			"ja-JP": "アクセス可能なすべてのSupabaseプロジェクトを一覧表示します。他の操作のためにproject_refを取得するために最初に使用してください。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "supabase:get_project",
		Name: "get_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific project.",
			"ja-JP": "特定のプロジェクトの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference (e.g., 'abcdefghijk'). Get from list_projects."},
			},
			Required: []string{"project_ref"},
		},
	},
	// Database Tools
	{
		ID:   "supabase:list_tables",
		Name: "list_tables",
		Descriptions: modules.LocalizedText{
			"en-US": "List all tables in the database with their schemas. Returns table names and column counts.",
			"ja-JP": "データベース内のすべてのテーブルをスキーマとともに一覧表示します。テーブル名とカラム数を返します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
				"schemas":     {Type: "array", Description: "Schemas to include (default: ['public'])"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:run_query",
		Name: "run_query",
		Descriptions: modules.LocalizedText{
			"en-US": "Execute a SQL query against the database. Supports both read and write operations.",
			"ja-JP": "データベースに対してSQLクエリを実行します。読み取りと書き込みの両方の操作をサポートします。",
		},
		Annotations: modules.AnnotateDestructive,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
				"query":       {Type: "string", Description: "SQL query to execute"},
			},
			Required: []string{"project_ref", "query"},
		},
	},
	{
		ID:   "supabase:list_migrations",
		Name: "list_migrations",
		Descriptions: modules.LocalizedText{
			"en-US": "List all database migrations that have been applied.",
			"ja-JP": "適用されたすべてのデータベースマイグレーションを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:apply_migration",
		Name: "apply_migration",
		Descriptions: modules.LocalizedText{
			"en-US": "Apply a new database migration. Use for DDL operations like CREATE TABLE, ALTER TABLE, etc.",
			"ja-JP": "新しいデータベースマイグレーションを適用します。CREATE TABLE、ALTER TABLEなどのDDL操作に使用します。",
		},
		Annotations: modules.AnnotateDestructive,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
				"name":        {Type: "string", Description: "Migration name in snake_case (e.g., add_users_table)"},
				"query":       {Type: "string", Description: "SQL DDL statements to apply"},
			},
			Required: []string{"project_ref", "name", "query"},
		},
	},
	// Debugging Tools
	{
		ID:   "supabase:get_logs",
		Name: "get_logs",
		Descriptions: modules.LocalizedText{
			"en-US": "Get logs for a specific service. Available services: api, postgres, edge-function, auth, storage, realtime.",
			"ja-JP": "特定のサービスのログを取得します。利用可能なサービス：api、postgres、edge-function、auth、storage、realtime。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
				"service":     {Type: "string", Description: "Service to get logs for: api, postgres, edge-function, auth, storage, realtime"},
				"start_time":  {Type: "string", Description: "ISO timestamp for start of log range (optional)"},
				"end_time":    {Type: "string", Description: "ISO timestamp for end of log range (optional)"},
			},
			Required: []string{"project_ref", "service"},
		},
	},
	{
		ID:   "supabase:get_security_advisors",
		Name: "get_security_advisors",
		Descriptions: modules.LocalizedText{
			"en-US": "Get security recommendations and potential issues for the project.",
			"ja-JP": "プロジェクトのセキュリティ推奨事項と潜在的な問題を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:get_performance_advisors",
		Name: "get_performance_advisors",
		Descriptions: modules.LocalizedText{
			"en-US": "Get performance recommendations and potential issues for the project.",
			"ja-JP": "プロジェクトのパフォーマンス推奨事項と潜在的な問題を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	// Development Tools
	{
		ID:   "supabase:get_project_url",
		Name: "get_project_url",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the base URL for a Supabase project.",
			"ja-JP": "SupabaseプロジェクトのベースURLを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:get_api_keys",
		Name: "get_api_keys",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the API keys for the project (anon key and service role key).",
			"ja-JP": "プロジェクトのAPIキー（匿名キーとサービスロールキー）を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	// Edge Function Tools
	{
		ID:   "supabase:list_edge_functions",
		Name: "list_edge_functions",
		Descriptions: modules.LocalizedText{
			"en-US": "List all Edge Functions deployed in the project.",
			"ja-JP": "プロジェクトにデプロイされたすべてのEdge Functionsを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:get_edge_function",
		Name: "get_edge_function",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific Edge Function.",
			"ja-JP": "特定のEdge Functionの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
				"slug":        {Type: "string", Description: "The slug/name of the Edge Function"},
			},
			Required: []string{"project_ref", "slug"},
		},
	},
	// Storage Tools
	{
		ID:   "supabase:list_storage_buckets",
		Name: "list_storage_buckets",
		Descriptions: modules.LocalizedText{
			"en-US": "List all storage buckets in the project.",
			"ja-JP": "プロジェクト内のすべてのストレージバケットを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:get_storage_config",
		Name: "get_storage_config",
		Descriptions: modules.LocalizedText{
			"en-US": "Get storage configuration for the project including file size limits and features.",
			"ja-JP": "ファイルサイズ制限や機能を含むプロジェクトのストレージ設定を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	// Composite Tools
	{
		ID:   "supabase:describe_project",
		Name: "describe_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a comprehensive overview of a Supabase project: settings, tables, API keys, edge functions, and storage.",
			"ja-JP": "Supabaseプロジェクトの全体像を取得：設定、テーブル、APIキー、Edge Functions、ストレージ。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
	{
		ID:   "supabase:inspect_health",
		Name: "inspect_health",
		Descriptions: modules.LocalizedText{
			"en-US": "Get security and performance recommendations for a Supabase project.",
			"ja-JP": "Supabaseプロジェクトのセキュリティとパフォーマンスの推奨事項を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_ref": {Type: "string", Description: "Project reference"},
			},
			Required: []string{"project_ref"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_organizations":        listOrganizations,
	"list_projects":             listProjects,
	"get_project":               getProject,
	"list_tables":               listTables,
	"run_query":                 runQuery,
	"list_migrations":           listMigrations,
	"apply_migration":           applyMigration,
	"get_logs":                  getLogs,
	"get_security_advisors":     getSecurityAdvisors,
	"get_performance_advisors":  getPerformanceAdvisors,
	"get_project_url":           getProjectURL,
	"get_api_keys":              getAPIKeys,
	"list_edge_functions":       listEdgeFunctions,
	"get_edge_function":         getEdgeFunction,
	"list_storage_buckets":      listStorageBuckets,
	"get_storage_config":        getStorageConfig,
	"describe_project":          describeProject,
	"inspect_health":            inspectHealth,
}

// =============================================================================
// Account Tools
// =============================================================================

func listOrganizations(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListOrganizations(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.GetProject(ctx, gen.GetProjectParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Database Tools
// =============================================================================

func executeQuery(ctx context.Context, projectRef, query string) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.RunDatabaseQuery(ctx, &gen.RunQueryRequest{Query: query}, gen.RunDatabaseQueryParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	// res is jx.Raw (raw JSON bytes), pretty-print it
	var parsed any
	if json.Unmarshal(res, &parsed) == nil {
		return toJSON(parsed)
	}
	return string(res), nil
}

func listTables(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)

	schemas := []string{"public"}
	if s, ok := params["schemas"].([]interface{}); ok && len(s) > 0 {
		schemas = make([]string, 0, len(s))
		for _, schema := range s {
			if str, ok := schema.(string); ok {
				schemas = append(schemas, str)
			}
		}
	}

	schemaList := make([]string, len(schemas))
	for i, s := range schemas {
		schemaList[i] = fmt.Sprintf("'%s'", s)
	}

	query := fmt.Sprintf(`
		SELECT
			schemaname as schema,
			tablename as name,
			(SELECT count(*)::int FROM information_schema.columns
			 WHERE table_schema = t.schemaname AND table_name = t.tablename) as column_count
		FROM pg_tables t
		WHERE schemaname = ANY(ARRAY[%s])
		ORDER BY schemaname, tablename
	`, strings.Join(schemaList, ","))

	return executeQuery(ctx, projectRef, query)
}

func runQuery(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	query, _ := params["query"].(string)
	return executeQuery(ctx, projectRef, query)
}

func listMigrations(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	query := `
		SELECT version, name
		FROM supabase_migrations.schema_migrations
		ORDER BY version DESC
	`
	return executeQuery(ctx, projectRef, query)
}

func applyMigration(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	name, _ := params["name"].(string)
	query, _ := params["query"].(string)

	_, err := executeQuery(ctx, projectRef, query)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"success": true, "migration": "%s"}`, name), nil
}

// =============================================================================
// Debugging Tools
// =============================================================================

func getLogs(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	projectRef, _ := params["project_ref"].(string)
	service, _ := params["service"].(string)

	// Map service name to log source table for SQL query
	sourceMap := map[string]string{
		"api":           "edge_logs",
		"postgres":      "postgres_logs",
		"edge-function": "function_edge_logs",
		"auth":          "auth_logs",
		"storage":       "storage_logs",
		"realtime":      "realtime_logs",
	}

	source, exists := sourceMap[service]
	if !exists {
		return "", fmt.Errorf("invalid service: %s. Valid services: api, postgres, edge-function, auth, storage, realtime", service)
	}

	p := gen.GetLogsParams{
		Ref: projectRef,
	}
	p.SQL.SetTo(fmt.Sprintf("select id, timestamp, event_message, metadata from %s limit 100", source))

	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		p.IsoTimestampStart.SetTo(startTime)
	}
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		p.IsoTimestampEnd.SetTo(endTime)
	}

	res, err := c.GetLogs(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getSecurityAdvisors(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.GetSecurityAdvisors(ctx, gen.GetSecurityAdvisorsParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getPerformanceAdvisors(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.GetPerformanceAdvisors(ctx, gen.GetPerformanceAdvisorsParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Development Tools
// =============================================================================

func getProjectURL(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	projectURL := fmt.Sprintf("https://%s.supabase.co", projectRef)
	return fmt.Sprintf(`{"url": "%s"}`, projectURL), nil
}

func getAPIKeys(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.GetApiKeys(ctx, gen.GetApiKeysParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Edge Function Tools
// =============================================================================

func listEdgeFunctions(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.ListEdgeFunctions(ctx, gen.ListEdgeFunctionsParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getEdgeFunction(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	slug, _ := params["slug"].(string)
	res, err := c.GetEdgeFunction(ctx, gen.GetEdgeFunctionParams{Ref: projectRef, Slug: slug})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Storage Tools
// =============================================================================

func listStorageBuckets(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.ListStorageBuckets(ctx, gen.ListStorageBucketsParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getStorageConfig(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)
	res, err := c.GetStorageConfig(ctx, gen.GetStorageConfigParams{Ref: projectRef})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Composite Tools
// =============================================================================

// describeProject returns a comprehensive overview of a Supabase project.
// Calls: getProject, listTables(SQL), getApiKeys, listEdgeFunctions, listStorageBuckets, getStorageConfig
// All calls are parallel with goroutines.
func describeProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)

	type result struct {
		key string
		val any
		err error
	}

	ch := make(chan result, 6)
	var wg sync.WaitGroup

	// 1. getProject
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.GetProject(ctx, gen.GetProjectParams{Ref: projectRef})
		ch <- result{"project", res, err}
	}()

	// 2. listTables (SQL via runDatabaseQuery)
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := `SELECT schemaname as schema, tablename as name,
			(SELECT count(*)::int FROM information_schema.columns
			 WHERE table_schema = t.schemaname AND table_name = t.tablename) as column_count
			FROM pg_tables t WHERE schemaname NOT IN ('pg_catalog','information_schema','supabase_migrations','extensions','graphql','graphql_public','realtime','_realtime','_analytics','vault','pgsodium','pgsodium_masks','pgtle','cron','net')
			ORDER BY schemaname, tablename`
		res, err := c.RunDatabaseQuery(ctx, &gen.RunQueryRequest{Query: query}, gen.RunDatabaseQueryParams{Ref: projectRef})
		if err != nil {
			ch <- result{"tables", nil, err}
			return
		}
		var parsed any
		json.Unmarshal(res, &parsed)
		ch <- result{"tables", parsed, nil}
	}()

	// 3. getApiKeys
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.GetApiKeys(ctx, gen.GetApiKeysParams{Ref: projectRef})
		if err != nil {
			ch <- result{"api_keys", nil, err}
			return
		}
		keys := make([]map[string]any, 0, len(res))
		for _, k := range res {
			m := map[string]any{"name": k.Name}
			if k.Type.Set && !k.Type.Null {
				m["type"] = k.Type.Value
			}
			keys = append(keys, m)
		}
		ch <- result{"api_keys", keys, nil}
	}()

	// 4. listEdgeFunctions
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.ListEdgeFunctions(ctx, gen.ListEdgeFunctionsParams{Ref: projectRef})
		if err != nil {
			ch <- result{"edge_functions", nil, err}
			return
		}
		funcs := make([]map[string]any, 0, len(res))
		for _, f := range res {
			funcs = append(funcs, map[string]any{
				"slug":    f.Slug,
				"name":    f.Name,
				"status":  f.Status,
				"version": f.Version,
			})
		}
		ch <- result{"edge_functions", funcs, nil}
	}()

	// 5. listStorageBuckets
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.ListStorageBuckets(ctx, gen.ListStorageBucketsParams{Ref: projectRef})
		if err != nil {
			ch <- result{"storage_buckets", nil, err}
			return
		}
		buckets := make([]map[string]any, 0, len(res))
		for _, b := range res {
			buckets = append(buckets, map[string]any{
				"name":   b.Name,
				"public": b.Public,
			})
		}
		ch <- result{"storage_buckets", buckets, nil}
	}()

	// 6. getStorageConfig
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.GetStorageConfig(ctx, gen.GetStorageConfigParams{Ref: projectRef})
		ch <- result{"storage_config", res, err}
	}()

	go func() { wg.Wait(); close(ch) }()

	out := map[string]any{}
	for r := range ch {
		if r.err != nil {
			continue // skip failed calls
		}
		if r.key == "project" {
			p := r.val.(*gen.Project)
			out["project"] = map[string]any{
				"name":       p.Name,
				"ref":        p.Ref,
				"region":     p.Region,
				"status":     p.Status,
				"created_at": p.CreatedAt,
				"url":        fmt.Sprintf("https://%s.supabase.co", projectRef),
			}
		} else if r.key == "storage_config" {
			sc := r.val.(*gen.StorageConfig)
			out[r.key] = map[string]any{
				"file_size_limit": sc.FileSizeLimit,
			}
		} else {
			out[r.key] = r.val
		}
	}

	out["_note"] = "Use individual tools for details: get_project, list_tables, get_api_keys, list_edge_functions, list_storage_buckets, get_storage_config"
	return toJSON(out)
}

// inspectHealth returns security and performance recommendations for a project.
// Calls: getSecurityAdvisors, getPerformanceAdvisors (parallel).
func inspectHealth(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectRef, _ := params["project_ref"].(string)

	type result struct {
		key string
		val any
		err error
	}

	ch := make(chan result, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.GetSecurityAdvisors(ctx, gen.GetSecurityAdvisorsParams{Ref: projectRef})
		ch <- result{"security", res, err}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.GetPerformanceAdvisors(ctx, gen.GetPerformanceAdvisorsParams{Ref: projectRef})
		ch <- result{"performance", res, err}
	}()

	go func() { wg.Wait(); close(ch) }()

	out := map[string]any{}
	for r := range ch {
		if r.err != nil {
			continue
		}
		advisors := r.val.(*gen.AdvisorsResponse)
		lints := make([]map[string]any, 0, len(advisors.Lints))
		for _, l := range advisors.Lints {
			m := map[string]any{
				"name":        l.Name,
				"level":       l.Level,
				"title":       l.Title,
				"description": l.Description,
			}
			if len(l.Categories) > 0 {
				m["categories"] = l.Categories
			}
			lints = append(lints, m)
		}
		out[r.key] = lints
	}

	out["_note"] = "Use get_security_advisors or get_performance_advisors for full details"
	return toJSON(out)
}
