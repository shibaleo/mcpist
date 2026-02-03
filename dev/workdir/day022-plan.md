# DAY022 計画

## 日付

2026-02-02～2026-02-03

---

## 概要

Sprint-006 6日目。DAY021ではGoogle Tasks モジュール、prompts MCP実装、Console テーマ改善が完了。本日は新規モジュール実装（Todoist, Google Docs, Trello, Asana, PostgreSQL）を進める。

---

## DAY021 の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| Google Tasks モジュール実装 | 9ツール、OAuth共有コールバック方式 |
| Microsoft To Do モジュール実装 | mcpist-dev で実装済み |
| prompts MCP 実装 | list/get、description/content 分離 |
| Console プロンプト管理 UI | description フィールド追加、楽観的更新 |
| Console ツール設定改善 | 楽観的更新パターン適用 |
| Console テーマ改善 | Liam ERD風ダークテーマ、アクセントカラー調整 |
| /services ページ分離 | ツール設定から接続管理を分離 |
| PKCE認証エラー修正 | skipBrowserRedirect で確実にクッキー設定 |
| 背景色プリセット機能削除 | slate/zinc/custom を削除 |

---

## 本日のタスク

### Phase 1: Todoist モジュール実装（優先度：高） ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-001 | `modules/todoist/module.go` 作成 | OAuth 2.0（リフレッシュトークンなし） | ✅ |
| D22-002 | OAuth アプリ設定を Supabase に登録 | todoist | ✅ |
| D22-003 | `main.go` に RegisterModule 追加 | | ✅ |
| D22-004 | Console に Todoist OAuth 連携 UI 追加 | | ✅ |
| D22-005 | 動作確認 | | ✅ |

**ツール一覧:**
- list_projects, get_project
- list_tasks, get_task, create_task, update_task, complete_task, delete_task

### Phase 2: Trello モジュール実装（優先度：高） ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-006 | `modules/trello/module.go` 作成 | **OAuth 1.0a 方式**（当初 API Key + Token を想定） | ✅ |
| D22-007 | Trello Power-Up / API Key 取得 | | ✅ |
| D22-008 | `main.go` に RegisterModule 追加 | | ✅ |
| D22-009 | Console に Trello OAuth 連携 UI 追加 | OAuth 1.0a フロー実装 | ✅ |
| D22-010 | 動作確認 | 全17ツール動作確認完了 | ✅ |

**注記:** Trello は OAuth 2.0 ではなく **OAuth 1.0a** を使用。HMAC-SHA1 署名生成、3-legged フロー（Request Token → Authorize → Access Token）を実装。

**ツール一覧（17ツール）:**
- list_boards, get_board, get_lists
- get_cards, get_card, create_card, update_card, move_card, delete_card
- get_checklists, create_checklist, delete_checklist
- get_checklist_items, add_checklist_item, update_checklist_item, delete_checklist_item

### Phase 3: GitHub OAuth 実装 ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-011 | GitHub OAuth authorize/callback ルート作成 | OAuth 2.0 | ✅ |
| D22-012 | oauth-apps.ts に GitHub 追加 | OAUTH_PROVIDERS, OAUTH_CONFIGS | ✅ |
| D22-013 | services/page.tsx に alternativeAuth パターン追加 | OAuth + Fine-grained PAT | ✅ |
| D22-014 | 動作確認 | get_user, list_repos 動作確認 | ✅ |

**注記:** GitHub は OAuth 2.0 を使用。トークンは無期限（revoke されるまで有効）、リフレッシュトークンなし。

**ツール一覧（20ツール）:**
- get_user, list_repos, get_repo, list_branches, list_commits, get_file_content
- list_issues, get_issue, create_issue, update_issue, add_issue_comment
- list_prs, get_pr, create_pr, list_pr_files
- search_repos, search_code, search_issues
- list_workflows, list_workflow_runs

### Phase 4: Google Docs モジュール実装 ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-015 | `modules/google_docs/module.go` 作成 | 既存 Google OAuth 基盤流用 | ✅ |
| D22-016 | Google Cloud Console でスコープ追加 | documents | ✅ |
| D22-017 | `main.go` に RegisterModule 追加 | | ✅ |
| D22-018 | 動作確認 | | ✅ |

**ツール一覧:**
- list_documents, get_document, create_document, batch_update

### Phase 4.5: Google Drive モジュール実装 ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-040 | `modules/google_drive/module.go` 作成 | 22ツール | ✅ |
| D22-041 | 動作確認 | ファイル一覧、検索、削除等 | ✅ |

### Phase 4.6: Google Sheets モジュール テスト ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-050 | google_sheets 全28ツールテスト | 動作確認完了 | ✅ |

**テスト済みツール:**
- スプレッドシート: create_spreadsheet, get_spreadsheet, search_spreadsheets
- シート操作: list_sheets, create_sheet, rename_sheet, duplicate_sheet, copy_sheet_to, delete_sheet
- 値の読み書き: get_values, batch_get_values, get_formulas, update_values, batch_update_values, append_values, clear_values
- 行・列操作: insert_rows, delete_rows, insert_columns, delete_columns
- 書式設定: format_cells, merge_cells, unmerge_cells, set_borders, auto_resize
- その他: find_replace, protect_range

**備考:** スプレッドシート削除は `google_drive:delete_file` を使用

### Phase 5: Asana モジュール実装 ✅

| ID      | タスク                            | 備考                                 | 状態  |
| ------- | ------------------------------ | ---------------------------------- | --- |
| D22-030 | `modules/asana/module.go` 作成   | OAuth 2.0（リフレッシュトークンあり）、12ツール      | ✅   |
| D22-031 | Asana Developer Console でアプリ登録 | Identity/OpenID Connect スコープ無効化必須  | ✅   |
| D22-032 | OAuth アプリ設定を Supabase に登録      | asana                              | ✅   |
| D22-033 | `main.go` に RegisterModule 追加  | server, tools-export 両方            | ✅   |
| D22-034 | Console に Asana OAuth 連携 UI 追加 | authorize/callback ルート             | ✅   |
| D22-035 | expires_at 型互換性修正              | FlexibleTime 型導入（ISO文字列 + Unix両対応） | ✅   |
| D22-036 | 動作確認                           | get_me, list_projects, list_tasks  | ✅   |

**注記:**
- Asana は OAuth 2.0 を使用。トークンは約1時間で期限切れ、リフレッシュトークンあり。
- Developer Console で Identity/OpenID Connect スコープを有効にすると `forbidden_scopes` エラー。
- `expires_at` を ISO 文字列で保存する Console と Unix タイムスタンプを期待する Go の互換性のため `FlexibleTime` 型を導入。

**ツール一覧（12ツール・読み取り専用）:**
- get_me, list_workspaces, get_workspace
- list_projects, get_project, list_sections
- list_tasks, get_task, list_subtasks, list_stories
- list_tags, search_tasks

### Phase 6: Google Apps Script モジュール実装 ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-060 | コミュニティ実装調査 | mohalmah, whichguy/gas_mcp 調査 | ✅ |
| D22-061 | ツール設計 | 17ツール（トリガー管理除外） | ✅ |
| D22-062 | `modules/google_apps_script/module.go` 作成 | 943行、全17ツール | ✅ |
| D22-063 | OAuth スコープ設定追加 | oauth-apps.ts に google-apps-script 追加 | ✅ |
| D22-064 | `main.go` に RegisterModule 追加 | server, tools-export 両方 | ✅ |
| D22-065 | Console UI に authConfig 追加 | services/page.tsx、module-data.ts | ✅ |
| D22-066 | tools.json 再生成 | Google Apps Script 17ツール追加 | ✅ |

**注記:**
- whichguy/gas_mcp の 50 ツールのうち、多くはローカルファイルシステム/Git/clasp 操作。mcpist のステートレスアーキテクチャでは実装不可。
- トリガー管理は公式 API が存在しないため除外。
- copy/delete は `google_drive` モジュールで代替可能（GAS プロジェクトは Drive ファイル）。

**ツール一覧（17ツール）:**
- プロジェクト: list_projects, get_project, create_project, get_content, update_content
- バージョン: list_versions, get_version, create_version
- デプロイメント: list_deployments, get_deployment, create_deployment, update_deployment, delete_deployment
- 実行: run_function, list_executions
- モニタリング: list_processes, get_metrics

### Phase 7: PostgreSQL モジュール実装 ✅

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-021 | `modules/postgresql/module.go` 作成 | Connection String 方式、7ツール | ✅ |
| D22-022 | `pgx` ドライバー追加 | pgx v5.7.2 | ✅ |
| D22-023 | `main.go` に RegisterModule 追加 | server, tools-export 両方 | ✅ |
| D22-024 | Console に接続文字列入力 UI 追加 | authConfig、icon追加 | ✅ |
| D22-025 | 動作確認 | 全7ツール動作確認済み | ✅ |
| D22-026 | UUID変換修正 | `[16]byte` → 文字列形式 | ✅ |

**ツール一覧（7ツール）:**
- test_connection（接続テスト）
- list_schemas（スキーマ一覧）
- list_tables（テーブル一覧）
- describe_table（テーブル定義）
- query（SELECT実行、max_rows制限付き）
- execute（INSERT/UPDATE/DELETE）
- execute_ddl（CREATE/ALTER/DROP）

**セキュリティ対策:**
- localhost/127.0.0.1/::1 接続禁止（SSRF対策）
- `sslmode=require` デフォルト
- SQL種別バリデーション（query→SELECTのみ、execute→DMLのみ、execute_ddl→DDLのみ）
- 行数制限（デフォルト1000、最大10000）
- タイムアウト設定（接続10秒、クエリ30秒）

**解決した問題:**
- IPv6接続エラー: Render は IPv6 未対応。Supabase Session pooler (port 6543) に切り替えて解決
- UUID表示問題: pgx は UUID を `[16]byte` で返す。`convertValue` 関数で文字列形式に変換

---

## 完了条件

- [x] Todoist モジュールが動作（list_tasks, create_task）
- [x] Trello モジュールが動作（OAuth 1.0a、全17ツール動作確認済み）
- [x] GitHub OAuth 実装（alternativeAuth パターン、20ツール）
- [x] Asana モジュールが動作（OAuth 2.0 + リフレッシュ、12ツール読み取り専用）
- [x] FlexibleTime 型導入（expires_at の ISO文字列/Unix両対応）
- [x] （stretch）Google Docs モジュールが動作
- [x] （stretch）Google Drive モジュールが動作（22ツール）
- [x] （stretch）Google Sheets モジュール全28ツール動作確認
- [x] （stretch）Google Apps Script モジュール実装（17ツール）
- [x] （stretch）PostgreSQL モジュールが動作（7ツール、UUID変換対応）

---

## タイムライン

| 時間帯 | タスク |
|--------|--------|
| 午前 | Phase 1: Todoist (D22-001〜005) |
| 午後前半 | Phase 2: Trello (D22-006〜010) |
| 午後後半 | Phase 3: Google Docs or Phase 4: Asana |

---

## 参考

- [day021-impl-modules.md](day021-impl-modules.md) - モジュール実装計画
- [backlog-open-tasks.md](day022-backlog-open-tasks.md) - バックログ
- [day021-worklog.md](day021-worklog.md) - 前日ログ
- [Todoist REST API](https://developer.todoist.com/rest/v2/)
- [Trello REST API](https://developer.atlassian.com/cloud/trello/rest/)
- [Google Docs API](https://developers.google.com/docs/api/reference/rest)
- [Asana API](https://developers.asana.com/docs)
