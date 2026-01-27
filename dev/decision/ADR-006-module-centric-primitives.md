---
title: ADR-006 モジュール中心アーキテクチャによる3プリミティブ統合
aliases:
  - ADR-006
  - module-centric-primitives
tags:
  - MCPist
  - ADR
  - architecture
  - MCP
document-type:
  - ADR
document-class: ADR
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# ADR-006: モジュール中心アーキテクチャによる3プリミティブ統合

## ステータス

採用

## コンテキスト

MCPist spec-sys.mdの初期設計では、MCP標準の3プリミティブ（Tools, Resources, Prompts）を以下の3層構造で分離していた:

**既存のシステム構成図（spec-sys.md § 1.1より）:**

```
┌──────────────────────────────────────────────────────────────────────┐
│  MCPサーバー (Go スクリプト, Koyebホスティング)                           │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │  認証ミドルウェア                                                │   │
│  │  - JWT検証（Authサーバー発行トークン）                               │   │
│  │  - user_accountID抽出                                            │   │
│  └───────────────────────────┬─────────────────────────────────────┘   │
│                              │                                          │
│  ┌───────────────────────────▼─────────────────────────────────────┐   │
│  │  MCPプロトコルハンドラ                                             │   │
│  │  - JSON-RPC 2.0リクエスト処理                                      │   │
│  │  - SSEセッション管理                                               │   │
│  │  - Tools/Resources/Promptsルーティング                             │   │
│  └─────┬─────────────┬─────────────┬──────────────────────────────┘   │
│        │             │             │                                   │
│        │(Tools)      │(Resources)  │(Prompts)                          │
│        │             │             │                                   │
│  ┌─────▼─────┐ ┌─────▼─────┐ ┌───▼───────┐                           │
│  │モジュール  │ │リソース    │ │プロンプト  │                           │
│  │レジストリ  │ │プロバイダ  │ │ライブラリ  │                           │
│  └─────┬─────┘ └─────┬─────┘ └───┬───────┘                           │
│        │             │             │                                   │
│        └─────────────┴─────────────┘                                   │
│                      │                                                 │
│  ┌───────────────────▼─────────────────────────────────────────────┐  │
│  │  外部モジュール群      (拡張可能)                                  │  │
│  │  ┌─────────┬─────────┬─────────┬─────────┬─────────┐            │  │
│  │  │ Notion  │ GitHub  │  Jira   │Confluence│ Supabase│  ...      │  │
│  │  └─────────┴─────────┴─────────┴─────────┴─────────┘            │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                              │                                          │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
    ┌──────────────────────────┼──────────────────────────┐
    │                          │                          │
    ▼                          ▼                          ▼
┌─────────────────┐  ┌─────────────────┐  ┌───────────────────┐
│   Authサーバー   │  │  Token Broker    │  │    外部API群       │
│   (OAuth2.1)    │  │ (暗号化Vault)     │  │                   │
└─────────────────┘  └─────────────────┘  └───────────────────┘
```

**既存の処理フロー（spec-sys.md § 2.1より）:**

```
リクエスト受信 (Bearer JWT)
    │
    ▼
[認証ミドルウェア] JWT検証 → user_id抽出
    │
    ▼
[MCPプロトコルハンドラ] リクエスト種別判定
    │
    ├─ tools/list, tools/call
    │   └─> [モジュールレジストリ] ツール実行
    │
    ├─ resources/list, resources/read, resources/subscribe
    │   └─> [リソースプロバイダ] リソース取得
    │        └─> [モジュールレジストリ] データ取得
    │
    └─ prompts/list, prompts/get
        └─> [プロンプトライブラリ] プロンプト返却
    │
    │  ※全てToken Brokerからトークン取得
    ▼
レスポンス返却
```

しかし、実装を進める中で以下の問題が明らかになった:

1. **責務の重複**: モジュールレジストリ、リソースプロバイダ、プロンプトライブラリは全て「外部サービス（GitHub, Notion等）への接続」という同じ役割を持つ
2. **認証コードの重複**: 各プリミティブ層が独立してToken Brokerと通信する設計では、GitHub/Notion等のサービスごとに3回認証ロジックを実装することになる（DRY原則違反）
3. **境界の曖昧さ**: 3層の違いが「返すデータ型」だけで、アーキテクチャ上の明確な分離理由がない

MCP仕様の確認:
- [MCP公式ドキュメント](https://modelcontextprotocol.io/docs/concepts/architecture)によると、Serverは`ServerCapabilities`で3プリミティブを宣言する
- 各プリミティブは独立した機能だが、同一のServer実装内で提供される
- 公式Go SDK（`github.com/mark3labs/mcp-go`）でもServer構造体が3プリミティブを統合管理

## 検討した選択肢

### 選択肢1: 3層分離アーキテクチャ（初期設計）

```
MCPプロトコルハンドラ
├─ tools/* → モジュールレジストリ（GitHub, Notion...）
├─ resources/* → リソースプロバイダ（GitHub, Notion...）
└─ prompts/* → プロンプトライブラリ（GitHub, Notion...）
```

**メリット:**
- プリミティブごとに責務が分離されている（に見える）
- MCP仕様のメソッド（`tools/*`, `resources/*`, `prompts/*`）との対応が明確

**デメリット:**
- 同じ外部サービス（GitHub等）への接続ロジックが3箇所に重複
- Token Broker呼び出しが各層で重複（DRY原則違反）
- 「プロバイダ」「ライブラリ」という命名だが、実態は同じモジュール群を参照
- 新しいサービス追加時に3箇所修正が必要

### 選択肢2: モジュール中心アーキテクチャ（採用）

```
MCPプロトコルハンドラ
├─ tools/* ──┐
├─ resources/*─┼─► モジュールレジストリ
└─ prompts/*──┘
                ├─ GitHub Module
                │   ├─ client.go (Token Broker → HTTP Client)
                │   ├─ tools.go
                │   ├─ resources.go
                │   └─ prompts.go
                ├─ Notion Module
                │   ├─ client.go (Token Broker → HTTP Client)
                │   ├─ tools.go
                │   ├─ resources.go
                │   └─ prompts.go
                └─ ...
```

**メリット:**
- **DRY原則遵守**: 各モジュールに1つの`client.go`で認証を集約
- **責務の明確化**: GitHub Module = GitHubに関する全てのプリミティブを提供
- **拡張性**: 新サービス追加時は1つのモジュールディレクトリを追加するだけ
- **実装の簡潔さ**: `tools.go`, `resources.go`, `prompts.go`は全て同じ`client`を使用

**デメリット:**
- 初期の3層構造から設計変更が必要

## 決定

**選択肢2（モジュール中心アーキテクチャ）を採用**

### アーキテクチャ構造

```
MCPプロトコルハンドラ (internal/mcp/handler.go)
├─ HandleToolsList() → modules.MetaTools()
├─ HandleToolsCall() → modules.CallModuleTool()
├─ HandleResourcesList() → modules.ListResources()
├─ HandleResourcesRead() → modules.ReadResource()
├─ HandlePromptsList() → modules.ListPrompts()
└─ HandlePromptsGet() → modules.GetPrompt()

モジュールレジストリ (internal/modules/registry.go)
├─ Registry map[string]ModuleDefinition
├─ MetaTools() → get_module_schema, call_module_tool
├─ ListResources(module) → []ResourceTemplate
├─ ReadResource(module, uri) → string
├─ ListPrompts(module) → []PromptTemplate
└─ GetPrompt(module, name) → Prompt

ModuleDefinition (internal/modules/types.go)
├─ Name, Description, APIVersion, TestedAt
├─ Tools []Tool + Handlers map[string]ToolHandler
├─ Resources []ResourceTemplate + ResourceResolver
└─ Prompts []PromptTemplate + PromptRenderer

具体的なモジュール実装例: internal/modules/github/
├─ module.go → Module() returns ModuleDefinition
├─ client.go → NewClient(tokenBroker, userID) *GitHubClient
├─ tools.go → registerTools() ([]Tool, map[string]ToolHandler)
├─ resources.go → registerResources() ([]ResourceTemplate, ResourceResolverFunc)
└─ prompts.go → registerPrompts() ([]PromptTemplate, PromptRendererFunc)
```

### 実装パターン

各モジュール（GitHub, Notion等）は以下の構造を持つ:

```go
// client.go - Token Brokerとの通信を集約
type GitHubClient struct {
    httpClient *http.Client
    baseURL    string
}

func NewClient(tokenBroker TokenBroker, userID, service string) (*GitHubClient, error) {
    token, err := tokenBroker.GetToken(userID, service)
    if err != nil {
        return nil, err
    }
    // トークンを使ってHTTPクライアント初期化
    return &GitHubClient{...}, nil
}

// tools.go, resources.go, prompts.go は全て同じclientを使用
func (c *GitHubClient) ListRepositories(params map[string]interface{}) (string, error) {...}
func (c *GitHubClient) GetFileContent(uri string) (string, error) {...}
func (c *GitHubClient) RenderIssuePrompt(name string, args map[string]interface{}) (string, error) {...}
```

## 根拠

### 1. DRY原則の徹底

**問題**: 初期設計では、GitHubに対して3箇所で認証が必要
- モジュールレジストリのGitHub Tools → Token Broker呼び出し
- リソースプロバイダのGitHub Resources → Token Broker呼び出し
- プロンプトライブラリのGitHub Prompts → Token Broker呼び出し

**解決**: モジュール中心設計では1箇所のみ
- `github/client.go`のみがToken Brokerを呼び出す
- `tools.go`, `resources.go`, `prompts.go`は`client`を共有

### 2. 単一責任の原則

**GitHub Module**は「GitHubとの統合」という1つの責任を持つ。その責任の中で:
- Tools: `list_repositories`, `create_issue`等の実行可能な操作
- Resources: `github://user/repo/file`形式のファイル内容取得
- Prompts: Issue作成のテンプレート等

全て「GitHubとの統合」という同じドメイン内の機能。これを3層に分散させる理由がない。

### 3. MCP仕様との整合性

MCP仕様では:
- `ServerCapabilities`が`tools`, `resources`, `prompts`を宣言
- これらは全て同一のServer実装内で提供される
- 公式Go SDKでも`mcp.Server`構造体が3プリミティブを統合管理

MCPist設計でも:
- `ModuleDefinition`が`Tools`, `Resources`, `Prompts`を宣言
- これらは全て同一のModule実装内で提供される
- MCP仕様の意図に沿った設計

### 4. 拡張性とメンテナンス性

新しいサービス（例: Slack）を追加する場合:

**初期設計（3層分離）**:
1. `modules/slack/tools.go`作成 → Token Broker接続実装
2. `resources/slack/provider.go`作成 → Token Broker接続実装（重複）
3. `prompts/slack/library.go`作成 → Token Broker接続実装（重複）
4. 3箇所でエラーハンドリング、ログ、監視を実装

**モジュール中心設計**:
1. `modules/slack/`ディレクトリ作成
2. `client.go`でToken Broker接続実装（1箇所のみ）
3. `tools.go`, `resources.go`, `prompts.go`は`client`を使用
4. エラーハンドリング、ログ、監視は`client.go`に集約

## 影響

### 実装変更が必要な箇所

1. **`internal/modules/types.go`** - `ModuleDefinition`の拡張
   ```go
   type ModuleDefinition struct {
       Name        string
       Description string
       APIVersion  string
       TestedAt    string

       // 既存
       Tools    []Tool
       Handlers map[string]ToolHandler

       // 新規追加
       Resources        []ResourceTemplate
       ResourceResolver ResourceResolverFunc
       Prompts          []PromptTemplate
       PromptRenderer   PromptRendererFunc
   }
   ```

2. **`internal/modules/registry.go`** - 新メソッド追加
   - `ListResources(moduleName string) ([]ResourceTemplate, error)`
   - `ReadResource(moduleName, uri string) (string, error)`
   - `ListPrompts(moduleName string) ([]PromptTemplate, error)`
   - `GetPrompt(moduleName, name string, args map[string]interface{}) (Prompt, error)`

3. **`internal/mcp/handler.go`** - 新ハンドラー追加
   - `HandleResourcesList()`
   - `HandleResourcesRead()`
   - `HandleResourcesSubscribe()` (将来)
   - `HandlePromptsList()`
   - `HandlePromptsGet()`

4. **`internal/mcp/types.go`** - `ServerCapabilities`拡張
   ```go
   type ServerCapabilities struct {
       Tools     *ToolsCapability     `json:"tools,omitempty"`
       Resources *ResourcesCapability `json:"resources,omitempty"`
       Prompts   *PromptsCapability   `json:"prompts,omitempty"`
   }
   ```

5. **各モジュール** - `resources.go`と`prompts.go`の追加
   - `internal/modules/github/resources.go`
   - `internal/modules/github/prompts.go`
   - `internal/modules/notion/resources.go`
   - `internal/modules/notion/prompts.go`
   - 他の全モジュールも同様

### spec-sys.mdの更新

2.1節のアーキテクチャ図を以下に変更:

```
MCPプロトコルハンドラ
├─ tools/* ──┐
├─ resources/*─┼─► モジュールレジストリ（GitHub, Notion, Supabase, Jira...）
└─ prompts/*──┘

各モジュールは以下の構造:
├─ client.go      ← Token Broker接続（1箇所のみ）
├─ tools.go       ← clientを使用
├─ resources.go   ← clientを使用
└─ prompts.go     ← clientを使用
```

リソースプロバイダとプロンプトライブラリの記述を削除し、モジュールレジストリが3プリミティブを統合提供することを明記。

### 後方互換性

- 既存の`get_module_schema`と`call_module_tool`は引き続き動作
- Toolsの実装は変更不要（ModuleDefinitionに`Resources`と`Prompts`フィールドが追加されるだけ）
- クライアント側（Claude Desktop等）への影響なし

## 参照

- [MCP Architecture - Official Documentation](https://modelcontextprotocol.io/docs/concepts/architecture)
- [spec-sys.md § 2.1 MCPサーバーの処理フロー](../spec-sys.md)
- [ADR-003: メタツール + 選択的スキーマ取得パターンの採用](../DAY2/ADR-003-meta-tool-lazy-loading.md)
- [ADR-005: RLSに依存しない認可設計](./ADR-005-no-rls-dependency.md)
