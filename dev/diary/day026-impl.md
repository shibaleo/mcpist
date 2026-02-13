# DAY026 実装計画: InputSchemaランタイムバリデーション + ogen PoC準備

## 日付

2026-02-11

---

## 分析結果のサマリ

### `map[string]any` は正当か？

**正当。MCPプロトコルが強制する境界型。** MCP JSON-RPCの `tools/call` は任意のJSONを送るため、受け取り側では `map[string]any` として到着する。これは避けられない。

ogenを導入しても `map[string]any` → ogen型の手書き変換は消えない:

```
JSON wire → map[string]any → 手書き変換 → ogen型 → HTTP → ogen型 → json.Marshal → string
            ^^^^^^^^^^^^^^^^                                        ^^^^^^^^^^^^^^^
            消えない                                                 増える
```

### 本当の問題: InputSchemaが飾り

現状、`InputSchema` は `tools/list` でクライアントに返すだけで、ランタイムバリデーションに使われていない。各ハンドラが `owner, _ := params["owner"].(string)` と書き、失敗時は空文字で突き進む。

**ogenの導入判断とは独立に、この問題は今すぐ解決できる。**

---

## 方針

### Phase 0（本日実装）: InputSchemaバリデーション共通化

- `modules/validate.go` を新設
- `InputSchema` の `Required` と `Properties` からランタイムバリデーションを行う関数を実装
- `ExecuteTool` の入口で呼び出し、required チェック + 型coerce（float64→int等）を自動化
- **全モジュールに適用可能** — ogen有無に関わらず有用

### Phase 1（後日）: ogen PoC（C案ディレクトリ構成）

```
apps/server/pkg/githubapi/
  gen/          ← ogen生成物
  client.go     ← SecuritySource実装
apps/server/internal/modules/github/
  module.go     ← アダプタ（今と同じシグネチャ）
```

- Phase 0のバリデーションが入った上でogen化し、ワークロードの差分を公平に比較
- ogenで明らかにステップが増えるなら B案（現状維持 + バリデーション）が最善

---

## Phase 0: 設計詳細

### validate.go の責務

```go
// ValidateParams checks params against InputSchema.
// - Required fields: returns error if missing or zero-value
// - Type coercion: float64 → int for "number"/"integer" properties
// Returns coerced params (new map) or error.
func ValidateParams(schema InputSchema, params map[string]any) (map[string]any, error)
```

**やること:**
1. `required` フィールドの存在チェック
2. 型の一致チェック（"string" → string, "number" → float64, "boolean" → bool, "array" → []any, "object" → map[string]any）
3. float64 → int の正規化（JSON numberは常にfloat64で来るため）

**やらないこと:**
- enum バリデーション（InputSchema.Property に Enum フィールドがない）
- ネストしたオブジェクトの再帰バリデーション（過剰）
- デフォルト値の適用（ハンドラ側の責務を維持）

### 適用箇所

`modules.Run()` 内、`m.ExecuteTool()` の**前**でバリデーション:

```go
func Run(ctx context.Context, moduleName, toolName string, params map[string]any) (*ToolCallResult, error) {
    m, ok := registry[moduleName]
    if !ok { ... }

    // Find tool schema
    tool, ok := findTool(m.Tools(), toolName)
    if !ok { ... }

    // Validate and coerce
    coerced, err := ValidateParams(tool.InputSchema, params)
    if err != nil {
        return &ToolCallResult{
            Content: []ContentBlock{{Type: "text", Text: err.Error()}},
            IsError: true,
        }, nil
    }

    result, err := m.ExecuteTool(ctx, toolName, coerced)
    ...
}
```

### 各モジュールへの影響

**変更不要。** バリデーションは `modules.Run()` レベルで入るので、各モジュールの `ExecuteTool` やハンドラ関数は一切変更しない。既存の `owner, _ := params["owner"].(string)` パターンはそのまま動く（required チェック済みなので空文字になることがなくなる）。

---

## Phase 0: 実装ファイル一覧

| ファイル | 変更 |
|---|---|
| `internal/modules/validate.go` | **新規**: ValidateParams + findTool |
| `internal/modules/modules.go` | **変更**: Run() にバリデーション追加 |
| `internal/modules/validate_test.go` | **新規**: テスト |

---

## Phase 1 (後日): ogen PoC の指標

| 指標 | 判断基準 |
|---|---|
| gen/ のサイズ | 1MB以下なら許容 |
| ビルド時間の増加 | 10秒以内なら許容 |
| アダプタのワークロード | 手書き版と同程度なら許容 |
| 仕様刈り込みの再現性 | スクリプト化されていること |
| json.MarshalIndent互換性 | ogen型が encoding/json で直接シリアライズ可能か |

---

## Phase 0: 実装完了

- `internal/modules/validate.go` — `ValidateParams()`, `checkType()`, `findTool()` 実装
- `internal/modules/validate_test.go` — required, 型チェック, 空schema, integer, findTool テスト全pass
- `internal/modules/modules.go` — `Run()` に `ValidateParams` 呼出追加

---

## Phase 1: ogen PoC → 全面移行完了

### PoC (3エンドポイント)

| 指標 | 結果 | 判定 |
|---|---|---|
| gen/ のサイズ | 98KB / 3301行 (3エンドポイント) | OK |
| cold ビルド時間 | 33.6s | 10s超だが許容 |
| warm ビルド時間 | 1.6s | OK |
| 追加依存 | +27 (otel系含む) | 許容 |
| json.MarshalIndent互換 | OK (生成型がMarshalJSON実装) | OK |
| レスポンスフィールド制限 | subset specで定義したフィールドのみ返却 | **好都合** (LLMにフルJSON送る予定なし) |

### 設計判断

- **subset spec = ツール設計書**: GitHub spec のコピーではなく、mcpistが「何を返すか」の宣言
- **field filter と TOON変換は別の関心事**: subset spec = 何を返す (schema), ToCompact() = どう返す (format)
- **otel依存は許容**: 既にLokiへデータpush済み、otel移行は自然な流れ (ただし後回し)

### 全面移行実績

subset spec を20→22エンドポイントに拡張し、全ハンドラをogen化:

| カテゴリ | ツール数 | エンドポイント数 |
|----------|---------|-----------------|
| User | 2 | 2 |
| Repositories | 5 | 5 |
| Issues | 5 | 6 (list+create が同パス) |
| Pull Requests | 4 | 5 (list+create が同パス) |
| Search | 3 | 3 |
| Actions | 2 | 3 (list_workflow_runs が2パスに分岐) |
| **合計** | **21** | **22** |

新規追加: `list_starred_repos` — `GET /users/{username}/starred` (任意ユーザーのスター済みリポジトリ)

### 削除したもの

- `httpclient` パッケージへの依存 (GitHub module から完全除去)
- `net/url` による手動クエリ構築
- `map[string]interface{}` によるリクエストボディ構築
- `headers()` 関数 (ogen SecuritySource に置換)

### 成果物

| ファイル | 状態 |
|---|---|
| `pkg/githubapi/openapi-subset.yaml` | 新規: 22エンドポイントの subset spec (~1180行) |
| `pkg/githubapi/ogen.yaml` | 新規: server生成無効化 |
| `pkg/githubapi/client.go` | 新規: SecuritySource アダプタ |
| `pkg/githubapi/gen/` | 生成: 10ファイル |
| `internal/modules/github/module.go` | 改修: 全21ハンドラをogen化 |
| `docs/003_design/modules/dsn-modules.md` | 新規: モジュールアーキテクチャ設計書 |
| `docs/003_design/modules/github.md` | 新規: GitHub エンドポイントカタログ |

---

## Next: ツール選別

現在の21ツールは GitHub API エンドポイントと1:1対応。
今後はツールを複合関数として再設計し、エンドポイント > ツール の関係にする。

subset spec のエンドポイントは原始関数としてそのまま維持し、
toolDefinitions + toolHandlers で公開するツールを絞り込む。

---

## 参考: 公式OpenAPI仕様の有無

### 直接利用可能

| モジュール | 形式 | ソース |
|---|---|---|
| GitHub | OpenAPI 3.0/3.1 | https://github.com/github/rest-api-description |
| Jira | OpenAPI 3.0 | https://developer.atlassian.com/cloud/jira/platform/swagger-v3.v3.json |
| Asana | OpenAPI 3.0 | https://github.com/Asana/openapi |
| Grafana | OpenAPI 2.0 + 3.0 | https://github.com/grafana/grafana (`public/openapi3.json`) |
| Supabase | OpenAPI | https://github.com/supabase/cli (`api/`) |
| Microsoft Todo | OpenAPI (Graph API) | https://github.com/microsoftgraph/msgraph-metadata |

### 対象外（手書き維持）

Notion, Trello, Todoist, TickTick, Dropbox, Airtable, PostgreSQL
