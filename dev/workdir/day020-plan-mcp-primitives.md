# DAY020 MCP Primitives 調査・計画

## 日付

2026-01-31

---

## 概要

MCPのコア機能（primitives）のうち、未実装のものを調査・計画する。

| Primitive | 現状 | 優先度 | 実装難易度 |
|-----------|------|--------|------------|
| tools | ✅ 実装済み | - | - |
| resources | ❌ 未実装 | 高 | 中 |
| prompts | ❌ 未実装（DB設計済み） | 中 | 低 |
| elicitation | ❌ 未実装 | 低 | 高 |

---

## 1. Resources

### 1.1 MCP仕様（2025-11-25）

Resourcesはサーバーがクライアントにデータを公開するための仕組み。URIで一意に識別される。

#### Capability宣言

```json
{
  "capabilities": {
    "resources": {
      "subscribe": true,    // リソース変更通知（オプション）
      "listChanged": true   // リスト変更通知（オプション）
    }
  }
}
```

#### メソッド

| メソッド | 説明 |
|----------|------|
| `resources/list` | リソース一覧取得（ページネーション対応） |
| `resources/read` | リソース内容取得 |
| `resources/templates/list` | テンプレート一覧取得 |
| `resources/subscribe` | リソース変更を購読 |

#### resources/list

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "resources/list",
  "params": { "cursor": "optional-cursor-value" }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "resources": [
      {
        "uri": "file:///project/src/main.rs",
        "name": "main.rs",
        "title": "Rust Application Main File",
        "description": "Primary application entry point",
        "mimeType": "text/x-rust"
      }
    ],
    "nextCursor": "next-page-cursor"
  }
}
```

#### resources/read

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "resources/read",
  "params": { "uri": "file:///project/src/main.rs" }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "contents": [
      {
        "uri": "file:///project/src/main.rs",
        "mimeType": "text/x-rust",
        "text": "fn main() {\n    println!(\"Hello world!\");\n}"
      }
    ]
  }
}
```

### 1.2 MCPistでの用途

| 用途 | URI例 | 説明 |
|------|-------|------|
| モジュールドキュメント | `mcpist://docs/{module}` | 各モジュールの使い方 |
| ユーザープロンプト | `mcpist://prompts/{id}` | 保存済みプロンプト |
| ツールリファレンス | `mcpist://tools/{module}/{tool}` | ツールの詳細仕様 |
| クレジット残高 | `mcpist://credits` | 現在の残高情報 |

### 1.3 実装方針

1. **Phase 1**: 静的リソース
   - モジュールドキュメント（ハードコード）
   - `resources/list`, `resources/read` のみ

2. **Phase 2**: 動的リソース
   - ユーザープロンプト（DBから取得）
   - クレジット残高

3. **Phase 3**: 購読機能
   - `subscribe` / `listChanged` 対応

### 1.4 実装タスク

- [ ] `handler.go` に `resources/list` ハンドラ追加
- [ ] `handler.go` に `resources/read` ハンドラ追加
- [ ] Capability宣言に `resources` 追加
- [ ] モジュールドキュメントリソース実装

---

## 2. Prompts

### 2.1 MCP仕様（2025-11-25）

Promptsはサーバーがクライアントにプロンプトテンプレートを提供する仕組み。ユーザーが明示的に選択して使用する（スラッシュコマンド等）。

#### Capability宣言

```json
{
  "capabilities": {
    "prompts": {
      "listChanged": true  // リスト変更通知（オプション）
    }
  }
}
```

#### メソッド

| メソッド | 説明 |
|----------|------|
| `prompts/list` | プロンプト一覧取得 |
| `prompts/get` | プロンプト内容取得（引数展開） |

#### prompts/list

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "prompts/list",
  "params": { "cursor": "optional-cursor-value" }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "prompts": [
      {
        "name": "code_review",
        "title": "Request Code Review",
        "description": "Asks the LLM to analyze code quality",
        "arguments": [
          {
            "name": "code",
            "description": "The code to review",
            "required": true
          }
        ]
      }
    ],
    "nextCursor": "next-page-cursor"
  }
}
```

#### prompts/get

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "prompts/get",
  "params": {
    "name": "code_review",
    "arguments": { "code": "def hello():\n    print('world')" }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "description": "Code review prompt",
    "messages": [
      {
        "role": "user",
        "content": {
          "type": "text",
          "text": "Please review this Python code:\ndef hello():\n    print('world')"
        }
      }
    ]
  }
}
```

### 2.2 既存設計（DB）

- `prompts` テーブル設計済み
- RPC: `list_my_prompts`, `get_my_prompt`, `upsert_my_prompt`, `delete_my_prompt`

### 2.3 MCPistでの用途

| 用途 | 説明 |
|------|------|
| ユーザー定義プロンプト | Consoleで作成したプロンプトをMCPで使用 |
| モジュール固有プロンプト | 特定モジュール向けのテンプレート |
| システムプロンプト | 共通のプロンプトテンプレート |

### 2.4 実装方針

1. **Phase 1**: DBプロンプト
   - `prompts/list` → `get_user_prompts` RPC（要作成）
   - `prompts/get` → `get_user_prompt` RPC（要作成）

2. **Phase 2**: システムプロンプト
   - 組み込みプロンプト（ハードコード）

### 2.5 実装タスク

- [ ] `handler.go` に `prompts/list` ハンドラ追加
- [ ] `handler.go` に `prompts/get` ハンドラ追加
- [ ] Capability宣言に `prompts` 追加
- [ ] `get_user_prompts` RPC作成（API Server用）
- [ ] `get_user_prompt` RPC作成（API Server用）

---

## 3. Elicitation

### 3.1 MCP仕様（2025-11-25）

Elicitationはサーバーがクライアントを通じてユーザーに情報を要求する仕組み。

#### 2つのモード

| モード | 説明 | 用途 |
|--------|------|------|
| `form` | 構造化データ収集（JSONスキーマ） | 一般的な入力要求 |
| `url` | 外部URLへのリダイレクト | OAuth、決済等のセンシティブな操作 |

#### Capability宣言（クライアント側）

```json
{
  "capabilities": {
    "elicitation": {
      "form": {},
      "url": {}
    }
  }
}
```

#### elicitation/create (form mode)

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "elicitation/create",
  "params": {
    "mode": "form",
    "message": "Please provide your GitHub username",
    "requestedSchema": {
      "type": "object",
      "properties": {
        "name": { "type": "string" }
      },
      "required": ["name"]
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "action": "accept",  // or "decline" or "cancel"
    "content": { "name": "octocat" }
  }
}
```

#### elicitation/create (url mode)

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "elicitation/create",
  "params": {
    "mode": "url",
    "elicitationId": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://mcp.example.com/ui/set_api_key",
    "message": "Please provide your API key to continue."
  }
}
```

### 3.2 技術的課題

| 課題 | 詳細 |
|------|------|
| SSE必須 | inline POSTでは使用不可（双方向通信が必要） |
| クライアントサポート | Claude Webがelicitationをサポートしているか不明 |
| ブロッキング | tools/callの処理中にelicitationを待機する必要あり |
| セッション管理 | elicitationIdとユーザーの紐付け |
| セキュリティ | form modeで機密情報を要求してはならない |

### 3.3 MCPistでの用途

| 用途 | モード | 説明 |
|------|--------|------|
| 確認ダイアログ | form | 破壊的操作の確認 |
| 追加パラメータ | form | ツール実行時の追加情報要求 |
| OAuth認可 | url | モジュール認証フロー |
| API Key入力 | url | センシティブな認証情報 |

### 3.4 実装方針

**現時点では実装を見送り**

理由：
1. Claude Webのelicitationサポート状況が不明
2. SSEセッション管理の複雑さ
3. resourcesとpromptsで十分な機能提供が可能

将来実装時の方針：
1. SSEセッションにelicitation対応を追加
2. 待機メカニズムの実装（channelベース）
3. Console側でのOAuth連携はurl modeで実装

### 3.5 将来検討（一般公開前）

**検討タイミング**: 破壊的操作ポリシー策定時

| 項目 | 内容 |
|------|------|
| 破壊的操作の定義 | delete系、bulk update等の基準 |
| 確認フロー | Elicitation (form) vs LLM確認 vs Console設定 |
| ユーザー設定 | 「確認なしで実行」オプション等 |
| ログ・監査 | 破壊的操作の記録 |

---

## 実装優先順位

| 順位 | Primitive | 理由 |
|------|-----------|------|
| 1 | prompts | DB設計済み、実装が最も単純 |
| 2 | resources | モジュールドキュメント提供に有用 |
| 3 | elicitation | クライアントサポート確認後 |

---

## 完了条件（コア機能）

| ID | 項目 | 説明 | 状態 |
|----|------|------|------|
| CORE-001 | Google Tasks MCP実装 | google_tasks モジュール追加 | ❌ |
| CORE-002 | prompts MCP実装 | `prompts/list`, `prompts/get` ハンドラ | ❌ |
| CORE-003 | Console プロンプト管理UI | ユーザーがカスタムプロンプトを定義可能 | ❌ |
| CORE-004 | チャットUIからテンプレ実行 | Claude Web等でプロンプト選択・実行 | ❌ |
| CORE-005 | resources MCP実装 | `resources/list`, `resources/read` ハンドラ | ❌ |
| CORE-006 | resources/list 動作確認 | Grafana or サーバーログで呼び出し確認 | ❌ |
| CORE-007 | profile リソース実装 | `mcpist://profile` - ユーザープロフィール | ❌ |
| CORE-008 | tasks リソース実装 | `mcpist://tasks` - タスク一覧（MS Todo + Google Tasks） | ❌ |
| CORE-009 | Claude Code E2E | ユーザーが `@` でリソース選択・実行 | ❌ |

---

## 次のアクション

1. **Google Tasks モジュール実装**
   - Google Tasks API連携
   - list_tasks, create_task, update_task, delete_task

2. **prompts実装**
   - `get_user_prompts` / `get_user_prompt` RPC作成
   - `handler.go` に `prompts/list`, `prompts/get` 追加
   - Capability宣言更新

3. **Console プロンプト管理UI**
   - プロンプト一覧ページ
   - 作成・編集・削除機能
   - 引数（arguments）定義UI

4. **E2Eテスト**
   - Claude Webでプロンプト一覧表示確認
   - プロンプト選択→ツール実行の流れ確認

5. **resources実装**（後回し）
   - 静的モジュールドキュメント作成
   - `handler.go` に `resources/list`, `resources/read` 追加

6. **elicitation調査**（後回し）
   - Claude Webでのelicitationサポート確認
   - 他のMCPクライアントでの実装状況調査

---

## 参考

- [MCP Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [MCP Features Guide - WorkOS](https://workos.com/blog/mcp-features-guide)
- [handler.go](../../apps/server/internal/mcp/handler.go)
- [grh-rpc-design.canvas](../docs/graph/grh-rpc-design.canvas)
