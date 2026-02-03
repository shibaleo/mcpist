# PostgreSQL モジュール実装計画

## 概要

PostgreSQL データベースに直接接続し、クエリ実行・スキーマ確認を行う MCP モジュール。
接続文字列（Connection String）方式で認証。

---

## 設計方針

### 認証方式

| 項目 | 内容 |
|------|------|
| 認証タイプ | `connection_string` (AuthTypeAPIKey 相当) |
| 保存先 | token-vault (`access_token` フィールドに接続文字列を保存) |
| 形式 | `postgresql://user:password@host:port/database?sslmode=require` |

### セキュリティ対策

| 対策           | 実装方法                                                              |
| ------------ | ----------------------------------------------------------------- |
| SQL インジェクション | クエリはそのまま実行（ユーザー責任）、ただし危険なステートメント検知                                |
| 危険操作ブロック     | `DROP`, `TRUNCATE`, `ALTER`, `CREATE`, `GRANT`, `REVOKE` をデフォルト禁止 |
| localhost 禁止 | `localhost`, `127.0.0.1`, `::1` への接続を禁止（SSRF 対策）                  |
| 行数制限         | `max_rows` パラメータ（デフォルト: 1000, 最大: 10000）                          |
| タイムアウト       | 接続: 10秒、クエリ: 30秒                                                  |
| 接続プール        | なし（リクエストごとに接続・切断）                                                 |
| SSL          | `sslmode=require` をデフォルト                                          |

### 実行モード

| モード     | 説明         | 対象ステートメント                     |     |
| ------- | ---------- | ----------------------------- | --- |
| `read`  | 読み取り専用     | SELECT                        |     |
| `write` | 書き込み許可     | INSERT, UPDATE, DELETE        |     |
| `admin` | 管理操作許可（危険） | DROP, ALTER, CREATE, TRUNCATE |     |

デフォルトは `read` モード。`write` / `admin` は明示的に指定が必要。

---

## ツール一覧（7ツール）

| ツール | 説明 | mode | readOnlyHint | destructiveHint |
|--------|------|------|--------------|-----------------|
| `test_connection` | 接続テスト | - | true | - |
| `list_schemas` | スキーマ一覧 | read | true | - |
| `list_tables` | テーブル一覧（スキーマ指定可） | read | true | - |
| `describe_table` | テーブル定義（カラム、型、制約） | read | true | - |
| `query` | SELECT クエリ実行 | read | true | - |
| `execute` | INSERT/UPDATE/DELETE 実行 | write | false | false |
| `execute_ddl` | DDL 実行（CREATE/ALTER/DROP） | admin | false | **true** |

---

## ツール詳細

### test_connection

接続テストを行い、PostgreSQL バージョンと接続情報を返す。

```json
{
  "name": "test_connection",
  "input": {}
}
```

**レスポンス例:**
```json
{
  "success": true,
  "version": "PostgreSQL 15.2",
  "database": "mydb",
  "user": "myuser",
  "host": "localhost",
  "port": 5432
}
```

### list_schemas

```json
{
  "name": "list_schemas",
  "input": {
    "include_system": false  // pg_catalog, information_schema を含めるか
  }
}
```

### list_tables

```json
{
  "name": "list_tables",
  "input": {
    "schema": "public",      // デフォルト: public
    "include_views": true    // ビューを含めるか（デフォルト: true）
  }
}
```

**レスポンス例:**
```json
{
  "tables": [
    {"name": "users", "type": "table", "rows_estimate": 1500},
    {"name": "orders", "type": "table", "rows_estimate": 50000},
    {"name": "user_stats", "type": "view", "rows_estimate": null}
  ]
}
```

### describe_table

```json
{
  "name": "describe_table",
  "input": {
    "table": "users",
    "schema": "public"  // デフォルト: public
  }
}
```

**レスポンス例:**
```json
{
  "table": "users",
  "schema": "public",
  "columns": [
    {"name": "id", "type": "uuid", "nullable": false, "default": "gen_random_uuid()", "primary_key": true},
    {"name": "email", "type": "text", "nullable": false, "default": null, "primary_key": false},
    {"name": "created_at", "type": "timestamp with time zone", "nullable": false, "default": "now()", "primary_key": false}
  ],
  "indexes": [
    {"name": "users_pkey", "columns": ["id"], "unique": true, "primary": true},
    {"name": "users_email_idx", "columns": ["email"], "unique": true, "primary": false}
  ],
  "foreign_keys": [],
  "row_count_estimate": 1500
}
```

### query

SELECT クエリ専用。

```json
{
  "name": "query",
  "input": {
    "sql": "SELECT * FROM users WHERE created_at > $1",
    "params": ["2024-01-01"],  // プレースホルダー（$1, $2, ...）
    "max_rows": 100            // デフォルト: 1000, 最大: 10000
  }
}
```

**レスポンス例:**
```json
{
  "columns": ["id", "email", "created_at"],
  "rows": [
    ["uuid-1", "user1@example.com", "2024-02-01T00:00:00Z"],
    ["uuid-2", "user2@example.com", "2024-02-15T00:00:00Z"]
  ],
  "row_count": 2,
  "truncated": false
}
```

### execute

INSERT/UPDATE/DELETE 用。

```json
{
  "name": "execute",
  "input": {
    "sql": "UPDATE users SET email = $1 WHERE id = $2",
    "params": ["new@example.com", "uuid-1"]
  }
}
```

**レスポンス例:**
```json
{
  "rows_affected": 1,
  "command": "UPDATE"
}
```

### execute_ddl

DDL（CREATE/ALTER/DROP/TRUNCATE）用。destructive フラグ付き。

```json
{
  "name": "execute_ddl",
  "input": {
    "sql": "CREATE INDEX idx_users_email ON users(email)"
  }
}
```

**レスポンス例:**
```json
{
  "success": true,
  "command": "CREATE INDEX"
}
```

---

## 実装タスク

### Phase 1: 基盤実装

| ID | タスク | 備考 |
|----|--------|------|
| PG-001 | `go.mod` に `pgx` ドライバー追加 | `github.com/jackc/pgx/v5` |
| PG-002 | `modules/postgresql/module.go` 作成 | Module interface 実装 |
| PG-003 | 接続管理関数実装 | `getConnection()`, 接続プーリングなし |
| PG-004 | `main.go` に RegisterModule 追加 | server, tools-export 両方 |

### Phase 2: ツール実装

| ID | タスク | 備考 |
|----|--------|------|
| PG-005 | `test_connection` 実装 | `SELECT version()` |
| PG-006 | `list_schemas` 実装 | `information_schema.schemata` |
| PG-007 | `list_tables` 実装 | `information_schema.tables` + `pg_stat_user_tables` |
| PG-008 | `describe_table` 実装 | `information_schema.columns` + `pg_indexes` |
| PG-009 | `query` 実装 | SELECT 専用、行数制限 |
| PG-010 | `execute` 実装 | INSERT/UPDATE/DELETE |
| PG-011 | `execute_ddl` 実装 | DDL、destructive フラグ |

### Phase 3: Console UI

| ID | タスク | 備考 |
|----|--------|------|
| PG-012 | services/page.tsx に PostgreSQL 追加 | 接続文字列入力フォーム |
| PG-013 | 接続文字列のバリデーション | URL パース、必須フィールド確認 |
| PG-014 | 接続テスト UI | 保存前に接続テスト実行 |

### Phase 4: テスト

| ID | タスク | 備考 |
|----|--------|------|
| PG-015 | ローカル PostgreSQL で動作確認 | Docker or ローカルインスタンス |
| PG-016 | Supabase PostgreSQL で動作確認 | 本番相当 |
| PG-017 | エラーハンドリング確認 | 接続エラー、権限エラー、タイムアウト |

---

## SQL クエリサンプル

### list_schemas

```sql
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
ORDER BY schema_name
```

### list_tables

```sql
SELECT
    t.table_name,
    t.table_type,
    COALESCE(s.n_live_tup, 0) as row_estimate
FROM information_schema.tables t
LEFT JOIN pg_stat_user_tables s
    ON t.table_name = s.relname
    AND t.table_schema = s.schemaname
WHERE t.table_schema = $1
ORDER BY t.table_name
```

### describe_table (columns)

```sql
SELECT
    c.column_name,
    c.data_type,
    c.is_nullable = 'YES' as nullable,
    c.column_default,
    EXISTS (
        SELECT 1 FROM information_schema.key_column_usage k
        JOIN information_schema.table_constraints tc
            ON k.constraint_name = tc.constraint_name
        WHERE k.table_schema = c.table_schema
            AND k.table_name = c.table_name
            AND k.column_name = c.column_name
            AND tc.constraint_type = 'PRIMARY KEY'
    ) as is_primary_key
FROM information_schema.columns c
WHERE c.table_schema = $1 AND c.table_name = $2
ORDER BY c.ordinal_position
```

### describe_table (indexes)

```sql
SELECT
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = $1 AND tablename = $2
```

---

## セキュリティ考慮事項

### 危険なステートメント検知

```go
var dangerousPatterns = []string{
    `(?i)^\s*DROP\s+`,
    `(?i)^\s*TRUNCATE\s+`,
    `(?i)^\s*ALTER\s+`,
    `(?i)^\s*CREATE\s+`,
    `(?i)^\s*GRANT\s+`,
    `(?i)^\s*REVOKE\s+`,
    `(?i);\s*DROP\s+`,  // インジェクション検知
    `(?i);\s*DELETE\s+FROM\s+\S+\s*$`,  // WHERE なし DELETE
    `(?i);\s*UPDATE\s+\S+\s+SET\s+.*\s*$`,  // WHERE なし UPDATE
}
```

### 接続文字列の検証

```go
func validateConnectionString(connStr string) error {
    // URL パース
    u, err := url.Parse(connStr)
    if err != nil {
        return fmt.Errorf("invalid connection string format")
    }

    // スキーム確認
    if u.Scheme != "postgresql" && u.Scheme != "postgres" {
        return fmt.Errorf("scheme must be postgresql or postgres")
    }

    // ホスト確認
    if u.Host == "" {
        return fmt.Errorf("host is required")
    }

    // localhost 禁止（SSRF 対策）
    host := u.Hostname()
    if host == "localhost" || host == "127.0.0.1" || host == "::1" {
        return fmt.Errorf("localhost connections are not allowed for security reasons")
    }

    // データベース名確認
    if u.Path == "" || u.Path == "/" {
        return fmt.Errorf("database name is required")
    }

    return nil
}
```

---

## Console UI 設計

### services/page.tsx

```tsx
// authConfig for postgresql
{
  type: "connection_string",
  label: "PostgreSQL",
  description: "PostgreSQL 接続文字列を入力",
  placeholder: "postgresql://user:password@host:5432/database?sslmode=require",
  helpText: "Supabase: Project Settings > Database > Connection string (URI)",
  testConnection: true,  // 保存前に接続テスト
}
```

### 接続文字列入力フォーム

- テキストインプット（type="password" で非表示）
- 「接続テスト」ボタン
- バリデーションエラー表示
- SSL モード選択（require/prefer/disable）

---

## 依存関係

### Go パッケージ

```bash
go get github.com/jackc/pgx/v5
```

### pgx の選定理由

| ライブラリ | メリット | デメリット |
|-----------|---------|-----------|
| `database/sql` + `lib/pq` | 標準的 | 古い、メンテナンス停止気味 |
| `pgx` | 高速、PostgreSQL 特化、アクティブ | PostgreSQL 専用 |
| `sqlx` | 汎用的 | 追加の抽象化層 |

→ **pgx** を採用（PostgreSQL 専用モジュールのため）

---

## 将来の拡張

1. **接続プール**: 高負荷時のパフォーマンス改善
2. **トランザクション**: `begin_transaction`, `commit`, `rollback`
3. **EXPLAIN**: クエリプラン表示
4. **pg_stat_statements**: クエリパフォーマンス分析
5. **レプリカ対応**: 読み取り専用レプリカへのルーティング

---

## 参考

- [pgx Documentation](https://github.com/jackc/pgx)
- [PostgreSQL System Catalogs](https://www.postgresql.org/docs/current/catalogs.html)
- [Supabase Database Connection](https://supabase.com/docs/guides/database/connecting-to-postgres)
