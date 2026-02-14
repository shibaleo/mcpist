# DSN: Three-Layer Architecture (spec / tool / format)

## Status

- **Status**: Draft
- **Date**: 2026-02-13

## Overview

mcpist のモジュールは 3 つの独立した層で構成される。各層は隣の層の存在を知らない。

```
[spec 層]   openapi-subset.yaml → ogen → 型安全な関数
[tool 層]   ハンドラ (パラメータ変換 → 関数呼び出し → JSON 返却)
[format 層] JSON → compact 表現 (CSV/MD/key-value)
```

## Motivation

DAY026-027 の ogen 全面移行で、HTTP 層の手書きが排除された。しかし format ロジックがハンドラに混在しており、以下の問題が発生していた:

- ハンドラが `format=json` パラメータを知っている (tool 層が format の存在を知っている)
- `compactWriteResult(params, jsonStr, "id", "title")` のようにフィールド選別がハンドラ内にある
- ハンドラを機械的に生成しようとすると format の知識が必要になる

これは 3 つの責務が混在している状態:

1. **tool 層**: API 呼び出しとパラメータ変換
2. **format 層**: JSON → compact 表現への変換
3. **ルーティング**: `format=json` の分岐判断

## Layer Definitions

### spec 層 (API Client)

**責務**: API エンドポイントから型安全な関数を生成する。

- openapi-subset.yaml を Single Source of Truth として ogen がコード生成
- MCP の存在を知らない
- レスポンスのフィールドフィルタはこの層で行う (subset spec に定義したフィールドのみ返る)

**成果物**: `pkg/<service>api/gen/*.go`

### tool 層 (Handler)

**責務**: MCP パラメータを受け取り、spec 層の関数を呼び、JSON を返す。

- 常に JSON を返す。format の存在を知らない
- `format` パラメータは tool のパラメータではない
- パラメータ変換 → 関数呼び出し → JSON シリアライズ、の機械的な処理のみ

**成果物**: `internal/modules/<service>/module.go`

```go
// tool 層のハンドラ: 常に JSON を返す
func createTask(ctx context.Context, params map[string]any) (string, error) {
    c, err := newOgenClient(ctx)
    if err != nil {
        return "", err
    }
    req := gen.CreateTaskReq{Title: params["title"].(string)}
    if v, ok := params["project_id"].(string); ok && v != "" {
        req.ProjectId.SetTo(v)
    }
    res, err := c.CreateTask(ctx, &req)
    if err != nil {
        return "", err
    }
    return toJSON(res)
}
```

### format 層 (Formatter)

**責務**: JSON を受け取り、compact 表現に変換する。

- tool を知らない。「渡された JSON を整形する」だけ
- いつ呼ばれるかは知らない (呼び出し判断は共通レイヤーが行う)
- params への依存なし

**成果物**: `internal/modules/<service>/format.go`

```go
// format 層: 純粋な変換関数。params を知らない
func formatCompact(toolName, jsonStr string) string {
    switch toolName {
    case "list_projects":
        return projectsToCSV(jsonStr)
    case "get_task":
        return taskToCompact(jsonStr)
    default:
        return jsonStr
    }
}
```

## Routing: format パラメータの分岐

`format=json` の判断は spec でも tool でも format でもなく、**MCP ハンドラ** (`handler.go`) が行う。

```go
// handler.go — MCP ハンドラ
result, err := modules.Run(ctx, moduleName, toolName, params)  // tool 層: 常に JSON
if !result.IsError {
    if f, _ := params["format"].(string); f != "json" {
        result.Content[0].Text = modules.ApplyCompact(moduleName, toolName, result.Content[0].Text)
    }
}
```

- `modules.Run()` は tool 層のみ。format の存在を知らない
- `ApplyCompact()` は `CompactConverter` インターフェースを持つモジュールにのみ適用
- batch 実行でも同じ `ApplyCompact()` が `output: true` のタスクに適用される

## Two-Stage Filtering

API レスポンスは 2 段階でフィルタされる。各段階は独立しており、目的が異なる。

```
API レスポンス (50 fields)
  → spec 層: 不要フィールド排除 (→ 15 fields)  ← 不可逆。捨てたら戻らない
  → format 層: 表示フィルタ (→ 5 fields)        ← 可逆。format=json で全部見える
```

### spec 層のフィルタ (不可逆)

subset spec に定義したフィールドのみ ogen がデシリアライズする。**mcpist として意味のないフィールドは入口で落とす。**

- 情報価値が少しでもあるフィールドは spec に含める
- 完全に不要なフィールド (内部 ID、deprecated フィールド等) だけを排除する
- 一度 spec から外したフィールドは `format=json` でも見えない

### format 層のフィルタ (可逆)

spec が通したフィールドのうち、デフォルト表示で見せるフィールドを選ぶ。**情報は保持するが、表示を絞る。**

- compact 表示で見せたいフィールドだけを CSV/key-value に含める
- 残りは `format=json` で全フィールドを閲覧可能
- フィールドの選別は人が判断する (ツールの用途に応じた設計)

### jx.Raw (any) のケース

Notion のようにレスポンスが構造化されていない API では、spec 層でのフィルタが効かない。この場合 format 層の負担が大きくなるが、アーキテクチャとしては同じ。spec と format は密結合ではなく、**同じ方向 (情報量の削減) に段階的に働く独立した層**である。

## Future: spec → tool Code Generation

### 背景

format 層を分離すると、tool 層のハンドラは純粋に「パラメータ変換 → 関数呼び出し → JSON 返却」だけになる。これは 100% 機械的な処理であり、コード生成の対象になる。

### 責務の分離: spec と tool

```
[spec 層] openapi-subset.yaml → ogen → 型安全な関数 (Go)
[tool 層] tools.yaml           → toolgen → ツール定義 + ハンドラ (Go)
```

- **spec** の責務: エンドポイントごとの関数生成。言語固有の型安全なクライアントを出力する。MCP の知識を持たない
- **tool** の責務: 関数の組み合わせ方を宣言し、MCP ツールとしての定義 + ハンドラの Go コードを生成する

spec に `x-mcp-*` 等の拡張を混ぜるのは責務違反。2 つの入力ファイルは独立して管理する。

### 言語非依存のパターン

「spec → 関数 → ツール」のパターンは言語に依存しない:

```
openapi-subset.yaml → ogen        → Go 関数
openapi-subset.yaml → openapi-ts  → TypeScript 関数
openapi-subset.yaml → openapi-gen → Python 関数
                         ↓
                    tools.yaml → toolgen → ツール定義 + ハンドラ
```

toolgen が見るのは「どの関数があるか」と「そのシグネチャ」だけ。関数の生成元が ogen でも他のジェネレータでも、tool 層の設計は変わらない。

### 標準ツール vs 拡張ツール

| 層 | 方式 | 対象 |
|---|---|---|
| 標準ツール | tools.yaml → toolgen → Go コード生成 | 単純 CRUD (format 分離後は ~100%) |
| 拡張ツール | Go ハンドラ手書き | composite、複数 API 合成 (例外のみ) |

拡張ツールの具体例:
- GitHub: composite ツール (describe_user/repo/pr — 5 API 並行呼び出し → マージ)
- Notion: ブロック → Markdown 変換

### テスト戦略

コード生成パイプラインの導入により、テスト対象が明確になる:

- **自動生成層** (ogen + toolgen) → 生成パイプラインの正しさを保証すればよい。生成物自体はテスト不要
- **手書き層** → Description、format.go のフィールド選別、拡張ツールのロジックだけをテストすれば十分

「テストすべきものが減った」のではなく、「テストすべき境界が見えた」ということ。

### 判断

toolgen は次スプリント以降の検討とする。先に format 層分離を完了し、「ハンドラが 100% 機械的」であることを実証する。それが toolgen の設計根拠になる。
