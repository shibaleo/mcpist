package supabase

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const supabaseAPIBase = "https://api.supabase.com/v1"

var client = httpclient.New()

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

// Description returns the module description
func (m *SupabaseModule) Description() string {
	return "Supabase Management API - Project management, DB operations, Migrations, Logs, and Storage"
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

// Resources returns all available resources (none for Supabase)
func (m *SupabaseModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *SupabaseModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for Supabase)
func (m *SupabaseModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *SupabaseModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
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
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "supabase")
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

	h := map[string]string{}

	// Supabase uses API Key (Bearer token)
	switch creds.AuthType {
	case store.AuthTypeAPIKey:
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Account Tools
	{
		Name:        "list_organizations",
		Description: "List all organizations you have access to.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		Name:        "list_projects",
		Description: "List all Supabase projects you have access to. Use this first to get project_ref for other operations.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		Name:        "get_project",
		Description: "Get details of a specific project.",
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
		Name:        "list_tables",
		Description: "List all tables in the database with their schemas. Returns table names and column counts.",
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
		Name:        "run_query",
		Description: "Execute a SQL query against the database. Supports both read and write operations.",
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
		Name:        "list_migrations",
		Description: "List all database migrations that have been applied.",
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
		Name:        "apply_migration",
		Description: "Apply a new database migration. Use for DDL operations like CREATE TABLE, ALTER TABLE, etc.",
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
		Name:        "get_logs",
		Description: "Get logs for a specific service. Available services: api, postgres, edge-function, auth, storage, realtime.",
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
		Name:        "get_security_advisors",
		Description: "Get security recommendations and potential issues for the project.",
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
		Name:        "get_performance_advisors",
		Description: "Get performance recommendations and potential issues for the project.",
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
		Name:        "get_project_url",
		Description: "Get the base URL for a Supabase project.",
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
		Name:        "get_api_keys",
		Description: "Get the API keys for the project (anon key and service role key).",
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
		Name:        "generate_typescript_types",
		Description: "Generate TypeScript type definitions from the database schema.",
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
		Name:        "list_edge_functions",
		Description: "List all Edge Functions deployed in the project.",
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
		Name:        "get_edge_function",
		Description: "Get details of a specific Edge Function.",
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
		Name:        "list_storage_buckets",
		Description: "List all storage buckets in the project.",
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
		Name:        "get_storage_config",
		Description: "Get storage configuration for the project including file size limits and features.",
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
	"generate_typescript_types": generateTypescriptTypes,
	"list_edge_functions":       listEdgeFunctions,
	"get_edge_function":         getEdgeFunction,
	"list_storage_buckets":      listStorageBuckets,
	"get_storage_config":        getStorageConfig,
}

// =============================================================================
// Account Tools
// =============================================================================

func listOrganizations(ctx context.Context, params map[string]any) (string, error) {
	endpoint := supabaseAPIBase + "/organizations"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	endpoint := supabaseAPIBase + "/projects"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Database Tools
// =============================================================================

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

func executeQuery(ctx context.Context, projectRef, query string) (string, error) {
	endpoint := fmt.Sprintf("%s/projects/%s/database/query", supabaseAPIBase, projectRef)
	payload := map[string]string{"query": query}
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), payload)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listMigrations(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	query := `
		SELECT version, name, executed_at
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
	projectRef, _ := params["project_ref"].(string)
	service, _ := params["service"].(string)

	serviceMap := map[string]string{
		"api":           "api_logs",
		"postgres":      "postgres_logs",
		"edge-function": "function_logs",
		"auth":          "auth_logs",
		"storage":       "storage_logs",
		"realtime":      "realtime_logs",
	}

	collection, exists := serviceMap[service]
	if !exists {
		return "", fmt.Errorf("invalid service: %s. Valid services: api, postgres, edge-function, auth, storage, realtime", service)
	}

	query := url.Values{}
	query.Set("collection", collection)
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		query.Set("start", startTime)
	}
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		query.Set("end", endTime)
	}

	endpoint := fmt.Sprintf("%s/projects/%s/analytics/endpoints/logs.all?%s", supabaseAPIBase, projectRef, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getSecurityAdvisors(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/advisors/security", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getPerformanceAdvisors(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/advisors/performance", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
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
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/api-keys", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func generateTypescriptTypes(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/types/typescript", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Edge Function Tools
// =============================================================================

func listEdgeFunctions(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/functions", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getEdgeFunction(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	slug, _ := params["slug"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/functions/%s", supabaseAPIBase, projectRef, slug)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Storage Tools
// =============================================================================

func listStorageBuckets(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/storage/buckets", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getStorageConfig(ctx context.Context, params map[string]any) (string, error) {
	projectRef, _ := params["project_ref"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s/config/storage", supabaseAPIBase, projectRef)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
