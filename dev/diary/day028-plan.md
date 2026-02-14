# DAY028 計画

## 日付

2026-02-13

---

## 概要

Sprint-007 延長 (7日目)。DAY027 で 10 モジュールの ogen 移行が完了したが、format ロジックがハンドラに混在している設計上の問題が判明。本日は format 層の分離リファクタを最優先で実施し、その上で残り 9 モジュールの ogen 移行を進める。

---

## DAY027 からの引き継ぎ

| 項目 | 状態 |
|------|------|
| ogen 移行 (10/19 モジュール) | ✅ 完了 |
| compact format (Notion, TickTick, Todoist, Trello) | ✅ 完了（ただしハンドラに混在） |
| httpclient 依存 (9 モジュール) | ❌ 未移行 |

### ogen 移行状況

| 状態 | モジュール |
|------|-----------|
| ✅ ogen 済 | GitHub, Supabase, Grafana, Asana, Jira, Confluence, Notion, TickTick, Todoist, Trello, Dropbox, Airtable, Microsoft Todo, Google Calendar, Google Tasks, Google Drive, Google Docs, Google Sheets, Google Apps Script |
| ❌ httpclient (未移行) | なし（全 19 モジュール移行完了） |
| — 対象外 | PostgreSQL (直接DB接続) |

---

## 本日のタスク

### 1. format 層の分離リファクタ（優先度：最高）

#### 問題

現在、format ロジックがハンドラ（tool 層）に混在している:

```go
// 現状: ハンドラが format を知っている
func createTask(ctx context.Context, params map[string]any) (string, error) {
    c, err := newOgenClient(ctx)
    req := gen.CreateTaskReq{Title: params["title"].(string)}
    res, err := c.CreateTask(ctx, &req)
    jsonStr, err := toJSON(res)
    return compactWriteResult(params, jsonStr, "id", "title")  // ← format が混入
}
```

これは 3 つの責務が混在:
- **tool 層**: API 呼び出しとパラメータ変換
- **format 層**: JSON → compact 表現への変換
- **ルーティング**: `format=json` の分岐判断

#### あるべき姿

```
spec層:   ogen 生成関数 (API 呼び出し)
tool層:   ハンドラ (パラメータ変換 → 関数呼び出し → JSON 返却)
format層: JSON → compact 表現 (format.go)
```

各層は隣の層の存在を知らない:
- tool 層は format パラメータの存在すら知らない。常に JSON を返す
- format 層は「渡された JSON を整形する」だけで、いつ呼ばれるかは知らない
- `format=json` の分岐は共通レイヤー (`Run()`) が行う

#### リファクタ後のコード

```go
// tool 層: 常に JSON を返す。format の存在を知らない
func createTask(ctx context.Context, params map[string]any) (string, error) {
    c, err := newOgenClient(ctx)
    req := gen.CreateTaskReq{Title: params["title"].(string)}
    res, err := c.CreateTask(ctx, &req)
    return toJSON(res)
}

// 共通レイヤー (modules.Run): format=json なら素通し、それ以外なら format 層へ
result, err := m.ExecuteTool(ctx, toolName, params)
if err == nil && params["format"] != "json" {
    result = applyFormat(moduleName, toolName, result)
}
```

#### タスク

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D28-001 | `modules.Run()` に format 分岐を追加 | `applyFormat(module, tool, json)` | ✅ 完了 |
| D28-002 | format.go を純粋な変換関数に書き換え | params 依存を除去 | ✅ 完了 |
| D28-003 | 全ハンドラから format 呼び出しを除去 | 常に JSON を返すように変更 | ✅ 完了 |
| D28-004 | format パラメータを toolDefinitions から除去 | tool 層のパラメータではない | ✅ 完了 |
| D28-005 | ビルド・テスト・動作確認 | format=json と compact の両方 | ✅ 完了 |

### 2. 残り 6 モジュールへの format.go 追加（優先度：高）

リファクタ後の正しい構造で format.go を追加。

| ID | タスク | ツール数 | 状態 |
|----|--------|---------|------|
| D28-006 | Jira format.go 追加 | 11 | ✅ 完了 |
| D28-007 | Confluence format.go 追加 | 12 | ✅ 完了 |
| D28-008 | GitHub format.go 追加 | 26 | ✅ 完了 |
| D28-009 | Asana format.go 追加 | 23 | ✅ 完了 |
| D28-010 | Supabase format.go 追加 | 18 | ✅ 完了 |
| D28-011 | Grafana format.go 追加 | 16 | ✅ 完了 |

### 3. 残り 9 モジュールの ogen 移行（優先度：高）

| ID | タスク | ツール数 | 備考 | 状態 |
|----|--------|---------|------|------|
| D28-012 | Dropbox ogen 移行 + format.go | 15 | Bearer (OAuth2) | ✅ 完了 |
| D28-013 | Airtable ogen 移行 + format.go | 11 | Bearer (PAT) | ✅ 完了 |
| D28-014 | Google Calendar ogen 移行 + format.go | 8 | Bearer (OAuth2) | ✅ 完了 |
| D28-015 | Google Tasks ogen 移行 + format.go | 9 | Bearer (OAuth2) | ✅ 完了 |
| D28-016 | Microsoft Todo ogen 移行 + format.go | 11 | Bearer (OAuth2) | ✅ 完了 |
| D28-017 | Google Docs ogen 移行 + format.go | 18 | Bearer (OAuth2) | ✅ 完了 |
| D28-018 | Google Drive ogen 移行 + format.go | 22 | Bearer (OAuth2) | ✅ 完了 |
| D28-019 | Google Sheets ogen 移行 + format.go | 27 | Bearer (OAuth2) | ✅ 完了 |
| D28-020 | Google Apps Script ogen 移行 + format.go | 17 | Bearer (OAuth2) | ✅ 完了 |

### 4. ツール定義テスト（優先度：中）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D28-021 | ツール定義テスト作成 | Description 非空、toolHandlers と toolDefinitions の一致 | ✅ 完了 |

---

## 実装方針

### 作業順序

1. **format 層分離 (D28-001〜005)** → 最優先。以降の全作業の前提
2. **format.go 追加 (D28-006〜011)** → 正しい構造で追加
3. **ogen 移行 (D28-012〜020)** → 小規模 → 大規模の順
4. **テスト (D28-021)** → 余裕があれば

### 3 層アーキテクチャ

```
[spec 層]   openapi-subset.yaml → ogen → 型安全な関数
[tool 層]   ハンドラ (パラメータ変換 → 関数呼び出し → JSON 返却)
[format 層] JSON → compact 表現 (CSV/MD/key-value)
```

各層の責務:
- **spec 層**: API エンドポイントから関数を生成する。MCP を知らない
- **tool 層**: MCP パラメータを受け取り、spec 層の関数を呼び、JSON を返す。format を知らない
- **format 層**: JSON を受け取り、compact 表現に変換する。tool を知らない

分岐は `modules.Run()` が行う:

```go
result := handler(ctx, params)  // tool 層
if params["format"] != "json" {
    result = format(result)     // format 層
}
```

### format.go の方針

- 純粋な変換関数: `func formatCompact(toolName, jsonStr string) string`
- params への依存なし
- CSV 出力は ` ```csv ` マークダウンコードブロックで囲む
- リスト系 → CSV、単体 → key-value テキスト

### 将来検討: spec → tool のコード生成

#### 背景

format 層を分離すると、ハンドラは純粋に「パラメータ変換 → 関数呼び出し → JSON 返却」だけになり、100% 機械的になる。

#### 責務の分離: spec と tool

```
[spec 層] openapi-subset.yaml → ogen → 型安全な関数 (Go)
[tool 層] tools.yaml           → toolgen → ツール定義 + ハンドラ (Go)
```

- **spec** の責務: エンドポイントごとの関数生成。言語固有の型安全なクライアントを出力する。MCP の知識を持たない
- **tool** の責務: 関数の組み合わせ方を宣言し、MCP ツールとしての定義 + ハンドラの Go コードを生成する

spec に `x-mcp-*` 等を混ぜるのは責務違反。2 つの関心事は分離する。

#### 言語非依存のパターン

「spec → 関数 → ツール」のパターンは言語に依存しない:

```
openapi-subset.yaml → ogen        → Go 関数
openapi-subset.yaml → openapi-ts  → TypeScript 関数
openapi-subset.yaml → openapi-gen → Python 関数
                         ↓
                    tools.yaml → toolgen → ツール定義 + ハンドラ
```

toolgen が見るのは「どの関数があるか」と「そのシグネチャ」だけ。

#### 標準ツール vs 拡張ツール

| 層 | 方式 | 対象 | 割合 |
|---|---|---|---|
| 標準ツール | tools.yaml → toolgen → Go コード生成 | 単純 CRUD | ~100% (format 分離後) |
| 拡張ツール | Go ハンドラ手書き | composite、複数 API 合成 | 例外のみ |

拡張ツールの具体例:
- GitHub: composite ツール (describe_user/repo/pr — 5 API 並行呼び出し → マージ)
- Notion: ブロック → Markdown 変換

#### 判断

DAY028 では **format 層分離を先に完成させる**。toolgen は次スプリント以降の検討とする。format 分離により「ハンドラが 100% 機械的」であることが実証されれば、toolgen の設計根拠が確立する。

### 設計書の削減: spec = 実装 = 設計書

#### 気づき

3 層分離の副次効果として、**モジュール別設計書の大半が不要になる**。

spec-tool が 1:1 対応しているツール（全体の ~100%）では:
- spec (openapi-subset.yaml) が公式 API 仕様のサブセットであり
- tool 層のハンドラは spec の機械的な変換であり
- つまり **spec が事実上の実装であり設計書**

ここに別途設計書を書いても「公式ドキュメントの劣化コピー」にしかならない。

#### 設計書が必要な箇所

設計書は**設計判断があるところだけ**書けばよい:

| 対象 | 設計判断 | 設計書 |
|---|---|---|
| 1:1 ツール (get/list/create/update/delete) | なし。spec = 実装 | **不要** |
| subset spec のフィールド選定 | 何を含めて何を落としたか | **spec 自体が設計書** |
| composite ツール (describe_user 等) | どの API を組み合わせるか | 必要 |
| format のフィールド選別 | compact で何を見せるか | **format.go 自体が設計書** |
| 認証方式の差異 (dual auth 等) | client.go の設計判断 | 必要に応じて |

#### 影響

これにより DAY028 以降のモジュール実装と設計書整備が一気に進む:

- 新規モジュールの ogen 移行時に設計書を別途書く必要がない
- 既存の github.md / supabase.md / asana.md 等のエンドポイントカタログも、composite ツールと認証の記述以外は不要
- dsn-layers.md (3 層アーキテクチャ) が全モジュール共通の設計書として機能する

#### 本質

設計書が多いのは設計が複雑なサインであり、ドキュメントの書き方で解決する問題ではなかった。層を正しく分離して機械的な部分をコード生成に任せれば、「書くべき設計書」自体が減る。

---

## 完了条件

- [x] format 層がハンドラから分離され、`modules.Run()` で適用される
- [x] 全ハンドラが常に JSON を返す
- [x] 全 19 モジュールに format.go が追加済み
- [x] 残り 9 モジュール全て ogen 化完了（全 19/19 モジュール移行完了）
- [x] ビルド pass
- [x] tools.json 再生成（差分なし、最新状態を確認済み）
- [x] ローカルサーバーテスト（6 Google モジュール Read/Write/Delete 確認済み）
- [x] OAuth2 token refresh のエラーボディ出力改善（全 6 Google モジュール）
- [x] ツール定義テスト作成（全 PASS）
- [x] テストで検出: grafana `query_datasource` の InputSchema.Type 欠落 → 修正済み
- [x] `internal/store` → `internal/broker` リネーム完了（全 23 ファイル）
- [x] OAuth2 リフレッシュを broker に集約（11 モジュールから削除、`OAuthRefreshConfig` テーブル駆動）
- [x] ローカルサーバーテスト（13 モジュール正常応答、asana/airtable リフレッシュ動作確認）

### 5. OAuth2 リフレッシュ共通化 + store → broker リネーム（優先度：高）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D28-022 | `internal/store/` → `internal/broker/` リネーム | パッケージ名・ディレクトリ移動 | ✅ 完了 |
| D28-023 | `OAuthRefreshConfig` テーブル + `refreshOAuthToken()` 実装 | 6 プロバイダ × 11 モジュール対応 | ✅ 完了 |
| D28-024 | `GetModuleToken()` にリフレッシュ統合 | fetchCredentials → needsRefresh → refresh の透過処理 | ✅ 完了 |
| D28-025 | 全 23 ファイルの import `store` → `broker` 更新 | `GetTokenStore()` → `GetTokenBroker()` | ✅ 完了 |
| D28-026 | 11 モジュールから refreshToken/needsRefresh 削除 | +679/-1,306 行 | ✅ 完了 |
| D28-027 | ローカルサーバーテスト | 13 モジュール正常応答、asana/airtable でリフレッシュ動作確認 | ✅ 完了 |

---

## 参考

- [day027-worklog.md](day027-worklog.md) - DAY027 作業ログ
- [day027-plan.md](day027-plan.md) - DAY027 計画
- [sprint007-plan.md](../sprint/sprint007-plan.md) - Sprint 007 計画
- [dsn-modules.md](../../docs/003_design/modules/dsn-modules.md) - モジュールアーキテクチャ設計書
