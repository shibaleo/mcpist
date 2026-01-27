---
title: ADR-008 LLM API抽象化層によるクライアント側ポータビリティ
aliases:
  - ADR-008
  - llm-api-abstraction
tags:
  - MCPist
  - ADR
  - architecture
  - LLM
document-type:
  - ADR
document-class: ADR
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# ADR-008: LLM API抽象化層によるクライアント側ポータビリティ

## ステータス

提案（Phase 2実装時）

## コンテキスト

### MCP仕様の守備範囲と限界

MCP (Model Context Protocol) は、LLMアプリケーションと外部ツール/データを接続するための標準プロトコルである。しかし、**MCP仕様はServer側の標準化に特化**しており、Host-Client間のインターフェースは各実装に委ねられている。

**MCP仕様のアーキテクチャ:**

```
Host (LLM) → Client (MCP Client) → Server (MCP Server)
    ↑             ↑                      ↑
  未定義        未定義                  標準化済み
```

MCP仕様が標準化しているのは:
- Server側のプロトコル（JSON-RPC 2.0）
- Tools/Resources/Promptsの定義形式
- `initialize`, `tools/list`, `tools/call`等のメソッド

MCP仕様が標準化**していない**のは:
- Host（LLM）の選択
- LLM APIの呼び出し方法
- LLM間のtool_use差異の吸収

### 既存MCPクライアントの実装状況

| MCPクライアント | 対応LLM | 制約 |
|---------------|---------|------|
| Claude Desktop | Claudeのみ | LLM固定、切り替え不可 |
| Cursor | Claude/GPT | 独自実装、設定で一部切り替え可 |
| カスタム実装 | 各自で対応 | **LLM API差異を各自が吸収** |

結果として:
- **LLM切り替え = MCPクライアント全取り替え**
- **各実装が独自にLLM API差異を吸収 = 車輪の再発明**

### 各LLMのtool_use実装差異

各LLMベンダーは独自のtool_use（function calling）形式を採用している:

| LLM | Tool Use API | ツール定義形式 | ストリーミング | 備考 |
|-----|-------------|--------------|--------------|------|
| **Claude** | `messages` + `tools` | Anthropic形式 | ✅ | 最も柔軟 |
| **GPT** | `messages` + `functions` | OpenAI形式 | ✅ | function_call/tools並存 |
| **Gemini** | `contents` + `function_declarations` | Google形式 | ✅ | 独自スキーマ |
| **Ollama** | `prompt` + `tools` | OpenAI互換 | ✅ | ローカル実行 |

**具体例: 同じツール定義の各LLM実装**

```json
// MCPのTool定義（共通）
{
  "name": "search_notion",
  "description": "Notionページを検索",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {"type": "string", "description": "検索クエリ"}
    },
    "required": ["query"]
  }
}

// Claude API形式
{
  "name": "search_notion",
  "description": "Notionページを検索",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {"type": "string", "description": "検索クエリ"}
    },
    "required": ["query"]
  }
}

// OpenAI API形式
{
  "type": "function",
  "function": {
    "name": "search_notion",
    "description": "Notionページを検索",
    "parameters": {
      "type": "object",
      "properties": {
        "query": {"type": "string", "description": "検索クエリ"}
      },
      "required": ["query"]
    }
  }
}

// Google AI形式
{
  "name": "search_notion",
  "description": "Notionページを検索",
  "parameters": {
    "type": "OBJECT",
    "properties": {
      "query": {"type": "STRING", "description": "検索クエリ"}
    },
    "required": ["query"]
  }
}
```

この差異を**各MCPクライアントが個別に実装**している現状がある。

### MCPistのミッション

**「LLMの選択を、ユーザーの自由にする」**

このミッションを実現するには:
- ❌ MCPサーバーのポータビリティだけでは不十分
- ❌ 既存MCPクライアント依存ではLLM固定
- ✅ **クライアント側（LLM API差異）の吸収が必須**

## 検討した選択肢

### 選択肢1: 既存MCPクライアント依存（Claude Desktop/Cursor）

各ベンダーが提供するMCPクライアントを使用。

**メリット:**
- 開発コスト低（Server実装のみ）
- 既存エコシステム活用

**デメリット:**
- LLM選択がベンダー依存（Claude Desktop = Claudeのみ）
- 複数LLM対応には複数クライアント必要
- オフラインLLM（Ollama）非対応
- MCPistのミッション「LLM選択の自由」を達成できない

### 選択肢2: LLM API抽象化層の実装（採用）

Host層（LLM API呼び出し）を抽象化し、統一インターフェースを提供。

**メリット:**
- **LLM選択の自由**: Ollama, Claude, GPT, Gemini全対応
- **透過的な切り替え**: UIで選択変更するだけ、他は不変
- **DRY原則**: tool_use差異吸収を1箇所に集約
- **MCPistのミッション達成**: 真のポータビリティ実現

**デメリット:**
- 開発コスト増（抽象化層 + 各LLM Provider実装）
- 各LLM SDKの学習コスト
- 新LLM追加時のProvider実装が必要

## 決定

**選択肢2（LLM API抽象化層）を採用**

MCPist Desktop（Phase 2）に**LLM API抽象化層**を実装し、クライアント側のポータビリティを実現する。

### アーキテクチャ

```
┌─────────────────────────────────────────────────────────────┐
│  MCPist Desktop                                             │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 軽量UI                                                 │  │
│  │ ・LLM選択（Ollama/Claude/GPT/Gemini）                   │  │
│  │ ・チャット画面                                          │  │
│  │ ・設定管理                                             │  │
│  └────────────────────┬────────────────────────────────────┘  │
│                       │                                      │
│  ┌────────────────────▼────────────────────────────────────┐  │
│  │ LLM API抽象化層 (internal/llm/)              ← ADR-008 │  │
│  │                                                         │  │
│  │ ┌─────────────────────────────────────────────────────┐ │  │
│  │ │ 統一インターフェース (Provider)                      │ │  │
│  │ │ ・Chat(messages, tools) → Response                 │ │  │
│  │ │ ・StreamChat(messages, tools) → <-chan Token       │ │  │
│  │ └─────────────────────────────────────────────────────┘ │  │
│  │                                                         │  │
│  │ ┌──────────┬──────────┬──────────┬──────────┐          │  │
│  │ │ Claude   │ GPT      │ Gemini   │ Ollama   │          │  │
│  │ │ Provider │ Provider │ Provider │ Provider │          │  │
│  │ └──────────┴──────────┴──────────┴──────────┘          │  │
│  │   ↓ SDK      ↓ SDK      ↓ SDK      ↓ SDK              │  │
│  └─────┼──────────┼──────────┼──────────┼──────────────────┘  │
│        │          │          │          │                     │
│  ┌─────▼──────────▼──────────▼──────────▼──────────────────┐  │
│  │ MCPクライアント (internal/mcp/client/)                   │  │
│  │ ・抽象化されたtool_call → MCP tools/call形式に変換       │  │
│  │ ・MCPサーバーとのJSON-RPC 2.0通信                        │  │
│  └────────────────────┬────────────────────────────────────┘  │
└─────────────────────────┼──────────────────────────────────────┘
                          │
                          ▼
                   MCPサーバー (Go)
                   ・埋め込みモード
                   ・Token Broker (SQLite)
```

### 統一インターフェース設計

```go
// internal/llm/provider.go
package llm

import "context"

// Provider は全LLMに共通の統一インターフェース
type Provider interface {
    // Chat は同期的にLLMを呼び出し、応答を取得
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)

    // StreamChat はストリーミングでLLMを呼び出し
    StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)

    // Name はProvider名を返す（"claude", "gpt", "gemini", "ollama"）
    Name() string
}

// ChatRequest は全LLMに共通のリクエスト形式
type ChatRequest struct {
    Messages []Message  // 会話履歴
    Tools    []Tool     // MCP Tool定義（そのまま）
    Config   Config     // LLM固有設定（temperature等）
}

// ChatResponse は全LLMに共通のレスポンス形式
type ChatResponse struct {
    Content   string      // テキスト応答
    ToolCalls []ToolCall  // ツール呼び出しリクエスト
    FinishReason string   // 終了理由
}

// ToolCall はLLMからのツール呼び出し要求（統一形式）
type ToolCall struct {
    ID        string                 // ツール呼び出しID
    Name      string                 // ツール名
    Arguments map[string]interface{} // 引数
}
```

### 各LLM Provider実装

```go
// internal/llm/claude/provider.go
package claude

import (
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/shibaleo/go-mcp-dev/internal/llm"
)

type ClaudeProvider struct {
    client *anthropic.Client
}

func (p *ClaudeProvider) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
    // 1. MCPのTool定義 → Anthropic形式に変換
    anthropicTools := convertToAnthropicTools(req.Tools)

    // 2. Anthropic APIコール
    resp, err := p.client.Messages.Create(ctx, anthropic.MessageCreateParams{
        Model:    "claude-3-5-sonnet-20241022",
        Messages: convertMessages(req.Messages),
        Tools:    anthropicTools,
    })

    // 3. Anthropicレスポンス → 統一形式に変換
    return llm.ChatResponse{
        Content:   extractContent(resp),
        ToolCalls: extractToolCalls(resp),
    }, nil
}

// convertToAnthropicTools はMCPのTool定義をAnthropic形式に変換
func convertToAnthropicTools(tools []llm.Tool) []anthropic.Tool {
    // input_schema変換等
}
```

同様に:
- `internal/llm/gpt/provider.go` - OpenAI SDK使用
- `internal/llm/gemini/provider.go` - Google AI SDK使用
- `internal/llm/ollama/provider.go` - Ollama SDK使用

### MCPクライアントとの統合

```go
// internal/mcp/client/client.go
package client

import (
    "github.com/shibaleo/go-mcp-dev/internal/llm"
    "github.com/shibaleo/go-mcp-dev/internal/mcp"
)

type MCPClient struct {
    llmProvider llm.Provider  // 抽象化されたLLM
    mcpServer   *mcp.Handler  // 埋め込みMCPサーバー
}

// Run はチャットループを実行
func (c *MCPClient) Run(ctx context.Context) error {
    for {
        // 1. ユーザー入力取得
        userMessage := getUserInput()

        // 2. MCPサーバーからツール一覧取得
        tools := c.mcpServer.ListTools()

        // 3. LLMに送信（抽象化されたインターフェース経由）
        resp, err := c.llmProvider.Chat(ctx, llm.ChatRequest{
            Messages: c.history,
            Tools:    tools,
        })

        // 4. LLMがツール呼び出しを要求した場合
        if len(resp.ToolCalls) > 0 {
            for _, toolCall := range resp.ToolCalls {
                // MCPサーバーのtools/callを実行
                result := c.mcpServer.CallTool(toolCall.Name, toolCall.Arguments)
                // 結果をLLMに返す...
            }
        }

        // 5. ユーザーに応答表示
        displayResponse(resp.Content)
    }
}
```

## 根拠

### 1. MCPistのポジショニング明確化

| 層 | MCP仕様 | Claude Desktop | MCPist Desktop |
|----|---------|---------------|---------------|
| **Host** | 未定義 | Claude固定 | **抽象化（ADR-008）** |
| **Client** | 未定義 | 独自実装 | **MCP準拠** |
| **Server** | 標準化 | MCP準拠 | **MCP準拠（ADR-006）** |

**MCPist = MCPの思想をクライアント側にも拡張**

MCP仕様はServerの標準化に留まるが、MCPistは:
- **Server側**: モジュール中心アーキテクチャ（ADR-006）
- **Client側**: LLM API抽象化（ADR-008）
- **Host側**: LLM選択の自由（ADR-007）

全層を統合した**完全なポータビリティ**を実現。

### 2. ユーザー体験の劇的改善

**Before（Claude Desktop）:**

```
タスク: Notionページ検索
→ Claude APIのみ使用可能
→ 月額課金必須（$20/月）
→ オフライン作業不可
→ API障害時に利用不可
```

**After（MCPist Desktop）:**

```
タスク: Notionページ検索

【シナリオ1: ローカル開発】
→ Ollama（Llama 3.1）選択
→ 完全無料・オフライン
→ プライバシー完全保護

【シナリオ2: 高精度が必要】
→ Claude API選択（UIで切り替え）
→ 同じツール、同じ認証、同じUI
→ 課金は必要な時だけ

【シナリオ3: コスト最適化】
→ GPT-4o mini選択（安価）
→ 大量処理時のコスト削減
```

全て**同じアプリケーション内で切り替え可能**。

### 3. DRY原則の徹底

**現状（各MCPクライアントが個別実装）:**

```
Claude Desktop → Anthropic tool_use変換実装
Cursor        → OpenAI + Anthropic変換実装
カスタム実装1 → 独自変換実装
カスタム実装2 → 独自変換実装
    ↓
同じロジックを各実装が重複実装 = 車輪の再発明
```

**MCPist Desktop（抽象化層で一元化）:**

```
LLM API抽象化層
├─ ClaudeProvider   → Anthropic変換（1箇所のみ）
├─ GPTProvider      → OpenAI変換（1箇所のみ）
├─ GeminiProvider   → Google変換（1箇所のみ）
└─ OllamaProvider   → Ollama変換（1箇所のみ）
    ↓
変換ロジックが集約、再利用可能
新LLM追加 = 新Providerのみ実装
```

### 4. 技術的実現可能性

各LLMベンダーは公式SDKを提供しており、統一ラッパー実装は十分実現可能:

| LLM | 公式Go SDK | 利用可否 |
|-----|-----------|---------|
| Claude | `anthropic-sdk-go` | ✅ |
| GPT | `openai-go` | ✅ |
| Gemini | `google-ai-go` (generativeai) | ✅ |
| Ollama | `ollama-go` | ✅ |

全てGoで実装可能、Phase 1のGoコードベースと統合しやすい。

### 5. ADR群との整合性

ADR-008は既存ADRの自然な帰結:

```
ADR-003（メタツール）
    ↓ Context Rot解決 → LLM推論品質向上
ADR-005（RLS非依存）
    ↓ SQLite対応設計 → ローカルアプリ化可能
ADR-006（モジュール中心）
    ↓ 認証統合 → Token Brokerの価値最大化
ADR-007（Host層ポータビリティ）
    ↓ LLM選択の自由 → 完全ポータビリティビジョン
        ↓
ADR-008（LLM API抽象化）← 具体的実装戦略
```

## 影響

### Phase 1（v1: クラウド版）への影響

**なし** - クラウド版はClaude Desktop/Cursor依存のまま進行。

Phase 1で実装するMCPサーバー層は、Phase 2で**そのまま埋め込み利用可能**な設計。

### Phase 2（v2: MCPist Desktop）への影響

**新規実装が必要:**

1. **LLM API抽象化層** (`internal/llm/`)
   ```
   internal/llm/
   ├─ provider.go           # 統一インターフェース定義
   ├─ types.go              # 共通型定義
   ├─ claude/
   │   └─ provider.go       # ClaudeProvider実装
   ├─ gpt/
   │   └─ provider.go       # GPTProvider実装
   ├─ gemini/
   │   └─ provider.go       # GeminiProvider実装
   └─ ollama/
       └─ provider.go       # OllamaProvider実装
   ```

2. **MCPクライアント** (`internal/mcp/client/`)
   ```
   internal/mcp/client/
   ├─ client.go             # MCPクライアント本体
   ├─ loop.go               # チャットループ
   └─ converter.go          # MCP ↔ LLM変換
   ```

3. **デスクトップUI** (`cmd/desktop/`)
   - LLM選択画面
   - チャットUI
   - 設定画面

4. **統合** (`cmd/desktop/main.go`)
   ```go
   // LLM Provider初期化（設定から選択）
   var llmProvider llm.Provider
   switch config.LLM {
   case "claude":
       llmProvider = claude.NewProvider(apiKey)
   case "gpt":
       llmProvider = gpt.NewProvider(apiKey)
   case "ollama":
       llmProvider = ollama.NewProvider()
   }

   // MCPサーバー埋め込み起動
   mcpServer := mcp.NewHandler()

   // MCPクライアント起動
   mcpClient := client.New(llmProvider, mcpServer)
   mcpClient.Run(ctx)
   ```

### ドキュメントへの影響

1. **README.md更新**
   ```markdown
   ## MCPistとは

   MCPistは「LLMの選択を、ユーザーの自由にする」ことを目指すMCPクライアント＆サーバーです。

   ### MCP仕様との違い

   - **MCP仕様**: Server側の標準化（tools/resources/prompts）
   - **MCPist**: Client側も含めた完全なポータビリティ
     - LLM選択の自由（Claude/GPT/Gemini/Ollama）
     - 統一認証（Token Broker）
     - モジュール中心アーキテクチャ
   ```

2. **spec-sys.md更新**
   - Phase 2の章に「2.2 LLM API抽象化層」を追加
   - アーキテクチャ図にLLM抽象化層を追記

3. **CONTRIBUTING.md作成**
   ```markdown
   ## 新しいLLMの追加方法

   1. `internal/llm/新LLM名/` ディレクトリ作成
   2. `Provider` インターフェース実装
   3. テスト作成
   4. UI設定画面に選択肢追加
   ```

### コミュニティへのメッセージング

**Phase 1リリース時:**
- "MCPist v1: Claude Desktop/Cursor用のポータブルMCPサーバー"
- "メタツールによるContext Rot解決"
- "Token Brokerによる統一認証"

**Phase 2告知:**
- "MCPist v2: LLM選択の自由を実現するデスクトップアプリ"
- "Ollama対応で完全オフライン動作"
- "Claude/GPT/Gemini切り替えがUIで完結"
- "MCPの思想をクライアント側にも拡張"

### 実装計画

**Phase 2.1: LLM抽象化層実装（2025 Q3）**
- [ ] `llm.Provider`インターフェース定義
- [ ] ClaudeProvider実装
- [ ] GPTProvider実装
- [ ] GeminiProvider実装
- [ ] OllamaProvider実装
- [ ] 単体テスト完備

**Phase 2.2: MCPクライアント実装（2025 Q3）**
- [ ] MCPクライアント基本実装
- [ ] LLM ↔ MCP変換ロジック
- [ ] チャットループ実装
- [ ] 埋め込みMCPサーバー統合

**Phase 2.3: デスクトップUI実装（2025 Q4）**
- [ ] 技術スタック決定（Tauri推奨）
- [ ] LLM選択UI
- [ ] チャットUI
- [ ] 設定画面
- [ ] クロスプラットフォームビルド

**Phase 2.4: 統合テスト・リリース（2025 Q4）**
- [ ] 全LLMでの動作確認
- [ ] パフォーマンステスト
- [ ] ドキュメント整備
- [ ] v2.0.0リリース

## 参照

- [ADR-003: メタツール + 選択的スキーマ取得パターンの採用](../DAY2/ADR-003-meta-tool-lazy-loading.md)
- [ADR-005: RLSに依存しない認可設計](./ADR-005-no-rls-dependency.md)
- [ADR-006: モジュール中心アーキテクチャによる3プリミティブ統合](./ADR-006-module-centric-primitives.md)
- [ADR-007: Host層を含めた完全ポータビリティの実現](./ADR-007-host-layer-portability.md)
- [MCP Specification - Architecture](https://modelcontextprotocol.io/docs/concepts/architecture)
- [Anthropic Tool Use Documentation](https://docs.anthropic.com/claude/docs/tool-use)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [Google AI Function Calling](https://ai.google.dev/gemini-api/docs/function-calling)
- [Ollama Tool Support](https://github.com/ollama/ollama/blob/main/docs/api.md#generate-a-chat-completion)
