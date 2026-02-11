# Supabase Module

## Status

- **Status**: Implemented (ogen)
- **Date**: 2026-02-11
- **API Version**: v1
- **Client**: ogen generated (`pkg/supabaseapi/`)

## Endpoint Catalog

### Account

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_organizations` | GET | `/organizations` | (none) |
| `list_projects` | GET | `/projects` | (none) |
| `get_project` | GET | `/projects/{ref}` | **project_ref** |

### Database

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_tables` | POST | `/projects/{ref}/database/query` | **project_ref**, schemas |
| `run_query` | POST | `/projects/{ref}/database/query` | **project_ref**, **query** |
| `list_migrations` | POST | `/projects/{ref}/database/query` | **project_ref** |
| `apply_migration` | POST | `/projects/{ref}/database/query` | **project_ref**, **name**, **query** |

### Debugging

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `get_logs` | GET | `/projects/{ref}/analytics/endpoints/logs.all` | **project_ref**, **service**, start_time, end_time |
| `get_security_advisors` | GET | `/projects/{ref}/advisors/security` | **project_ref** |
| `get_performance_advisors` | GET | `/projects/{ref}/advisors/performance` | **project_ref** |

### Development

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `get_project_url` | (none) | — | **project_ref** |
| `get_api_keys` | GET | `/projects/{ref}/api-keys` | **project_ref** |
| `generate_typescript_types` | GET | `/projects/{ref}/types/typescript` | **project_ref** |

### Edge Functions

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_edge_functions` | GET | `/projects/{ref}/functions` | **project_ref** |
| `get_edge_function` | GET | `/projects/{ref}/functions/{slug}` | **project_ref**, **slug** |

### Storage

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_storage_buckets` | GET | `/projects/{ref}/storage/buckets` | **project_ref** |
| `get_storage_config` | GET | `/projects/{ref}/config/storage` | **project_ref** |

### Composite Tools

| Tool | 構成 API | Params (required **bold**) |
|------|----------|---------------------------|
| `describe_project` | getProject, listTables(SQL), getApiKeys, listEdgeFunctions, listStorageBuckets, getStorageConfig + URL計算 | **project_ref** |
| `inspect_health` | getSecurityAdvisors, getPerformanceAdvisors | **project_ref** |

## Summary

- **Total**: 19 tools (GET: 10, POST: 1, SQL composite: 3, URL compute: 1, Composite: 2 + list_tables/list_migrations/apply_migration が runDatabaseQuery にマッピング)
- **ogen operations**: 13

## Tool Classification

| 分類 | ツール | 説明 |
|------|--------|------|
| **ogen 単体** | list_organizations, list_projects, get_project, run_query, get_logs, get_security_advisors, get_performance_advisors, get_api_keys, generate_typescript_types, list_edge_functions, get_edge_function, list_storage_buckets, get_storage_config | ogen client の 1 メソッド呼出 → `toJSON(res)` |
| **SQL 複合** | list_tables, list_migrations, apply_migration | SQL を構築 → `runDatabaseQuery` (ogen) で実行 |
| **URL 計算** | get_project_url | API 呼出なし (`https://{ref}.supabase.co` を返すだけ) |
| **Composite** | describe_project, inspect_health | 複数 ogen メソッドを goroutine 並行呼出 → フィールド選別 → JSON |

## Response Schemas

subset spec で定義しているスキーマ:

| Schema | Used by |
|--------|---------|
| Organization | listOrganizations |
| Project, ProjectDatabase | listProjects, getProject |
| RunQueryRequest | runDatabaseQuery |
| AnalyticsResponse | getLogs |
| AdvisorsResponse, Lint, LintMetadata | getSecurityAdvisors, getPerformanceAdvisors |
| ApiKey | getApiKeys |
| TypescriptResponse | generateTypescriptTypes |
| EdgeFunction | listEdgeFunctions, getEdgeFunction |
| StorageBucket | listStorageBuckets |
| StorageConfig, StorageFeatures, FeatureToggle | getStorageConfig |

## Composite Tool Response Format

すべて Level 2 (Field Selection) で JSON を返す。各 API は goroutine で並行呼出。

### describe_project

```json
{
  "project": { "name", "ref", "region", "status", "created_at", "url" },
  "tables": [{ "schema", "name", "column_count" }],
  "api_keys": [{ "name", "type" }],
  "edge_functions": [{ "slug", "name", "status", "version" }],
  "storage_buckets": [{ "name", "public" }],
  "storage_config": { "file_size_limit" },
  "_note": "Use individual tools for details: get_project, list_tables, ..."
}
```

### inspect_health

```json
{
  "security": [{ "name", "level", "title", "description", "categories" }],
  "performance": [{ "name", "level", "title", "description", "categories" }],
  "_note": "Use get_security_advisors or get_performance_advisors for full details"
}
```

## Notes

### Analytics API の SQL 制約

Supabase Analytics API (BigQuery ベース) は `select *` を許可しない。
`get_logs` ハンドラは `service` パラメータからテーブル名を解決し、固定カラムで SQL を構築する:

```sql
select id, timestamp, event_message, metadata from {table} limit 100
```

テーブル名マッピング:

| service | テーブル名 |
|---------|-----------|
| `api` | `edge_logs` |
| `postgres` | `postgres_logs` |
| `edge-function` | `function_edge_logs` |
| `auth` | `auth_logs` |
| `storage` | `storage_logs` |
| `realtime` | `realtime_logs` |

### deprecated `collection` パラメータ

旧 Supabase Logs API は `collection=api_logs` クエリパラメータを使用していたが、
現在は `sql` パラメータに移行している。`collection` を送ると `Unrecognized key(s) in object: 'collection'` エラーになる。

### runDatabaseQuery のレスポンス

SQL 結果は任意の構造を持つため、ogen spec では `schema: {}` (free-form JSON) として定義。
Go 側では `jx.Raw` 型で受け取り、`json.Unmarshal` → `toJSON` で整形する。

### AnalyticsResponse の nullable result

Analytics API は結果がない場合 `result: null` を返す。
spec で `nullable: true` を指定し、ogen が `OptNilAnyArray` 型を生成。

### get_project_url

API 呼出を行わない唯一のツール。`https://{project_ref}.supabase.co` を計算して返す。

### タイムスタンプ形式

Analytics API のタイムスタンプパラメータは ISO 8601 文字列 (`2026-02-11T00:00:00Z`) を受け取る。
spec では `type: string` (format なし) として定義し、ユーザー入力をそのまま渡す。
`format: date-time` にすると ogen が Go の `time.Time` にエンコードし、Supabase が受理しない形式になる。
