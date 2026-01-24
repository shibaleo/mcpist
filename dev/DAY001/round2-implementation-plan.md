# Round 2: モジュール拡張 - 詳細実装計画

**期間**: 4日（Day 6-9）
**目的**: 検証済み基盤の上でモジュールを横展開
**状態**: ✅ 完了

---

## 追加完了: Supabaseモジュール拡張（17ツール）

Round 2の一環として、Supabaseモジュールを2ツールから17ツールに拡張。

### Supabaseツール一覧

**Account**:
| ツール名 | 説明 |
|---------|------|
| `supabase_list_organizations` | 組織一覧取得 |
| `supabase_list_projects` | プロジェクト一覧 |
| `supabase_get_project` | プロジェクト詳細 |

**Database**:
| ツール名 | 説明 |
|---------|------|
| `supabase_list_tables` | テーブル一覧取得 |
| `supabase_run_query` | SQL実行 |
| `supabase_list_migrations` | マイグレーション履歴取得 |
| `supabase_apply_migration` | マイグレーション適用 |

**Debugging**:
| ツール名 | 説明 |
|---------|------|
| `supabase_get_logs` | ログ取得 |
| `supabase_get_security_advisors` | セキュリティ推奨事項取得 |
| `supabase_get_performance_advisors` | パフォーマンス推奨事項取得 |

**Development**:
| ツール名 | 説明 |
|---------|------|
| `supabase_get_project_url` | プロジェクトURL取得 |
| `supabase_get_api_keys` | APIキー取得 |
| `supabase_generate_typescript_types` | TypeScript型定義生成 |

**Edge Functions**:
| ツール名 | 説明 |
|---------|------|
| `supabase_list_edge_functions` | Edge Functions一覧取得 |
| `supabase_get_edge_function` | Edge Function詳細取得 |

**Storage**:
| ツール名 | 説明 |
|---------|------|
| `supabase_list_storage_buckets` | ストレージバケット一覧取得 |
| `supabase_get_storage_config` | ストレージ設定取得 |

---

## 参考実装

TypeScript実装（`C:\Users\shiba\HOBBY\dwhbi\packages\console\src\app\api\mcp\modules`）を参照。
認証はVault経由だが、go-mcp-devでは環境変数の固定トークンを使用。

---

## 共通パターン

各モジュールは以下の構造で実装:

```
internal/modules/{module}/
└── module.go    # ModuleDefinition を返す Module() 関数
```

**実装パターン**:
```go
package {module}

import (
    "github.com/shibaleo/go-mcp-dev/internal/httpclient"
    "github.com/shibaleo/go-mcp-dev/internal/modules"
)

func Module() modules.ModuleDefinition {
    return modules.ModuleDefinition{
        Name:        "{module}",
        Description: "...",
        Tools:       tools,
        Handlers:    handlers,
    }
}

var tools = []modules.Tool{...}
var handlers = map[string]modules.ToolHandler{...}
```

---

## Day 6: Notionモジュール

### 概要

| 項目 | 内容 |
|------|------|
| API | Notion API v1 |
| 認証 | `NOTION_TOKEN` (Integration Token) |
| ベースURL | `https://api.notion.com/v1` |
| ヘッダー | `Authorization: Bearer {token}`, `Notion-Version: 2022-06-28` |

### ツール一覧（14ツール）

| ツール名 | 説明 | 必須パラメータ |
|---------|------|---------------|
| `notion_search` | ページ・データベース検索 | - |
| `notion_get_page` | ページ取得 | `page_id` |
| `notion_get_page_content` | ページ内容（ブロック）取得 | `page_id` |
| `notion_create_page` | ページ作成 | `title`, `parent_page_id` or `parent_database_id` |
| `notion_update_page` | ページ更新 | `page_id`, `properties` |
| `notion_get_database` | データベース取得 | `database_id` |
| `notion_query_database` | データベースクエリ | `database_id` |
| `notion_append_blocks` | ブロック追加 | `block_id`, `blocks` |
| `notion_delete_block` | ブロック削除 | `block_id` |
| `notion_list_comments` | コメント一覧 | `block_id` |
| `notion_add_comment` | コメント追加 | `page_id`, `content` |
| `notion_list_users` | ユーザー一覧 | - |
| `notion_get_user` | ユーザー取得 | `user_id` |
| `notion_get_bot_user` | Bot情報取得 | - |

### タスク

- [x] `internal/modules/notion/module.go` 作成
- [x] ツール定義（14ツール）
- [x] ハンドラー実装
- [x] main.goにモジュール登録
- [ ] デプロイ + 実運用検証

### API エンドポイント

```
POST /search                    # notion_search
GET  /pages/{page_id}           # notion_get_page
GET  /blocks/{block_id}/children # notion_get_page_content
POST /pages                     # notion_create_page
PATCH /pages/{page_id}          # notion_update_page
GET  /databases/{database_id}   # notion_get_database
POST /databases/{database_id}/query # notion_query_database
PATCH /blocks/{block_id}/children # notion_append_blocks
DELETE /blocks/{block_id}       # notion_delete_block
GET  /comments?block_id=...     # notion_list_comments
POST /comments                  # notion_add_comment
GET  /users                     # notion_list_users
GET  /users/{user_id}           # notion_get_user
GET  /users/me                  # notion_get_bot_user
```

---

## Day 7: GitHubモジュール

### 概要

| 項目 | 内容 |
|------|------|
| API | GitHub REST API v3 |
| 認証 | `GITHUB_TOKEN` (PAT) |
| ベースURL | `https://api.github.com` |
| ヘッダー | `Authorization: Bearer {token}`, `X-GitHub-Api-Version: 2022-11-28` |

### ツール一覧（22ツール）

**User**:
| ツール名 | 説明 |
|---------|------|
| `github_get_user` | 認証ユーザー情報取得 |

**Repositories**:
| ツール名 | 説明 |
|---------|------|
| `github_list_repos` | リポジトリ一覧 |
| `github_get_repo` | リポジトリ詳細 |
| `github_list_branches` | ブランチ一覧 |
| `github_list_commits` | コミット一覧 |
| `github_get_file_content` | ファイル内容取得 |

**Issues**:
| ツール名 | 説明 |
|---------|------|
| `github_list_issues` | Issue一覧 |
| `github_get_issue` | Issue詳細 |
| `github_create_issue` | Issue作成 |
| `github_update_issue` | Issue更新 |
| `github_add_issue_comment` | Issueコメント追加 |

**Pull Requests**:
| ツール名 | 説明 |
|---------|------|
| `github_list_prs` | PR一覧 |
| `github_get_pr` | PR詳細 |
| `github_create_pr` | PR作成 |
| `github_list_pr_commits` | PRコミット一覧 |
| `github_list_pr_files` | PR変更ファイル一覧 |
| `github_list_pr_reviews` | PRレビュー一覧 |

**Search**:
| ツール名 | 説明 |
|---------|------|
| `github_search_repos` | リポジトリ検索 |
| `github_search_code` | コード検索 |
| `github_search_issues` | Issue/PR検索 |

**Actions**:
| ツール名 | 説明 |
|---------|------|
| `github_list_workflows` | ワークフロー一覧 |
| `github_list_workflow_runs` | ワークフロー実行一覧 |
| `github_get_workflow_run` | ワークフロー実行詳細 |

### タスク

- [x] `internal/modules/github/module.go` 作成
- [x] ツール定義（22ツール）
- [x] ハンドラー実装
- [x] main.goにモジュール登録
- [ ] デプロイ + 実運用検証

---

## Day 8: Jiraモジュール

### 概要

| 項目 | 内容 |
|------|------|
| API | Jira Cloud REST API v3 |
| 認証 | Basic認証 (`JIRA_EMAIL:JIRA_API_TOKEN`) |
| ベースURL | `https://{domain}.atlassian.net/rest/api/3` |
| 環境変数 | `JIRA_DOMAIN`, `JIRA_EMAIL`, `JIRA_API_TOKEN` |

### ツール一覧（13ツール）

| ツール名 | 説明 | 必須パラメータ |
|---------|------|---------------|
| `jira_get_myself` | 現在のユーザー情報 | - |
| `jira_list_projects` | プロジェクト一覧 | - |
| `jira_get_project` | プロジェクト詳細 | `projectKey` |
| `jira_search` | JQL検索 | `jql` |
| `jira_get_issue` | Issue詳細 | `issueKey` |
| `jira_create_issue` | Issue作成 | `projectKey`, `issueType`, `summary` |
| `jira_update_issue` | Issue更新 | `issueKey` |
| `jira_get_transitions` | 遷移一覧取得 | `issueKey` |
| `jira_transition_issue` | ステータス遷移 | `issueKey`, `transitionId` |
| `jira_get_comments` | コメント一覧 | `issueKey` |
| `jira_add_comment` | コメント追加 | `issueKey`, `body` |
| `jira_get_worklogs` | 作業ログ一覧 | `issueKey` |
| `jira_add_worklog` | 作業ログ追加 | `issueKey`, `timeSpentSeconds` |

### タスク

- [x] `internal/modules/jira/module.go` 作成
- [x] ツール定義（13ツール）
- [x] ハンドラー実装
- [x] main.goにモジュール登録
- [ ] デプロイ + 実運用検証

### API エンドポイント

```
GET  /myself                           # jira_get_myself
GET  /project/search                   # jira_list_projects
GET  /project/{projectKey}             # jira_get_project
POST /search                           # jira_search (JQL)
GET  /issue/{issueKey}                 # jira_get_issue
POST /issue                            # jira_create_issue
PUT  /issue/{issueKey}                 # jira_update_issue
GET  /issue/{issueKey}/transitions     # jira_get_transitions
POST /issue/{issueKey}/transitions     # jira_transition_issue
GET  /issue/{issueKey}/comment         # jira_get_comments
POST /issue/{issueKey}/comment         # jira_add_comment
GET  /issue/{issueKey}/worklog         # jira_get_worklogs
POST /issue/{issueKey}/worklog         # jira_add_worklog
```

---

## Day 9: Confluenceモジュール

### 概要

| 項目 | 内容 |
|------|------|
| API | Confluence Cloud REST API v2 |
| 認証 | Basic認証 (`CONFLUENCE_EMAIL:CONFLUENCE_API_TOKEN`) |
| ベースURL | `https://{domain}.atlassian.net/wiki/api/v2` |
| 環境変数 | `CONFLUENCE_DOMAIN`, `CONFLUENCE_EMAIL`, `CONFLUENCE_API_TOKEN` |

### ツール一覧（12ツール）

| ツール名 | 説明 | 必須パラメータ |
|---------|------|---------------|
| `confluence_list_spaces` | スペース一覧 | - |
| `confluence_get_space` | スペース詳細 | `spaceIdOrKey` |
| `confluence_get_pages` | ページ一覧 | `spaceId` |
| `confluence_get_page` | ページ取得 | `pageId` |
| `confluence_create_page` | ページ作成 | `spaceId`, `title`, `body` |
| `confluence_update_page` | ページ更新 | `pageId`, `title`, `body`, `version` |
| `confluence_delete_page` | ページ削除 | `pageId` |
| `confluence_search` | CQL検索 | `cql` |
| `confluence_get_page_comments` | コメント一覧 | `pageId` |
| `confluence_add_page_comment` | コメント追加 | `pageId`, `body` |
| `confluence_get_page_labels` | ラベル一覧 | `pageId` |
| `confluence_add_page_label` | ラベル追加 | `pageId`, `label` |

### タスク

- [x] `internal/modules/confluence/module.go` 作成
- [x] ツール定義（12ツール）
- [x] ハンドラー実装
- [x] main.goにモジュール登録
- [ ] デプロイ + 実運用検証

### API エンドポイント

```
GET  /spaces                           # confluence_list_spaces
GET  /spaces/{id}                      # confluence_get_space (by ID)
GET  /spaces?keys={key}                # confluence_get_space (by key)
GET  /spaces/{id}/pages                # confluence_get_pages
GET  /pages/{id}                       # confluence_get_page
POST /pages                            # confluence_create_page
PUT  /pages/{id}                       # confluence_update_page
DELETE /pages/{id}                     # confluence_delete_page
GET  /wiki/rest/api/search?cql=...     # confluence_search (v1 API)
GET  /pages/{id}/footer-comments       # confluence_get_page_comments
POST /pages/{id}/footer-comments       # confluence_add_page_comment
GET  /pages/{id}/labels                # confluence_get_page_labels
POST /pages/{id}/labels                # confluence_add_page_label
```

---

## 環境変数追加

```bash
# .env.example に追加
# Notion
NOTION_TOKEN=

# GitHub
GITHUB_TOKEN=

# Jira
JIRA_DOMAIN=
JIRA_EMAIL=
JIRA_API_TOKEN=

# Confluence
CONFLUENCE_DOMAIN=
CONFLUENCE_EMAIL=
CONFLUENCE_API_TOKEN=
```

---

## 成功条件

### Day 6 完了条件
- [x] Notionモジュール（14ツール）が動作する
- [ ] Claude Codeからnotion_searchが呼び出せる

### Day 7 完了条件
- [x] GitHubモジュール（22ツール）が動作する
- [ ] Claude Codeからgithub_list_reposが呼び出せる

### Day 8 完了条件
- [x] Jiraモジュール（13ツール）が動作する
- [ ] Claude Codeからjira_searchが呼び出せる

### Day 9 完了条件
- [x] Confluenceモジュール（12ツール）が動作する
- [ ] Claude Codeからconfluence_searchが呼び出せる

---

## Round 2 完了後のファイル構成

```
go-mcp-dev/
├── internal/
│   └── modules/
│       ├── registry.go
│       ├── types.go
│       ├── supabase/
│       │   └── module.go           # 既存
│       ├── notion/
│       │   └── module.go           # NEW (14ツール)
│       ├── github/
│       │   └── module.go           # NEW (22ツール)
│       ├── jira/
│       │   └── module.go           # NEW (13ツール)
│       └── confluence/
│           └── module.go           # NEW (12ツール)
├── cmd/
│   └── server/
│       └── main.go                 # モジュール登録追加
└── .env.example                    # 環境変数追加
```

---

## リスクと対策

### 256MB メモリ制限
- **リスク**: 全モジュール読み込みでメモリ不足
- **対策**: メタツールで遅延読み込み済み。スキーマはオンデマンド

### レート制限
- **リスク**: GitHub/Atlassian APIのレート制限
- **対策**: レスポンスヘッダーでレート制限を監視、Lokiにログ

### 認証情報不足
- **リスク**: ユーザーが環境変数を設定していない
- **対策**: get_module_schemaで認証不足をエラー返却

---

## 優先度

1. **高**: GitHub（よく使う）
2. **高**: Jira（業務で必須）
3. **中**: Notion（ドキュメント管理）
4. **中**: Confluence（Jiraと同時利用が多い）

※ 実装順序はDay順（Notion → GitHub → Jira → Confluence）だが、
   個人の利用頻度によって順番入れ替え可能

---

*作成日: 2026-01-10*
*最終更新: 2026-01-10 - 全モジュール実装完了（84ツール）、ローカルDockerテスト完了*

---

## 実装済みモジュール一覧

| モジュール | ツール数 | 状態 |
|-----------|---------|------|
| Supabase | 18 | ✅ |
| Notion | 15 | ✅ |
| GitHub | 24 | ✅ |
| Jira | 14 | ✅ |
| Confluence | 13 | ✅ |
| **合計** | **84** | ✅ |

### ローカルDockerテスト結果（2026-01-10）

| モジュール | テスト内容 | 結果 |
|-----------|-----------|------|
| Supabase | run_query, list_projects | ✅ |
| Notion | get_bot_user | ✅ |
| GitHub | get_user | ✅ |
| Jira | get_myself | ✅ |
| Confluence | list_spaces | ✅ |

### 次のステップ

- [ ] デプロイ（Koyeb）
- [ ] 実運用検証（Claude Codeから各モジュールのツール呼び出し）
- [ ] 256MBメモリで全モジュール載るか検証
- [ ] 各APIのレート制限の実態調査
