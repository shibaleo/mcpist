# DSN: Module Architecture

## Status

- **Status**: Implemented
- **Date**: 2026-02-11

## Overview

mcpist のモジュールは **2 層構造**で外部 API をラップする。

1. **Module Interface 層** (`internal/modules/`) — MCP プリミティブ (Tools, Resources) を定義し、`map[string]any` で MCP プロトコルと接合する
2. **API Client 層** (`pkg/<service>api/`) — OpenAPI subset spec から ogen で生成した型付きクライアント

```
MCP Host (Claude Code 等)
  │  JSON-RPC: tools/call {module:"github", tool:"get_repo", params:{owner:"x",repo:"y"}}
  ▼
┌──────────────────────────────────────────────────────────┐
│  Module Registry                                          │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ ValidateParams(InputSchema, params)                 │  │  ← Phase 0: 実行前バリデーション
│  └─────────────────────┬───────────────────────────────┘  │
│                        ▼                                  │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ module.ExecuteTool(ctx, name, map[string]any)       │  │  ← Module Interface 層
│  │   └→ handler(ctx, params)                           │  │
│  │        └→ ogenClient.ReposGet(ctx, typed params)    │  │  ← API Client 層
│  │              └→ HTTP GET api.github.com/repos/x/y   │  │
│  └─────────────────────────────────────────────────────┘  │
│                        │                                  │
│                        ▼                                  │
│  json.MarshalIndent(typed response) → string              │
│                        │                                  │
│  (将来) CompactConverter.ToCompact() → TOON/Markdown      │
└──────────────────────────────────────────────────────────┘
```

## Design Decisions

### map[string]any は正当

MCP プロトコルの `tools/call` は任意の JSON を `params` として受け取る。
これは MCP 仕様自体が強制する境界型であり、Go 側で `map[string]any` として受け取るのは不可避。

ただしバリデーションなしで使うのは危険なので、**Phase 0** で `ValidateParams()` を導入した:

- `InputSchema.Required` に基づく必須チェック
- `InputSchema.Properties[key].Type` に基づく型チェック
- `modules.Run()` 内で `ExecuteTool()` の **前** に自動実行

### subset spec = ツール設計

ogen で使う OpenAPI subset spec は「GitHub の API spec のコピー」ではない。
mcpist が **何を返すか** を宣言するスキーマ定義である。

- 必要なフィールドだけを定義 → レスポンスが自動的にフィルタされる
- ツール追加 = subset spec にエンドポイント追記 → `ogen` 再生成 → ハンドラ実装
- 「field filter」と「TOON 変換」は別の関心事:
  - **subset spec** = 何を返すか (schema)
  - **ToCompact()** = どう返すか (format)

### ogen 採用基準

| 条件 | 採用可否 |
|------|----------|
| 公式 OpenAPI spec が存在する API (GitHub, Jira) | ogen 推奨 |
| OpenAPI spec がない/不完全な API (Notion, Trello) | 手書き httpclient |

ogen を使うモジュールと使わないモジュールで、ハンドラのワークロードが大きく変わらないようにする。
違いは「HTTP クライアントの呼び方」だけで、ハンドラの構造 (params 展開 → API 呼出 → JSON 返却) は同一。

## File Structure

### ogen 採用モジュール (GitHub)

```
apps/server/
├── pkg/githubapi/                    # API Client 層
│   ├── openapi-subset.yaml           # ★ subset spec (ツール設計書)
│   ├── ogen.yaml                     # ogen 設定 (server 生成無効化)
│   ├── client.go                     # SecuritySource アダプタ
│   └── gen/                          # ogen 自動生成 (編集不可)
│       ├── oas_client_gen.go         # 型付きクライアントメソッド
│       ├── oas_schemas_gen.go        # レスポンス/リクエスト型
│       ├── oas_parameters_gen.go     # パスパラメータ/クエリ型
│       ├── oas_json_gen.go           # JSON エンコーダ/デコーダ
│       ├── oas_security_gen.go       # Bearer 認証
│       ├── oas_validators_gen.go     # レスポンスバリデータ
│       ├── oas_request_encoders_gen.go
│       ├── oas_response_decoders_gen.go
│       ├── oas_operations_gen.go     # オペレーション名定数
│       └── oas_cfg_gen.go            # クライアント設定
├── internal/modules/
│   ├── types.go                      # Module interface, Tool, InputSchema
│   ├── modules.go                    # Registry, Run() with validation
│   ├── validate.go                   # ValidateParams(), checkType()
│   ├── validate_test.go
│   └── github/
│       └── module.go                 # Module Interface 層 (ハンドラ群)
```

### 手書きモジュール (Notion 等)

```
apps/server/
├── internal/modules/
│   └── notion/
│       ├── module.go                 # Module interface
│       ├── client.go                 # httpclient ベース
│       ├── tools.go                  # Tool 定義 + ハンドラ
│       ├── toon.go                   # TOON 変換
│       └── markdown.go              # Markdown 変換
```

## Handler Pattern

### ogen ハンドラ (GitHub)

```go
func getIssue(ctx context.Context, params map[string]any) (string, error) {
    c, err := newOgenClient(ctx)            // 認証情報取得 → ogen Client 生成
    if err != nil {
        return "", err
    }
    owner, _ := params["owner"].(string)     // map[string]any → Go 値
    repo, _ := params["repo"].(string)
    issueNumber, _ := params["issue_number"].(float64)

    res, err := c.IssuesGet(ctx, gen.IssuesGetParams{  // 型付き呼出
        Owner: owner, Repo: repo, IssueNumber: int(issueNumber),
    })
    if err != nil {
        return "", err
    }
    return toJSON(res)                       // 型付き構造体 → JSON 文字列
}
```

### POST ハンドラ (リクエストボディあり)

```go
func createIssue(ctx context.Context, params map[string]any) (string, error) {
    c, err := newOgenClient(ctx)
    if err != nil {
        return "", err
    }
    owner, _ := params["owner"].(string)
    repo, _ := params["repo"].(string)
    title, _ := params["title"].(string)

    req := &gen.CreateIssueRequest{Title: title}
    if b, ok := params["body"].(string); ok {
        req.Body.SetTo(b)                     // OptString への変換
    }
    if labels, ok := params["labels"].([]interface{}); ok {
        req.Labels = toStringSlice(labels)    // []interface{} → []string
    }

    res, err := c.IssuesCreate(ctx, req, gen.IssuesCreateParams{Owner: owner, Repo: repo})
    if err != nil {
        return "", err
    }
    return toJSON(res)
}
```

## Validation Layer

`modules.Run()` は `ExecuteTool()` を呼ぶ前に自動バリデーションを行う:

```go
// modules.go
func (r *ModuleRegistry) Run(ctx, moduleName, toolName, params) (*ToolCallResult, error) {
    // ...
    if tool, found := findTool(m.Tools(), toolName); found {
        validated, err := ValidateParams(tool.InputSchema, params)
        if err != nil {
            return &ToolCallResult{Content: [{Text: err.Error()}], IsError: true}, nil
        }
        params = validated
    }
    result, err := m.ExecuteTool(ctx, toolName, params)
    // ...
}
```

バリデーション内容:
- `Required` フィールドの存在チェック (空文字列も拒否)
- `Properties[key].Type` に基づく型チェック (`string`, `number`, `boolean`, `array`, `object`)
- 未宣言パラメータはスルー (OpenWorld)

## Regeneration Workflow

ツール追加・変更時のワークフロー:

```
1. openapi-subset.yaml を編集 (エンドポイント追加/フィールド変更)
2. ogen 再生成:
   ogen.exe -package gen -target pkg/githubapi/gen -config pkg/githubapi/ogen.yaml -clean openapi-subset.yaml
3. module.go にハンドラ追加
4. go build ./... で検証
```
