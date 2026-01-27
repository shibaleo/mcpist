# レポート: MCP Tool Annotations 仕様と各プラットフォーム対応

## 日付

2026-01-27

---

## 概要

MCP (Model Context Protocol) はツール定義に `annotations` オブジェクトを持ち、ツールの性質をクライアントに伝えるメタデータ仕様を規定している。OpenAI (ChatGPT Apps SDK) と Anthropic (Claude MCP Directory) は共にこの仕様を採用しており、ディレクトリ審査でも必須要件となっている。

---

## MCP Tool Annotations 仕様

### フィールド定義

| フィールド | 型 | デフォルト | 意味 |
|---|---|---|---|
| `title` | string | - | UI表示用の人間可読な名前 |
| `readOnlyHint` | boolean | **false** | `true`: 環境を変更しない（読み取りのみ） |
| `destructiveHint` | boolean | **true** | `true`: 破壊的更新の可能性あり |
| `idempotentHint` | boolean | **false** | `true`: 同一引数での再実行で追加効果なし |
| `openWorldHint` | boolean | **true** | `true`: 外部エンティティと相互作用する |

> **重要**: デフォルト値が「最も危険側」に設定されている。未設定のツールは「書き込み可能 / 破壊的 / 非冪等 / 外部到達」と見なされる。

### フィールド間の関係

```
readOnlyHint = true → destructiveHint, idempotentHint は意味を持たない
                        （読み取り専用なので破壊も冪等もない）

readOnlyHint = false → destructiveHint, idempotentHint が意味を持つ
                         destructiveHint = true  → 削除・上書きの可能性
                         destructiveHint = false → 作成・更新だが非破壊的
                         idempotentHint = true   → 再実行しても追加効果なし
```

### ツール定義構造

```typescript
{
  name: string;              // ツール識別子
  title?: string;            // UI表示用名前
  description?: string;      // 説明文
  inputSchema: {             // JSON Schema
    type: "object",
    properties: { ... }
  },
  annotations?: {            // メタデータヒント
    title?: string;
    readOnlyHint?: boolean;
    destructiveHint?: boolean;
    idempotentHint?: boolean;
    openWorldHint?: boolean;
  }
}
```

---

## 各プラットフォームの対応状況

### OpenAI (ChatGPT Apps SDK)

**エリシテーション（確認UI）の挙動:**

| readOnlyHint | destructiveHint | openWorldHint | 確認が必要か |
|---|---|---|---|
| true | - | - | 不要 |
| false | true | false | 不要 |
| false | false | false | 不要 |
| false | true | true | **必要** |
| false | false | true | **必要** |
| 未設定 | 未設定 | 未設定 | **必要**（最悪ケース） |

> open-world な書き込みのみ確認が必要。closed-world な操作は確認不要。

**アプリ審査要件:**

- `readOnlyHint` または `destructiveHint` は全ツールに**必須**
- 不正確なアノテーションはリジェクトの主要因
- 提出時に各アノテーションの根拠を説明する必要がある
- 読み取りと書き込みが混在する場合は別ツールに分割を推奨

**参考リンク:**

- [Define Tools](https://developers.openai.com/apps-sdk/plan/tools/)
- [Optimize Metadata](https://developers.openai.com/apps-sdk/guides/optimize-metadata/)
- [App Submission Guidelines](https://developers.openai.com/apps-sdk/app-submission-guidelines/)
- [API Reference](https://developers.openai.com/apps-sdk/reference/)

### Anthropic (Claude MCP Directory)

**審査要件:**

- `readOnlyHint` と `destructiveHint` は全ツールに**必須**（ハード要件）
- `title` は推奨
- 欠落は即リジェクト（コード変更が必要）
- Revision Request の最頻出理由が「Missing tool annotations」

**アノテーション判定マトリックス（Anthropic公式）:**

| ツールの振る舞い | アノテーション | 例 |
|---|---|---|
| 読み取り専用 | `readOnlyHint: true, destructiveHint: false` | search, get, list, fetch, read |
| データ変更 | `destructiveHint: true, readOnlyHint: false` | create, update, delete, send |
| 一時ファイル作成 | `destructiveHint: true` | 一時的な書き込みも対象 |
| 外部リクエスト | `destructiveHint: true` | メール、通知、Webhook |
| 内部キャッシュのみ | `readOnlyHint: true` | 内部最適化は許容 |

**参考リンク:**

- [Remote MCP Server Submission Guide](https://support.claude.com/en/articles/12922490-remote-mcp-server-submission-guide)
- [Local MCP Server Submission Guide](https://support.claude.com/en/articles/12922832-local-mcp-server-submission-guide)

---

## MCPist への適用

### 現状

```json
// tools.json (現在)
{
  "id": "delete_event",
  "name": "delete_event",
  "description": "Delete an event from a calendar.",
  "dangerous": true,         // ← 独自フィールド
  "defaultEnabled": false    // ← 独自フィールド
}
```

### MCP annotations 準拠案

```json
// tools.json (MCP準拠)
{
  "id": "delete_event",
  "name": "delete_event",
  "description": "Delete an event from a calendar.",
  "annotations": {
    "title": "Delete Event",
    "readOnlyHint": false,
    "destructiveHint": true,
    "idempotentHint": true,
    "openWorldHint": false
  },
  "defaultEnabled": false
}
```

### MCPist ツール別アノテーション設計例

#### Google Calendar

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_calendars | true | false | - | false |
| get_calendar | true | false | - | false |
| list_events | true | false | - | false |
| get_event | true | false | - | false |
| create_event | false | false | false | false |
| update_event | false | false | true | false |
| delete_event | false | **true** | true | false |
| quick_add | false | false | false | false |

#### Notion

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| search | true | false | - | false |
| get_page | true | false | - | false |
| get_page_content | true | false | - | false |
| create_page | false | false | false | false |
| update_page | false | false | true | false |
| get_database | true | false | - | false |
| query_database | true | false | - | false |
| append_blocks | false | false | false | false |
| delete_block | false | **true** | true | false |
| list_comments | true | false | - | false |
| add_comment | false | false | false | false |
| list_users | true | false | - | false |
| get_user | true | false | - | false |
| get_bot_user | true | false | - | false |

#### Microsoft To Do

| ツール | readOnly | destructive | idempotent | openWorld |
|---|---|---|---|---|
| list_lists | true | false | - | false |
| get_list | true | false | - | false |
| create_list | false | false | false | false |
| update_list | false | false | true | false |
| delete_list | false | **true** | true | false |
| list_tasks | true | false | - | false |
| get_task | true | false | - | false |
| create_task | false | false | false | false |
| update_task | false | false | true | false |
| complete_task | false | false | true | false |
| delete_task | false | **true** | true | false |

### defaultEnabled との関係

`annotations` はツールの性質を記述するメタデータ。`defaultEnabled` はユーザーの初期設定。両者は独立した概念だが、以下のルールが自然：

- `destructiveHint: true` のツール → `defaultEnabled: false` が妥当
- `readOnlyHint: true` のツール → `defaultEnabled: true` が妥当

---

## セキュリティに関する注意

MCP仕様より:

> All properties in ToolAnnotations are **hints** and not guaranteed to provide a faithful description of tool behavior. Clients should **never make security-critical decisions based solely on annotations**.

アノテーションはUXガイダンスであり、セキュリティバリアではない。サーバー側で独自に認可チェックを行う必要がある（MCPist の Sieve / Permission Gate がこの役割を担う）。

---

## 参考リンク

- [MCP Specification - Tools](https://modelcontextprotocol.io/legacy/concepts/tools)
- [MCP Schema - ToolAnnotations](https://modelcontextprotocol.io/specification/2025-11-25/schema#toolannotations)
- [OpenAI Apps SDK - Define Tools](https://developers.openai.com/apps-sdk/plan/tools/)
- [OpenAI Apps SDK - Optimize Metadata](https://developers.openai.com/apps-sdk/guides/optimize-metadata/)
- [OpenAI Apps SDK - App Submission Guidelines](https://developers.openai.com/apps-sdk/app-submission-guidelines/)
- [OpenAI Apps SDK - Reference](https://developers.openai.com/apps-sdk/reference/)
- [Anthropic Remote MCP Server Submission Guide](https://support.claude.com/en/articles/12922490-remote-mcp-server-submission-guide)
