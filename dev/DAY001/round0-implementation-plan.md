# Round 0: 詳細実装計画

## 目的

ローカル環境でGo MCPサーバーを起動し、Claude CodeからSSE経由で`supabase_run_query`を呼び出す最小構成を確立する。

## 前提条件

- Go 1.22以上がインストール済み
- Dockerは使用しない（ローカルGo直接実行）

## 成功条件

- [x] `go run ./cmd/server`でサーバーが起動する
- [x] `/health`エンドポイントがレスポンスを返す
- [x] `/mcp`エンドポイントがHTTP POST（Inline JSON-RPC）を受け付ける
- [x] JSON-RPC 2.0で`initialize`、`tools/list`、`tools/call`が動作する
- [x] `supabase_list_projects`ツールがプロジェクト一覧を取得できる
- [x] `supabase_run_query`ツールがSupabase Management APIを呼び出せる
- [x] Claude Codeの`.mcp.json`に追加し、実際にツールを呼び出せる

**完了日: 2026-01-10**

---

## Phase 1: プロジェクト基盤構築

### 1.1 ディレクトリ構造作成

```
go-mcp-supabase/
├── cmd/
│   └── server/
│       └── main.go           # エントリーポイント
├── internal/
│   ├── auth/
│   │   └── middleware.go     # Bearer token検証
│   ├── mcp/
│   │   ├── handler.go        # SSE + JSON-RPC handler
│   │   └── types.go          # JSON-RPC 2.0 型定義
│   └── modules/
│       └── supabase/
│           └── run_query.go  # supabase_run_query実装
├── .env
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

### 1.2 go.mod 初期化

```go
module github.com/shibadogcatfrog/go-mcp-supabase

go 1.22
```

外部依存: なし（標準ライブラリのみ）

### 1.3 環境変数

**.env.example**
```bash
# MCPサーバー認証
INTERNAL_SECRET=your-local-secret-here

# Supabase Management API
SUPABASE_ACCESS_TOKEN=sbp_xxxxxxxxxxxx
```

**.env**はGit管理外（.gitignoreに追加済み）

---

## Phase 2: HTTPサーバー + 認証

### 2.1 main.go（エントリーポイント）

```go
// cmd/server/main.go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/shibadogcatfrog/go-mcp-supabase/internal/auth"
    "github.com/shibadogcatfrog/go-mcp-supabase/internal/mcp"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    handler := mcp.NewHandler()
    authMiddleware := auth.NewMiddleware(os.Getenv("INTERNAL_SECRET"))

    http.HandleFunc("/health", healthHandler)
    http.Handle("/mcp", authMiddleware(handler))

    log.Printf("Starting MCP server on :%s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}
```

### 2.2 認証ミドルウェア

```go
// internal/auth/middleware.go
package auth

import (
    "net/http"
    "strings"
)

func NewMiddleware(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := r.Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            token := strings.TrimPrefix(auth, "Bearer ")
            if token != secret {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Phase 3: MCP/SSE実装

### 3.1 JSON-RPC 2.0 型定義

```go
// internal/mcp/types.go
package mcp

// JSON-RPC 2.0 Request
type Request struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params,omitempty"`
}

// JSON-RPC 2.0 Response
type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *Error      `json:"error,omitempty"`
}

// JSON-RPC 2.0 Error
type Error struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// MCP Error Codes (JSON-RPC 2.0 + MCP拡張)
const (
    ParseError     = -32700
    InvalidRequest = -32600
    MethodNotFound = -32601
    InvalidParams  = -32602
    InternalError  = -32603
)

// MCP Protocol Types
type InitializeParams struct {
    ProtocolVersion string            `json:"protocolVersion"`
    Capabilities    ClientCapabilities `json:"capabilities"`
    ClientInfo      ClientInfo        `json:"clientInfo"`
}

type ClientCapabilities struct {
    Roots        *RootsCapability    `json:"roots,omitempty"`
    Sampling     *SamplingCapability `json:"sampling,omitempty"`
}

type RootsCapability struct {
    ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct{}

type ClientInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type InitializeResult struct {
    ProtocolVersion string             `json:"protocolVersion"`
    Capabilities    ServerCapabilities `json:"capabilities"`
    ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
    Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
    ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type Tool struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
    Type       string              `json:"type"`
    Properties map[string]Property `json:"properties"`
    Required   []string            `json:"required,omitempty"`
}

type Property struct {
    Type        string `json:"type"`
    Description string `json:"description"`
}

type ToolsListResult struct {
    Tools []Tool `json:"tools"`
}

type ToolCallParams struct {
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
}

type ToolCallResult struct {
    Content []ContentBlock `json:"content"`
    IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
    Type string `json:"type"`
    Text string `json:"text"`
}
```

### 3.2 SSE Handler

```go
// internal/mcp/handler.go
package mcp

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "net/http"

    "github.com/shibadogcatfrog/go-mcp-supabase/internal/modules/supabase"
)

type Handler struct {
    tools map[string]ToolExecutor
}

type ToolExecutor interface {
    Execute(args map[string]interface{}) (string, error)
    Definition() Tool
}

func NewHandler() *Handler {
    h := &Handler{
        tools: make(map[string]ToolExecutor),
    }

    // Register supabase tools
    h.RegisterTool(supabase.NewRunQueryTool())

    return h
}

func (h *Handler) RegisterTool(tool ToolExecutor) {
    def := tool.Definition()
    h.tools[def.Name] = tool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    // Send endpoint event (MCP SSE protocol)
    fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
    flusher.Flush()

    // Read JSON-RPC messages from request body
    scanner := bufio.NewScanner(r.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if line == "" {
            continue
        }

        var req Request
        if err := json.Unmarshal([]byte(line), &req); err != nil {
            h.sendError(w, flusher, nil, ParseError, "Parse error")
            continue
        }

        h.handleRequest(w, flusher, &req)
    }
}

func (h *Handler) handleRequest(w http.ResponseWriter, flusher http.Flusher, req *Request) {
    var result interface{}
    var rpcErr *Error

    switch req.Method {
    case "initialize":
        result = h.handleInitialize(req)
    case "initialized":
        // Notification, no response needed
        return
    case "tools/list":
        result = h.handleToolsList()
    case "tools/call":
        result, rpcErr = h.handleToolCall(req)
    default:
        rpcErr = &Error{Code: MethodNotFound, Message: "Method not found"}
    }

    if rpcErr != nil {
        h.sendError(w, flusher, req.ID, rpcErr.Code, rpcErr.Message)
        return
    }

    h.sendResult(w, flusher, req.ID, result)
}

func (h *Handler) handleInitialize(req *Request) *InitializeResult {
    return &InitializeResult{
        ProtocolVersion: "2024-11-05",
        Capabilities: ServerCapabilities{
            Tools: &ToolsCapability{},
        },
        ServerInfo: ServerInfo{
            Name:    "go-mcp-supabase",
            Version: "0.1.0",
        },
    }
}

func (h *Handler) handleToolsList() *ToolsListResult {
    tools := make([]Tool, 0, len(h.tools))
    for _, executor := range h.tools {
        tools = append(tools, executor.Definition())
    }
    return &ToolsListResult{Tools: tools}
}

func (h *Handler) handleToolCall(req *Request) (*ToolCallResult, *Error) {
    paramsBytes, err := json.Marshal(req.Params)
    if err != nil {
        return nil, &Error{Code: InvalidParams, Message: "Invalid params"}
    }

    var params ToolCallParams
    if err := json.Unmarshal(paramsBytes, &params); err != nil {
        return nil, &Error{Code: InvalidParams, Message: "Invalid params structure"}
    }

    executor, ok := h.tools[params.Name]
    if !ok {
        return nil, &Error{Code: InvalidParams, Message: fmt.Sprintf("Unknown tool: %s", params.Name)}
    }

    result, err := executor.Execute(params.Arguments)
    if err != nil {
        return &ToolCallResult{
            Content: []ContentBlock{{Type: "text", Text: err.Error()}},
            IsError: true,
        }, nil
    }

    return &ToolCallResult{
        Content: []ContentBlock{{Type: "text", Text: result}},
    }, nil
}

func (h *Handler) sendResult(w http.ResponseWriter, flusher http.Flusher, id interface{}, result interface{}) {
    resp := Response{
        JSONRPC: "2.0",
        ID:      id,
        Result:  result,
    }
    data, _ := json.Marshal(resp)
    fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
    flusher.Flush()
    log.Printf("Sent response: %s", data)
}

func (h *Handler) sendError(w http.ResponseWriter, flusher http.Flusher, id interface{}, code int, message string) {
    resp := Response{
        JSONRPC: "2.0",
        ID:      id,
        Error:   &Error{Code: code, Message: message},
    }
    data, _ := json.Marshal(resp)
    fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
    flusher.Flush()
    log.Printf("Sent error: %s", data)
}
```

---

## Phase 4: Supabase Run Query ツール

### 4.1 ツール実装

```go
// internal/modules/supabase/run_query.go
package supabase

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"

    "github.com/shibadogcatfrog/go-mcp-supabase/internal/mcp"
)

type RunQueryTool struct {
    accessToken string
    httpClient  *http.Client
}

func NewRunQueryTool() *RunQueryTool {
    return &RunQueryTool{
        accessToken: os.Getenv("SUPABASE_ACCESS_TOKEN"),
        httpClient:  &http.Client{},
    }
}

func (t *RunQueryTool) Definition() mcp.Tool {
    return mcp.Tool{
        Name:        "supabase_run_query",
        Description: "Execute a SQL query against a Supabase project database using the Management API",
        InputSchema: mcp.InputSchema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "project_ref": {
                    Type:        "string",
                    Description: "The Supabase project reference ID",
                },
                "query": {
                    Type:        "string",
                    Description: "The SQL query to execute",
                },
            },
            Required: []string{"project_ref", "query"},
        },
    }
}

func (t *RunQueryTool) Execute(args map[string]interface{}) (string, error) {
    projectRef, ok := args["project_ref"].(string)
    if !ok {
        return "", fmt.Errorf("project_ref must be a string")
    }

    query, ok := args["query"].(string)
    if !ok {
        return "", fmt.Errorf("query must be a string")
    }

    // Supabase Management API endpoint
    url := fmt.Sprintf("https://api.supabase.com/v1/projects/%s/database/query", projectRef)

    payload := map[string]string{"query": query}
    body, err := json.Marshal(payload)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequest("POST", url, bytes.NewReader(body))
    if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+t.accessToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("failed to read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
    }

    // Pretty print JSON response
    var result interface{}
    if err := json.Unmarshal(respBody, &result); err != nil {
        return string(respBody), nil
    }

    prettyJSON, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return string(respBody), nil
    }

    return string(prettyJSON), nil
}
```

---

## Phase 5: 開発環境構成

### 5.1 Makefile

```makefile
.PHONY: run build test health clean

# サーバー起動（ホットリロードなし）
run:
	go run ./cmd/server

# ビルド
build:
	go build -o bin/server ./cmd/server

# ヘルスチェック
health:
	curl -s http://localhost:8080/health

# SSE接続テスト
test-sse:
	curl -N -X POST http://localhost:8080/mcp \
		-H "Authorization: Bearer test-secret" \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# クリーンアップ
clean:
	rm -rf bin/
```

### 5.2 .env読み込み

Goは標準で.envを読まないため、起動時に環境変数を設定:

**PowerShell (Windows)**
```powershell
# .envファイルを読み込んで実行
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^=]+)=(.*)$') {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
    }
}
go run ./cmd/server
```

**Bash (Linux/Mac)**
```bash
# .envファイルを読み込んで実行
export $(cat .env | xargs) && go run ./cmd/server
```

または、godotenvパッケージを使用（後述のmain.goで対応）

---

## Phase 6: Claude Code連携

### 6.1 MCP設定（claude_desktop_config.jsonまたはClaude Code設定）

Claude Codeでローカルホストに接続するには、以下の設定を追加:

```json
{
  "mcpServers": {
    "go-mcp-supabase": {
      "url": "http://localhost:8080/mcp",
      "transport": "sse",
      "headers": {
        "Authorization": "Bearer your-local-secret-here"
      }
    }
  }
}
```

### 6.2 動作確認手順

1. サーバー起動
   ```bash
   make dev
   ```

2. ヘルスチェック
   ```bash
   make health
   ```

3. Claude Codeから接続テスト
   - Claude Codeを起動
   - MCPサーバーとして`go-mcp-supabase`が認識されることを確認
   - `supabase_run_query`ツールが利用可能なことを確認

4. クエリ実行テスト
   ```
   Claude Codeで: "supabase_run_queryを使って、テーブル一覧を取得して"
   ```

---

## 実装チェックリスト

### Day 1: 基盤実装

- [x] ディレクトリ構造作成
- [x] go.mod初期化
- [x] .env.example作成
- [x] main.go実装
- [x] auth/middleware.go実装
- [x] mcp/types.go実装
- [x] mcp/handler.go実装（GET: SSE、POST: Inline JSON-RPC対応）
- [x] Makefile作成
- [x] `go run ./cmd/server`でサーバー起動確認
- [x] `/health`エンドポイント動作確認

### Day 2: Supabase連携 + Claude Code接続

- [x] modules/supabase/list_projects.go実装
- [x] modules/supabase/run_query.go実装
- [x] Supabase Access Token取得・設定
- [x] ローカルでsupabase_list_projects動作確認
- [x] ローカルでsupabase_run_query動作確認
- [x] Claude Code `.mcp.json`設定追加
- [x] Claude Codeからの接続確認
- [x] 実際のクエリ実行テスト（rawスキーマのテーブル一覧取得）

---

## 技術的注意点

### SSE実装

MCP SSEプロトコルでは:
- 初回接続時に`endpoint`イベントを送信
- JSON-RPCメッセージは`message`イベントで送信
- クライアントからのリクエストはPOST bodyで受信

### JSON-RPC 2.0準拠

- `jsonrpc`フィールドは必ず`"2.0"`
- `id`はリクエストに含まれていれば同じ値をレスポンスに含める
- `notifications`（`id`なし）にはレスポンスを返さない

### エラーハンドリング

- Supabase APIエラーは`isError: true`でツール結果として返す
- JSON-RPCレベルのエラーは標準エラーコードを使用

---

## 次のステップ（Round 1への準備）

Round 0完了後、以下をRound 1で実施:
- SSE/JSON-RPCの改善（実運用での挙動フィードバック反映）
- Supabase残りツール追加（get_table_schema等）
- 最小CI（GitHub Actions）構築
- Loki送信実装（Round 0では省略、ローカル確認優先）

---

## 実装メモ

### 変更点（計画からの差分）

1. **MCP通信方式**: SSE専用ではなく、HTTP POST Inline JSON-RPC方式を採用
   - Claude Codeの`type: "http"`設定に対応
   - GET: SSEセッション確立、POST: JSON-RPCメッセージ処理
   - sessionIdなしのPOSTはInline応答（SSEなし）

2. **ツール追加**: `supabase_list_projects`を追加
   - パラメータなしでプロジェクト一覧取得

3. **エラーハンドリング**: HTTPステータス200-299を成功として扱う
   - Supabase APIがクエリ成功時に201を返すため

### 最終ディレクトリ構造

```
go-mcp-supabase/
├── cmd/server/main.go
├── internal/
│   ├── auth/middleware.go
│   ├── mcp/
│   │   ├── handler.go
│   │   └── types.go
│   └── modules/supabase/
│       ├── list_projects.go
│       └── run_query.go
├── .env
├── .env.example
├── .mcp.json
├── go.mod
└── Makefile
```

---

## 起動手順まとめ

```powershell
# 1. 環境変数設定
cp .env.example .env
# .envを編集してトークンを設定

# 2. 環境変数読み込み + サーバー起動
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^=]+)=(.*)$') {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
    }
}
go run ./cmd/server

# 3. 別ターミナルでヘルスチェック
curl http://localhost:8080/health
```
