# DSN: Module Architecture

## Status

- **Status**: Implemented
- **Date**: 2026-02-14

## Overview

mcpist のモジュールは 3 層構造で外部 API をラップする。各層の責務と詳細は [dsn-layers.md](dsn-layers.md) を参照。

## Design Decisions

### map[string]any は正当

MCP プロトコルの `tools/call` は任意の JSON を `params` として受け取る。
これは MCP 仕様自体が強制する境界型であり、Go 側で `map[string]any` として受け取るのは不可避。

`ValidateParams()` で `InputSchema` に基づく必須・型チェックを `ExecuteTool()` 前に自動実行する。

### subset spec = ツール設計

ogen で使う OpenAPI subset spec は「元 API spec のコピー」ではない。
mcpist が **何を返すか** を宣言するスキーマ定義である。

- 必要なフィールドだけを定義 → レスポンスが自動的にフィルタされる
- ツール追加 = subset spec にエンドポイント追記 → `ogen` 再生成 → ハンドラ実装
- subset spec (何を返すか) と format 層 (どう返すか) は別の関心事

### ogen 採用基準

| 条件 | 採用可否 |
|------|----------|
| 公式 OpenAPI spec が存在する API (GitHub, Supabase, Grafana, Jira 等) | ogen 推奨 |
| OpenAPI spec がない/不完全な API (Notion, Trello 等) | 手書き httpclient |

## Composite Tool Design

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

```
1. openapi-subset.yaml を編集 (エンドポイント追加/フィールド変更)
2. ogen 再生成:
   cd apps/server/pkg/<service>api
   ogen.exe -package gen -target gen -config ogen.yaml -clean openapi-subset.yaml
3. module.go にハンドラ追加
4. go build ./... で検証
```
