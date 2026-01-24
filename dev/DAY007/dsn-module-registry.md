# DSN: Module Registry - 2層抽象化アーキテクチャ

## Status
- **Status**: Prototype Complete
- **Branch**: `dev/prototype-module-registry`
- **Date**: 2026-01-16

## Overview

Module Registry は MCP プリミティブを 2 層で抽象化するアーキテクチャを定義する。

**従来の MCP:**
```
MCP Host → Tools/Resources/Prompts (フラット)
```

**Module Registry:**
```
MCP Host → Module Registry → Module → Tools/Resources/Prompts (2層)
```

この 2 層構造により：
- **モジュール単位の管理**: 関連するツール群をモジュールとしてグループ化
- **遅延読み込み**: `get_module_schema` で必要なモジュールのみスキーマを取得
- **トークン効率**: 出力を TOON/Markdown 形式に変換して削減
- **バッチ実行**: モジュール横断で複数ツールを DAG ベースで並列実行

## System Architecture

Module Registry は MCPist システムの中核コンポーネントである。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCP Host (Claude Code 等)                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       │ MCP Protocol (JSON-RPC)
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCPist Server                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐  │
│  │  Auth Middleware │───▶│ Permission Gate │───▶│  MCP Protocol Handler   │  │
│  │  (JWT検証)       │    │  (ツールマスク)  │    │  (tools/list, call)     │  │
│  └─────────────────┘    └─────────────────┘    └───────────┬─────────────┘  │
│                                                            │                │
│  ┌─────────────────────────────────────────────────────────▼─────────────┐  │
│  │                     ★ Module Registry ★                              │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │  Meta Tools: get_module_schema, call_module_tool, batch          │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                              │                                        │  │
│  │         ┌────────────────────┼────────────────────┐                   │  │
│  │         ▼                    ▼                    ▼                   │  │
│  │  ┌────────────┐       ┌────────────┐       ┌────────────┐             │  │
│  │  │   Notion   │       │   GitHub   │       │    Jira    │  ...        │  │
│  │  │  Module    │       │   Module   │       │   Module   │             │  │
│  │  ├────────────┤       ├────────────┤       ├────────────┤             │  │
│  │  │ Tools      │       │ Tools      │       │ Tools      │             │  │
│  │  │ Resources  │       │ Resources  │       │ Resources  │             │  │
│  │  │ Prompts    │       │ Prompts    │       │ Prompts    │             │  │
│  │  └─────┬──────┘       └─────┬──────┘       └─────┬──────┘             │  │
│  └────────┼────────────────────┼────────────────────┼────────────────────┘  │
│           │                    │                    │                       │
│  ┌────────▼────────────────────▼────────────────────▼────────────────────┐  │
│  │                         Token Vault                                   │  │
│  │                    (Supabase Vault + Edge Functions)                  │  │
│  │              サービス認証情報の安全な保管・取得                          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       │ HTTPS
                                       ▼
              ┌────────────────────────────────────────────────┐
              │              External APIs                      │
              │  Notion API │ GitHub API │ Jira API │ ...       │
              └────────────────────────────────────────────────┘
```

**Module Registry の役割:**
- **Meta Tools 提供**: `get_module_schema`, `call`, `batch` の 3 つのエントリーポイント
- **モジュールルーティング**: リクエストを適切なモジュールに振り分け
- **出力形式変換**: JSON → TOON/Markdown でトークン効率を最適化
- **バッチ実行**: DAG ベースの依存関係解決と並列実行

## Goals

1. **MCP 準拠**: 各モジュールが Tools, Resources, Prompts を持つ
2. **トークン効率**: JSON より 90%+ 小さい出力形式（TOON/Markdown）
3. **並列実行**: DAG ベースの依存関係解決と goroutine による並列処理
4. **段階的移行**: Legacy モジュールとの後方互換性

## Architecture

### Module Interface

```go
// Module defines the interface that all modules must implement.
// Each module provides Tools, Resources, and Prompts (MCP 3 primitives).
type Module interface {
    // Metadata
    Name() string
    Description() string
    APIVersion() string

    // Tools - LLM executes, has side effects
    Tools() []Tool
    ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error)

    // Resources - LLM reads, no side effects
    Resources() []Resource
    ReadResource(ctx context.Context, uri string) (string, error)

    // Prompts - Templates/workflows
    Prompts() []Prompt
    GetPrompt(ctx context.Context, name string, args map[string]any) (string, error)
}
```

### CompactConverter Interface

```go
// CompactConverter provides optional compact format conversion (TOON/Markdown)
// Modules that implement this can convert their JSON output to token-efficient formats
type CompactConverter interface {
    // ToCompact converts JSON result to compact format (TOON or Markdown)
    // toolName is used to select the appropriate format for each tool
    ToCompact(toolName string, jsonResult string) string
}
```

### Registry Structure

```
┌─────────────────────────────────────────────────────────────┐
│                      Registry                                │
├─────────────────────────────────────────────────────────────┤
│  registry map[string]Module    (new interface)              │
│  Registry map[string]ModuleDefinition  (legacy, deprecated) │
└─────────────────────────────────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   notion    │   │   github    │   │    jira     │
│  (Module)   │   │  (Legacy)   │   │  (Legacy)   │
├─────────────┤   └─────────────┘   └─────────────┘
│ Tools       │
│ Resources   │
│ Prompts     │
│ ToCompact() │ ← CompactConverter
└─────────────┘
```

## Meta Tools

レジストリは 3 つのメタツールを提供する：

### 1. get_module_schema

複数モジュールのスキーマ（Tools, Resources, Prompts）を一括取得。

**入力:**
```json
{
  "modules": ["notion", "jira"]
}
```

**出力:**
```json
[
  {
    "module": "notion",
    "description": "Notion API - ページ・データベース・ブロック操作",
    "api_version": "2022-06-28",
    "tools": [...],
    "resources": [...],
    "prompts": [...]
  },
  {
    "module": "jira",
    "description": "Jira API - Issue/Project操作",
    "api_version": "3",
    "tools": [...],
    "resources": [...],
    "prompts": [...]
  }
]
```

### 2. call

単発のツール実行。

```json
{
  "module": "notion",
  "tool": "search",
  "params": {"query": "設計"}
}
```

### 3. batch

複数ツールの一括実行（JSONL 形式、DAG ベース並列実行）。

**フィールド:**
| Field | Required | Description |
|-------|----------|-------------|
| id | Yes | タスク識別子 |
| module | Yes | モジュール名 |
| tool | Yes | ツール名 |
| params | No | ツールパラメータ |
| after | No | 依存タスク ID 配列 |
| output | No | true で TOON/MD 形式で結果を返却 |
| raw_output | No | true で JSON 形式で結果を返却（output より優先）|

**変数参照:**
```
${taskId.results[index].field}
```

**例: 連鎖処理**
```jsonl
{"id":"search","module":"notion","tool":"search","params":{"query":"設計"}}
{"id":"page","module":"notion","tool":"get_page_content","params":{"page_id":"${search.results[0].id}"},"after":["search"],"output":true}
```

## Output Format

### 出力形式の選択

| output | raw_output | 結果 |
|--------|------------|------|
| false | false | 出力なし（変数参照用） |
| true | false | TOON/Markdown 形式 |
| false | true | JSON 形式 |
| true | true | JSON 形式（raw_output 優先） |

### TOON Format

2D データ（リスト、テーブル）向けのトークン効率フォーマット。

```
pages[3]{id,title,type}:
  abc123,設計ドキュメント,page
  def456,API仕様,page
  ghi789,DB設計,database
```

**JSON (1,500+ chars) → TOON (150 chars) = 90%+ 削減**

### Markdown Format

ページコンテンツ向けの可読フォーマット。

```markdown
# タイトル

これは本文です。

- リスト項目1
- リスト項目2

```code
function example() {}
```
```

## Implementation Details

### Notion Module Structure

```
internal/modules/notion/
├── module.go      # Module interface implementation
├── client.go      # HTTP client, auth headers
├── tools.go       # Tool definitions and handlers
├── toon.go        # TOON format converter
├── markdown.go    # Markdown format converter
├── resources.go   # Resource definitions (optional)
└── prompts.go     # Prompt definitions (optional)
```

### Module Implementation Pattern

```go
// NotionModule implements the Module interface
type NotionModule struct{}

func New() *NotionModule {
    return &NotionModule{}
}

// Metadata
func (m *NotionModule) Name() string        { return "notion" }
func (m *NotionModule) Description() string { return "Notion API - ページ・データベース・ブロック操作" }
func (m *NotionModule) APIVersion() string  { return "2022-06-28" }

// Tools
func (m *NotionModule) Tools() []modules.Tool {
    return toolDefinitions()
}

func (m *NotionModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
    handler, ok := toolHandlers[name]
    if !ok {
        return "", fmt.Errorf("unknown tool: %s", name)
    }
    return handler(params) // Always returns JSON
}

// CompactConverter
func (m *NotionModule) ToCompact(toolName string, jsonResult string) string {
    return ToTOON(toolName, jsonResult) // Converts to TOON or Markdown
}
```

### Batch Execution Flow

```
1. Parse JSONL commands
2. Validate dependencies (detect cycles with DFS)
3. Execute tasks with goroutines
   - Wait for dependencies (channel sync)
   - Resolve variable references from resultStore
   - Execute tool (always JSON internally)
   - Store result for dependent tasks
4. Build response
   - Apply CompactConverter if output:true
   - Return JSON if raw_output:true
```

## Auto-Pagination

`get_page_content` は `fetch_all: true` パラメータで自動ページネーションをサポート。

```json
{
  "page_id": "xxx",
  "fetch_all": true,
  "page_size": 100
}
```

- `fetch_all: false` (default): 1 回のリクエストで `page_size` 件取得
- `fetch_all: true`: `has_more` が `false` になるまでループ

## Migration Path

### Legacy → New Interface

1. **Phase 1**: `ModuleDefinition` で動作確認済み（現状）
2. **Phase 2**: `Module` インターフェース実装（Notion 完了）
3. **Phase 3**: `CompactConverter` 実装でトークン削減
4. **Phase 4**: 他モジュールを順次移行

### Backward Compatibility

```go
// Registry checks new interface first, then falls back to legacy
if m, ok := registry[moduleName]; ok {
    return m.ExecuteTool(ctx, toolName, params)
}
// Fallback to legacy
if module, ok := Registry[moduleName]; ok {
    return module.Handlers[toolName](params)
}
```

## Performance Results

### Token Reduction (Notion get_page_content)

| Format | Size | Reduction |
|--------|------|-----------|
| JSON | 1,946 chars | - |
| Markdown | 180 chars | 90.8% |

### Batch Execution

- **並列タスク**: goroutine で同時実行
- **依存タスク**: channel で同期、順次実行
- **ページネーション**: `fetch_all: true` で自動ループ（順次、並列化不可）

## Files Changed

- `internal/modules/types.go` - Module, CompactConverter interfaces
- `internal/modules/registry.go` - Registry, MetaTools, Batch execution
- `internal/modules/notion/module.go` - Module implementation
- `internal/modules/notion/tools.go` - Tool definitions, fetch_all
- `internal/modules/notion/toon.go` - TOON converter
- `internal/modules/notion/markdown.go` - Markdown converter
- `internal/mcp/handler.go` - Handler routing

## Future Work

1. **Resources/Prompts 実装**: 各モジュールで Resources, Prompts を定義
2. **他モジュール移行**: GitHub, Jira, Confluence 等を新インターフェースに移行
3. **キャッシュ**: 頻繁にアクセスするリソースのキャッシュ
4. **Rate Limiting**: API レート制限の考慮
