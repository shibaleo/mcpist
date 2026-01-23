# MCP Handler 詳細仕様書（dtl-spc-hdl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | MCP Handler Detail Specification |

---

## 概要

本ドキュメントはMCP Handler（HDL）の内部実装詳細を記述する。コンポーネント間のやり取りについては[itr-hdl.md](./itr-hdl.md)を参照。

**準拠仕様:** [Model Context Protocol Specification 2025-03-26](https://spec.modelcontextprotocol.io/specification/2025-03-26/)

---

## メタツール

HDLは以下のメタツールを提供する。

| ツール名 | 説明 |
|----------|------|
| get_module_schema | 指定モジュールのスキーマ（利用可能プリミティブ一覧）を取得 |
| run | 単一ツールを実行 |
| batch | 複数ツールを一括実行 |

### get_module_schema

指定モジュールのスキーマを取得する。**ユーザーが無効化したツールはレスポンスから除外される。**

**処理フロー:**
1. DSTからtool_settingsを取得
2. MODからモジュールスキーマを取得
3. tool_settingsに基づいて無効ツールをフィルタリング
4. フィルタリング済みスキーマを返却

**リクエスト:**
```json
{
  "module": "notion"
}
```

**レスポンス（delete_pageが無効化されている場合）:**
```json
{
  "module": "notion",
  "description": "Notion integration module",
  "tools": [
    {
      "name": "search",
      "description": "Search pages and databases",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": { "type": "string", "description": "Search query" }
        },
        "required": ["query"]
      }
    },
    {
      "name": "create_page",
      "description": "Create a new page",
      "inputSchema": {
        "type": "object",
        "properties": {
          "parent_id": { "type": "string" },
          "title": { "type": "string" },
          "content": { "type": "string" }
        },
        "required": ["parent_id", "title"]
      }
    }
  ],
  "resources": [],
  "prompts": []
}
```

### run

**リクエスト:**
```json
{
  "module": "notion",
  "tool": "search",
  "params": {
    "query": "設計ドキュメント"
  }
}
```

**レスポンス:**
```json
{
  "success": true,
  "result": {
    "pages": [
      {
        "id": "page-123",
        "title": "システム設計ドキュメント",
        "url": "https://notion.so/..."
      }
    ]
  }
}
```

### batch

**リクエスト:**
```json
{
  "calls": [
    {
      "module": "notion",
      "tool": "search",
      "params": { "query": "meeting" }
    },
    {
      "module": "google_calendar",
      "tool": "list_events",
      "params": { "date": "2024-01-15" }
    }
  ]
}
```

**レスポンス:**
```json
{
  "results": [
    {
      "success": true,
      "result": { "pages": [...] }
    },
    {
      "success": true,
      "result": { "events": [...] }
    }
  ]
}
```

---

## MCPメソッド処理

### tools/list

**フロー:**
1. HDLがtools/listリクエストを受信
2. DSTからユーザーの許可モジュール・ツール一覧を取得
3. メタツール形式でレスポンスを構築

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "get_module_schema",
        "description": "Get schema for a specific module",
        "inputSchema": {
          "type": "object",
          "properties": {
            "module": { "type": "string" }
          },
          "required": ["module"]
        }
      },
      {
        "name": "run",
        "description": "Execute a tool in a module",
        "inputSchema": {
          "type": "object",
          "properties": {
            "module": { "type": "string" },
            "tool": { "type": "string" },
            "params": { "type": "object" }
          },
          "required": ["module", "tool"]
        }
      },
      {
        "name": "batch",
        "description": "Execute multiple tools in batch",
        "inputSchema": {
          "type": "object",
          "properties": {
            "calls": {
              "type": "array",
              "items": {
                "type": "object",
                "properties": {
                  "module": { "type": "string" },
                  "tool": { "type": "string" },
                  "params": { "type": "object" }
                }
              }
            }
          },
          "required": ["calls"]
        }
      }
    ]
  }
}
```

### tools/call

**フロー:**
1. HDLがtools/callリクエストを受信
2. パラメータからmodule, tool, paramsを抽出
3. DSTで権限チェック
4. MODにツール実行を委譲
5. 結果をJSON-RPC形式で返却

**リクエスト例:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "run",
    "arguments": {
      "module": "notion",
      "tool": "search",
      "params": {
        "query": "設計ドキュメント"
      }
    }
  }
}
```

### prompts/list, prompts/get

**フロー:**
1. HDLがprompts/list または prompts/get リクエストを受信
2. DSTからユーザー定義プロンプトを取得
3. MODからモジュール提供プロンプトを取得
4. マージしてレスポンスを構築

**prompts/list レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "prompts": [
      {
        "name": "daily-summary",
        "description": "Generate daily summary from calendar and tasks"
      }
    ]
  }
}
```

### resources/list, resources/read

**フロー:**
1. HDLがresources/list または resources/read リクエストを受信
2. DSTで権限チェック
3. MODにリソース取得を委譲
4. レスポンスを構築

---

## サポートするMCPメソッド

| メソッド | 説明 | 処理 |
|----------|------|------|
| tools/list | 利用可能なツール一覧 | DST権限確認 → メタツール返却 |
| tools/call | ツール実行 | DST権限確認 → MOD委譲 |
| resources/list | リソース一覧 | DST権限確認 → MOD委譲 |
| resources/read | リソース取得 | DST権限確認 → MOD委譲 |
| prompts/list | プロンプト一覧 | DST + MOD から取得 |
| prompts/get | プロンプト取得 | DST + MOD から取得 |
| initialize | セッション初期化 | HDL内部 |
| ping | ヘルスチェック | HDL内部 |

---

## エラーレスポンス

### メソッド不明

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found"
  },
  "id": 1
}
```

### パラメータ不正

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params: module is required"
  },
  "id": 1
}
```

### アカウント停止

```json
{
  "success": false,
  "error": {
    "code": "ACCOUNT_SUSPENDED",
    "message": "Account is suspended"
  }
}
```

### クレジット不足

```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_CREDIT",
    "message": "Insufficient credit balance"
  }
}
```

### モジュール未有効

```json
{
  "success": false,
  "error": {
    "code": "MODULE_NOT_ENABLED",
    "message": "Module 'notion' is not enabled for this user"
  }
}
```

### ツール無効

```json
{
  "success": false,
  "error": {
    "code": "TOOL_DISABLED",
    "message": "Tool 'delete_page' is disabled for this user"
  }
}
```

### モジュール不存在

```json
{
  "success": false,
  "error": {
    "code": "MODULE_NOT_FOUND",
    "message": "Module 'unknown' does not exist"
  }
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-hdl.md](./itr-hdl.md) | MCP Handler インタラクション仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store詳細仕様 |
