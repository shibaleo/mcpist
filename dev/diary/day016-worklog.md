# DAY016 作業ログ

## 日付

2026-01-27

---

## 作業内容

### D16-001: サービス接続時のデフォルトツール設定自動保存

BL-001の対応。サービス接続成功時にtools.jsonのannotationsを元にデフォルトツール設定をDBに自動保存する機能を実装。

#### MCP Tool Annotations移行

tools.jsonのカスタムフィールド(`defaultEnabled`/`dangerous`)をMCP仕様（2025-11-25）の`annotations`に移行。

| フィールド | 説明 | デフォルト |
|-----------|------|-----------|
| `readOnlyHint` | 環境を変更しない | false |
| `destructiveHint` | 破壊的な更新を行う可能性がある | true |
| `idempotentHint` | 繰り返し呼び出しても追加的影響がない | false |
| `openWorldHint` | 外部エンティティとやり取りする | true |

#### ヘルパー関数

| 関数 | ロジック |
|------|---------|
| `isDefaultEnabled(tool)` | `readOnlyHint === true` のツールをデフォルト有効 |
| `isDangerous(tool)` | `readOnlyHint !== true && destructiveHint !== false` のツールを危険表示 |

#### 自動保存の実装

`saveDefaultToolSettings()` をトークン保存後に呼び出す:
- API Key接続: `token-vault.ts` の `upsertTokenWithVerification()` 内
- OAuth接続: `google/callback` と `microsoft/callback` のRoute Handler内

#### 再接続時のユーザー設定保持

`saveDefaultToolSettings()` に既存設定チェックを追加。設定が存在する場合はスキップし、ユーザーのカスタム設定を保持する。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/lib/tools.json` | 全115ツールを`annotations`形式に変換 |
| `apps/console/src/lib/module-data.ts` | `ToolAnnotations`型、`isDefaultEnabled()`、`isDangerous()` 追加 |
| `apps/console/src/lib/tool-settings.ts` | `saveDefaultToolSettings()` 追加、既存設定スキップロジック |
| `apps/console/src/lib/token-vault.ts` | 接続成功時に`saveDefaultToolSettings()`呼び出し |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | 接続成功時にデフォルト設定保存 |
| `apps/console/src/app/api/oauth/microsoft/callback/route.ts` | 接続成功時にデフォルト設定保存 |
| `apps/console/src/lib/supabase/database.types.ts` | `get_my_tool_settings`、`upsert_my_tool_settings` 型定義追加 |

#### ビルドエラー修正

| エラー | 原因 | 修正 |
|--------|------|------|
| `"get_my_tool_settings"` not assignable | `database.types.ts` に型定義がなかった | 型定義追加 |
| `Property 'error' does not exist on type 'Json'` | `upsert_my_tool_settings` の戻り値が`Json`型 | `Record<string, unknown>` にキャスト |

---

### Console UI改善

#### ツール設定UIの改善

| 改善内容 | 詳細 |
|----------|------|
| MCP Annotationsバッジ | ReadOnly（青）、Destructive（黄）、Idempotent（灰）のバッジ表示 |
| Switch (トグル) | Checkboxを`@radix-ui/react-switch`ベースのSwitchに置換（左側配置） |
| 全解除ボタン | 「デフォルト」「全選択」に加え「全解除」ボタンを追加 |
| 認証ヘルプリンク | シークレット入力ダイアログにサービスごとのトークン取得ページへのリンクを追加 |

#### 認証ヘルプリンク

| サービス | URL |
|----------|-----|
| Notion | https://www.notion.so/profile/integrations |
| GitHub | https://github.com/settings/tokens?type=beta |
| Jira/Confluence | https://id.atlassian.com/manage-profile/security/api-tokens |
| Supabase | https://supabase.com/dashboard/account/tokens |

#### ダッシュボード

「有効なツール」カードにDBから取得した有効ツール数/全ツール数を表示（モック値`-`を置換）。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/(console)/tools/page.tsx` | Switch、全解除、認証リンク、Annotationsバッジ |
| `apps/console/src/components/ui/switch.tsx` | Switchコンポーネント（新規作成） |
| `apps/console/package.json` | `@radix-ui/react-switch` 追加 |
| `apps/console/src/app/(console)/dashboard/page.tsx` | 有効ツール数表示 |

---

### DB検証

Supabase MCP経由で本番DBのツール設定を検証。

| 検証内容 | 結果 |
|----------|------|
| 全8モジュールのツール設定登録 | OK（supabase, airtable, notion, github, jira, confluence, google_calendar, microsoft_todo） |
| Annotations基準のデフォルト値 | OK（readOnlyHint=trueのツールが有効） |
| ユーザーカスタム設定の保存 | OK |
| 切断後のツール設定残存 | 確認（孤立データとして残存） |
| 再接続時の設定保持（スキップロジック） | OK（既存設定が保持された） |

---

### D16-007: get_module_schema 複数モジュール対応 + ツールフィルタリング

plan-meta-tool-refactor.md の全ステップを実装。

#### 実装内容

| 項目 | 詳細 |
|------|------|
| 配列入力対応 | `module` 引数を `string \| string[]` に変更。1回で複数モジュールのスキーマ取得可能 |
| ツールフィルタリング | `DisabledTools` を参照し、無効ツールをスキーマから除外 |
| tools/list 動的化 | `DynamicMetaTools()` で接続済み＆有効ツール>0のモジュールのみ description に含める |
| 後方互換 | 単一文字列 `"notion"` でも `["notion"]` でも動作 |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/types.go` | `Property.Items` フィールド追加 |
| `apps/server/internal/modules/modules.go` | `filterTools()`, `availableModuleNames()`, `DynamicMetaTools()`, `GetModuleSchemas()` 実装 |
| `apps/server/internal/mcp/handler.go` | `handleToolsList(ctx)`, `handleGetModuleSchema(ctx, args)` 配列対応、ctx伝播 |

---

### D16-008: Observability — Loki統合 + X-Request-IDトレーシング

Grafana Lokiへの構造化ログ送信と、Worker → Go Server → Loki → DB のエンドツーエンドリクエストトレーシングを実装。

#### Loki統合

| 関数 | 用途 |
|------|------|
| `LogToolCall()` | ツール実行のログ（module, tool, duration_ms, status, request_id） |
| `LogRequest()` | HTTPリクエストログ（method, path, status_code, duration_ms） |
| `LogError()` | エラーログ（context, error） |
| `LogSecurityEvent()` | セキュリティイベントログ（Layer 3 Detection用。`maybe_attacked: true` ラベル付き） |

#### X-Request-ID トレーシング

| レイヤー | 実装 |
|----------|------|
| Worker | `crypto.randomUUID()` で生成し `X-Request-ID` ヘッダに設定 |
| Go Server middleware | `X-Request-ID` ヘッダから取得（なければ `crypto/rand` で fallback 生成） |
| Go Server handler | `middleware.GetRequestID(ctx)` でコンテキストから取得 |
| Loki | `request_id` フィールドとして送信 |
| DB | `credit_transactions.request_id` に保存（冪等性チェック用） |

#### 検証結果

| テスト | 結果 |
|--------|------|
| `X-Request-ID: trace-verify-001` でcurl → Loki + DB に記録 | OK |
| `X-Request-ID: final-trace-test` でcurl → Loki + DB に記録 | OK |
| Worker UUID v4 生成 → Go Server → DB | OK |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/observability/loki.go` | `LokiClient`, `Push()`, `LogToolCall()`, `LogRequest()`, `LogError()`, `LogSecurityEvent()` |
| `apps/server/internal/middleware/authz.go` | `RequestIDKey` コンテキストキー、`GetRequestID()`, `generateRequestID()` |
| `apps/server/internal/modules/modules.go` | `Run()` 内で `middleware.GetRequestID(ctx)` → `LogToolCall()` |
| `apps/worker/src/index.ts` | `fetchBackend()` に `X-Request-ID: crypto.randomUUID()` ヘッダ追加 |

---

### D16-009: Batch権限チェック + クレジット残高事前検証

handleBatch に All-or-Nothing 権限チェックとクレジット残高の事前検証を実装。

#### 設計: エラーメッセージの情報レベル

| 対象 | 情報レベル | 理由 |
|------|-----------|------|
| MCP Client (LLM) | 曖昧: `"batch rejected: one or more tools are not permitted"` | Layer 1 (Filter) の「見せない」方針と整合 |
| サーバーログ (Loki) | 具体的: `denied_tools=[notion:create_page(TOOL_DISABLED)]` | Layer 3 (Detection) 攻撃検知用 |

#### 実装内容

| 機能 | 詳細 |
|------|------|
| `checkBatchPermissions()` | JSONL全コマンドの module/tool 権限を事前チェック。1つでも拒否→全体拒否 |
| クレジット残高チェック | `TotalCredits() < toolCount` で事前検証。不足時はエラー返却 |
| セキュリティログ | `LogSecurityEvent()` で denied_tools を Loki に送信（`maybe_attacked: true`） |
| run のクレジットチェック | `CanAccessTool(module, tool, 1)` で既に実装済み（変更なし） |

#### 脆弱性修正

batch で `creditCost=0` でツール実行後にクレジット消費していたため、残高不足でも実行できてしまう問題を修正。事前に `TotalCredits() < toolCount` で検証するようにした。

#### 将来拡張コメント

ツールごとにコスト（単価）が変わる場合の拡張ポイントを英語コメントで記載:
- `handleRun`: `"Currently 1 credit per tool. To support per-tool pricing, replace with e.g. modules.CreditCost(moduleName, toolName)."`
- `checkBatchPermissions`: `"Credit balance check: currently 1 credit per tool. To support per-tool pricing, replace toolCount with a sum of per-tool costs..."`

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/mcp/handler.go` | `checkBatchPermissions()` 新規追加、`handleBatch()` に権限チェック挿入、クレジットコメント追加 |

---

### D16-010: Go Server MCP Tool Annotations 実装（plan-tool-annotations Step 5-6）

Go Server 側の `Dangerous bool` を完全削除し、MCP仕様（2025-11-25）準拠の `ToolAnnotations` に移行。

#### 実装内容

| 項目 | 詳細 |
|------|------|
| `ToolAnnotations` 構造体 | `ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`, `OpenWorldHint` (*bool) |
| `boolPtr()` ヘルパー | `*bool` 生成用 |
| プリセット定数 | `AnnotateReadOnly`, `AnnotateCreate`, `AnnotateUpdate`, `AnnotateDelete`, `AnnotateDestructive` |
| 全8モジュール115ツール | 各ツールに適切な annotations を設定 |
| `Dangerous bool` 完全削除 | types.go + 全モジュール + tools-export から削除。grep 0件を確認 |

#### 分類ルール

| 操作パターン | プリセット |
|-------------|-----------|
| list / get / search / query | `AnnotateReadOnly` |
| create / add / append | `AnnotateCreate` |
| update / transition / complete | `AnnotateUpdate` |
| delete | `AnnotateDelete` |
| run_query / apply_migration | `AnnotateDestructive` |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/types.go` | `Dangerous bool` 削除、`ToolAnnotations` 構造体・プリセット定数追加 |
| `apps/server/internal/modules/notion/tools.go` | 14ツールに annotations 設定 |
| `apps/server/internal/modules/github/module.go` | 20ツールに annotations 設定 |
| `apps/server/internal/modules/jira/module.go` | 11ツールに annotations 設定 |
| `apps/server/internal/modules/confluence/module.go` | 12ツールに annotations 設定 |
| `apps/server/internal/modules/supabase/module.go` | 17ツールに annotations 設定 |
| `apps/server/internal/modules/airtable/module.go` | 11ツールに annotations 設定 |
| `apps/server/internal/modules/google_calendar/module.go` | 8ツールに annotations 設定 |
| `apps/server/internal/modules/microsoft_todo/module.go` | 11ツールに annotations 設定 |
| `apps/server/cmd/tools-export/main.go` | `ToolDef` から `Dangerous`/`DefaultEnabled` 削除、`Annotations` 追加 |

---

### D16-003: next.config.ts デバッグログ削除

B-006対応。ビルド時に出力されるデバッグ用 `console.log` を削除。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/next.config.ts` | `console.log('[next.config] NEXT_PUBLIC_SUPABASE_URL:', ...)` 削除 |

---

### D16-004: VitePress docsビルド修正

VitePress config のサイドバーパス（`/specification/`）と実ファイルパス（`docs/002_specification/`）の不一致を修正。

#### 問題

- ファイルは `docs/002_specification/`, `docs/003_design/` 等の番号付きディレクトリに配置
- VitePress config の sidebar は `/specification/` 等の番号なしパスを参照
- サイドバーリンクが全て 404

#### 修正

VitePress `rewrites` で番号付きディレクトリをクリーンURLにマッピング。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `docs/.vitepress/config.ts` | `rewrites` 追加（`002_specification` → `specification` 等、全6セクション） |
| `docs/index.md` | リンクを `/002_specification/` → `/specification/` 等に修正 |

---

### D16-011: Console MCP接続設定から type:sse 削除

MCPエンドポイント設定UIに表示されていた `"type": "sse"` を削除。現在のMCP仕様ではSSEはデフォルトのため不要。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/(console)/mcp/page.tsx` | JSON設定例から `"type": "sse",` 行を削除 |

---

### D16-012: Console /mcp ルートを /connections にリネーム

Consoleページルート `/mcp` を `/connections` にリネーム。`/mcp` はMCPプロトコルエンドポイントとして標準的に使われるパスのため、Consoleページルートとの混同を避ける。

#### 変更内容

| 項目 | 詳細 |
|------|------|
| ページ移動 | `(console)/mcp/page.tsx` → `(console)/connections/page.tsx` |
| アクション移動 | `(console)/mcp/actions.ts` → `(console)/connections/actions.ts` |
| サイドバー更新 | `sidebar.tsx` の href を `/mcp` → `/connections` に変更 |
| 旧ディレクトリ削除 | `(console)/mcp/` ディレクトリを削除 |
| ビルド修正 | `@radix-ui/react-switch` パッケージ追加（ビルドエラー解消） |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/(console)/connections/page.tsx` | 旧 `/mcp/page.tsx` の内容で上書き（リダイレクトスタブを置換） |
| `apps/console/src/app/(console)/connections/actions.ts` | 旧 `/mcp/actions.ts` を移動 |
| `apps/console/src/components/sidebar.tsx` | `href: "/mcp"` → `href: "/connections"` |
| `apps/console/package.json` | `@radix-ui/react-switch` 依存追加 |

---

### D16-013: gRPC移行構想の実装可能性評価

`prop-grpc-specify.md`（gRPC移行構想）の実装可能性を調査・評価し、**REJECTED** 判定を記録。

#### 調査内容

| 項目 | 結果 |
|------|------|
| Google Cloud ブログ参照PRの状況 | Python MCP SDK [PR #1936](https://github.com/modelcontextprotocol/python-sdk/pull/1936) が2026-01-23にreject。「MCP仕様外」が理由 |
| MCP仕様レベルの議論 | #966 → SEP-1352 (Draft段階でClose)、SEP-1319 (JSON-RPCデカップリング) が根本的ブロッカー |
| 通信パターン分析 | MCPはLSP由来の双方向メッセージングプロトコル。gRPCの一方向RPCモデルと根本的に不適合 (Adrian Cole の指摘) |
| MCPist への適用判断 | 時期尚早。MCPトランスポートには不向き。内部マイクロサービス通信としては将来検討の余地あり |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `dev/workdir/prop-grpc-specify.md` | REJECTED ラベル追加、調査結果セクション追加（PR状況、仕様議論、通信パターン分析、推奨アクション） |

---

## コミット履歴

| コミット | メッセージ |
|----------|-----------|
| `d4ea428` | feat(server): support array input and tool filtering in get_module_schema |
| `79807cd` | feat(console): show enabled tool count on dashboard and fix switch position |
| `ddbdeb5` | feat(console): add switch toggle, deselect-all button, and auth help links |
| `a7fc850` | fix(console): skip default tool settings if user settings already exist |
| `645b79d` | feat(console): adopt MCP tool annotations and auto-save default tool settings |

---

## タスク進捗

| ID | タスク | 状態 |
|----|--------|------|
| D16-001 | サービス接続時のデフォルトツール設定自動保存 | **完了** |
| D16-002 | 仕様書の実装追従更新 | 未着手 |
| D16-003 | next.config.ts デバッグログ削除 | **完了** |
| D16-004 | VitePress docsビルド修正 | **完了** |
| D16-005 | Phase 4: UI要件定義 | 未着手 |
| D16-006 | E2Eテスト設計 | 未着手 |
| D16-007 | get_module_schema 複数モジュール対応 + ツールフィルタリング | **完了** |
| D16-008 | Observability — Loki統合 + X-Request-IDトレーシング | **完了** |
| D16-009 | Batch権限チェック + クレジット残高事前検証 | **完了** |
| D16-010 | Go Server MCP Tool Annotations 実装 | **完了** |
| D16-011 | Console MCP接続設定から type:sse 削除 | **完了** |
| D16-012 | Console /mcp → /connections リネーム | **完了** |
| D16-013 | gRPC移行構想の実装可能性評価 | **完了** |

---

## バックログ更新

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| BL-001 | サービス接続時にデフォルトツール設定を自動保存 | **完了** | D16-001で実装。MCP Annotations移行も含む |
| BL-002 | 切断時のツール設定クリーンアップ | 保留 | DB migration必要。ユーザー設定復元のためのデータ構造変更が前提 |
| BL-025 | Go Server annotations 実装（plan-tool-annotations Step 5-6） | **完了** | D16-010で実装。Dangerous bool 完全削除、全115ツールに annotations 設定 |
| BL-026 | ツール定義マスタDB管理（plan-tool-defaults-master） | **廃止** | annotations からの導出方式に決定。DB管理・管理画面は不要 |
