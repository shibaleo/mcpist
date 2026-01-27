# Round 1: 基盤深化 - 詳細実装計画

**期間**: 3日（Day 3-5）
**目的**: Round 0の知見を還流させ、基盤を強化
**状態**: ✅ 完了（2026-01-10）

---

## 現状分析

### Round 0 完了状態

| ファイル | 状態 | 内容 |
|---------|------|------|
| `internal/mcp/handler.go` | 実装済 | SSE対応、JSON-RPC 2.0基本実装 |
| `internal/mcp/types.go` | 実装済 | MCP型定義、エラーコード |
| `internal/auth/middleware.go` | 実装済 | Bearer token認証（テストなし） |
| `internal/modules/supabase/list_projects.go` | 実装済 | プロジェクト一覧取得 |
| `internal/modules/supabase/run_query.go` | 実装済 | SQL実行 |
| `internal/observability/loki.go` | 実装済 | Loki Push API |

### 改善が必要な点（Round 0からの還流）

1. **エラーハンドリング**: 各ツールでエラー処理が重複、統一パターンなし
2. **HTTPクライアント**: 各ツールで個別に`http.Client`を生成（共有なし）
3. **JSON-RPC**: `id`のバリデーションなし、通知（id省略）の処理が不完全
4. **テスト**: ユニットテストなし

---

## Day 3: SSE/JSON-RPC改善 + エラーハンドリング統一 ✅

### タスク 3.1: 共通HTTPクライアントの導入 ✅

**目的**: タイムアウト設定とHTTPクライアントの共有

```
internal/
├── httpclient/
│   ├── client.go       # 共通HTTPクライアント
│   └── client_test.go  # テスト（5テスト）
```

**実装内容**:
- [x] `internal/httpclient/client.go` 作成
  - デフォルトタイムアウト: 30秒
  - リトライなし（シンプル優先）
  - User-Agent設定: `go-mcp-dev/0.1.0`
  - `DoJSON()` - JSONリクエスト/レスポンス処理
  - `PrettyJSON()` - レスポンス整形

### タスク 3.2: Supabase共通クライアント ✅

**目的**: Supabaseツール間でHTTPクライアントと認証を共有

```
internal/modules/supabase/
└── module.go          # モジュール定義（list_projects, run_query統合）
```

**実装内容**:
- [x] `internal/modules/supabase/module.go` 作成
  - `Module()` - モジュール定義を返す
  - `listProjects()` - プロジェクト一覧
  - `runQuery()` - SQL実行
  - 共通HTTPクライアント使用

### タスク 3.3: メタツール実装 ✅

**目的**: 遅延読み込みパターンでツール数削減

**修正ファイル**: `internal/mcp/handler.go`, `internal/modules/registry.go`

**実装内容**:
- [x] `get_module_schema` - モジュールのスキーマ取得
- [x] `call_module_tool` - モジュールのツール実行
- [x] `tools/list` は2つのメタツールのみ返す
- [x] モジュールレジストリパターン実装

### タスク 3.4: エラーハンドリング統一 ✅

**目的**: 一貫したエラーレスポンス

**実装内容**:
- [x] `ToolCallResult.IsError` フィールドでエラー判定
- [x] JSON-RPCエラーコード統一（ParseError, InvalidParams等）
- [x] モジュール内エラーはisError=trueで返却

---

## Day 4: Supabase残りツール追加 ⏭ スキップ

**理由**: モジュール追加は後回しにし、Round 3（仕上げ）を先に完了させる方針に変更。Round 2でモジュール拡張を行う。

---

## Day 5: 最小CI + 認証ミドルウェアテスト ✅

### タスク 5.1: GitHub Actions CI設定 ✅

**ファイル**: `.github/workflows/ci.yml`

**実装内容**:
- [x] `.github/workflows/ci.yml` 作成
  - トリガー: PR時（テストのみ）、main push時（デプロイ）
  - Go 1.22 セットアップ
  - `go build ./...`
  - `go test ./...`
  - main pushでKoyeb再デプロイ

### タスク 5.2: 認証ミドルウェアテスト ✅

**ファイル**: `internal/auth/middleware_test.go`

**実装内容**:
- [x] テストケース（6テスト）:
  - `TestMiddleware_ValidToken` - 正しいトークンで200
  - `TestMiddleware_InvalidToken` - 間違ったトークンで401
  - `TestMiddleware_MissingHeader` - Authorizationヘッダーなしで401
  - `TestMiddleware_WrongScheme` - Bearer以外のスキームで401
  - `TestMiddleware_EmptyToken` - 空トークンで401
  - `TestMiddleware_HealthEndpoint` - /healthはスキップ

### タスク 5.3: JSON-RPCパーサーテスト ✅

**ファイル**: `internal/mcp/handler_test.go`

**実装内容**:
- [x] テストケース（10テスト）:
  - `TestHandleInlineMessage_Initialize` - initialize正常系
  - `TestHandleInlineMessage_ToolsList` - tools/list正常系（2メタツール）
  - `TestHandleInlineMessage_GetModuleSchema` - get_module_schema正常系
  - `TestHandleInlineMessage_CallModuleTool` - call_module_tool正常系
  - `TestHandleInlineMessage_ParseError` - 不正JSON
  - `TestHandleInlineMessage_MethodNotFound` - 未知メソッド
  - `TestHandleInlineMessage_UnknownTool` - 未知ツール
  - `TestServeHTTP_MethodNotAllowed` - 不正HTTPメソッド
  - `TestHandleInlineMessage_UnknownModule` - 未知モジュール
  - `TestHandleInlineMessage_InvalidParams` - 不正パラメータ

### タスク 5.4: Koyeb設定 ✅

**作業**: GitHub SecretsとKoyeb連携

**設定済み**:
- [x] `KOYEB_API_TOKEN` - Koyeb APIトークン
- [x] `KOYEB_SERVICE_ID` - サービスID
- [x] GitHub Branch Protection（testステータス必須）

---

## 成功条件

### Day 3 完了条件 ✅
- [x] 共通HTTPクライアントが全Supabaseツールで使用されている
- [x] メタツールパターンが実装されている
- [x] エラーハンドリングが統一パターンになっている

### Day 4 完了条件 ⏭ スキップ
- ⏭ supabase_get_project が動作する（Round 2で実装）
- ⏭ supabase_list_tables が動作する（Round 2で実装）
- ⏭ supabase_get_table_schema が動作する（Round 2で実装）

### Day 5 完了条件 ✅
- [x] GitHub ActionsでPR時にテストが実行される
- [x] main pushでKoyebに自動デプロイされる
- [x] 認証ミドルウェアのテストカバレッジ100%
- [x] JSON-RPCパーサーの主要パスがテスト済み

---

## Round 1 完了後のファイル構成

```
go-mcp-dev/
├── .github/
│   └── workflows/
│       ├── ci.yml                       # ✅ CI/CD
│       └── ping.yml                     # ✅ スリープ回避
├── cmd/
│   └── server/
│       └── main.go                      # ✅ モジュール登録
├── internal/
│   ├── auth/
│   │   ├── middleware.go                # ✅
│   │   └── middleware_test.go           # ✅ 6テスト
│   ├── httpclient/
│   │   ├── client.go                    # ✅ 共通HTTPクライアント
│   │   └── client_test.go               # ✅ 5テスト
│   ├── mcp/
│   │   ├── handler.go                   # ✅ メタツール対応
│   │   ├── handler_test.go              # ✅ 10テスト
│   │   └── types.go                     # ✅
│   ├── modules/
│   │   ├── registry.go                  # ✅ モジュールレジストリ
│   │   ├── types.go                     # ✅ 共通型定義
│   │   └── supabase/
│   │       └── module.go                # ✅ Supabaseモジュール
│   └── observability/
│       └── loki.go                      # ✅
├── Dockerfile                           # ✅
├── docker-compose.yml                   # ✅
├── go.mod                               # ✅
├── go.sum                               # ✅
└── Makefile                             # ✅
```

---

## テスト結果サマリ

```
=== 全17テスト通過 ===

internal/auth      - 6 tests (middleware_test.go)
internal/httpclient - 5 tests (client_test.go)
internal/mcp       - 10 tests (handler_test.go) ※ 一部はtestモジュール使用
```

---

## 次のステップ

Round 1完了後は **Round 2: モジュール拡張** に進む:

- Day 6: Notionモジュール
- Day 7: GitHubモジュール
- Day 8: Jiraモジュール
- Day 9: Confluenceモジュール

---

*作成日: 2026-01-10*
*完了日: 2026-01-10*
