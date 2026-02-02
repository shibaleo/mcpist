# DAY022 計画

## 日付

2026-02-02

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

### Phase 3: Google Docs モジュール実装（優先度：中）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-011 | `modules/google_docs/module.go` 作成 | 既存 Google OAuth 基盤流用 | |
| D22-012 | Google Cloud Console でスコープ追加 | documents | |
| D22-013 | `main.go` に RegisterModule 追加 | | |
| D22-014 | 動作確認 | | |

**ツール一覧:**
- list_documents, get_document, create_document, batch_update

### Phase 4: Asana モジュール実装（時間があれば）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-015 | `modules/asana/module.go` 作成 | OAuth 2.0（リフレッシュトークンあり） | |
| D22-016 | Asana Developer Console でアプリ登録 | | |
| D22-017 | OAuth アプリ設定を Supabase に登録 | asana | |
| D22-018 | `main.go` に RegisterModule 追加 | | |
| D22-019 | Console に Asana OAuth 連携 UI 追加 | | |
| D22-020 | 動作確認 | | |

### Phase 5: PostgreSQL モジュール実装（時間があれば）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D22-021 | `modules/postgresql/module.go` 作成 | Connection String 方式 | |
| D22-022 | `pgx` ドライバー追加 | | |
| D22-023 | `main.go` に RegisterModule 追加 | | |
| D22-024 | Console に接続文字列入力 UI 追加 | | |
| D22-025 | 動作確認 | | |

**ツール一覧:**
- query (SELECT のみ), list_tables, describe_table, execute (INSERT/UPDATE/DELETE)

**セキュリティ考慮:**
- SQL インジェクション対策（プリペアドステートメント必須）
- 接続制限（1接続/リクエスト、タイムアウト）
- 行数制限（max_rows）
- 危険操作禁止（DROP/TRUNCATE/ALTER）

---

## 完了条件

- [x] Todoist モジュールが動作（list_tasks, create_task）
- [x] Trello モジュールが動作（OAuth 1.0a、全17ツール動作確認済み）
- [ ] （stretch）Google Docs モジュールが動作
- [ ] （stretch）Asana モジュールが動作
- [ ] （stretch）PostgreSQL モジュールが動作

---

## タイムライン

| 時間帯 | タスク |
|--------|--------|
| 午前 | Phase 1: Todoist (D22-001〜005) |
| 午後前半 | Phase 2: Trello (D22-006〜010) |
| 午後後半 | Phase 3: Google Docs or Phase 4: Asana |

---

## 参考

- [day021-impl-modules.md](./day021-impl-modules.md) - モジュール実装計画
- [backlog-open-tasks.md](./backlog-open-tasks.md) - バックログ
- [day021-worklog.md](./day021-worklog.md) - 前日ログ
- [Todoist REST API](https://developer.todoist.com/rest/v2/)
- [Trello REST API](https://developer.atlassian.com/cloud/trello/rest/)
- [Google Docs API](https://developers.google.com/docs/api/reference/rest)
- [Asana API](https://developers.asana.com/docs)
