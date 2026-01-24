---
title: MCPist 設計仕様書（spec-dsn）
aliases:
  - spec-dsn
  - MCPist-design-specification
tags:
  - MCPist
  - specification
  - design
document-type:
  - specification
document-class: specification
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist 設計仕様書（spec-dsn）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v3.0 (DAY5) |
| Base | DAY4からコピー |
| Note | DAY5で詳細化予定 |

---

本ドキュメントは、MCPistの詳細設計を定義する。

---

## 1. API設計

### 1.1 MCPサーバーエンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/health` | ヘルスチェック |
| POST | `/mcp` | MCP Protocol (JSON-RPC 2.0 over SSE) |

### 1.2 MCPプロトコル

**対応プロトコルバージョン:** 2025-11-25

#### 1.2.1 サポートメソッド

| メソッド                 | 説明                      | Phase |
| -------------------- | ----------------------- | ----- |
| `initialize`         | 接続初期化、サーバー情報返却          | 1     |
| `tools/list`         | メタツール一覧を返却              | 1     |
| `tools/call`         | メタツール実行                 | 1     |
| `tasks/get`          | タスク状態取得                 | 1     |
| `tasks/result`       | タスク結果取得（ブロッキング）         | 1     |
| `tasks/list`         | タスク一覧取得                 | 1     |
| `tasks/cancel`       | タスクキャンセル                | 1     |
| `elicitation/create` | URL Elicitation（認可フロー等） | 1     |

#### 1.2.2 initialize レスポンス

```json
{
  "protocolVersion": "2025-11-25",
  "capabilities": {
    "tools": {
      "listChanged": false
    },
    "tasks": {
      "list": {},
      "cancel": {},
      "requests": {
        "tools": { "call": {} }
      }
    }
  },
  "serverInfo": { "name": "mcpist", "version": "1.0.0" }
}
```

#### 1.2.3 tools/list レスポンス

3つのメタツール:
- `get_module_schema`: モジュールのツール定義を取得（複数モジュール同時取得可能、1セッション1回推奨）
- `call`: モジュールのツール実行（Tasks対応可）
- `batch`: 複数ツールを一括実行（JSONL形式、依存関係指定可能）

#### 1.2.4 Tasks（非同期タスク管理）

長時間実行される処理を非同期で管理する MCP 標準機能。

**タスク作成（`_meta.taskCreation` で指定）:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "call",
    "arguments": { "module": "supabase", "tool_name": "execute_sql", "params": {...} },
    "_meta": { "taskCreation": { "ttl": 60000 } }
  }
}
```

**レスポンス:**
```json
{
  "result": {
    "_meta": { "task": { "id": "786512e2-...", "status": "working" } }
  }
}
```

**タスク操作:**
- `tasks/get` - 状態取得
- `tasks/result` - 結果取得（ブロッキング）
- `tasks/cancel` - キャンセル

#### 1.2.5 URL Elicitation（認可フロー）

サーバーがクライアントにURL遷移を要求し、ユーザーに外部認可（OAuth等）を促す MCP 標準機能。

**用途:**
- 外部サービスへのOAuth認可（Token Broker未設定時）
- 支払い処理
- サードパーティ認証

**リクエスト:**
```json
{
  "method": "elicitation/create",
  "params": {
    "mode": "url",
    "elicitationId": "550e8400-...",
    "url": "https://mcpist.example.com/oauth/notion/authorize",
    "message": "Notion APIへのアクセス権限が必要です。"
  }
}
```

**レスポンス:**
```json
{
  "result": {
    "action": "accept"
  }
}
```

`action: "accept"` はユーザーの同意を示す。実際の認可完了は out-of-band（URL遷移先）で行われる。

**完了通知:**
```json
{
  "method": "notifications/elicitation/complete",
  "params": {
    "elicitationId": "550e8400-..."
  }
}
```

**セキュリティ要件:**
- URLにユーザー資格情報を含めない
- 事前認証済みURLを提供しない
- HTTPSを使用（開発環境除く）
- elicitationIdとユーザーIDを紐付けて検証

---

## 2. メタツール詳細設計

### 2.1 get_module_schema

**入力:** `{"modules": ["notion", "github"]}` （配列で複数指定可能）

**出力:** モジュールごとのスキーマ配列。各要素は: モジュール名、description、apiVersion、tools配列（name/description/inputSchema/outputSchema/dangerousフラグ）

**outputSchema について:**

各ツールは `outputSchema` で TOON 形式の返却フィールドを宣言する。

```json
{
  "name": "github_list_issues",
  "inputSchema": { ... },
  "outputSchema": {
    "format": "toon",
    "fields": ["number", "title", "state", "user", "html_url"]
  }
}
```

```json
{
  "name": "notion_search",
  "inputSchema": { ... },
  "outputSchema": {
    "format": "toon",
    "fields": ["id", "title", "url", "created_time"]
  }
}
```

LLM はこれを見て `items[N]{number,title,state,user,html_url}:` 形式で返ってくると理解できる。

### 2.2 統一レスポンス形式

すべてのツールは [TOON 形式](https://github.com/toon-format/toon) で結果を返す。

**形式:** `items[件数]{フィールド,...}: 値,値,...`（1行1レコード、ヘッダがスキーマ）

```
items[3]{id,title,status}:
  task1,買い物,notStarted
  task2,掃除,completed
  task3,料理,inProgress
```

- `items` = 固定キー（全ツール共通）
- `[3]` = 件数
- `{id,title,status}` = 各行のフィールド順
- `:` 以降 = CSV 形式のデータ（1行1件）

**エスケープルール（TOON 仕様準拠）:**
- 値にカンマ `,` を含む場合は `"..."` でクォート
- 値にダブルクォートを含む場合は `""` でエスケープ
- 改行を含む値は `\n` でエスケープ

```
items[1]{number,title,state}:
  17863,"[Bug] Temporary files with ""tmpclaude"" pattern",open
```

#### 例

**一覧取得:**
```
items[5]{id,displayName}:
  AQMk...AAAA,Tasks
  AQMk...Kw==,Daily Routine
  AQMk...6g==,Shopping
  AQMk...Lg==,Wishlist
  AQMk...Og==,Flagged Emails
```

**単一取得/作成/更新/削除:**
```
items[1]{id,title,status}:
  task1,買い物,completed
```

**空結果:**
```
items[0]{}:
```

**エラー:**
```
error[1]{code,message}:
  NOT_FOUND,タスクが見つかりません
```

**設計方針:**
- 全ツールで `items` を使用（配列名の統一）
- ヘッダ `[N]` で件数を表現（JSON 比 30-40% トークン削減）
- 外部 API のレスポンスを正規化し、必要なフィールドのみ 2次元で返却

**変数参照と展開:**

LLM は JSON 的な感覚で変数参照を生成:
```
${lists.items[0].id}      → lists の 0番目の id
${tasks.items[1].title}   → tasks の 1番目の title
${lists.items.length}     → lists の件数
```

Go（MCPist）がレスポンス形式をパースして変数を展開する。

#### モジュール別フィールドマッピング

各モジュールは外部 API のネストしたレスポンスをフラット化して返す。
色・装飾・メタデータ等は除外し、操作に必要なフィールドのみ抽出。

| モジュール | リソース | 抽出フィールド |
|-----------|---------|---------------|
| microsoft_todo | list | id, displayName |
| microsoft_todo | task | id, title, status, dueDateTime |
| notion | page | id, title, url, created_time |
| notion | database | id, title, url |
| github | issue | id, number, title, state, html_url |
| github | repo | id, name, full_name, html_url |
| jira | issue | key, summary, status, assignee |
| google_calendar | event | id, summary, start, end |

**変換例（Notion search）:**

外部 API のレスポンス（ネスト深い）:
```json
{"results": [{"id": "2e62cd76...", "properties": {"Name": {"title": [{"plain_text": "タスク"}]}}, ...}]}
```

MCPist が正規化して返却:
```
items[1]{id,title,url,created_time}:
  2e62cd76...,タスク,https://notion.so/...,2026-01-12T20:14:00Z
```

#### Tool Sieve によるレスポンス形式の切り替え

Tool Sieve（§3.2）でユーザーと開発者向けにツールを分離できる。

| 対象 | ツール例 | レスポンス形式 | 用途 |
|------|---------|---------------|------|
| ユーザー | `notion_search` | TOON（正規化済み） | 日常利用、トークン効率 |
| 開発者 | `notion_search_raw` | JSON（生データ） | デバッグ、高度な処理 |

**設定例:**
```yaml
user_tools:
  - notion_search        # TOON
  - notion_get_page      # TOON
  - github_list_issues   # TOON

developer_tools:
  - notion_search_raw    # JSON
  - notion_get_page_raw  # JSON
  - github_api_raw       # JSON
```

**自動分類ルール:**
```yaml
rules:
  - pattern: "*_raw"
    role: developer
    response_format: json
  - pattern: "*"
    role: user
    response_format: toon
```

### 2.3 統一入力形式（JSONL）

`call` と `batch` は同じ JSONL 形式で入力する。1行なら単発実行、複数行なら並列/連鎖実行。

**フィールド:**
- `module` (必須): モジュール名
- `tool` (必須): ツール名
- `params`: パラメータ
- `id`: タスク識別子（単一行でも必須）
- `after`: 依存タスクID配列
- `output`: `true` で結果を LLM に返却

**出力:**
- 1行: §2.2 統一レスポンス形式（TOON）に準拠
- 複数行: JSON でラップし、各 ID の値に TOON を格納

```json
{
  "results": {
    "tasks": "items[3]{id,title,status}:\n  task1,買い物,notStarted\n  task2,掃除,completed\n  task3,料理,inProgress",
    "daily": "items[0]{}:"
  },
  "errors": {}
}
```

**変数参照:** `${id.items[index].field}` 形式で前タスクの出力参照

### 2.4 入力パターン

#### パターン1: 単発取得（1行）

```jsonl
{"id": "task", "module":"microsoft_todo","tool":"mstodo_list_lists","params":{}}
```

#### パターン2: 特定リソース取得（1行）

```jsonl
{"id": "task", "module":"microsoft_todo","tool":"mstodo_list_tasks","params":{"listId":"AQMkADAwATM3ZmYAZS0zNjNl..."}}
```

#### パターン3: 並列取得（複数行、依存なし）

全リストのタスクを同時に取得:

```jsonl
{"id":"tasks","module":"microsoft_todo","tool":"mstodo_list_tasks","params":{"listId":"AQMk...AAAA"},"output":true}
{"id":"daily","module":"microsoft_todo","tool":"mstodo_list_tasks","params":{"listId":"AQMk...Kw=="},"output":true}
{"id":"shopping","module":"microsoft_todo","tool":"mstodo_list_tasks","params":
```

`after` がないため3つを並列実行。

#### パターン4: 連鎖処理（複数行、依存あり）

GitHub Issue を Notion/Confluence に転記:

```jsonl
{"id":"issues","module":"github","tool":"github_list_issues","params":{"owner":"org","repo":"app","state":"open"},"output":true}
{"id":"notion","module":"notion","tool":"notion_create_page","params":{"parent_page_id":"abc","title":"${issues.items[0].title}","body":"${issues.items[0].body}"},"after":["issues"]}
{"id":"confluence","module":"confluence","tool":"confluence_create_page","params":{"spaceId":"PROJ","parentId":"456","title":"${issues.items[1].title}","body":"${issues.items[1].body}"},"after":["issues"]}
```

実行順序:
```
issues ─┬→ notion
        └→ confluence  (並列実行可能)
```

#### パターン5: 動的件数（2往復）

事前に件数が不明な場合は2往復で処理。

**1回目:** タスク一覧を取得
```jsonl
{"id": "task", "module":"microsoft_todo","tool":"mstodo_list_tasks","params":{"listId":"AQMk...AAAA"}}
```

→ 結果:
```
items[3]{id,title,status}:
  task1,買い物,notStarted
  task2,掃除,inProgress
  task3,料理,notStarted
```

**2回目:** LLM が件数（3件）を見て生成
```jsonl
{"id":"t1","module":"microsoft_todo","tool":"mstodo_complete_task","params":{"listId":"AQMk...AAAA","taskId":"task1"}}
{"id":"t2","module":"microsoft_todo","tool":"mstodo_complete_task","params":{"listId":"AQMk...AAAA","taskId":"task2"}}
{"id":"t3","module":"microsoft_todo","tool":"mstodo_complete_task","params":{"listId":"AQMk...AAAA","taskId":"task3"}}
```

LLM が結果を見て次のアクションを決める MCP の思想に沿った設計。

### 2.5 パターン選択ガイド

| ケース | パターン | 行数 |
|--------|----------|------|
| 単発取得/操作 | 1, 2 | 1行 |
| 複数サービス並列取得 | 3 | N行（依存なし） |
| 前の結果を使う連鎖処理 | 4 | N行（依存あり） |
| 動的件数のループ処理 | 5 | 2往復 |

`output: true` を指定したタスクのみ結果が LLM に返却される（トークン消費削減）。

### 2.6 実行エンジン（Go 実装方針）

**実行ルール:**
- `after` なし → 即座に goroutine で並列実行
- `after` あり → 依存タスク完了を待ってから実行

**実装:**
- 結果共有: `sync.Map`
- 待機: 依存タスクの結果が `sync.Map` に入るまでループ or channel
- 変数解決: 実行直前に `${id.items[N].field}` を正規表現で置換、TOON をパースして値取得

**エラー時:**
- 循環依存 → パース時に検出、即失敗
- 依存タスク失敗 → 依存先もスキップ、errors に記録

**ページネーション:**

外部 API のページネーションはモジュール実装側の責務。LLM は意識しない。

```go
// モジュール実装例
func (m *GitHubModule) ListIssues(ctx context.Context, params Params) (string, error) {
    var allIssues []Issue
    page := 1
    for {
        issues, hasMore := m.client.ListIssues(params.Owner, params.Repo, page)
        allIssues = append(allIssues, issues...)
        if !hasMore || len(allIssues) >= 500 { // 上限
            break
        }
        page++
    }
    return toTOON(allIssues), nil
}
```

- 各モジュールが全件取得（上限まで）してから TOON で返却
- デフォルト上限: 500件（モジュールごとに設定可能）

**レートリミット:**

外部 API のレートリミット制御はモジュール実装側の責務。各モジュールがサービスごとの制限に応じたレートリミッターを持ち、バーストを防ぐ。

---

## 3. データモデル

データは責務別に3つの保存先に分類される（§5.3 データ分類と暗号化要件 参照）。

### 3.1 Authサーバー（Supabase Auth）

ユーザー認証情報。Supabase Authが管理するため、アプリケーション側でのテーブル定義は不要。

| 管理項目 | 説明 |
|---------|------|
| ユーザーID | UUID形式の一意識別子 |
| メールアドレス | 認証用メールアドレス |
| 認証プロバイダ | Google, GitHub等のOAuthプロバイダ |
| セッション情報 | JWT発行・検証に使用 |

### 3.2 Tool Sieve（Supabase DB）

ユーザー管理とツール権限設定。暗号化不要（設定情報のみで、漏洩時の影響が限定的）。

#### tenants テーブル

組織/テナント情報。Phase 1はシングルテナントだが、将来のマルチテナント対応を見据えて設計。

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | テナントID（PK） |
| name | VARCHAR | テナント名（組織名） |
| created_at | TIMESTAMP | 作成日時 |

#### users テーブル

MCPistが管理するユーザー。Supabase Authの1アカウント=1ユーザー制約を論理的にグループ化。

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | ユーザーID（PK） |
| tenant_id | UUID | テナントID（FK） |
| system_role | VARCHAR | システムロール（admin / user） |
| display_name | VARCHAR | 表示名 |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

#### auth_accounts テーブル

ソーシャルログインアカウントとMCPistユーザーの紐付け。1ユーザーに複数のソーシャルアカウントを関連付け可能。

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | 認証アカウントID（PK） |
| user_id | UUID | ユーザーID（FK → users） |
| supabase_auth_id | UUID | Supabase AuthのユーザーID |
| provider | VARCHAR | プロバイダ（google, github等） |
| created_at | TIMESTAMP | 作成日時 |

#### roles テーブル

ツール権限パターンの定義。管理者が作成し、ユーザーに割り当てる。

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | ロールID（PK） |
| tenant_id | UUID | テナントID（FK） |
| name | VARCHAR | ロール名（営業閲覧者、開発者フル等） |
| description | TEXT | 説明 |
| created_by | UUID | 作成者（admin）のユーザーID（FK） |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

#### user_roles テーブル

ユーザーとロールの関連。管理者がユーザーにロールを割り当て。

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | 割り当てID（PK） |
| user_id | UUID | ユーザーID（FK） |
| role_id | UUID | ロールID（FK） |
| created_at | TIMESTAMP | 作成日時 |

**UNIQUE制約**: (user_id, role_id)

#### role_permissions テーブル

ロール別のツール権限設定。

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | 権限ID（PK） |
| role_id | UUID | ロールID（FK） |
| enabled_modules | TEXT[] | 有効モジュール一覧 |
| tool_masks | JSONB | ツール別の有効/無効設定 |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

#### module_registry テーブル

モジュールの状態管理。管理者がモジュールの有効/無効、APIバージョン情報を管理。

| カラム | 型 | 説明 |
|--------|-----|------|
| module_name | VARCHAR | モジュール名（PK） |
| display_name | VARCHAR | 表示名 |
| description | TEXT | 説明 |
| api_version | VARCHAR | 対応APIバージョン |
| status | VARCHAR | stable / deprecated / broken |
| last_verified_at | TIMESTAMP | 最終動作確認日時 |
| notes | TEXT | 管理者メモ（障害情報等） |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

**status の意味:**
- `stable`: 正常動作（ツール一覧に表示）
- `deprecated`: 非推奨（警告付きで表示）
- `broken`: 動作不可（ツール一覧から除外）

**運用フロー:**
1. 新モジュール追加時: 管理者がレジストリに登録（status=stable）
2. API変更検知時: notes に情報記載、必要に応じて status 変更
3. 修正完了時: status を stable に戻し、api_version を更新

**バージョン監視（spec-inf §9.2 参照）:**
- version-notify.yml（週1実行）が公開APIからバージョン取得
- `api_version` と差分があればGitHub Issue自動作成
- 管理者がIssue確認後、手動で `api_version` と `last_verified_at` を更新

### 3.3 Token Broker（Supabase Vault）

OAuthトークン。暗号化必須（外部サービスへのアクセスキー）。

#### oauth_tokens テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | トークンID（PK） |
| role_id | UUID | ロールID（FK）**必須** |
| user_id | UUID | ユーザーID（FK）**NULL可** |
| service | VARCHAR | サービス名（notion, github等） |
| access_token | TEXT | アクセストークン（**AES-256-GCM暗号化**） |
| refresh_token | TEXT | リフレッシュトークン（**AES-256-GCM暗号化**） |
| expires_at | TIMESTAMP | 有効期限 |
| scopes | TEXT[] | 許可スコープ |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

**UNIQUE制約**: (role_id, user_id, service) ※user_idはNULL許容

**トークン所有タイプ:**
- `user_id = NULL` → ロール共有トークン（adminが設定、ロール内全員が使用）
- `user_id = 値あり` → 個人トークン（userが自分のアカウントで認可）

**Token Broker解決順序:**
1. user_id + role_id + service で個人トークンを検索
2. なければ role_id + service で共有トークンを検索
3. どちらもなければエラー（要連携）

### 3.4 ER図

```
Supabase Auth                    Supabase DB (Tool Sieve)
┌──────────────┐                 ┌──────────────┐
│   auth.users │                 │   tenants    │
│  (Auth管理)   │                 └──────┬───────┘
└──────┬───────┘                        │ 1:N
       │                                ▼
       │ supabase_auth_id        ┌──────────────┐      ┌──────────────┐
       │                         │    users     │      │    roles     │
       ▼                         │(system_role) │      │ (権限パターン) │
┌──────────────┐                 └──────┬───────┘      └──────┬───────┘
│auth_accounts │◀──  user_id  ──────────┘                     │
└──────────────┘                        │                     │
                                        │    ┌────────────────┘
                                        ▼    ▼
                                 ┌──────────────────┐
                                 │   user_roles     │
                                 └────────┬─────────┘
                                          │
                                          ▼
                                 ┌──────────────────┐
                                 │ role_permissions │
                                 │   (ツール権限)    │
                                 └────────┬─────────┘
                                          │
                                          ▼
                          Supabase Vault (Token Broker)
                                 ┌──────────────────┐
                                 │   oauth_tokens   │
                                 │ (role_id必須)    │
                                 │ (user_id任意)    │
                                 │    (暗号化)      │
                                 └──────────────────┘

トークン解決:
  個人トークン優先 → 共有トークンにフォールバック
  (user_id+role_id+service) → (role_id+service)
```

### 3.5 アクセス制御モデル

| 操作 | admin | user |
|------|-------|------|
| テナント設定 | ○ | × |
| ユーザー管理 | ○ | × |
| ロール作成 | ○ | × |
| ロールへのユーザー割当 | ○ | × |
| ロールの権限設定 | ○ | × |
| OAuthアプリ設定（Client ID/Secret） | ○ | × |
| 共有トークン設定（タイプA/B） | ○ | × |
| 個人アカウント連携（OAuth認可） | ○ | ○（自分のアカウントのみ） |
| ツール一覧閲覧 | ○ | ○（自分の権限内のみ） |
| サービス連携状況確認 | ○ | ○（自分の連携状況のみ） |
| ツール実行（LLMクライアント経由） | ○ | ○（割り当てロールの権限内） |

### 3.6 検証要件

**ツール一覧の整合性検証:**

管理画面に表示されるツール一覧と、LLMクライアントが呼び出せるツール一覧は完全に一致する必要がある。

| 検証項目 | 説明 |
|---------|------|
| `/api/profile/tools` の結果 | ユーザーの全ロールから集約した利用可能ツール一覧 |
| `get_module_schema` の結果 | Tool Sieveがフィルタリングした利用可能ツール一覧 |
| **整合性** | 上記2つの結果は完全一致すること |

**テスト要件:**
- 同一ユーザーに対して、管理UI API（`/api/profile/tools`）とMCP API（`get_module_schema`）の結果を比較
- 差分がある場合はテスト失敗
- ロール権限変更時に両方が即座に反映されることを確認

---

## 4. エラーハンドリング

### 4.1 エラーコード体系

| コード範囲 | カテゴリ |
|-----------|---------|
| 1000-1999 | 認証エラー |
| 2000-2999 | 入力検証エラー |
| 3000-3999 | 外部APIエラー |
| 4000-4999 | 内部エラー |

### 4.2 エラーコード一覧

| コード | 名前 | 説明 |
|--------|------|------|
| 1001 | INVALID_JWT | JWTが無効 |
| 1002 | JWT_EXPIRED | JWTが期限切れ |
| 1003 | UNAUTHORIZED | 認証が必要 |
| 2001 | INVALID_MODULE | 存在しないモジュール |
| 2002 | INVALID_TOOL | 存在しないツール |
| 2003 | INVALID_PARAMS | パラメータ不正 |
| 3001 | EXTERNAL_API_ERROR | 外部APIエラー |
| 3002 | TOKEN_REFRESH_FAILED | トークンリフレッシュ失敗 |
| 3003 | RATE_LIMITED | レート制限 |
| 4001 | INTERNAL_ERROR | 内部エラー |
| 4002 | TIMEOUT | タイムアウト |

### 4.3 エラーレスポンス形式

JSON-RPC 2.0形式: `{"jsonrpc": "2.0", "id": N, "error": {"code": N, "message": "...", "data": {...}}}`

### 4.4 リトライ戦略

| エラー種別 | リトライ | 戦略 |
|-----------|---------|------|
| 認証エラー | × | 即座にエラー返却 |
| レート制限 | ○ | Exponential backoff |
| 一時的エラー | ○ | 最大3回、1秒間隔 |
| 永続的エラー | × | 即座にエラー返却 |

---

## 5. セキュリティ設計

### 5.1 暗号化仕様

| 項目 | 仕様 |
|------|------|
| アルゴリズム | AES-256-GCM |
| キー長 | 256bit |
| IV長 | 96bit |
| 認証タグ長 | 128bit |

### 5.2 トークン暗号化

Supabase Vaultが暗号化・復号化を内部で処理。アプリケーション側での実装は不要。

- 保存時: plaintext → Vault（AES-256-GCM暗号化） → DB
- 取得時: DB → Vault（復号化） → plaintext

### 5.3 データ分類と暗号化要件

MCPistのデータは機密性に応じて暗号化要件が異なる。

| コンポーネント | 管理対象 | 保存先 | 暗号化 | 理由 |
|--------------|---------|--------|-------|------|
| Authサーバー | ユーザー認証情報 | Supabase Auth | Supabase管理 | 認証基盤が責任を持つ |
| Tool Sieve | tenants, users, auth_accounts, roles, user_roles, role_permissions | Supabase DB | **不要** | 設定情報のみ、漏洩時の影響が限定的 |
| Token Broker | oauth_tokens（アクセス/リフレッシュトークン） | Supabase Vault | **必要**（AES-256-GCM） | 外部サービスへのアクセスキー |

**暗号化判断の根拠:**

- **Tool Sieve各テーブル**: テナント情報、ユーザー情報、ロール設定、ツール権限設定等。漏洩しても「誰が何を使えるか」が分かるだけで、外部サービスへの不正アクセスには繋がらない
- **oauth_tokens**: 外部サービス（Notion, GitHub等）へのアクセスキー。漏洩時は外部サービスへの不正アクセスが可能となるため、暗号化が必須

### 5.4 危険操作フラグ

以下の操作は`dangerous: true`としてマーク:

| モジュール | ツール | 理由 |
|-----------|--------|------|
| Notion | delete_page | データ削除 |
| GitHub | delete_repository | データ削除 |
| GitHub | merge_pull_request | 変更の確定 |
| Jira | delete_issue | データ削除 |
| Supabase | execute_sql | 任意SQL実行 |

---

## 6. 管理UI設計

### 6.1 URL設計方針

**想定ユースケース:**
- admin = 社内情報システム担当者
- user = 社員（LLMクライアント経由でツールを利用）
- adminが外部サービス（Google, Microsoft等）のOAuthアプリを作成・設定し、社員がLLM経由で操作できるようにする

**RESTful原則に従った設計:**
- 同一リソースには単一のURLを割り当て（REST原則）
- 権限による表示制御はURLではなく、ミドルウェア＋画面内で実現
- 近年のSPA/MPAフレームワーク（Next.js App Router, Remix等）の慣習に準拠

**責務分離:**
- OAuthアプリ作成・Client ID/Secret登録 → admin
- 共有トークン設定（タイプA/B） → admin
- ロール・権限管理 → admin
- 個人アカウント連携（OAuth認可） → user（adminが設定したOAuthアプリを使用）
- ツール利用 → user（LLMクライアント経由）

**MCPist への認証方式:**

| ユーザー種別 | 認証方式 | MFA | 備考 |
|-------------|---------|-----|------|
| admin | ソーシャルログイン | 任意（TOTP） | Google / Microsoft / GitHub |
| user | ソーシャルログイン | - | Google / Microsoft / GitHub |
| テストアカウント | メール + パスワード | なし | テスト環境のみ |

**ソーシャルログイン統一の理由:**
- Google Workspace / Microsoft 365 のセキュリティ機能に委ねる方が堅牢
- パスワード漏洩リスクを排除
- フィッシング耐性が高い（WebAuthn/Passkey 対応）
- パスワード管理の運用負荷を削減

**admin の追加セキュリティ（任意）:**
- TOTP を追加設定可能（Supabase MFA）
- 設定画面: `/profile` → 「二段階認証を有効化」

**ログイン画面の環境別表示:**

| 環境 | ソーシャルログイン | メール + パスワード |
|------|------------------|-------------------|
| 本番 | ✅ 表示 | ❌ 非表示 |
| テスト | ✅ 表示 | ✅ 表示 |

```typescript
// 環境変数で制御
const showPasswordLogin = process.env.NEXT_PUBLIC_SHOW_PASSWORD_LOGIN === 'true';
```

**外部サービス連携時の認証方式の優先順位:**
外部サービス連携時は、以下の優先順位で認証方式を選択する。
1. **OIDC（OpenID Connect）** - 対応サービス: Google, Microsoft, Slack等
2. **OAuth 2.0** - OIDC非対応サービス: GitHub, Notion, Jira, Discord等

**OIDC優先の理由:**
- OIDCはOAuth 2.0の上位互換であり、ID Tokenによるユーザー識別が標準化されている
- **放置運用の実現**: OIDCはトークンリフレッシュが標準化されており、管理者の手動介入なしで長期運用が可能
- OAuth 2.0のみのサービスはリフレッシュトークンの仕様がサービスごとに異なり、定期的なメンテナンスが必要になる場合がある

**adminの役割:**
- 初回セットアップ時: 最初にログインした許可リストユーザーがadminとして設定される
- 追加admin: 既存adminが他ユーザーの`system_role`を`admin`に変更可能
- adminは複数人存在でき、全員が同等の管理権限を持つ

### 6.2 画面遷移（SPA最適化設計）

**設計方針:**
- トップレベルは独立したページ（フルページ遷移）
- 詳細・編集はモーダル/パネルで開く（SPAの状態管理を簡略化）
- URL queryパラメータで選択状態を表現（ブラウザバック対応）
- 深いネストを避け、コンテキスト切り替えを明確にする

```
# 基本ページ（フルページ）
/                                        # ダッシュボード
/login                                   # ログイン
/profile                                 # 自分のプロファイル
/tools                                   # ツール一覧（全員）
/users                                   # ユーザー管理（adminのみ）
/roles                                   # ロール管理（adminのみ）
/logs                                    # 監査ログ（adminのみ）

# モーダル/パネル（query parameterで表現）
/tools?connect=notion                    # 個人アカウント連携モーダル
/users?id=xxx                            # ユーザー詳細パネル
/users?id=xxx&tab=roles                  # ユーザーのロール割当タブ
/roles?id=xxx                            # ロール詳細パネル
/roles?id=xxx&tab=permissions            # 権限設定タブ
/roles?id=xxx&tab=services               # サービス一覧タブ
/roles?id=xxx&service=notion             # サービス設定モーダル
/roles?new=true                          # ロール新規作成モーダル
```

**URLパターン:**

| パターン | 用途 | 例 |
|---------|------|-----|
| `/page` | ベースページ | `/users`, `/roles` |
| `/page?id=xxx` | 詳細パネル表示 | `/users?id=abc123` |
| `/page?id=xxx&tab=xxx` | 詳細内のタブ切替 | `/roles?id=abc&tab=permissions` |
| `/page?action=xxx` | アクション系モーダル | `/tools?connect=notion` |

**利点:**
- ブラウザの戻る/進むでモーダル状態を復元可能
- React/Next.jsの状態管理がシンプル（URL単一ソース）
- 深いルートネストによる複雑さを回避
- サーバーコンポーネントとクライアントコンポーネントの境界が明確

### 6.3 権限による表示制御

| 画面 | admin | user |
|------|-------|------|
| `/` | 全体概要・管理メニュー | 割り当てロール一覧 |
| `/profile` | 自分のプロファイル | 自分のプロファイル |
| `/tools` | ツール一覧+連携状況 | ツール一覧+連携状況（UX仕様参照） |
| `/tools?connect=xxx` | 個人アカウント連携 | 個人アカウント連携 |
| `/users` | 全操作可 | 403 |
| `/roles` | 全操作可 | 403 |
| `/logs` | 全ログ閲覧 | 403 |

**userの画面:**
- ダッシュボード: 自分に割り当てられたロールの確認
- プロファイル: 表示名等の編集
- ツール一覧: 使えるツールと連携状況（共有/個人/未連携）の確認
- 個人アカウント連携: adminが設定したOAuthアプリを使って自分のアカウントで認可
- 実際のツール利用はLLMクライアント（Claude Code, Cursor等）経由

### 6.4 サービス設定画面（admin向け）

管理者が外部サービスの認証情報を設定する統一画面。

**画面構成:**
```
┌─────────────────────────────────────────────────┐
│ サービス設定                                      │
├─────────────────────────────────────────────────┤
│ プロバイダ: [Google Calendar ▼]                  │
│                                                 │
│ 認証方式:  ● OIDC (推奨)                        │
│            ○ OAuth 2.0                          │
│            ○ APIキー                            │
│                                                 │
│ ───────────────────────────────────────────── │
│ Client ID:     [________________________]       │
│ Client Secret: [________________________]       │
│                                                 │
│ [テスト接続]  [保存]                             │
└─────────────────────────────────────────────────┘
```

**プロバイダ別の対応認証方式:**

| プロバイダ | OIDC | OAuth 2.0 | APIキー | 推奨 |
|-----------|------|-----------|---------|------|
| Google (Calendar等) | ✅ | ✅ | - | OIDC |
| Microsoft (Todo, Graph) | ✅ | ✅ | - | OIDC |
| Slack | ✅ | ✅ | - | OIDC |
| GitHub | - | ✅ | ✅ | OAuth 2.0 |
| Notion | - | ✅ | - | OAuth 2.0 |
| Jira | - | ✅ | ✅ | OAuth 2.0 |
| Confluence | - | ✅ | ✅ | OAuth 2.0 |
| Discord | - | ✅ | ✅ | OAuth 2.0 |
| Supabase | - | - | ✅ | APIキー |

**動的フォーム:**
- プロバイダ選択時、対応する認証方式のみ選択可能
- OIDC対応サービスは「(推奨)」ラベル付きで優先表示
- 認証方式に応じて入力項目が動的に変化:
  - OIDC/OAuth 2.0: Client ID, Client Secret
  - APIキー: API Key

**保存先:** Supabase Vault（暗号化保存）

### 6.5 ツール一覧のUX仕様（user向け）

userの `/tools` 画面では、権限の有無によってツールの表示方法を変える。

**表示状態:**

| 状態 | 表示 | 操作 |
|------|------|------|
| 利用可能（連携済み） | 通常表示、緑バッジ | 連携解除ボタン |
| 利用可能（未連携） | 通常表示、黄バッジ | 「連携する」ボタン |
| 利用不可（権限なし） | **折りたたみ表示** | 展開可能 |

**利用不可ツールの表示仕様:**

```
┌─────────────────────────────────────────────────────┐
│ ▼ 利用できないツール（12件）                          │ ← デフォルトは折りたたみ
├─────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────┐ │
│ │ [グレーアウト] Notion                           │ │
│ │  ・search_pages                                │ │
│ │  ・create_page                                 │ │
│ │  ・...                                         │ │
│ │                      [利用を申請] ボタン        │ │
│ └─────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────┐ │
│ │ [グレーアウト] GitHub                           │ │
│ │  ・create_issue                                │ │
│ │  ・...                                         │ │
│ │                      [利用を申請] ボタン        │ │
│ └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**UXポイント:**
- **デフォルト折りたたみ**: 利用できないツールは邪魔にならないよう折りたたむ
- **グレーアウト表示**: 展開時は使えないことが視覚的に明確
- **「利用を申請」ボタン**: adminへの申請フローを提供（Phase 1はメール通知等）
- **ツール名の列挙**: どのツールが含まれるか確認可能

**「利用を申請」ボタンの動作（Phase 1）:**
1. クリックでモーダル表示
2. 申請理由を入力（任意）
3. 送信 → adminにメール通知
4. admin側で `/users?id=xxx&tab=roles` からロール割当

**将来拡張（Phase 2以降）:**
- 申請履歴の管理画面
- adminの承認ワークフロー
- 申請状態の表示（申請中/承認済み/却下）

### 6.6 API エンドポイント（管理UI用）

#### 認証

| メソッド | パス | 説明 | 権限 |
|---------|------|------|------|
| POST | `/api/auth/login` | ログイン | all |
| POST | `/api/auth/logout` | ログアウト | all |
| GET | `/api/auth/me` | 現在のユーザー情報 | all |

#### プロファイル

| メソッド | パス | 説明 | 権限 |
|---------|------|------|------|
| GET | `/api/profile` | 自分のプロファイル | all |
| PUT | `/api/profile` | プロファイル更新 | all |
| GET | `/api/profile/roles` | 自分のロール一覧 | all |
| GET | `/api/profile/tools` | 自分が使えるツール一覧（連携状況含む） | all |
| GET | `/api/profile/services` | 自分のサービス連携状況一覧 | all |
| GET | `/api/profile/services/:service` | 特定サービスの連携状況 | all |
| POST | `/api/profile/services/:service/oauth/start` | 個人アカウントOAuth認可開始 | all |
| DELETE | `/api/profile/services/:service/token` | 個人トークン削除 | all |
| POST | `/api/profile/tools/request` | ツール利用申請（adminへ通知） | all |

#### ユーザー管理

| メソッド | パス | 説明 | 権限 |
|---------|------|------|------|
| GET | `/api/users` | ユーザー一覧 | admin |
| POST | `/api/users` | ユーザー作成 | admin |
| GET | `/api/users/:id` | ユーザー詳細 | admin |
| PUT | `/api/users/:id` | ユーザー更新 | admin |
| DELETE | `/api/users/:id` | ユーザー削除 | admin |
| GET | `/api/users/:id/roles` | ユーザーのロール一覧 | admin |
| POST | `/api/users/:id/roles` | ユーザーにロール割当 | admin |
| DELETE | `/api/users/:id/roles/:roleId` | ユーザーからロール削除 | admin |

#### ロール管理

| メソッド | パス | 説明 | 権限 |
|---------|------|------|------|
| GET | `/api/roles` | ロール一覧 | admin |
| POST | `/api/roles` | ロール作成 | admin |
| GET | `/api/roles/:id` | ロール詳細 | admin |
| PUT | `/api/roles/:id` | ロール更新 | admin |
| DELETE | `/api/roles/:id` | ロール削除 | admin |
| GET | `/api/roles/:id/permissions` | 権限取得 | admin |
| PUT | `/api/roles/:id/permissions` | 権限更新 | admin |

#### サービス・トークン管理

| メソッド   | パス                                             | 説明                                | 権限    |
| ------ | ---------------------------------------------- | --------------------------------- | ----- |
| GET    | `/api/roles/:id/services`                      | サービス一覧（設定状況）                      | admin |
| GET    | `/api/roles/:id/services/:service`             | サービス設定取得                          | admin |
| PUT    | `/api/roles/:id/services/:service`             | 設定更新（Client ID/Secret or APIトークン） | admin |
| POST   | `/api/roles/:id/services/:service/oauth/start` | OAuth認可開始（タイプB）                   | admin |
| DELETE | `/api/roles/:id/services/:service/token`       | トークン削除                            | admin |
| GET    | `/api/oauth/:service/callback`                 | OAuthコールバック（固定パス）                 | -     |

**備考:**
- タイプA（長期トークン型）: `PUT /api/roles/:id/services/:service` でAPIトークンを直接登録（共有トークン）
- タイプB（OAuth型）: `PUT` でClient ID/Secret登録後、`POST .../oauth/start` で認可フロー実行（共有トークン）
- 個人トークン: userが `/api/profile/services/:service/oauth/start` で自分のアカウントで認可
- OAuthコールバック: 共有/個人の両方で `/api/oauth/:service/callback` を使用（stateパラメータで判別）

#### 監査ログ

| メソッド | パス | 説明 | 権限 |
|---------|------|------|------|
| GET | `/api/logs` | 監査ログ取得 | admin |

---

## 7. モジュール設計

### 7.1 モジュールインターフェース

各モジュールは以下を実装:
- `Name()`, `Description()`, `APIVersion()`: メタ情報
- `Tools()`: ツール定義一覧
- `Execute(ctx, tool, params)`: ツール実行

### 7.2 モジュール登録

8モジュールをレジストリに登録: notion, github, jira, confluence, supabase, google_calendar, microsoft_todo, rag

---

## 8. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [インフラ仕様書](spec-inf.md) | インフラ構成 |
| [運用仕様書](spec-ops.md) | 運用設計 |
