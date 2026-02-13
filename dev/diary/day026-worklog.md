# DAY026 作業ログ

## 日付

2026-02-11

---

## コミット一覧

| # | ハッシュ | 時刻 | メッセージ |
|---|---------|------|-----------|
| 1 | `980174c` | 19:19 | feat(server): migrate GitHub module to ogen-generated client with input validation |
| 2 | `2dbbde9` | 19:31 | fix(server): version up golang 1.23 to 1.24 |
| 3 | `ff0422f` | 20:14 | feat(server): add describe_user composite tool for GitHub user analysis |
| 4 | `075a599` | 21:38 | feat(server): add describe_repo and describe_pr composite tools with compact JSON output |
| 5 | `c70d5d3` | 22:21 | feat(server): migrate Supabase module to ogen-generated client with composite tools |
| 6 | `d1e577f` | 22:37 | feat(server): migrate Grafana module to ogen-generated client with dual auth support |
| 7 | `7c5b469` | 22:49 | fix(server): remove generate_typescript_types tool and fix list_migrations SQL |
| 8 | `12314f9` | 23:01 | perf(server): use compact JSON for tool responses to reduce token usage |
| 9 | `3586be8` | 23:30 | feat(server): migrate Asana module to ogen-generated client with subset spec |

---

## 完了タスク

### Phase 0: InputSchema バリデーション共通化 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D26-001 | `validate.go` 新設 (ValidateParams, checkType, findTool) | ✅ | required チェック + 型チェック |
| D26-002 | `validate_test.go` 新設 | ✅ | 全テスト pass |
| D26-003 | `modules.go` に ValidateParams 呼出追加 | ✅ | ExecuteTool の前で自動実行 |

### Phase 1: ogen 全面移行 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D26-004 | GitHub module ogen 移行 | ✅ | 22 endpoints, 21→26 tools (composite 3 + list_starred_repos 追加) |
| D26-005 | Go 1.23 → 1.24 バージョンアップ | ✅ | go.mod 更新 |
| D26-006 | describe_user 複合ツール | ✅ | 5 API 並行呼出 → フィールド選別 → 1 JSON |
| D26-007 | describe_repo / describe_pr 複合ツール | ✅ | repo: 5 API, pr: 2 API 並行呼出 |
| D26-008 | Supabase module ogen 移行 | ✅ | 12 endpoints, 18 tools (composite 2: describe_project, inspect_health) |
| D26-009 | Grafana module ogen 移行 | ✅ | 16 endpoints, 16 tools (dual auth: Bearer/Basic) |
| D26-010 | Supabase generate_typescript_types 削除 + list_migrations SQL 修正 | ✅ | 不要ツール除去 |
| D26-011 | レスポンス compact JSON 化 (json.Marshal → json.MarshalIndent 削除) | ✅ | トークン使用量削減 |
| D26-012 | Asana module ogen 移行 | ✅ | 26 operations, 23 tools, data envelope パターン |
| D26-013 | Asana 本番 MCP ツールテスト | ✅ | 22/23 pass (search_tasks は Premium 制約で 402) |
| D26-014 | 設計書作成 (dsn-modules.md, github.md, supabase.md, asana.md) | ✅ | ドキュメント整備 |

### ヘルパー共通化 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D26-015 | `modules/helpers.go` 新設 (ToJSON, ToStringSlice) | ✅ | 4モジュールから重複排除 |
| D26-016 | GitHub/Supabase/Grafana/Asana の toJSON/toStringSlice を共通ヘルパーに置換 | ✅ | ビルド・テスト・tools.json diff なし確認済み |

---

## 作業詳細

### 1. ogen モジュールアーキテクチャ

2層構造で外部 API をラップ:

```
MCP Host → JSON-RPC → Module Interface 層 → API Client 層 (ogen) → HTTP
```

- **Module Interface 層** (`internal/modules/<service>/module.go`) — map[string]any で MCP と接合
- **API Client 層** (`pkg/<service>api/`) — OpenAPI subset spec から ogen 生成

### 2. 移行結果サマリ

| モジュール | ツール数 | ogen operations | 固有の課題 |
|-----------|---------|-----------------|-----------|
| GitHub | 26 | 22 | composite ツール 3 (describe_user/repo/pr) |
| Supabase | 18 | 12 | composite ツール 2 (describe_project, inspect_health), jx.Raw レスポンス |
| Grafana | 16 | 16 | dual auth (Bearer/Basic), 動的 URL, jx.Raw レスポンス |
| Asana | 23 | 26 | data envelope, 条件付きエンドポイント (list_projects×2, list_tasks×3), OAuth refresh, 2段階 create_task |
| **合計** | **83** | **76** | |

### 3. subset spec = ツール設計書

ogen の subset spec は「GitHub API spec のコピー」ではなく、mcpist が何を返すかの宣言:

- 必要なフィールドだけ定義 → レスポンスが自動フィルタ
- ツール追加 = spec にエンドポイント追記 → ogen 再生成 → ハンドラ実装

### 4. レスポンスフォーマット戦略

| Level | 方式 | 使い所 |
|-------|------|--------|
| Level 1: Spec Level | subset spec による自動フィルタ | 単一 API (ogen) |
| Level 2: Field Selection | ハンドラ内フィールド選別 + toJSON | 複合ツール |
| Level 3: Format Conversion | MD/CSV/TOON 変換 | 手書きモジュール (Notion 等) |

### 5. ヘルパー共通化

4モジュールに重複していた `toJSON` / `toStringSlice` を `modules/helpers.go` に集約:

| 関数 | 移動元 | 方式 |
|------|--------|------|
| `ToJSON(v any) (string, error)` | GitHub, Supabase, Grafana, Asana | 公開関数として定義 |
| `ToStringSlice(v []interface{}) []string` | GitHub, Asana | 公開関数として定義 |

各モジュールでは `var toJSON = modules.ToJSON` で参照。ハンドラ側の呼び出しは変更不要。

---

## 変更ファイル

### 新規ファイル

| ファイル | 内容 |
|----------|------|
| `internal/modules/validate.go` | ValidateParams, checkType, findTool |
| `internal/modules/validate_test.go` | バリデーションテスト |
| `internal/modules/helpers.go` | ToJSON, ToStringSlice 共通ヘルパー |
| `pkg/githubapi/openapi-subset.yaml` | GitHub subset spec (22 endpoints, ~1180行) |
| `pkg/githubapi/ogen.yaml` | ogen 設定 (server 生成無効化) |
| `pkg/githubapi/client.go` | SecuritySource アダプタ |
| `pkg/githubapi/gen/` | ogen 自動生成 (10ファイル) |
| `pkg/supabaseapi/openapi-subset.yaml` | Supabase subset spec (12 endpoints) |
| `pkg/supabaseapi/ogen.yaml` | ogen 設定 |
| `pkg/supabaseapi/client.go` | SecuritySource アダプタ (Bearer, 固定URL) |
| `pkg/supabaseapi/client_test.go` | 統合テスト |
| `pkg/supabaseapi/gen/` | ogen 自動生成 |
| `pkg/grafanaapi/openapi-subset.yaml` | Grafana subset spec (16 endpoints, dual auth) |
| `pkg/grafanaapi/ogen.yaml` | ogen 設定 |
| `pkg/grafanaapi/client.go` | SecuritySource アダプタ (Bearer/Basic, 動的URL) |
| `pkg/grafanaapi/gen/` | ogen 自動生成 |
| `pkg/asanaapi/openapi-subset.yaml` | Asana subset spec (26 operations, data envelope) |
| `pkg/asanaapi/ogen.yaml` | ogen 設定 |
| `pkg/asanaapi/client.go` | SecuritySource アダプタ (Bearer, 固定URL) |
| `pkg/asanaapi/client_test.go` | 統合テスト (13 tests) |
| `pkg/asanaapi/gen/` | ogen 自動生成 |
| `docs/003_design/modules/dsn-modules.md` | モジュールアーキテクチャ設計書 |
| `docs/003_design/modules/github.md` | GitHub エンドポイントカタログ |
| `docs/003_design/modules/supabase.md` | Supabase エンドポイントカタログ |
| `docs/003_design/modules/asana.md` | Asana エンドポイントカタログ |

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `go.mod` / `go.sum` | Go 1.24 + ogen 依存追加 |
| `internal/modules/modules.go` | Run() に ValidateParams 呼出追加 |
| `internal/modules/github/module.go` | 全ハンドラ ogen 化 + composite ツール追加 + toJSON/toStringSlice 共通化 |
| `internal/modules/supabase/module.go` | 全ハンドラ ogen 化 + toJSON 共通化 |
| `internal/modules/grafana/module.go` | 全ハンドラ ogen 化 + toJSON 共通化 |
| `internal/modules/asana/module.go` | 全ハンドラ ogen 化 (httpclient 除去) + toJSON/toStringSlice 共通化 |

---

## 設計判断

### ogen 採用基準

| 条件 | 判断 |
|------|------|
| 公式 OpenAPI spec あり (GitHub, Supabase, Grafana, Jira, Asana) | ogen 推奨 |
| OpenAPI spec なし/不完全 (Notion, Trello, Todoist, etc.) | 手書き httpclient |

### 共通化の判断

| 対象 | 判断 | 理由 |
|------|------|------|
| toJSON / toStringSlice | ✅ 共通化 | 完全同一の関数が4箇所、リスクゼロ |
| Module struct boilerplate | ❌ 見送り | 各1-3行で短い、embed するとむしろ読みにくい |
| getCredentials | ❌ 見送り | モジュール名が異なる、Asana は OAuth リフレッシュ追加 |
| newOgenClient | ❌ 見送り | 型が異なる、Grafana は認証方式分岐あり |

---

## DAY026 サマリ

| 項目 | 内容 |
|------|------|
| InputSchema バリデーション | 全モジュール共通の実行前バリデーション導入 |
| ogen 移行 | GitHub, Supabase, Grafana, Asana の 4 モジュール完了 (83 tools, 76 ogen ops) |
| 複合ツール | describe_user, describe_repo, describe_pr, describe_project, inspect_health |
| ヘルパー共通化 | ToJSON, ToStringSlice を modules パッケージに集約 |
| 設計書 | dsn-modules.md + 3 モジュール別ドキュメント作成 |
| コミット数 | 9 |
| テスト | ビルド成功、テスト pass、tools.json diff なし |

---

## 次回の作業

1. ヘルパー共通化コミット (未コミット: helpers.go + 4 module.go 変更)
2. Jira module ogen 移行 (公式 OpenAPI 3.0 spec あり、11 tools)
3. Confluence module ogen 移行 (Atlassian OpenAPI spec あり、12 tools)
