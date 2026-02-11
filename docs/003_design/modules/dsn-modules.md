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

## Response Format Strategy

ツールのレスポンスは **3 レベル** で制御する。

### Level 1: Spec Level (フィールドフィルタ)

ogen の subset spec が自動的にフィールドフィルタとして機能する。

- `Decode()`: JSON 内の未定義フィールドは `d.Skip()` で読み飛ばし
- `Encode()`: struct に定義されたフィールドのみシリアライズ

例: GitHub `get_repo` は API の 80+ フィールドから subset spec 定義の 22 フィールドのみ返す。

**→ 単一ツール (ogen) は `toJSON(res)` でそのまま返す。追加のフィールドフィルタは不要。**

### Level 2: Field Selection (複合ツール)

複数 API を並行呼出しする複合ツールは、ハンドラ内でフィールドを選別する。

- 各 API レスポンスから **概要に必要なフィールドだけ** を `map[string]any` に抽出
- 長文テキスト (body, README) は切り詰め
- `_note` フィールドで「詳細は個別ツールで取得」と補足
- 結果は JSON (`toJSON`) で返す — LLM が直接解析する用途に最適

例: `describe_repo` = get_repo + README + branches + issues + PRs → 概要 JSON

### Level 3: Format Conversion (構造変換)

複雑なネスト構造を持つ API (Notion 等) は、JSON → 人間/LLM 可読フォーマットに変換する。

| Format | Use Case |
|--------|----------|
| Markdown | ページ内容、リッチテキスト (Notion blocks → MD) |
| CSV/Table | テーブルデータ、一覧 (Notion DB → CSV) |
| TOON | 汎用コンパクト形式 (将来) |

ogen モジュールでは Level 1 で十分なため Level 3 は不要。
手書きモジュール (Notion 等) で必要に応じて実装する。

### 判断基準

```
ツール種別の判定:
  単一 API (ogen)  → Level 1: toJSON(res) そのまま
  複合ツール       → Level 2: フィールド選別 + toJSON
  複雑構造 (手書き) → Level 3: MD/CSV/TOON 変換
```

## Composite Tool Design

### 定義

**複合ツール** = 複数 API を goroutine で並行呼出し → フィールド選別 → 1 JSON で返すツール。

### 設計基準

1. **API 呼出回数が固定**であること (入力によって変動しない)
2. **構成する API 群が自明**であること (常に同じ組み合わせ)
3. **概要把握の需要がある**こと (個別に呼ぶと手間)

### 不採用にしたもの

| 候補 | 理由 |
|------|------|
| `describe_issue` | 単一 API (get_issue) で十分。合成の価値なし |
| `user_stats` (list 系全呼出) | ページネーション = API 呼出回数が無制限 |
| `describe_ci` | workflow_id が入力依存で呼出回数不定 |
| `search_and_read` | 検索結果に依存して呼出回数不定 |

## Regeneration Workflow

ツール追加・変更時のワークフロー:

```
1. openapi-subset.yaml を編集 (エンドポイント追加/フィールド変更)
2. ogen 再生成:
   ogen.exe -package gen -target pkg/githubapi/gen -config pkg/githubapi/ogen.yaml -clean openapi-subset.yaml
3. module.go にハンドラ追加
4. go build ./... で検証
```
