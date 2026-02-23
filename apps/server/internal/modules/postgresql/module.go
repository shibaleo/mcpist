package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
)

// Connection settings
const (
	connectTimeout = 10 * time.Second
	queryTimeout   = 30 * time.Second
	defaultMaxRows = 1000
	maxMaxRows     = 10000
)

// convertValue converts PostgreSQL-specific types to JSON-friendly formats
// - UUID ([16]byte) -> string format "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
func convertValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	// Check for [16]byte (UUID)
	if b, ok := v.([16]byte); ok {
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	}

	// Check for []byte that might be a UUID (16 bytes)
	if b, ok := v.([]byte); ok && len(b) == 16 {
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	}

	return v
}

// convertRow applies convertValue to all values in a row
func convertRow(values []interface{}) []interface{} {
	result := make([]interface{}, len(values))
	for i, v := range values {
		result[i] = convertValue(v)
	}
	return result
}

// PostgreSQLModule implements the Module interface for PostgreSQL
type PostgreSQLModule struct{}

// New creates a new PostgreSQLModule instance
func New() *PostgreSQLModule {
	return &PostgreSQLModule{}
}

// Name returns the module name
func (m *PostgreSQLModule) Name() string {
	return "postgresql"
}

// moduleDescriptions holds module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "PostgreSQL Database - Direct connection for query execution and schema inspection",
	"ja-JP": "PostgreSQL データベース - クエリ実行とスキーマ確認のための直接接続",
}

// Descriptions returns multilingual module descriptions
func (m *PostgreSQLModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *PostgreSQLModule) Description() string {
	return moduleDescriptions["en-US"]
}

// APIVersion returns the PostgreSQL module version
func (m *PostgreSQLModule) APIVersion() string {
	return "v1"
}

// Tools returns all available tools
func (m *PostgreSQLModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *PostgreSQLModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for PostgreSQL)
func (m *PostgreSQLModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *PostgreSQLModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Connection Management
// =============================================================================

func getConnectionString(ctx context.Context) (string, error) {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return "", fmt.Errorf("authentication required")
	}

	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "postgresql")
	if err != nil {
		return "", fmt.Errorf("failed to get credentials: %w", err)
	}
	if credentials == nil || credentials.AccessToken == "" {
		return "", fmt.Errorf("PostgreSQL connection string not configured")
	}

	return credentials.AccessToken, nil
}

func validateConnectionString(connStr string) error {
	u, err := url.Parse(connStr)
	if err != nil {
		return fmt.Errorf("invalid connection string format")
	}

	// Scheme check
	if u.Scheme != "postgresql" && u.Scheme != "postgres" {
		return fmt.Errorf("scheme must be postgresql or postgres")
	}

	// Host check
	if u.Host == "" {
		return fmt.Errorf("host is required")
	}

	// localhost prohibition (SSRF protection)
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("localhost connections are not allowed for security reasons")
	}

	// Database name check
	if u.Path == "" || u.Path == "/" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

func getConnection(ctx context.Context) (*pgx.Conn, error) {
	connStr, err := getConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	if err := validateConnectionString(connStr); err != nil {
		return nil, err
	}

	// Add default sslmode if not specified
	if !strings.Contains(connStr, "sslmode=") {
		if strings.Contains(connStr, "?") {
			connStr += "&sslmode=require"
		} else {
			connStr += "?sslmode=require"
		}
	}

	// Set connect timeout
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	conn, err := pgx.Connect(connectCtx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return conn, nil
}

// =============================================================================
// SQL Safety
// =============================================================================

// Dangerous patterns for DDL operations
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*DROP\s+`),
	regexp.MustCompile(`(?i)^\s*TRUNCATE\s+`),
	regexp.MustCompile(`(?i)^\s*ALTER\s+`),
	regexp.MustCompile(`(?i)^\s*CREATE\s+`),
	regexp.MustCompile(`(?i)^\s*GRANT\s+`),
	regexp.MustCompile(`(?i)^\s*REVOKE\s+`),
	regexp.MustCompile(`(?i);\s*DROP\s+`),
	regexp.MustCompile(`(?i);\s*TRUNCATE\s+`),
	regexp.MustCompile(`(?i);\s*ALTER\s+`),
	regexp.MustCompile(`(?i);\s*CREATE\s+`),
}

// Write operation patterns
var writePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*INSERT\s+`),
	regexp.MustCompile(`(?i)^\s*UPDATE\s+`),
	regexp.MustCompile(`(?i)^\s*DELETE\s+`),
}

func isDDL(sql string) bool {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(sql) {
			return true
		}
	}
	return false
}

func isWriteOperation(sql string) bool {
	for _, pattern := range writePatterns {
		if pattern.MatchString(sql) {
			return true
		}
	}
	return false
}

func isSelectOnly(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	return strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(trimmed), "WITH")
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	{
		ID:   "postgresql:test_connection",
		Name: "test_connection",
		Descriptions: modules.LocalizedText{
			"en-US": "Test PostgreSQL connection and return server version and connection info.",
			"ja-JP": "PostgreSQL 接続をテストし、サーバーバージョンと接続情報を返します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "postgresql:list_schemas",
		Name: "list_schemas",
		Descriptions: modules.LocalizedText{
			"en-US": "List all schemas in the database.",
			"ja-JP": "データベース内のすべてのスキーマを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"include_system": {Type: "boolean", Description: "Include system schemas (pg_catalog, information_schema). Default: false"},
			},
		},
	},
	{
		ID:   "postgresql:list_tables",
		Name: "list_tables",
		Descriptions: modules.LocalizedText{
			"en-US": "List all tables in a schema with row count estimates.",
			"ja-JP": "スキーマ内のすべてのテーブルを推定行数とともに一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"schema":        {Type: "string", Description: "Schema name. Default: public"},
				"include_views": {Type: "boolean", Description: "Include views. Default: true"},
			},
		},
	},
	{
		ID:   "postgresql:describe_table",
		Name: "describe_table",
		Descriptions: modules.LocalizedText{
			"en-US": "Get table structure including columns, types, constraints, and indexes.",
			"ja-JP": "カラム、型、制約、インデックスを含むテーブル構造を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"table":  {Type: "string", Description: "Table name"},
				"schema": {Type: "string", Description: "Schema name. Default: public"},
			},
			Required: []string{"table"},
		},
	},
	{
		ID:   "postgresql:query",
		Name: "query",
		Descriptions: modules.LocalizedText{
			"en-US": "Execute a SELECT query. Supports parameterized queries with $1, $2, etc.",
			"ja-JP": "SELECT クエリを実行します。$1, $2 などのパラメータ化クエリをサポートします。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"sql":      {Type: "string", Description: "SELECT query to execute"},
				"params":   {Type: "array", Description: "Query parameters for $1, $2, etc.", Items: &modules.Property{Type: "string"}},
				"max_rows": {Type: "integer", Description: "Maximum rows to return. Default: 1000, Max: 10000"},
			},
			Required: []string{"sql"},
		},
	},
	{
		ID:   "postgresql:execute",
		Name: "execute",
		Descriptions: modules.LocalizedText{
			"en-US": "Execute INSERT/UPDATE/DELETE statement. Returns affected row count.",
			"ja-JP": "INSERT/UPDATE/DELETE 文を実行します。影響を受けた行数を返します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"sql":    {Type: "string", Description: "INSERT/UPDATE/DELETE statement to execute"},
				"params": {Type: "array", Description: "Query parameters for $1, $2, etc.", Items: &modules.Property{Type: "string"}},
			},
			Required: []string{"sql"},
		},
	},
	{
		ID:   "postgresql:execute_ddl",
		Name: "execute_ddl",
		Descriptions: modules.LocalizedText{
			"en-US": "Execute DDL statement (CREATE/ALTER/DROP/TRUNCATE). Use with caution.",
			"ja-JP": "DDL 文（CREATE/ALTER/DROP/TRUNCATE）を実行します。注意して使用してください。",
		},
		Annotations: modules.AnnotateDestructive,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"sql": {Type: "string", Description: "DDL statement to execute"},
			},
			Required: []string{"sql"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"test_connection": testConnection,
	"list_schemas":    listSchemas,
	"list_tables":     listTables,
	"describe_table":  describeTable,
	"query":           queryTool,
	"execute":         executeTool,
	"execute_ddl":     executeDDL,
}

// =============================================================================
// Tool Implementations
// =============================================================================

func testConnection(ctx context.Context, params map[string]any) (string, error) {
	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var version string
	err = conn.QueryRow(queryCtx, "SELECT version()").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	// Parse connection info from connection string
	connStr, _ := getConnectionString(ctx)
	u, _ := url.Parse(connStr)

	result := map[string]interface{}{
		"success":  true,
		"version":  version,
		"host":     u.Hostname(),
		"port":     u.Port(),
		"database": strings.TrimPrefix(u.Path, "/"),
	}
	if u.User != nil {
		result["user"] = u.User.Username()
	}

	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

func listSchemas(ctx context.Context, params map[string]any) (string, error) {
	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	includeSystem := false
	if v, ok := params["include_system"].(bool); ok {
		includeSystem = v
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var query string
	if includeSystem {
		query = `
			SELECT schema_name
			FROM information_schema.schemata
			ORDER BY schema_name
		`
	} else {
		query = `
			SELECT schema_name
			FROM information_schema.schemata
			WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			ORDER BY schema_name
		`
	}

	rows, err := conn.Query(queryCtx, query)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", fmt.Errorf("scan failed: %w", err)
		}
		schemas = append(schemas, name)
	}

	result := map[string]interface{}{
		"schemas": schemas,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

func listTables(ctx context.Context, params map[string]any) (string, error) {
	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	schema := "public"
	if v, ok := params["schema"].(string); ok && v != "" {
		schema = v
	}

	includeViews := true
	if v, ok := params["include_views"].(bool); ok {
		includeViews = v
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var tableTypes string
	if includeViews {
		tableTypes = "('BASE TABLE', 'VIEW')"
	} else {
		tableTypes = "('BASE TABLE')"
	}

	query := fmt.Sprintf(`
		SELECT
			t.table_name,
			t.table_type,
			COALESCE(s.n_live_tup, 0) as row_estimate
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s
			ON t.table_name = s.relname
			AND t.table_schema = s.schemaname
		WHERE t.table_schema = $1
			AND t.table_type IN %s
		ORDER BY t.table_name
	`, tableTypes)

	rows, err := conn.Query(queryCtx, query, schema)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	type tableInfo struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		RowsEstimate int64  `json:"rows_estimate"`
	}

	var tables []tableInfo
	for rows.Next() {
		var t tableInfo
		var tableType string
		if err := rows.Scan(&t.Name, &tableType, &t.RowsEstimate); err != nil {
			return "", fmt.Errorf("scan failed: %w", err)
		}
		if tableType == "BASE TABLE" {
			t.Type = "table"
		} else {
			t.Type = "view"
		}
		tables = append(tables, t)
	}

	result := map[string]interface{}{
		"schema": schema,
		"tables": tables,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

func describeTable(ctx context.Context, params map[string]any) (string, error) {
	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	table, ok := params["table"].(string)
	if !ok || table == "" {
		return "", fmt.Errorf("table is required")
	}

	schema := "public"
	if v, ok := params["schema"].(string); ok && v != "" {
		schema = v
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Get columns
	columnQuery := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as nullable,
			c.column_default,
			EXISTS (
				SELECT 1 FROM information_schema.key_column_usage k
				JOIN information_schema.table_constraints tc
					ON k.constraint_name = tc.constraint_name
					AND k.table_schema = tc.table_schema
				WHERE k.table_schema = c.table_schema
					AND k.table_name = c.table_name
					AND k.column_name = c.column_name
					AND tc.constraint_type = 'PRIMARY KEY'
			) as is_primary_key
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := conn.Query(queryCtx, columnQuery, schema, table)
	if err != nil {
		return "", fmt.Errorf("column query failed: %w", err)
	}

	type columnInfo struct {
		Name       string  `json:"name"`
		Type       string  `json:"type"`
		Nullable   bool    `json:"nullable"`
		Default    *string `json:"default"`
		PrimaryKey bool    `json:"primary_key"`
	}

	var columns []columnInfo
	for rows.Next() {
		var col columnInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Default, &col.PrimaryKey); err != nil {
			rows.Close()
			return "", fmt.Errorf("scan failed: %w", err)
		}
		columns = append(columns, col)
	}
	rows.Close()

	if len(columns) == 0 {
		return "", fmt.Errorf("table %s.%s not found", schema, table)
	}

	// Get indexes
	indexQuery := `
		SELECT
			indexname,
			indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2
	`

	rows, err = conn.Query(queryCtx, indexQuery, schema, table)
	if err != nil {
		return "", fmt.Errorf("index query failed: %w", err)
	}

	type indexInfo struct {
		Name       string `json:"name"`
		Definition string `json:"definition"`
	}

	var indexes []indexInfo
	for rows.Next() {
		var idx indexInfo
		if err := rows.Scan(&idx.Name, &idx.Definition); err != nil {
			rows.Close()
			return "", fmt.Errorf("scan failed: %w", err)
		}
		indexes = append(indexes, idx)
	}
	rows.Close()

	// Get row count estimate
	var rowCount int64
	err = conn.QueryRow(queryCtx, `
		SELECT COALESCE(n_live_tup, 0)
		FROM pg_stat_user_tables
		WHERE schemaname = $1 AND relname = $2
	`, schema, table).Scan(&rowCount)
	if err != nil {
		rowCount = 0 // Ignore error, just set to 0
	}

	result := map[string]interface{}{
		"table":              table,
		"schema":             schema,
		"columns":            columns,
		"indexes":            indexes,
		"row_count_estimate": rowCount,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

func queryTool(ctx context.Context, params map[string]any) (string, error) {
	sql, ok := params["sql"].(string)
	if !ok || sql == "" {
		return "", fmt.Errorf("sql is required")
	}

	// Validate SELECT only
	if !isSelectOnly(sql) {
		return "", fmt.Errorf("only SELECT queries are allowed. Use 'execute' for INSERT/UPDATE/DELETE or 'execute_ddl' for DDL")
	}

	// Check for DDL injection
	if isDDL(sql) {
		return "", fmt.Errorf("DDL statements are not allowed in query tool")
	}

	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	// Parse params
	var queryParams []interface{}
	if p, ok := params["params"].([]interface{}); ok {
		queryParams = p
	}

	// Max rows
	maxRows := defaultMaxRows
	if v, ok := params["max_rows"].(float64); ok {
		maxRows = int(v)
		if maxRows > maxMaxRows {
			maxRows = maxMaxRows
		}
		if maxRows < 1 {
			maxRows = 1
		}
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := conn.Query(queryCtx, sql, queryParams...)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columnNames[i] = string(fd.Name)
	}

	// Fetch rows
	var resultRows [][]interface{}
	rowCount := 0
	truncated := false

	for rows.Next() {
		if rowCount >= maxRows {
			truncated = true
			break
		}

		values, err := rows.Values()
		if err != nil {
			return "", fmt.Errorf("scan failed: %w", err)
		}
		resultRows = append(resultRows, convertRow(values))
		rowCount++
	}

	result := map[string]interface{}{
		"columns":   columnNames,
		"rows":      resultRows,
		"row_count": rowCount,
		"truncated": truncated,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

func executeTool(ctx context.Context, params map[string]any) (string, error) {
	sql, ok := params["sql"].(string)
	if !ok || sql == "" {
		return "", fmt.Errorf("sql is required")
	}

	// Validate write operation
	if !isWriteOperation(sql) {
		return "", fmt.Errorf("only INSERT/UPDATE/DELETE statements are allowed. Use 'query' for SELECT or 'execute_ddl' for DDL")
	}

	// Check for DDL injection
	if isDDL(sql) {
		return "", fmt.Errorf("DDL statements are not allowed in execute tool")
	}

	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	// Parse params
	var queryParams []interface{}
	if p, ok := params["params"].([]interface{}); ok {
		queryParams = p
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tag, err := conn.Exec(queryCtx, sql, queryParams...)
	if err != nil {
		return "", fmt.Errorf("execute failed: %w", err)
	}

	result := map[string]interface{}{
		"rows_affected": tag.RowsAffected(),
		"command":       tag.String(),
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

func executeDDL(ctx context.Context, params map[string]any) (string, error) {
	sql, ok := params["sql"].(string)
	if !ok || sql == "" {
		return "", fmt.Errorf("sql is required")
	}

	// Validate DDL
	if !isDDL(sql) {
		return "", fmt.Errorf("only DDL statements (CREATE/ALTER/DROP/TRUNCATE) are allowed. Use 'query' for SELECT or 'execute' for INSERT/UPDATE/DELETE")
	}

	conn, err := getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tag, err := conn.Exec(queryCtx, sql)
	if err != nil {
		return "", fmt.Errorf("execute failed: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"command": tag.String(),
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}
