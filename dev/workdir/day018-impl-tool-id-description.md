# ツールID化 + モジュール説明（多言語）対応 - 詳細実装計画

## 概要

本計画は `day018-plan-tool-id-description.md` の方針に基づき、以下を実装する:

1. **tool_id**: ツール設定の安定キーとして `{module}:{tool_name}` 形式のIDを導入
2. **多言語対応**: モジュール/ツール説明を `en-US` / `ja-JP` の2言語で提供
3. **ユーザー追記**: モジュール説明にユーザー独自の説明を追加可能に

---

## 現状分析

### Go Server

| ファイル | 現状 |
|----------|------|
| `internal/modules/types.go` | `Tool.Name`, `Tool.Description` (単一言語) |
| `internal/modules/modules.go` | `filterTools()` は `tool.Name` で判定 |
| `internal/mcp/handler.go` | `GetModuleSchemas()` は言語パラメータなし |
| `*/module.go` | `Description() string` - 単一言語 |
| `*/tools.go` | `Tool{Name: "search", Description: "..."}` |

### DB

| テーブル | カラム | 現状 |
|----------|--------|------|
| `tool_settings` | `tool_name TEXT` | PRIMARY KEY の一部 |
| `module_settings` | - | `description` カラムなし |

### Console

| ファイル | 現状 |
|----------|------|
| `lib/module-data.ts` | `tools.json` / `services.json` を import |
| `lib/tool-settings.ts` | `tool_name` で RPC 呼び出し |
| `lib/tools.json` | `tool.id` = `tool.name` (同一値) |
| `(console)/tools/page.tsx` | `tool.id` で設定管理 |

---

## Phase 1: Go Server - 型定義と多言語対応

### 1.1 types.go の更新

```go
// apps/server/internal/modules/types.go

// LocalizedText は多言語テキストを保持
// key: BCP47 言語コード (en-US, ja-JP)
type LocalizedText map[string]string

// GetLocalizedText は指定言語のテキストを取得、なければ en-US にフォールバック
func GetLocalizedText(texts LocalizedText, lang string) string {
    if text, ok := texts[lang]; ok && text != "" {
        return text
    }
    if text, ok := texts["en-US"]; ok {
        return text
    }
    return ""
}

// Tool represents an MCP tool definition
type Tool struct {
    ID           string           `json:"id"`                      // 安定ID (e.g., "notion:search")
    Name         string           `json:"name"`                    // 表示名/実行時キー
    Description  string           `json:"description"`             // 実行時の説明 (言語選択後)
    Descriptions LocalizedText    `json:"descriptions,omitempty"`  // 多言語説明 (export用)
    InputSchema  InputSchema      `json:"inputSchema"`
    Annotations  *ToolAnnotations `json:"annotations,omitempty"`
}
```

### 1.2 Module インターフェースの拡張

```go
// types.go

type Module interface {
    Name() string
    Descriptions() LocalizedText              // 追加: 多言語説明
    Description(lang string) string           // 変更: 言語指定版
    APIVersion() string
    Tools() []Tool
    ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error)
    // ... Resources, Prompts は変更なし
}
```

---

## Phase 2: 全モジュールの多言語対応

### 2.1 対象モジュール (8個)

1. notion
2. github
3. jira
4. confluence
5. supabase
6. airtable
7. google_calendar
8. microsoft_todo

### 2.2 module.go の変更パターン

```go
// 例: notion/module.go

var moduleDescriptions = modules.LocalizedText{
    "en-US": "Notion API - Page, Database, Block, Comment, and User operations",
    "ja-JP": "Notion API - ページ、データベース、ブロック、コメント、ユーザー操作",
}

func (m *NotionModule) Descriptions() modules.LocalizedText {
    return moduleDescriptions
}

func (m *NotionModule) Description(lang string) string {
    return modules.GetLocalizedText(moduleDescriptions, lang)
}
```

### 2.3 tools.go の変更パターン

```go
// 例: notion/tools.go

func toolDefinitions() []modules.Tool {
    return []modules.Tool{
        {
            ID:   "notion:search",
            Name: "search",
            Descriptions: modules.LocalizedText{
                "en-US": "Search pages and databases in Notion by title. Returns pages and databases shared with the integration.",
                "ja-JP": "Notionのページとデータベースをタイトルで検索します。インテグレーションと共有されているページとデータベースを返します。",
            },
            Annotations: modules.AnnotateReadOnly,
            InputSchema: modules.InputSchema{...},
        },
        // ... 他のツール
    }
}
```

### 2.4 ツール数一覧

| モジュール | ツール数 |
|------------|----------|
| notion | 14 |
| github | 20 |
| jira | 11 |
| confluence | 12 |
| supabase | 18 |
| airtable | 11 |
| google_calendar | 8 |
| microsoft_todo | 11 |
| **合計** | **105** |

---

## Phase 3: modules.go の更新

### 3.1 filterTools の tool.ID 対応

```go
// modules.go

// filterTools returns tools excluding disabled ones for a given module.
// disabledTools now uses tool.ID as key
func filterTools(moduleName string, tools []Tool, disabledTools map[string][]string) []Tool {
    if disabledTools == nil {
        return tools
    }
    disabled, ok := disabledTools[moduleName]
    if !ok {
        return tools
    }
    disabledSet := make(map[string]bool, len(disabled))
    for _, id := range disabled {
        disabledSet[id] = true
    }
    var filtered []Tool
    for _, tool := range tools {
        if !disabledSet[tool.ID] {  // Name -> ID に変更
            filtered = append(filtered, tool)
        }
    }
    return filtered
}
```

### 3.2 GetModuleSchemas の言語対応

```go
// modules.go

// GetModuleSchemas returns schemas for multiple modules with tool filtering and localization.
func GetModuleSchemas(
    moduleNames []string,
    enabledModules []string,
    disabledTools map[string][]string,
    lang string,                        // 追加: 言語コード
    userDescriptions map[string]string, // 追加: ユーザー追記
) (*ToolCallResult, error) {
    var schemas []ModuleSchema

    for _, name := range moduleNames {
        m, ok := registry[name]
        if !ok {
            continue
        }

        // 説明の結合: default + user追記
        defaultDesc := m.Description(lang)
        userDesc := userDescriptions[name]
        combinedDesc := defaultDesc
        if userDesc != "" {
            combinedDesc = defaultDesc + "\n\n" + userDesc
        }

        // ツールの言語選択
        tools := filterTools(name, m.Tools(), disabledTools)
        for i := range tools {
            tools[i].Description = GetLocalizedText(tools[i].Descriptions, lang)
        }

        schemas = append(schemas, ModuleSchema{
            Module:      m.Name(),
            Description: combinedDesc,
            APIVersion:  m.APIVersion(),
            Tools:       tools,
            Resources:   m.Resources(),
            Prompts:     m.Prompts(),
        })
    }
    // ...
}
```

### 3.3 DynamicMetaTools の多言語対応

```go
// modules.go

// メタツールの多言語説明
var metaToolDescriptions = map[string]LocalizedText{
    "get_module_schema": {
        "en-US": "Get tool definitions for modules. Important: Call only once per module per session. Schemas are cached in conversation history, so use run directly for subsequent calls to the same module.",
        "ja-JP": "モジュールのツール定義を取得。重要: 各モジュールにつき1セッション1回のみ呼び出すこと。スキーマは会話履歴にキャッシュされるため、同一モジュールへの2回目以降の呼び出しはrunを直接使用すること。",
    },
    "run": {
        "en-US": "Execute a single module tool.\n\n[Available Modules]\n%s\n\n[Usage]\n1. get_module_schema(module) to check available tools and parameters\n2. run(module, tool, params) to execute",
        "ja-JP": "モジュールのツールを呼び出す。\n\n【利用可能モジュール】\n%s\n\n【使い方】\n1. get_module_schema(module) でツール一覧とパラメータを確認\n2. run(module, tool, params) で実行",
    },
    "batch": {
        "en-US": "Execute multiple tools in batch (JSONL format, with dependency and parallel execution support)...",
        "ja-JP": "複数ツールをバッチ実行（JSONL形式、依存関係と並列実行をサポート）...",
    },
}

// DynamicMetaTools returns meta tools with dynamic module lists based on user's enabled modules and tool settings.
func DynamicMetaTools(enabledModules []string, disabledTools map[string][]string, lang string) []Tool {
    available := availableModuleNames(enabledModules, disabledTools)
    moduleList := strings.Join(available, ", ")

    // Build module description lines for run tool
    var moduleLines []string
    for _, name := range available {
        m, ok := registry[name]
        if !ok {
            continue
        }
        moduleLines = append(moduleLines, fmt.Sprintf("- %s: %s", name, m.Description(lang)))
    }
    moduleDesc := strings.Join(moduleLines, "\n")

    return []Tool{
        {
            Name:        "get_module_schema",
            Description: GetLocalizedText(metaToolDescriptions["get_module_schema"], lang),
            // ...
        },
        {
            Name:        "run",
            Description: fmt.Sprintf(GetLocalizedText(metaToolDescriptions["run"], lang), moduleDesc),
            // ...
        },
        {
            Name:        "batch",
            Description: GetLocalizedText(metaToolDescriptions["batch"], lang),
            // ...
        },
    }
}
```

---

## Phase 4: Handler の更新

### 4.1 AuthContext に言語追加

```go
// middleware/authz.go

type AuthContext struct {
    UserID          string
    EnabledModules  []string
    DisabledTools   map[string][]string
    Language        string              // 追加: user.preferences.language
    // ...
}
```

### 4.2 handleGetModuleSchema の更新

```go
// mcp/handler.go

func (h *Handler) handleGetModuleSchema(ctx context.Context, args map[string]interface{}) (*ToolCallResult, *Error) {
    // ... moduleNames の解析

    authCtx := middleware.GetAuthContext(ctx)
    if authCtx == nil {
        return nil, &Error{Code: InternalError, Message: "auth context missing"}
    }

    // ユーザー追記を取得 (Phase 5 で実装)
    userDescriptions := h.getUserModuleDescriptions(authCtx.UserID)

    result, err := modules.GetModuleSchemas(
        moduleNames,
        authCtx.EnabledModules,
        authCtx.DisabledTools,
        authCtx.Language,       // 言語
        userDescriptions,       // ユーザー追記
    )
    // ...
}
```

---

## Phase 5: DBマイグレーション

### 5.1 マイグレーションファイル

```sql
-- supabase/migrations/00000000000020_tool_id_and_module_description.sql

-- =============================================================================
-- Tool ID Migration + Module Description
-- =============================================================================
-- 1. tool_settings.tool_name -> tool_id (破壊的変更)
-- 2. module_settings.description 追加
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. tool_settings: tool_name -> tool_id
-- 既存データは互換性なし（新しいID形式に移行）
-- -----------------------------------------------------------------------------

-- 既存データを削除（tool_name -> tool_id への自動変換は複雑なため）
TRUNCATE TABLE mcpist.tool_settings;

-- カラム名変更
ALTER TABLE mcpist.tool_settings RENAME COLUMN tool_name TO tool_id;

-- コメント追加
COMMENT ON COLUMN mcpist.tool_settings.tool_id IS 'Tool ID in format: {module}:{tool_name} (e.g., notion:search)';

-- -----------------------------------------------------------------------------
-- 2. module_settings: description カラム追加
-- -----------------------------------------------------------------------------

ALTER TABLE mcpist.module_settings
    ADD COLUMN description TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN mcpist.module_settings.description IS 'User-defined additional description for the module';

-- -----------------------------------------------------------------------------
-- 3. RPC関数の更新
-- -----------------------------------------------------------------------------

-- get_tool_settings: tool_name -> tool_id
CREATE OR REPLACE FUNCTION mcpist.get_tool_settings(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,        -- 変更: tool_name -> tool_id
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ts.tool_id,         -- 変更
        ts.enabled
    FROM mcpist.tool_settings ts
    JOIN mcpist.modules m ON m.id = ts.module_id
    WHERE ts.user_id = p_user_id
      AND (p_module_name IS NULL OR m.name = p_module_name)
    ORDER BY m.name, ts.tool_id;
END;
$$;

-- public wrapper
CREATE OR REPLACE FUNCTION public.get_tool_settings(
    p_user_id UUID,
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_tool_settings(p_user_id, p_module_name);
$$;

-- upsert_tool_settings: tool_name -> tool_id
CREATE OR REPLACE FUNCTION mcpist.upsert_tool_settings(
    p_user_id UUID,
    p_module_name TEXT,
    p_enabled_tools TEXT[],   -- Now expects tool_id format
    p_disabled_tools TEXT[]   -- Now expects tool_id format
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
BEGIN
    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    -- 有効ツールをUPSERT
    IF p_enabled_tools IS NOT NULL AND array_length(p_enabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT p_user_id, v_module_id, unnest(p_enabled_tools), true
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = true;
    END IF;

    -- 無効ツールをUPSERT
    IF p_disabled_tools IS NOT NULL AND array_length(p_disabled_tools, 1) > 0 THEN
        INSERT INTO mcpist.tool_settings (user_id, module_id, tool_id, enabled)
        SELECT p_user_id, v_module_id, unnest(p_disabled_tools), false
        ON CONFLICT (user_id, module_id, tool_id)
        DO UPDATE SET enabled = false;
    END IF;

    RETURN jsonb_build_object(
        'success', true,
        'module', p_module_name,
        'enabled_count', COALESCE(array_length(p_enabled_tools, 1), 0),
        'disabled_count', COALESCE(array_length(p_disabled_tools, 1), 0)
    );
END;
$$;

-- public wrapper
CREATE OR REPLACE FUNCTION public.upsert_tool_settings(
    p_user_id UUID,
    p_module_name TEXT,
    p_enabled_tools TEXT[],
    p_disabled_tools TEXT[]
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_tool_settings(p_user_id, p_module_name, p_enabled_tools, p_disabled_tools);
$$;

-- get_my_tool_settings: 戻り値型変更
CREATE OR REPLACE FUNCTION mcpist.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM mcpist.get_tool_settings(auth.uid(), p_module_name);
END;
$$;

-- public wrapper
CREATE OR REPLACE FUNCTION public.get_my_tool_settings(
    p_module_name TEXT DEFAULT NULL
)
RETURNS TABLE (
    module_name TEXT,
    tool_id TEXT,
    enabled BOOLEAN
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_tool_settings(p_module_name);
$$;

-- upsert_my_tool_settings (変更なし、引数の意味が変わるだけ)

-- -----------------------------------------------------------------------------
-- 4. モジュール説明取得/更新用RPC
-- -----------------------------------------------------------------------------

-- get_my_module_descriptions: ユーザーのモジュール説明を取得
CREATE OR REPLACE FUNCTION mcpist.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.name AS module_name,
        ms.description
    FROM mcpist.module_settings ms
    JOIN mcpist.modules m ON m.id = ms.module_id
    WHERE ms.user_id = auth.uid()
      AND ms.description IS NOT NULL
      AND ms.description != ''
    ORDER BY m.name;
END;
$$;

CREATE OR REPLACE FUNCTION public.get_my_module_descriptions()
RETURNS TABLE (
    module_name TEXT,
    description TEXT
)
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT * FROM mcpist.get_my_module_descriptions();
$$;

GRANT EXECUTE ON FUNCTION mcpist.get_my_module_descriptions() TO authenticated;
GRANT EXECUTE ON FUNCTION public.get_my_module_descriptions() TO authenticated;

-- upsert_my_module_description: モジュール説明を更新
CREATE OR REPLACE FUNCTION mcpist.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = mcpist, public
AS $$
DECLARE
    v_module_id UUID;
BEGIN
    SELECT id INTO v_module_id
    FROM mcpist.modules
    WHERE name = p_module_name;

    IF v_module_id IS NULL THEN
        RETURN jsonb_build_object('error', 'Module not found: ' || p_module_name);
    END IF;

    INSERT INTO mcpist.module_settings (user_id, module_id, enabled, description)
    VALUES (auth.uid(), v_module_id, true, p_description)
    ON CONFLICT (user_id, module_id)
    DO UPDATE SET description = p_description;

    RETURN jsonb_build_object('success', true, 'module', p_module_name);
END;
$$;

CREATE OR REPLACE FUNCTION public.upsert_my_module_description(
    p_module_name TEXT,
    p_description TEXT
)
RETURNS JSONB
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT mcpist.upsert_my_module_description(p_module_name, p_description);
$$;

GRANT EXECUTE ON FUNCTION mcpist.upsert_my_module_description(TEXT, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION public.upsert_my_module_description(TEXT, TEXT) TO authenticated;
```

---

## Phase 6: tools-export の更新

### 6.1 出力形式の変更

```go
// cmd/tools-export/main.go

// tools.json 用
type ToolDef struct {
    ID           string            `json:"id"`           // notion:search
    Name         string            `json:"name"`         // search
    Descriptions map[string]string `json:"descriptions"` // 多言語
    Annotations  *ToolAnnotations  `json:"annotations,omitempty"`
}

type ModuleDef struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Descriptions map[string]string `json:"descriptions"` // 多言語
    APIVersion   string            `json:"apiVersion"`
    Tools        []ToolDef         `json:"tools"`
}

// services.json 用
type ServiceDef struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Descriptions map[string]string `json:"descriptions"` // 多言語
    APIVersion   string            `json:"apiVersion"`
}
```

### 6.2 出力例

**tools.json:**
```json
{
  "modules": [{
    "id": "notion",
    "name": "Notion",
    "descriptions": {
      "en-US": "Notion API - Page, Database, Block, Comment, and User operations",
      "ja-JP": "Notion API - ページ、データベース、ブロック、コメント、ユーザー操作"
    },
    "apiVersion": "2022-06-28",
    "tools": [{
      "id": "notion:search",
      "name": "search",
      "descriptions": {
        "en-US": "Search pages and databases in Notion by title.",
        "ja-JP": "Notionのページとデータベースをタイトルで検索します。"
      },
      "annotations": {
        "readOnlyHint": true,
        "openWorldHint": false
      }
    }]
  }]
}
```

**services.json:**
```json
{
  "services": [{
    "id": "notion",
    "name": "Notion",
    "descriptions": {
      "en-US": "Notion API - Page, Database, Block, Comment, and User operations",
      "ja-JP": "Notion API - ページ、データベース、ブロック、コメント、ユーザー操作"
    },
    "apiVersion": "2022-06-28"
  }]
}
```

---

## Phase 7: Console の更新

### 7.1 module-data.ts の型更新

```typescript
// lib/module-data.ts

export interface ToolDef {
  id: string           // notion:search (新形式)
  name: string         // search
  descriptions: Record<string, string>  // 多言語
  annotations: ToolAnnotations
}

export interface ModuleDef {
  id: string
  name: string
  descriptions: Record<string, string>  // 多言語
  apiVersion: string
  tools: ToolDef[]
}

// ヘルパー関数
export function getLocalizedText(
  texts: Record<string, string>,
  lang: string = 'ja-JP'
): string {
  return texts[lang] ?? texts['en-US'] ?? ''
}

// 後方互換: description プロパティのゲッター
export function getModuleDescription(mod: ModuleDef, lang: string = 'ja-JP'): string {
  return getLocalizedText(mod.descriptions, lang)
}

export function getToolDescription(tool: ToolDef, lang: string = 'ja-JP'): string {
  return getLocalizedText(tool.descriptions, lang)
}
```

### 7.2 tool-settings.ts の型更新

```typescript
// lib/tool-settings.ts

export interface ToolSetting {
  module_name: string
  tool_id: string      // 変更: tool_name -> tool_id
  enabled: boolean
}

// toToolSettingsMap も tool_id を使用
export function toToolSettingsMap(settings: ToolSetting[]): ToolSettingsMap {
  const map: ToolSettingsMap = {}
  for (const setting of settings) {
    if (!map[setting.module_name]) {
      map[setting.module_name] = {}
    }
    map[setting.module_name][setting.tool_id] = setting.enabled  // 変更
  }
  return map
}
```

### 7.3 tools/page.tsx の更新

```tsx
// (console)/tools/page.tsx

// 表示時の言語選択
const lang = 'ja-JP' // TODO: ユーザー設定から取得

// ツール説明の表示
<p className="text-sm text-muted-foreground">
  {getToolDescription(tool, lang)}
</p>

// ツール設定の保存時は tool.id を使用（既存のコードで対応済み）
```

---

## 実装順序チェックリスト

### Phase 1: Go Server 型定義
- [x] `types.go`: `LocalizedText` 型追加
- [x] `types.go`: `GetLocalizedText()` ヘルパー追加
- [x] `types.go`: `Tool` に `ID`, `Descriptions` 追加
- [x] `types.go`: `Module` インターフェース更新

### Phase 2: モジュール多言語対応
- [x] notion/module.go: `Descriptions()`, `Description(lang)` 実装
- [x] notion/tools.go: 全ツールに `ID`, `Descriptions` 追加 (14個)
- [x] github/module.go + tools.go (20個)
- [x] jira/module.go + tools.go (11個)
- [x] confluence/module.go + tools.go (12個)
- [x] supabase/module.go + tools.go (18個)
- [x] airtable/module.go + tools.go (11個)
- [x] google_calendar/module.go + tools.go (8個)
- [x] microsoft_todo/module.go + tools.go (11個)

### Phase 3: modules.go + メタツール多言語
- [x] `filterTools()`: `tool.ID` で判定
- [x] `GetModuleSchemas()`: `lang`, `userDescriptions` パラメータ追加
- [x] `DynamicMetaTools(enabledModules, lang)`: 言語パラメータ追加
- [x] `get_module_schema`, `run`, `batch` の説明を多言語化

### Phase 4: Handler
- [x] `AuthContext`: `Language` フィールド追加
- [x] authz middleware: `user.preferences.language` を AuthContext にセット
- [x] `handleGetModuleSchema()`: 言語とユーザー追記を渡す
- [x] `handleToolsList()`: 言語パラメータを `DynamicMetaTools` に渡す

### Phase 5: DBマイグレーション
- [x] マイグレーションファイル作成 (020-028)
- [x] ローカルでテスト
- [x] 本番適用
- [x] `get_user_context` RPC最適化: `enabled_modules` を `enabled_tools` のキーから導出

### Phase 6: tools-export
- [x] `ToolDef`, `ModuleDef` 型更新 (`descriptions` 追加)
- [x] `ServiceDef` 型更新 (`descriptions` 追加)
- [x] `exportTools()`: `descriptions` 出力
- [x] `exportServices()`: `descriptions` 出力
- [x] tools.json / services.json 再生成

### Phase 7: Console
- [x] `module-data.ts`: 型・ヘルパー更新
- [x] `tool-settings.ts`: `tool_id` 対応
- [x] `tools/page.tsx`: 言語選択表示
- [x] `settings/page.tsx`: 言語設定UI追加
- [x] `user-settings.ts`: ユーザー設定管理追加
- [ ] ~~(オプション) モジュール説明編集UI~~ 不要

### 完了 ✓
全Phase完了。ローカルテスト（curl）で日本語表示を確認済み。

---

## 影響範囲まとめ

| コンポーネント | 影響 |
|----------------|------|
| `tool_settings` テーブル | **破壊的**: 既存データ削除 |
| Go Server | 全モジュール修正 (8個) |
| Console | 型定義・表示ロジック |
| tools.json / services.json | フォーマット変更 |

---

## 注意事項

1. **既存データ移行**: `tool_settings` の既存データは `TRUNCATE` で削除
   - ユーザーがいないため破壊的変更OK
   - 移行スクリプト・告知は不要

2. **tool_id 命名規則**: `{module}:{tool_name}`
   - 例: `notion:search`, `github:list_repos`
   - モジュール名とツール名の間は `:` で区切る

3. **言語フォールバック**: 常に `en-US` にフォールバック
   - 日本語が未定義の場合は英語を表示

4. **後方互換**: `Tool.Name` は実行時キーとして維持
   - `ExecuteTool(ctx, name, params)` は `Name` で呼び出し
   - DB・設定は `ID` を使用

---

## 見積もり

| Phase | 工数 |
|-------|------|
| Phase 1: 型定義 | 0.5h |
| Phase 2: モジュール多言語 (105ツール) | 3-4h |
| Phase 3: modules.go + メタツール多言語 | 1.5h |
| Phase 4: Handler | 0.5h |
| Phase 5: DBマイグレーション | 1h |
| Phase 6: tools-export (tools.json + services.json) | 0.5h |
| Phase 7: Console | 1h |
| **合計** | **8-10h** |

---

## 設計決定（レビュー結果）

1. **tool_id**: `{module}:{tool_name}` 形式を採用（グローバル一意なID）
   - DB の `tool_settings` でも `notion:search` 形式で保存
   - `module_id` との冗長性はあるが、ID としての一貫性を優先

2. **メタツールの多言語対応**: `DynamicMetaTools` も多言語化する
   - `handleToolsList` 時点で `AuthContext` から言語を取得可能
   - `get_module_schema`, `run`, `batch` の説明も `en-US` / `ja-JP` 対応

3. **services.json**: `descriptions` 対応を追加（tools.json と同様）

4. **テスト**: 後回し、発見的に実装する
